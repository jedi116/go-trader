-- AI recommendations extended tables
CREATE TABLE IF NOT EXISTS ai_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument VARCHAR(50) NOT NULL,
    direction VARCHAR(4) NOT NULL CHECK (direction IN ('BUY', 'SELL')),
    units DECIMAL(15,2) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,
    rationale TEXT NOT NULL,
    stop_loss DECIMAL(15,8),
    take_profit DECIMAL(15,8),
    time_to_live TIMESTAMPTZ NOT NULL,
    market_context JSONB NOT NULL,
    news_context JSONB,
    historical_context JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    approved_at TIMESTAMPTZ,
    executed_trade_id UUID REFERENCES trades(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID REFERENCES ai_recommendations(id),
    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    response_time_ms INTEGER NOT NULL,
    claude_model VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS market_analysis_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instruments VARCHAR(200) NOT NULL,
    analysis_data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);


