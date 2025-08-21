package models

import "time"

type RecommendationStatus string

const (
	RecommendationStatusPending  RecommendationStatus = "PENDING"
	RecommendationStatusExecuted RecommendationStatus = "EXECUTED"
)

type Recommendation struct {
	ID               string               `db:"id" json:"id"`
	Instrument       string               `db:"instrument" json:"instrument"`
	Direction        string               `db:"direction" json:"direction"`
	Units            float64              `db:"units" json:"units"`
	Rationale        *string              `db:"rationale" json:"rationale,omitempty"`
	ConfidenceScore  *float64             `db:"confidence_score" json:"confidence_score,omitempty"`
	MarketConditions []byte               `db:"market_conditions" json:"market_conditions,omitempty"`
	Status           RecommendationStatus `db:"status" json:"status"`
	TradeID          *string              `db:"trade_id" json:"trade_id,omitempty"`
	CreatedAt        time.Time            `db:"created_at" json:"created_at"`
	ExecutedAt       *time.Time           `db:"executed_at" json:"executed_at,omitempty"`
}
