# Go-Trader AI Service Implementation Plan

## Architecture Correction

The original MCP design was flawed - MCP servers are meant to run locally, not as production services. This plan corrects the architecture to implement a proper AI service that aggregates trading data and calls Claude's API directly.

## New Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   Mobile App    │    │  gRPC Client    │
│   (Dashboard)   │    │   (Trading)     │    │  (Services)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │     Go-Trader Server    │
                    │  ┌─────────────────────┐│
                    │  │   REST Endpoints    ││
                    │  │  • /api/v1/trades   ││
                    │  │  • /api/v1/orders   ││
                    │  │  • /api/v1/analyze  ││
                    │  │  • /api/v1/ai       ││  ← NEW AI endpoint
                    │  └─────────────────────┘│
                    │  ┌─────────────────────┐│
                    │  │   gRPC Services     ││
                    │  │  • TradeService     ││
                    │  │  • AnalysisService  ││
                    │  │  • AIService        ││  ← NEW AI service
                    │  └─────────────────────┘│
                    │  ┌─────────────────────┐│
                    │  │   AI Service        ││  ← NEW COMPONENT
                    │  │  • Data Aggregator  ││
                    │  │  • Claude API       ││
                    │  │  • Recommendation   ││
                    │  │  • Auto Execute     ││
                    │  └─────────────────────┘│
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │       Claude API        │  ← External API
                    │   (api.anthropic.com)   │
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │    PostgreSQL DB        │
                    │  • trades               │
                    │  • recommendations      │
                    │  • market_data          │
                    │  • analysis_results     │
                    └─────────────────────────┘
```

## Core AI Service Component

### Data Aggregation Flow
1. **Market Data Collection**: Real-time prices, candles, positions from OANDA
2. **News Analysis**: Financial news from Brave Search with sentiment scoring
3. **Historical Analysis**: Past trade performance, market patterns
4. **Context Assembly**: Create comprehensive trading context for Claude
5. **API Request**: Send structured prompt to Claude API
6. **Response Processing**: Parse recommendation JSON from Claude
7. **User Approval**: Present recommendation to user via REST/gRPC
8. **Trade Execution**: Execute approved trades via OANDA

### New AI Service Implementation

#### Core Service (`internal/ai/service.go`)
```go
type AIService interface {
    GenerateRecommendation(ctx context.Context, request *RecommendationRequest) (*Recommendation, error)
    ExecuteRecommendation(ctx context.Context, id string) (*Trade, error)
    GetRecommendationStatus(ctx context.Context, id string) (*RecommendationStatus, error)
}

type RecommendationRequest struct {
    Instruments []string `json:"instruments"`
    RiskLevel   string   `json:"risk_level"` // conservative, moderate, aggressive
    TimeHorizon string   `json:"time_horizon"` // short, medium, long
    MaxRisk     float64  `json:"max_risk"` // maximum position size
    Context     string   `json:"context,omitempty"` // optional user context
}

type Recommendation struct {
    ID          string    `json:"id"`
    Instrument  string    `json:"instrument"`
    Direction   string    `json:"direction"` // BUY/SELL
    Units       int64     `json:"units"`
    Confidence  float64   `json:"confidence"` // 0-1
    Rationale   string    `json:"rationale"`
    StopLoss    *float64  `json:"stop_loss,omitempty"`
    TakeProfit  *float64  `json:"take_profit,omitempty"`
    TimeToLive  time.Time `json:"time_to_live"`
    MarketData  MarketContext `json:"market_data"`
    NewsContext []NewsItem    `json:"news_context"`
}
```

#### Data Aggregator (`internal/ai/aggregator.go`)
```go
type DataAggregator interface {
    GatherMarketData(ctx context.Context, instruments []string) (*MarketContext, error)
    GatherNewsData(ctx context.Context, instruments []string) ([]NewsItem, error)
    GatherHistoricalData(ctx context.Context, instruments []string) (*HistoricalContext, error)
    AssembleContext(market *MarketContext, news []NewsItem, historical *HistoricalContext) *TradingContext
}

