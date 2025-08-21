package models

import "time"

type AIRecommendationStatus string

const (
	AIRecommendationStatusPending  AIRecommendationStatus = "PENDING"
	AIRecommendationStatusApproved AIRecommendationStatus = "APPROVED"
	AIRecommendationStatusRejected AIRecommendationStatus = "REJECTED"
	AIRecommendationStatusExecuted AIRecommendationStatus = "EXECUTED"
)

type AIRecommendation struct {
	ID                string                 `db:"id" json:"id"`
	Instrument        string                 `db:"instrument" json:"instrument"`
	Direction         string                 `db:"direction" json:"direction"`
	Units             float64                `db:"units" json:"units"`
	Confidence        float64                `db:"confidence" json:"confidence"`
	Rationale         string                 `db:"rationale" json:"rationale"`
	StopLoss          *float64               `db:"stop_loss" json:"stop_loss,omitempty"`
	TakeProfit        *float64               `db:"take_profit" json:"take_profit,omitempty"`
	TimeToLive        time.Time              `db:"time_to_live" json:"time_to_live"`
	MarketContext     []byte                 `db:"market_context" json:"market_context"`
	NewsContext       []byte                 `db:"news_context" json:"news_context,omitempty"`
	HistoricalContext []byte                 `db:"historical_context" json:"historical_context,omitempty"`
	Status            AIRecommendationStatus `db:"status" json:"status"`
	ApprovedAt        *time.Time             `db:"approved_at" json:"approved_at,omitempty"`
	ExecutedTradeID   *string                `db:"executed_trade_id" json:"executed_trade_id,omitempty"`
	CreatedAt         time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at" json:"updated_at"`
}
