package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jedi116/go-trader/internal/config"
	"github.com/jedi116/go-trader/pkg/models"
	_ "github.com/lib/pq"
)

type Postgres struct {
	DB *sql.DB
}

func (p *Postgres) audit(ctx context.Context, entity string, entityID string, action string, details map[string]interface{}) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO audit_logs(entity, entity_id, action, details) VALUES ($1,$2,$3,$4)`, entity, entityID, action, details)
	return err
}

func NewPostgres(cfg *config.Config) (*Postgres, error) {
	dsn := os.Getenv("DATABASE_URL")
	via := "env"
	if dsn == "" {
		sslmode := cfg.Database.SSLMode
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database, sslmode)
		via = "config"
	}
	if u, err := url.Parse(dsn); err == nil {
		q := u.Query()
		sslmode := q.Get("sslmode")
		dbName := strings.TrimPrefix(u.Path, "/")
		if u.User != nil {
			if _, has := u.User.Password(); has {
				u.User = url.User(u.User.Username())
			}
		}
		log.Printf("[DB] Connecting host=%s db=%s sslmode=%s via=%s", u.Host, dbName, sslmode, via)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &Postgres{DB: db}, nil
}

func (p *Postgres) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.DB.PingContext(ctx)
}

// Recommendation CRUD
func (p *Postgres) CreateRecommendation(ctx context.Context, r *models.Recommendation) (string, error) {
	query := `INSERT INTO recommendations (id, instrument, direction, units, rationale, confidence_score, market_conditions, status, trade_id, created_at, executed_at)
              VALUES (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()),$2,$3,$4,$5,$6,$7,$8,$9,NOW(),$10)
              RETURNING id`
	var id string
	if err := p.DB.QueryRowContext(ctx, query, r.ID, r.Instrument, r.Direction, r.Units, r.Rationale, r.ConfidenceScore, r.MarketConditions, r.Status, r.TradeID, r.ExecutedAt).Scan(&id); err != nil {
		return "", err
	}
	_ = p.audit(ctx, "recommendations", id, "CREATE", map[string]interface{}{"instrument": r.Instrument, "direction": r.Direction, "units": r.Units})
	return id, nil
}

func (p *Postgres) ListRecommendations(ctx context.Context) ([]models.Recommendation, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT id, instrument, direction, units, rationale, confidence_score, market_conditions, status, trade_id, created_at, executed_at FROM recommendations WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Recommendation
	for rows.Next() {
		var r models.Recommendation
		if err := rows.Scan(&r.ID, &r.Instrument, &r.Direction, &r.Units, &r.Rationale, &r.ConfidenceScore, &r.MarketConditions, &r.Status, &r.TradeID, &r.CreatedAt, &r.ExecutedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Postgres) MarkRecommendationExecuted(ctx context.Context, id string, tradeID string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE recommendations SET status='EXECUTED', trade_id=$2, executed_at=NOW() WHERE id=$1`, id, tradeID)
	if err == nil {
		_ = p.audit(ctx, "recommendations", id, "EXECUTE", map[string]interface{}{"trade_id": tradeID})
	}
	return err
}

// Trade persistence (minimal)
func (p *Postgres) CreateTrade(ctx context.Context, t *models.Trade) error {
	query := `INSERT INTO trades (id, instrument, direction, units, entry_price, exit_price, profit_loss, commission, swap, status, oanda_trade_id, created_at, updated_at, closed_at)
              VALUES (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()),$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW(),$12)`
	_, err := p.DB.ExecContext(ctx, query, t.ID, t.Instrument, t.Direction, t.Units, t.EntryPrice, t.ExitPrice, t.ProfitLoss, t.Commission, t.Swap, t.Status, t.OandaTradeID, t.ClosedAt)
	if err == nil {
		_ = p.audit(ctx, "trades", t.ID, "CREATE", map[string]interface{}{"instrument": t.Instrument, "direction": t.Direction, "units": t.Units})
	}
	return err
}

func (p *Postgres) Close() error { return p.DB.Close() }