type TradingContext struct {
    Timestamp     time.Time         `json:"timestamp"`
    MarketData    *MarketContext    `json:"market_data"`
    NewsAnalysis  []NewsItem        `json:"news_analysis"`
    Historical    *HistoricalContext `json:"historical"`
    RiskMetrics   *RiskContext      `json:"risk_metrics"`
}
```

#### Claude API Client (`internal/ai/claude.go`)
```go
type ClaudeClient interface {
    GenerateRecommendation(ctx context.Context, tradingContext *TradingContext, request *RecommendationRequest) (*Recommendation, error)
}

type ClaudeRequest struct {
    Model       string `json:"model"`
    MaxTokens   int    `json:"max_tokens"`
    Temperature float64 `json:"temperature"`
    Messages    []Message `json:"messages"`
}

type ClaudeResponse struct {
    Content []ContentBlock `json:"content"`
    Usage   Usage         `json:"usage"`
}
```

## Implementation Steps

### Phase 1: Remove MCP Components
- [ ] Remove `internal/mcp/` package completely
- [ ] Remove `cmd/mcp/` command
- [ ] Remove MCP-related documentation
- [ ] Clean up any MCP references in other files

### Phase 2: Implement AI Service Core
- [ ] Create `internal/ai/` package structure
- [ ] Implement `service.go` with core AI service interface
- [ ] Implement `aggregator.go` for data collection
- [ ] Implement `claude.go` for Claude API integration
- [ ] Add proper error handling and logging

### Phase 3: Data Context Assembly
- [ ] Enhance market data collection with technical indicators
- [ ] Improve news sentiment analysis
- [ ] Add historical performance analysis
- [ ] Create structured prompts for Claude API

### Phase 4: API Integration
- [ ] Add new REST endpoints for AI service
- [ ] Add new gRPC service for AI functionality
- [ ] Implement user approval workflow
- [ ] Add recommendation persistence

### Phase 5: Enhanced Features
- [ ] Risk management and position sizing
- [ ] Portfolio optimization suggestions  
- [ ] Performance tracking and learning
- [ ] Advanced market analysis

## New REST API Endpoints

### AI Recommendation Endpoints
- `POST /api/v1/ai/recommend` - Generate new recommendation
- `GET /api/v1/ai/recommendations` - List pending recommendations
- `POST /api/v1/ai/recommendations/:id/approve` - Approve and execute
- `DELETE /api/v1/ai/recommendations/:id` - Reject recommendation
- `GET /api/v1/ai/status` - AI service health and statistics

### Enhanced Analysis Endpoints
- `POST /api/v1/ai/analyze` - Comprehensive market analysis
- `GET /api/v1/ai/insights` - Market insights and patterns
- `POST /api/v1/ai/portfolio` - Portfolio optimization suggestions

## New gRPC Services

### AIService Methods
```protobuf
service AIService {
    rpc GenerateRecommendation(GenerateRecommendationRequest) returns (Recommendation);
    rpc ListRecommendations(ListRecommendationsRequest) returns (ListRecommendationsResponse);
    rpc ApproveRecommendation(ApproveRecommendationRequest) returns (Trade);
    rpc RejectRecommendation(RejectRecommendationRequest) returns (Empty);
    rpc GetMarketAnalysis(GetMarketAnalysisRequest) returns (MarketAnalysis);
}
```

## Claude API Integration Strategy

### Prompt Engineering
- **System Prompt**: Define role as professional forex trading analyst
- **Context Prompt**: Provide current market data, news, historical performance
- **Request Prompt**: Specific request for recommendation with risk parameters
- **Output Format**: Structured JSON response with required fields

### Example Prompt Structure
```
System: You are a professional forex trading analyst with 20+ years of experience.

Context: 
- Current EUR/USD: 1.0750 (up 0.2% today)
- Recent news: ECB hints at rate hike, USD showing weakness
- Historical: Last 5 EUR/USD trades: 3 wins, 2 losses, +2.5% net
- Risk profile: Moderate (max 2% account risk)

