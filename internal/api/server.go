package api

import (
	"encoding/json"
	"log"
	"math"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jedi116/go-trader/internal/ai"
	"github.com/jedi116/go-trader/internal/broker"
	"github.com/jedi116/go-trader/internal/config"
	"github.com/jedi116/go-trader/internal/database"
	"github.com/jedi116/go-trader/internal/news"
	"github.com/jedi116/go-trader/pkg/models"
)

type Server struct {
	config    *config.Config
	router    *gin.Engine
	mt4Client *broker.OandaMT4Client
	brave     *news.BraveClient
	db        *database.Postgres
	ai        ai.Service
}

func NewServer(cfg *config.Config, mt4Client *broker.OandaMT4Client, brave *news.BraveClient, db *database.Postgres, aiSvc ai.Service) *Server {
	router := gin.Default()

	// CORS middleware
	router.Use(cors.Default())

	server := &Server{
		config:    cfg,
		router:    router,
		mt4Client: mt4Client,
		brave:     brave,
		db:        db,
		ai:        aiSvc,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		api.GET("/health", s.healthCheck)
		api.GET("/health/db", s.dbHealth)
		api.GET("/market/:symbol", s.getMarketData)
		api.POST("/orders", s.placeOrder)
		api.GET("/positions", s.getPositions)
		api.GET("/trades", s.listTrades)
		api.DELETE("/trades/:id", s.deleteTrade)
		api.GET("/news/:query", s.searchNews)
		api.POST("/recommendations", s.createRecommendation)
		api.GET("/recommendations", s.listRecommendations)
		api.POST("/recommendations/:id/accept", s.acceptRecommendation)
		api.DELETE("/recommendations/:id", s.deleteRecommendation)
		// AI endpoints
		api.POST("/ai/recommend", s.aiGenerateRecommendation)
		api.GET("/ai/status", s.aiStatus)
	}
}