func (p *Postgres) ListTrades(ctx context.Context, limit int) ([]models.Trade, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := p.DB.QueryContext(ctx, `SELECT id, instrument, direction, units, entry_price, exit_price, profit_loss, commission, swap, status, oanda_trade_id, created_at, updated_at, closed_at FROM trades WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Trade
	for rows.Next() {
		var t models.Trade
		if err := rows.Scan(&t.ID, &t.Instrument, &t.Direction, &t.Units, &t.EntryPrice, &t.ExitPrice, &t.ProfitLoss, &t.Commission, &t.Swap, &t.Status, &t.OandaTradeID, &t.CreatedAt, &t.UpdatedAt, &t.ClosedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Soft deletes
func (p *Postgres) SoftDeleteRecommendation(ctx context.Context, id string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE recommendations SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err == nil {
		_ = p.audit(ctx, "recommendations", id, "DELETE", map[string]interface{}{})
	}
	return err
}

func (p *Postgres) SoftDeleteTrade(ctx context.Context, id string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE trades SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err == nil {
		_ = p.audit(ctx, "trades", id, "DELETE", map[string]interface{}{})
	}
	return err
}

// Market data persistence
func (p *Postgres) UpsertMarketData(ctx context.Context, rows []models.MarketData) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO market_data (id, instrument, timestamp, open_price, high_price, low_price, close_price, volume, timeframe, created_at)
        VALUES (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()),$2,$3,$4,$5,$6,$7,$8,$9,NOW())
        ON CONFLICT (instrument, timestamp, timeframe)
        DO UPDATE SET open_price=EXCLUDED.open_price, high_price=EXCLUDED.high_price, low_price=EXCLUDED.low_price, close_price=EXCLUDED.close_price, volume=EXCLUDED.volume
    `)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, r.ID, r.Instrument, r.Timestamp, r.OpenPrice, r.HighPrice, r.LowPrice, r.ClosePrice, r.Volume, r.Timeframe); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (p *Postgres) ListMarketData(ctx context.Context, instrument string, timeframe string, limit int) ([]models.MarketData, error) {
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := p.DB.QueryContext(ctx, `
        SELECT id, instrument, timestamp, open_price, high_price, low_price, close_price, volume, timeframe, created_at
        FROM market_data
        WHERE deleted_at IS NULL AND instrument = $1 AND timeframe = $2
        ORDER BY timestamp DESC
        LIMIT $3
    `, instrument, timeframe, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.MarketData
	for rows.Next() {
		var m models.MarketData
		if err := rows.Scan(&m.ID, &m.Instrument, &m.Timestamp, &m.OpenPrice, &m.HighPrice, &m.LowPrice, &m.ClosePrice, &m.Volume, &m.Timeframe, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ---- AI tables ----
func (p *Postgres) CreateAIRecommendation(ctx context.Context, r *models.AIRecommendation) (string, error) {
	query := `INSERT INTO ai_recommendations (id, instrument, direction, units, confidence, rationale, stop_loss, take_profit, time_to_live, market_context, news_context, historical_context, status, approved_at, executed_trade_id, created_at, updated_at)
              VALUES (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()),$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW(),NOW())
              RETURNING id`
	var id string
	if err := p.DB.QueryRowContext(ctx, query, r.ID, r.Instrument, r.Direction, r.Units, r.Confidence, r.Rationale, r.StopLoss, r.TakeProfit, r.TimeToLive, r.MarketContext, r.NewsContext, r.HistoricalContext, r.Status, r.ApprovedAt, r.ExecutedTradeID).Scan(&id); err != nil {
		return "", err
	}
	_ = p.audit(ctx, "ai_recommendations", id, "CREATE", map[string]interface{}{"instrument": r.Instrument, "direction": r.Direction, "units": r.Units})
	return id, nil
}

func (p *Postgres) UpdateAIRecommendationStatus(ctx context.Context, id string, status models.AIRecommendationStatus) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE ai_recommendations SET status=$2, updated_at=NOW() WHERE id=$1`, id, status)
	return err
}

// MarkAIRecommendationExecuted sets status to EXECUTED and stores the executed trade id
func (p *Postgres) MarkAIRecommendationExecuted(ctx context.Context, id string, tradeID string) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE ai_recommendations SET status='EXECUTED', executed_trade_id=$2, updated_at=NOW() WHERE id=$1`, id, tradeID)
	if err == nil {
		_ = p.audit(ctx, "ai_recommendations", id, "EXECUTE", map[string]interface{}{"trade_id": tradeID})
	}
	return err
}

func (p *Postgres) ListAIRecommendations(ctx context.Context, limit int) ([]models.AIRecommendation, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := p.DB.QueryContext(ctx, `SELECT id, instrument, direction, units, confidence, rationale, stop_loss, take_profit, time_to_live, market_context, news_context, historical_context, status, approved_at, executed_trade_id, created_at, updated_at FROM ai_recommendations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AIRecommendation
	for rows.Next() {
		var r models.AIRecommendation
		if err := rows.Scan(&r.ID, &r.Instrument, &r.Direction, &r.Units, &r.Confidence, &r.Rationale, &r.StopLoss, &r.TakeProfit, &r.TimeToLive, &r.MarketContext, &r.NewsContext, &r.HistoricalContext, &r.Status, &r.ApprovedAt, &r.ExecutedTradeID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AI usage logs
func (p *Postgres) CreateAIUsageLog(ctx context.Context, recommendationID string, promptTokens, completionTokens, totalTokens, responseTimeMs int, model string) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO ai_usage_logs (recommendation_id, prompt_tokens, completion_tokens, total_tokens, response_time_ms, claude_model) VALUES ($1,$2,$3,$4,$5,$6)`, recommendationID, promptTokens, completionTokens, totalTokens, responseTimeMs, model)
	return err
}

// Market analysis cache insert
func (p *Postgres) InsertMarketAnalysisCache(ctx context.Context, instruments string, analysisData []byte, expiresAt time.Time) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO market_analysis_cache (instruments, analysis_data, expires_at) VALUES ($1,$2,$3)`, instruments, analysisData, expiresAt)
	return err
}
