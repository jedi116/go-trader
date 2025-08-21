# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
This is a production-ready AI-powered forex trading backend written in Go with complete PostgreSQL persistence, multiple API interfaces (REST, gRPC), and comprehensive trading capabilities. The system integrates with OANDA for trading, Brave Search for news analysis, and provides AI-powered recommendations via Claude API integration.

## Core Architecture
The system follows a modular, production-ready architecture with complete database persistence:

- **main.go**: Primary HTTP API server with database integration
- **cmd/grpcserver**: High-performance gRPC server (3 services, 6 methods)
- **cmd/migrate**: Database migration runner with tracking
- **internal/api**: REST endpoints with database persistence
- **internal/database**: PostgreSQL operations with connection pooling
- **internal/broker**: Complete OANDA trading client
- **internal/config**: Configuration with environment variable expansion
- **internal/ai**: AI service for Claude API integration and recommendation engine
- **pkg/models**: Database models with proper tags and relationships

The application supports three operational modes:
1. **REST API Server**: Full trading API with database persistence
2. **gRPC Server**: High-performance service-to-service communication
3. **Migration Runner**: Database schema management

## Development Commands

### Database Setup (Required)
```bash
# Run migrations before first use
go run ./cmd/migrate --dir scripts/migrations --dsn "$DATABASE_URL"
```

### Build and Run
```bash
# Build the project
go build -o go-trader.exe .

# Run primary HTTP API server (11 endpoints with DB)
go run .

# Run gRPC server (3 services, 6 methods)
bash scripts/gen-proto.sh  # Generate protobuf code once
go run -tags=grpc ./cmd/grpcserver

# Run migrations
go run ./cmd/migrate --dir scripts/migrations --dsn "your-postgres-dsn"
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Test specific package
go test ./internal/database
go test ./internal/api
```

### Code Quality
```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy
```

## Configuration
The system uses `config.yaml` with environment variable expansion (`${VAR}` syntax). 

**Database (Required):**
- `DATABASE_URL`: PostgreSQL connection string
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: Database components

**Server:**
- `SERVER_HOST`, `SERVER_PORT`: HTTP API server configuration

**Trading (Required):**
- `OANDA_API_KEY`, `OANDA_ACCOUNT_ID`: OANDA trading credentials
- `OANDA_BASE_URL`: Environment (practice: api-fxpractice.oanda.com, live: api-fxtrade.oanda.com)

**News (Required):**
- `BRAVE_API_KEY`: Brave Search API key
- `BRAVE_BASE_URL`: Brave Search base URL (optional)

**AI (Required for Recommendations):**
- `ANTHROPIC_API_KEY`: Claude API key for AI-powered trade recommendations
- `ANTHROPIC_MODEL`: Claude model to use (default: claude-3-sonnet-20240229)
- `ANTHROPIC_MAX_TOKENS`: Maximum tokens per request (default: 2000)

## Key Integration Points

### Database Layer (`internal/database/postgres.go`)
- **Complete PostgreSQL Integration**: Connection pooling, health checks, audit trails
- **CRUD Operations**: Trades, recommendations, market data with soft deletes
- **Migration System**: Tracked migrations in `scripts/migrations/`
- **Performance**: Proper indexing, connection limits, timeouts

### REST API Endpoints (`internal/api/server.go`)
**Core Trading Endpoints:**
- `GET /api/v1/health`, `/health/db`: Service and database health
- `GET /api/v1/market/:symbol`: Real-time data with DB persistence
- `POST /api/v1/orders`: Market orders with trade persistence
- `GET /api/v1/trades`, `DELETE /api/v1/trades/:id`: Trade management
- `GET /api/v1/positions`: OANDA account positions
- `GET /api/v1/news/:query`: Financial news search

**AI Recommendation Endpoints:**
- `POST /api/v1/ai/recommend`: Generate Claude-powered recommendations
- `GET /api/v1/ai/recommendations`: List pending recommendations
- `POST /api/v1/ai/recommendations/:id/approve`: Approve and execute trades
- `DELETE /api/v1/ai/recommendations/:id`: Reject recommendations

### gRPC Services (`cmd/grpcserver/main.go`)
**3 Services with Enhanced Methods:**
- **TradeService**: PlaceOrder, ListTrades
- **AIService**: GenerateRecommendation, ListRecommendations, ApproveRecommendation
- **AnalysisService**: GetCandles, GetMarketAnalysis

### AI Service System (`internal/ai/`)
**Claude API Integration:**
- **Data Aggregation**: Combines market data, news, and historical analysis
- **Prompt Engineering**: Structured prompts for trading recommendations
- **Response Processing**: Parses and validates Claude's JSON recommendations
- **User Approval Workflow**: Presents recommendations before execution
- **Trade Execution**: Converts approved recommendations to OANDA trades

### OANDA Trading Client (`internal/broker/market.data.go`)
- **Complete API Coverage**: Prices, candles, positions, orders, account data
- **Environment Support**: Practice and live trading environments
- **Trade Execution**: Market order placement with response tracking

## Development Notes

### Dependency Management
- **Go Modules**: Complete dependency management with `go.mod` and `go.sum`
- **Key Dependencies**: Gin (HTTP), gRPC, lib/pq (PostgreSQL), Viper (config)
- **Build Tags**: gRPC server uses `grpc` build tag for conditional compilation

### Code Organization
- **Multi-Interface Architecture**: REST, gRPC, and MCP servers with shared components
- **Database-First Design**: All operations persist to PostgreSQL with fallback support
- **Modular Structure**: Clear separation between API layers, database operations, and external integrations
- **Production-Ready**: Connection pooling, health checks, audit logging, soft deletes

### Current Capabilities
- **Complete Database Persistence**: All trades, recommendations, and market data stored
- **Multi-API Support**: REST and gRPC interfaces with comprehensive endpoints
- **Trading Operations**: Full OANDA integration with order execution and tracking
- **AI-Powered Recommendations**: Claude API integration for intelligent trade suggestions
- **Migration System**: Database schema versioning and deployment support

### Extension Points
- **Advanced Analytics**: Historical data foundation ready for complex analysis
- **Portfolio Optimization**: AI service can be extended for portfolio management
- **Risk Management**: Enhanced position sizing and risk assessment
- **Additional Markets**: Architecture supports extending beyond forex
- **Multi-Model AI**: Support for different AI providers and models

## Testing Strategy

### REST API Testing
```bash
# Health checks
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/health/db

# Trading operations
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"instrument":"EUR_USD","units":100}'

curl http://localhost:8080/api/v1/trades
curl http://localhost:8080/api/v1/positions

# Market data with DB persistence
curl http://localhost:8080/api/v1/market/EUR_USD

# AI recommendations
curl -X POST http://localhost:8080/api/v1/ai/recommend \
  -H "Content-Type: application/json" \
  -d '{"instruments":["EUR_USD"],"risk_level":"moderate","time_horizon":"medium"}'

curl http://localhost:8080/api/v1/ai/recommendations
```

### gRPC Testing
```bash
# Install grpcurl for testing
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Test services
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9090 gotrader.v1.TradeService/ListTrades
grpcurl -plaintext localhost:9090 gotrader.v1.AIService/ListRecommendations
```

### Database Testing
```bash
# Test migrations
go run ./cmd/migrate --dir scripts/migrations --dsn "$DATABASE_URL"

# Connect to database and verify schema
psql $DATABASE_URL -c "\dt"  # List tables
```