-- Enable pgcrypto for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- trades table
CREATE TABLE IF NOT EXISTS trades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument VARCHAR(50) NOT NULL,
    direction VARCHAR(4) NOT NULL CHECK (direction IN ('BUY','SELL')),
    units DECIMAL(15,2) NOT NULL,
    entry_price DECIMAL(15,8),
    exit_price DECIMAL(15,8),
    profit_loss DECIMAL(15,2),
    commission DECIMAL(15,2),
    swap DECIMAL(15,2),
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    oanda_trade_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument VARCHAR(50) NOT NULL,
    direction VARCHAR(4) NOT NULL CHECK (direction IN ('BUY','SELL')),
    units DECIMAL(15,2) NOT NULL,
    rationale TEXT,
    confidence_score DECIMAL(3,2),
    market_conditions JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    trade_id UUID REFERENCES trades(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    executed_at TIMESTAMPTZ
);

-- market_data table
CREATE TABLE IF NOT EXISTS market_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    open_price DECIMAL(15,8) NOT NULL,
    high_price DECIMAL(15,8) NOT NULL,
    low_price DECIMAL(15,8) NOT NULL,
    close_price DECIMAL(15,8) NOT NULL,
    volume BIGINT,
    timeframe VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(instrument, timestamp, timeframe)
);

-- indexes
CREATE INDEX IF NOT EXISTS idx_trades_instrument ON trades(instrument);
CREATE INDEX IF NOT EXISTS idx_recs_instrument ON recommendations(instrument);
CREATE INDEX IF NOT EXISTS idx_market_data_instrument_timeframe ON market_data(instrument, timeframe);