func (s *Server) Run() error {
	return s.router.Run(s.config.Server.Host + ":" + s.config.Server.Port)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

func (s *Server) dbHealth(c *gin.Context) {
	if s.db == nil {
		c.JSON(503, gin.H{"status": "db not configured"})
		return
	}
	if err := s.db.Health(c.Request.Context()); err != nil {
		c.JSON(503, gin.H{"status": "db error", "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "db ok"})
}

// Placeholder handlers
func (s *Server) getMarketData(c *gin.Context) {
	symbol := c.Param("symbol")
	// fetch candles and return latest price; also persist snapshot to DB if configured
	candles, err := s.mt4Client.GetCandles(symbol, "M5", 50, nil, nil)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if s.db != nil && candles != nil {
		// map to market_data upsert
		rows := make([]models.MarketData, 0, len(candles.Candles))
		for _, cdl := range candles.Candles {
			// time comes as RFC3339 from OANDA client type
			ts := cdl.Time
			// we don't have volume in Mid; set nil
			rows = append(rows, models.MarketData{
				ID:         "", // DB default UUID via UNIQUE on (instrument,timestamp,timeframe) handles conflict
				Instrument: candles.Instrument,
				Timestamp:  ts,
				OpenPrice:  parseDecimal(cdl.Mid.Open),
				HighPrice:  parseDecimal(cdl.Mid.High),
				LowPrice:   parseDecimal(cdl.Mid.Low),
				ClosePrice: parseDecimal(cdl.Mid.Close),
				Volume:     nil,
				Timeframe:  candles.Granularity,
			})
		}
		_ = s.db.UpsertMarketData(c.Request.Context(), rows)
	}
	c.JSON(200, candles)
}

func (s *Server) placeOrder(c *gin.Context) {
	var req struct {
		Instrument string   `json:"instrument"`
		Units      float64  `json:"units"`
		StopLoss   *float64 `json:"stop_loss,omitempty"`
		TakeProfit *float64 `json:"take_profit,omitempty"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	var resp *broker.OrderCreateResponse
	var err error
	if req.StopLoss != nil || req.TakeProfit != nil {
		resp, err = s.mt4Client.PlaceMarketOrderWithBrackets(req.Instrument, req.Units, req.StopLoss, req.TakeProfit)
	} else {
		resp, err = s.mt4Client.PlaceMarketOrder(req.Instrument, req.Units)
	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if s.db != nil && resp != nil {
		// Compute entry price from current mid
		entry := 0.0
		if prices, perr := s.mt4Client.GetPrices([]string{req.Instrument}); perr == nil && len(prices) > 0 && len(prices[0].Bids) > 0 && len(prices[0].Asks) > 0 {
			b := parseDecimal(prices[0].Bids[0].Price)
			a := parseDecimal(prices[0].Asks[0].Price)
			if b > 0 && a > 0 {
				entry = (b + a) / 2
			}
		}
		tr := &models.Trade{
			ID:         "", // let DB assign UUID
			Instrument: req.Instrument,
			Direction: func() string {
				if req.Units >= 0 {
					return "BUY"
				}
				return "SELL"
			}(),
			Units:        req.Units,
			EntryPrice:   &entry,
			Status:       models.TradeStatusOpen,
			OandaTradeID: func() *string { id := resp.OrderCreateTransaction.ID; return &id }(),
		}
		_ = s.db.CreateTrade(c.Request.Context(), tr)
	}
	c.JSON(200, gin.H{"order": resp})
}

func (s *Server) getPositions(c *gin.Context) {
	positions, errors := s.mt4Client.GetPositions()
	if errors != nil {
		log.Printf("Error getting positions: %v", errors)
		c.JSON(500, gin.H{"status": "error"})
	}

	c.JSON(200, gin.H{"message": positions})
}

func (s *Server) listTrades(c *gin.Context) {
	if s.db == nil {
		c.JSON(503, gin.H{"error": "db not configured"})
		return
	}
	trades, err := s.db.ListTrades(c.Request.Context(), 200)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, trades)
}

func (s *Server) deleteTrade(c *gin.Context) {
	if s.db == nil {
		c.JSON(503, gin.H{"error": "db not configured"})
		return
	}
	id := c.Param("id")
	if err := s.db.SoftDeleteTrade(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"deleted": id})
}

func (s *Server) searchNews(c *gin.Context) {
	query := c.Param("query")
	items, err := s.brave.SearchNews(c.Request.Context(), query, 10)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}

// --- Simple in-memory recommendations store (temporary placeholder) ---
type recommendation struct {
	ID         string  `json:"id"`
	Instrument string  `json:"instrument"`
	Direction  string  `json:"direction"` // BUY or SELL
	Units      float64 `json:"units"`
	Rationale  string  `json:"rationale"`
	CreatedAt  int64   `json:"createdAt"`
}

var recs = make(map[string]recommendation)

func (s *Server) createRecommendation(c *gin.Context) {
	var r recommendation
	if err := c.BindJSON(&r); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if r.ID == "" {
		r.ID = time.Now().Format("20060102150405")
	}
	r.CreatedAt = time.Now().Unix()
	// persist to DB primarily; keep in-memory as fallback only
	if s.db != nil {
		rationale := r.Rationale
		status := models.RecommendationStatusPending
		rec := &models.Recommendation{
			ID:         r.ID,
			Instrument: r.Instrument,
			Direction:  r.Direction,
			Units:      r.Units,
			Rationale:  &rationale,
			Status:     status,
		}
		if id, err := s.db.CreateRecommendation(c.Request.Context(), rec); err == nil {
			r.ID = id
		} else {
			// fallback to memory if DB fails
			recs[r.ID] = r
		}
	} else {
		recs[r.ID] = r
	}

	c.JSON(201, r)
}

func (s *Server) listRecommendations(c *gin.Context) {
	if s.db != nil {
		recsDB, err := s.db.ListRecommendations(c.Request.Context())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, recsDB)
		return
	}
	list := make([]recommendation, 0, len(recs))
	for _, v := range recs {
		list = append(list, v)
	}
	c.JSON(200, list)
}

func (s *Server) acceptRecommendation(c *gin.Context) {
	id := c.Param("id")
	var r recommendation
	var ok bool
	var sl, tp *float64
	// First try legacy recommendations table
	if s.db != nil {
		list, err := s.db.ListRecommendations(c.Request.Context())
		if err == nil {
			for _, item := range list {
				if item.ID == id {
					r = recommendation{ID: item.ID, Instrument: item.Instrument, Direction: item.Direction, Units: item.Units, Rationale: func() string {
						if item.Rationale != nil {
							return *item.Rationale
						}
						return ""
					}(), CreatedAt: item.CreatedAt.Unix()}
					ok = true
					break
				}
			}
		}
	}
	// If not found, try AI recommendations table
	isAI := false
	if s.db != nil && !ok {
		if list, err := s.db.ListAIRecommendations(c.Request.Context(), 200); err == nil {
			for _, item := range list {
				if item.ID == id {
					isAI = true
					dir := item.Direction
					units := item.Units
					r = recommendation{ID: item.ID, Instrument: item.Instrument, Direction: dir, Units: units, Rationale: item.Rationale, CreatedAt: item.CreatedAt.Unix()}
					// capture SL/TP from AI rec
					sl = item.StopLoss
					tp = item.TakeProfit
					ok = true
					break
				}
			}
		}
	}
	if !ok {
		if mem, found := recs[id]; found {
			r = mem
			ok = true
		}
	}
	if !ok {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	units := r.Units
	if strings.ToUpper(r.Direction) == "SELL" {
		units = -units
	}
	var resp *broker.OrderCreateResponse
	var err error
	// Use brackets if we have SL/TP from AI
	if sl != nil || tp != nil {
		resp, err = s.mt4Client.PlaceMarketOrderWithBrackets(r.Instrument, units, sl, tp)
	} else {
		resp, err = s.mt4Client.PlaceMarketOrder(r.Instrument, units)
	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// mark executed in DB and create trade record
	if s.db != nil && resp != nil {
		if isAI {
			_ = s.db.MarkAIRecommendationExecuted(c.Request.Context(), id, resp.OrderCreateTransaction.ID)
		} else {
			_ = s.db.MarkRecommendationExecuted(c.Request.Context(), id, resp.OrderCreateTransaction.ID)
		}
		// Create trade row
		entry := 0.0
		if prices, perr := s.mt4Client.GetPrices([]string{r.Instrument}); perr == nil && len(prices) > 0 && len(prices[0].Bids) > 0 && len(prices[0].Asks) > 0 {
			b := parseDecimal(prices[0].Bids[0].Price)
			a := parseDecimal(prices[0].Asks[0].Price)
			if b > 0 && a > 0 {
				entry = (b + a) / 2
			}
		}
		trade := &models.Trade{
			ID:         "",
			Instrument: r.Instrument,
			Direction: func() string {
				if units >= 0 {
					return "BUY"
				}
				return "SELL"
			}(),
			Units:        units,
			EntryPrice:   &entry,
			Status:       models.TradeStatusOpen,
			OandaTradeID: func() *string { id := resp.OrderCreateTransaction.ID; return &id }(),
		}
		_ = s.db.CreateTrade(c.Request.Context(), trade)
	}
	c.JSON(200, gin.H{"accepted": r, "order": resp})
}

func (s *Server) deleteRecommendation(c *gin.Context) {
	if s.db == nil {
		c.JSON(503, gin.H{"error": "db not configured"})
		return
	}
	id := c.Param("id")
	if err := s.db.SoftDeleteRecommendation(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"deleted": id})
}

// ---- AI endpoints ----
func (s *Server) aiGenerateRecommendation(c *gin.Context) {
	var req ai.RecommendationRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if s.ai == nil {
		c.JSON(503, gin.H{"error": "ai service not configured"})
		return
	}
	log.Printf("[AI] recommend start instruments=%v risk=%s horizon=%s units=%d risk_percent=%.4f sl_pips=%.2f", req.Instruments, req.RiskLevel, req.TimeHorizon, req.Units, req.RiskPercent, req.StopLossPips)
	start := time.Now()
	rec, err := s.ai.GenerateRecommendation(c.Request.Context(), &req)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Enrich with SL/TP using price/candles
	var mid float64
	if prices, err := s.mt4Client.GetPrices([]string{rec.Instrument}); err == nil && len(prices) > 0 && len(prices[0].Bids) > 0 && len(prices[0].Asks) > 0 {
		b := parseDecimal(prices[0].Bids[0].Price)
		a := parseDecimal(prices[0].Asks[0].Price)
		if b > 0 && a > 0 {
			mid = (b + a) / 2
		}
	}
	if mid == 0 {
		if candles, err := s.mt4Client.GetCandles(rec.Instrument, "M5", 1, nil, nil); err == nil && candles != nil && len(candles.Candles) > 0 {
			mid = parseDecimal(candles.Candles[len(candles.Candles)-1].Mid.Close)
		}
	}
	if mid > 0 {
		pip := 0.0001
		if strings.Contains(rec.Instrument, "JPY") {
			pip = 0.01
		}
		distPips := 20.0
		switch strings.ToLower(req.RiskLevel) {
		case "low":
			distPips = 30
		case "high":
			distPips = 10
		}
		rr := 2.0
		sl := mid
		tp := mid
		if strings.ToUpper(rec.Direction) == "BUY" {
			sl = mid - distPips*pip
			tp = mid + rr*distPips*pip
		} else {
			sl = mid + distPips*pip
			tp = mid - rr*distPips*pip
		}
		rec.StopLoss = &sl
		rec.TakeProfit = &tp
	}

	// Position sizing
	if req.Units > 0 {
		rec.Units = req.Units
	} else if (req.RiskPercent > 0 && req.StopLossPips > 0) || (req.RiskPercent > 0 && rec.StopLoss != nil) {
		// Estimate pip value for EUR/USD ~ $10 per pip per 100k units; scale linearly
		pipValuePerUnit := 10.0 / 100000.0 // USD per pip per unit
		// Determine stop distance in pips
		var slPips float64
		if req.StopLossPips > 0 {
			slPips = req.StopLossPips
		} else if rec.StopLoss != nil {
			// approximate from mid price
			mid := 0.0
			if prices, err := s.mt4Client.GetPrices([]string{rec.Instrument}); err == nil && len(prices) > 0 && len(prices[0].Bids) > 0 && len(prices[0].Asks) > 0 {
				b := parseDecimal(prices[0].Bids[0].Price)
				a := parseDecimal(prices[0].Asks[0].Price)
				if b > 0 && a > 0 {
					mid = (b + a) / 2
				}
			}
			if mid > 0 {
				pip := 0.0001
				if strings.Contains(rec.Instrument, "JPY") {
					pip = 0.01
				}
				slPips = math.Abs(mid-*rec.StopLoss) / pip
			}
		}
		// Get account NAV
		account, accErr := s.mt4Client.GetAccount()
		if accErr == nil && slPips > 0 {
			riskUSD := account.NAV * req.RiskPercent
			units := riskUSD / (slPips * pipValuePerUnit)
			if units < 1 {
				units = 1
			}
			rec.Units = int64(math.Round(units))
		}
	}

	// Attempt to persist AI recommendation with contexts
	var persistedID string
	if s.db != nil {
		marketJSON, _ := json.Marshal(rec.MarketData)
		newsJSON, _ := json.Marshal(rec.NewsContext)
		histJSON, _ := json.Marshal(struct {
			Notes string `json:"notes"`
		}{Notes: "pending"})

		// Ensure we don't pass a non-UUID ID (e.g., "simulated") to the DB
		safeID := rec.ID
		if !isUUIDLike(safeID) {
			safeID = ""
		}

		aiRow := &models.AIRecommendation{
			ID:                safeID,
			Instrument:        rec.Instrument,
			Direction:         rec.Direction,
			Units:             float64(rec.Units),
			Confidence:        rec.Confidence,
			Rationale:         rec.Rationale,
			StopLoss:          rec.StopLoss,
			TakeProfit:        rec.TakeProfit,
			TimeToLive:        rec.TimeToLive,
			MarketContext:     marketJSON,
			NewsContext:       newsJSON,
			HistoricalContext: histJSON,
			Status:            models.AIRecommendationStatusPending,
		}
		if id, err := s.db.CreateAIRecommendation(c.Request.Context(), aiRow); err == nil {
			rec.ID = id
			persistedID = id
			log.Printf("[AI] recommendation persisted id=%s instrument=%s dir=%s units=%d", id, rec.Instrument, rec.Direction, rec.Units)

			// Mirror into legacy recommendations for compatibility with existing endpoints
			rationale := rec.Rationale
			status := models.RecommendationStatusPending
			var confPtr *float64
			if rec.Confidence > 0 {
				v := rec.Confidence
				confPtr = &v
			}
			legacy := &models.Recommendation{
				ID:               "", // let DB generate
				Instrument:       rec.Instrument,
				Direction:        rec.Direction,
				Units:            float64(rec.Units),
				Rationale:        &rationale,
				ConfidenceScore:  confPtr,
				MarketConditions: marketJSON,
				Status:           status,
			}
			if rid, err := s.db.CreateRecommendation(c.Request.Context(), legacy); err == nil {
				log.Printf("[AI] legacy recommendation mirrored id=%s from ai_id=%s", rid, id)
			} else {
				log.Printf("[AI] mirror to legacy recommendations failed: %v", err)
			}
		} else {
			log.Printf("[AI] persist recommendation error: %v", err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("[AI] recommend done instrument=%s dir=%s units=%d elapsed=%s", rec.Instrument, rec.Direction, rec.Units, elapsed)

	// Write AI usage log (approximate tokens based on payload sizes)
	if s.db != nil && persistedID != "" {
		promptTokens := len(req.Instruments)*4 + 20
		completionTokens := 60
		total := promptTokens + completionTokens
		model := "simulated"
		_ = s.db.CreateAIUsageLog(c.Request.Context(), persistedID, promptTokens, completionTokens, total, int(elapsed.Milliseconds()), model)
	}

	// Optional: write a small market analysis cache record for the instrument
	if s.db != nil && len(req.Instruments) > 0 {
		inst := req.Instruments[0]
		if candles, err := s.mt4Client.GetCandles(inst, "M5", 20, nil, nil); err == nil && candles != nil {
			summary := map[string]interface{}{"instrument": inst, "granularity": candles.Granularity, "count": len(candles.Candles)}
			buf, _ := json.Marshal(summary)
			expires := time.Now().Add(10 * time.Minute)
			_ = s.db.InsertMarketAnalysisCache(c.Request.Context(), inst, buf, expires)
		}
	}

	c.JSON(200, rec)
}

// isUUIDLike performs a lightweight UUID format validation (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
func isUUIDLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, ch := range s {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

func (s *Server) aiStatus(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
