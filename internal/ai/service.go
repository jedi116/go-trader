package ai

import (
	"context"
	"time"
)

type Service interface {
	GenerateRecommendation(ctx context.Context, request *RecommendationRequest) (*Recommendation, error)
	ExecuteRecommendation(ctx context.Context, id string) (*Trade, error)
	GetRecommendationStatus(ctx context.Context, id string) (*RecommendationStatus, error)
}

type RecommendationRequest struct {
	Instruments  []string `json:"instruments"`
	RiskLevel    string   `json:"risk_level"`
	TimeHorizon  string   `json:"time_horizon"`
	MaxRisk      float64  `json:"max_risk"`
	Context      string   `json:"context,omitempty"`
	Units        int64    `json:"units,omitempty"`
	RiskPercent  float64  `json:"risk_percent,omitempty"`
	StopLossPips float64  `json:"stop_loss_pips,omitempty"`
}

type MarketContext struct {
	Instruments map[string]interface{} `json:"instruments"`
}

type NewsItem struct {
	Title     string `json:"title"`
	Url       string `json:"url"`
	Snippet   string `json:"snippet"`
	Source    string `json:"source"`
	Published string `json:"published"`
}

type HistoricalContext struct {
	Notes string `json:"notes"`
}

type TradingContext struct {
	Timestamp    time.Time          `json:"timestamp"`
	MarketData   *MarketContext     `json:"market_data"`
	NewsAnalysis []NewsItem         `json:"news_analysis"`
	Historical   *HistoricalContext `json:"historical"`
}

type Recommendation struct {
	ID          string         `json:"id"`
	Instrument  string         `json:"instrument"`
	Direction   string         `json:"direction"`
	Units       int64          `json:"units"`
	Confidence  float64        `json:"confidence"`
	Rationale   string         `json:"rationale"`
	StopLoss    *float64       `json:"stop_loss,omitempty"`
	TakeProfit  *float64       `json:"take_profit,omitempty"`
	TimeToLive  time.Time      `json:"time_to_live"`
	MarketData  *MarketContext `json:"market_data"`
	NewsContext []NewsItem     `json:"news_context"`
}

type RecommendationStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Trade struct {
	ID         string  `json:"id"`
	Instrument string  `json:"instrument"`
	Units      float64 `json:"units"`
}
