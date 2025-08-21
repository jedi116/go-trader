package models

import "time"

type MarketData struct {
	ID         string    `db:"id" json:"id"`
	Instrument string    `db:"instrument" json:"instrument"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
	OpenPrice  float64   `db:"open_price" json:"open_price"`
	HighPrice  float64   `db:"high_price" json:"high_price"`
	LowPrice   float64   `db:"low_price" json:"low_price"`
	ClosePrice float64   `db:"close_price" json:"close_price"`
	Volume     *int64    `db:"volume" json:"volume,omitempty"`
	Timeframe  string    `db:"timeframe" json:"timeframe"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