Request: Generate a trading recommendation for EUR/USD with medium-term outlook.

Required JSON response format:
{
  "instrument": "EUR_USD",
  "direction": "BUY|SELL",
  "units": 1000,
  "confidence": 0.75,
  "rationale": "Detailed explanation...",
  "stop_loss": 1.0700,
  "take_profit": 1.0850
}
```

### Error Handling
- API rate limiting and retries
- Fallback to heuristic recommendations
- Response validation and parsing
- Graceful degradation

## Configuration Updates

### New Environment Variables
```yaml
# Claude API Configuration
ANTHROPIC_API_KEY=sk-ant-your-key-here
ANTHROPIC_BASE_URL=https://api.anthropic.com
ANTHROPIC_MODEL=claude-3-sonnet-20240229
ANTHROPIC_MAX_TOKENS=2000
ANTHROPIC_TEMPERATURE=0.3

# AI Service Configuration
AI_SERVICE_ENABLED=true
AI_RECOMMENDATION_TTL=3600  # 1 hour
AI_MAX_DAILY_REQUESTS=100
AI_FALLBACK_MODE=heuristic
```

## Database Schema Changes

### New Tables
```sql
-- AI recommendations with enhanced tracking
CREATE TABLE ai_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument VARCHAR(50) NOT NULL,
    direction VARCHAR(4) NOT NULL CHECK (direction IN ('BUY', 'SELL')),
    units DECIMAL(15,2) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,
    rationale TEXT NOT NULL,
    stop_loss DECIMAL(15,8),
    take_profit DECIMAL(15,8),
    time_to_live TIMESTAMP WITH TIME ZONE NOT NULL,
    market_context JSONB NOT NULL,
    news_context JSONB,
    historical_context JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    approved_at TIMESTAMP WITH TIME ZONE,
    executed_trade_id UUID REFERENCES trades(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- AI service usage tracking
CREATE TABLE ai_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID REFERENCES ai_recommendations(id),
    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    response_time_ms INTEGER NOT NULL,
    claude_model VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Market analysis cache
CREATE TABLE market_analysis_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instruments VARCHAR(200) NOT NULL,
    analysis_data JSONB NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Testing Strategy

### Unit Tests
- AI service components
- Data aggregation logic
- Claude API client
- Recommendation processing

### Integration Tests  
- End-to-end recommendation flow
- Database persistence
- API error handling
- Trading execution

### Load Testing
- Claude API rate limits
- Concurrent recommendations
- Database performance
- Memory usage patterns

## Deployment Considerations

### Environment Setup
- Claude API key management
- Rate limiting configuration  
- Fallback mode testing
- Database migrations

### Monitoring
- Claude API usage metrics
- Recommendation success rates
- Trading performance tracking
- Error rate monitoring

### Security
- API key encryption
- Request/response logging limits
- User data protection
- Trading authorization

## Timeline

### Week 1: Core Implementation
- Remove MCP components
- Implement AI service foundation
- Create data aggregation system
- Add Claude API client

### Week 2: API Integration  
- Add REST and gRPC endpoints
- Implement user approval workflow
- Add database persistence
- Create configuration system

### Week 3: Enhancement & Testing
- Add advanced analysis features
- Implement comprehensive testing
- Performance optimization
- Documentation updates

### Week 4: Deployment & Monitoring
- Production deployment
- Monitoring and alerting
- Performance tuning
- User feedback integration

## Success Metrics

- **Functional**: AI service generates valid recommendations
- **Performance**: Sub-5s response time for recommendations
- **Reliability**: 99%+ uptime with graceful fallbacks
- **Accuracy**: Track recommendation success rates
- **Cost**: Optimize Claude API usage costs

This corrected architecture provides a proper AI-powered trading service that aggregates data locally and uses Claude's API for intelligent recommendations, with user approval before trade execution.