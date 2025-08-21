package models

import "time"

type TradeStatus string

const (
	TradeStatusOpen   TradeStatus = "OPEN"
	TradeStatusClosed TradeStatus = "CLOSED"
)

type Trade struct {
	ID           string      `db:"id" json:"id"`
	Instrument   string      `db:"instrument" json:"instrument"`
	Direction    string      `db:"direction" json:"direction"` // BUY or SELL
	Units        float64     `db:"units" json:"units"`
	EntryPrice   *float64    `db:"entry_price" json:"entry_price,omitempty"`
	ExitPrice    *float64    `db:"exit_price" json:"exit_price,omitempty"`
	ProfitLoss   *float64    `db:"profit_loss" json:"profit_loss,omitempty"`
	Commission   *float64    `db:"commission" json:"commission,omitempty"`
	Swap         *float64    `db:"swap" json:"swap,omitempty"`
	Status       TradeStatus `db:"status" json:"status"`
	OandaTradeID *string     `db:"oanda_trade_id" json:"oanda_trade_id,omitempty"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at" json:"updated_at"`
	ClosedAt     *time.Time  `db:"closed_at" json:"closed_at,omitempty"`
}
