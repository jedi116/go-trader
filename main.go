package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jedi116/go-trader/internal/ai"
	"github.com/jedi116/go-trader/internal/api"
	"github.com/jedi116/go-trader/internal/broker"
	"github.com/jedi116/go-trader/internal/config"
	"github.com/jedi116/go-trader/internal/database"
	"github.com/jedi116/go-trader/internal/news"
	"github.com/jedi116/go-trader/pkg/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	oandaAPIKey := os.Getenv("OANDA_API_KEY")
	oandaAccountID := os.Getenv("OANDA_ACCOUNT_ID")
	braveAPIKey := os.Getenv("BRAVE_API_KEY")
	braveBaseURL := cfg.Brave.BaseURL
	// isLive := os.Getenv("OANDA_ENV") == "live"

	oandaMT4Client := broker.NewOandaMT4Client(oandaAPIKey, oandaAccountID, false)
	braveClient := news.NewBraveClient(braveAPIKey, braveBaseURL)

	// Initialize database if configured
	pg, err := database.NewPostgres(cfg)
	if err != nil {
		log.Printf("database init failed: %v (continuing without DB)", err)
		pg = nil
	}

	// Wire AI service with real market/news aggregation and logging
	agg := ai.NewAggregator(
		func(ctx context.Context, instruments []string) (*ai.MarketContext, error) {
			start := time.Now()
			log.Printf("[AI] Gathering market data for instruments=%v granularity=M5 count=50", instruments)
			marketInfo := map[string]interface{}{"list": instruments}
			for _, inst := range instruments {
				candles, err := oandaMT4Client.GetCandles(inst, "M5", 50, nil, nil)
				if err != nil {
					log.Printf("[AI] GetCandles error instrument=%s: %v", inst, err)
					continue
				}
				if pg != nil && candles != nil {
					rows := make([]models.MarketData, 0, len(candles.Candles))
					for _, cdl := range candles.Candles {
						rows = append(rows, models.MarketData{
							ID:         "",
							Instrument: candles.Instrument,
							Timestamp:  cdl.Time,
							OpenPrice:  parseFloat(cdl.Mid.Open),
							HighPrice:  parseFloat(cdl.Mid.High),
							LowPrice:   parseFloat(cdl.Mid.Low),
							ClosePrice: parseFloat(cdl.Mid.Close),
							Volume:     nil,
							Timeframe:  candles.Granularity,
						})
					}
					if err := pg.UpsertMarketData(ctx, rows); err != nil {
						log.Printf("[AI] UpsertMarketData error instrument=%s: %v", inst, err)
					} else {
						log.Printf("[AI] UpsertMarketData ok instrument=%s rows=%d", inst, len(rows))
					}
				}
				lastClose := 0.0
				if n := len(candles.Candles); n > 0 {
					lastClose = parseFloat(candles.Candles[n-1].Mid.Close)
				}
				marketInfo[inst] = map[string]interface{}{
					"granularity": candles.Granularity,
					"last_close":  lastClose,
					"count":       len(candles.Candles),
				}
			}
			log.Printf("[AI] Market data gathered in %s", time.Since(start))
			return &ai.MarketContext{Instruments: marketInfo}, nil
		},
		func(ctx context.Context, instruments []string) ([]ai.NewsItem, error) {
			start := time.Now()
			query := instruments[0] + " forex"
			log.Printf("[AI] Fetching news via Brave query=%q", query)
			items, err := braveClient.SearchNews(ctx, query, 5)
			if err != nil {
				return nil, err
			}
			out := make([]ai.NewsItem, 0, len(items))
			for _, it := range items {
				out = append(out, ai.NewsItem{Title: it.Title, Url: it.Url, Snippet: it.Snippet, Source: it.Source, Published: it.Published})
			}
			log.Printf("[AI] News fetched count=%d in %s", len(out), time.Since(start))
			return out, nil
		},
		func(ctx context.Context, instruments []string) (*ai.HistoricalContext, error) {
			return &ai.HistoricalContext{Notes: "pending"}, nil
		},
	)
	claude := ai.NewClaudeClient(http.DefaultClient)
	aiSvc := ai.NewService(agg, claude)

	server := api.NewServer(cfg, oandaMT4Client, braveClient, pg, aiSvc)
	if err := server.Run(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// parseFloat converts a numeric string to float64, returning 0 on error.
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
