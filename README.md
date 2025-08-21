## go-trader

### Overview
Production-ready AI-powered forex trading backend written in Go. Features complete PostgreSQL persistence, multiple API interfaces (REST, gRPC), OANDA trading integration, and AI-driven recommendations. Supports real-time trading, historical analysis, and seamless AI model integration.

### Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST Client   │    │   gRPC Client   │    │   AI Service    │
│   (Web/Mobile)  │    │   (Services)    │    │  (Claude API)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │   Go-Trader Server      │
                    │  ┌─────────────────────┐│
                    │  │   11 REST Endpoints ││
                    │  │   6 gRPC Methods    ││
                    │  │     AI Service      ││
                    │  └─────────────────────┘│
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │    PostgreSQL DB        │
                    │  • trades (with audit)  │
                    │  • recommendations      │
                    │  • market_data (OHLC)   │
                    │  • audit_logs           │
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │   External APIs         │
                    │  • OANDA (Trading)      │
                    │  • Brave (News)         │
                    │  • Anthropic (AI)       │
                    └─────────────────────────┘
```

### What's implemented ✅
**Database Layer (PostgreSQL)**
- Complete schema: trades, recommendations, market_data, audit_logs
- Migration system with tracking (`cmd/migrate`)
- CRUD operations with soft deletes and audit trails
- Connection pooling and health monitoring

**REST API (Gin) - 11 Endpoints**
- `GET /api/v1/health` - Service health check
- `GET /api/v1/health/db` - Database connectivity check  
- `GET /api/v1/market/:symbol` - Real-time market data with DB persistence
- `POST /api/v1/orders` - Place market orders via OANDA
- `GET /api/v1/positions` - Current positions from OANDA
- `GET /api/v1/trades` - Historical trades from database
- `DELETE /api/v1/trades/:id` - Soft delete trades
- `GET /api/v1/news/:query` - Financial news via Brave API
- `POST /api/v1/recommendations` - Create AI recommendations
- `GET /api/v1/recommendations` - List all recommendations
- `POST /api/v1/recommendations/:id/accept` - Execute recommendations
- `DELETE /api/v1/recommendations/:id` - Soft delete recommendations

**gRPC Server (3 Services, 6 Methods)**
- `TradeService`: PlaceOrder, ListTrades
- `RecommendationService`: CreateRecommendation, ListRecommendations, AcceptRecommendation
- `AnalysisService`: GetCandles
- Protocol buffers defined with proper Go code generation

**MCP Server (JSON-RPC) - 4 Tools (deprecated; being replaced by direct AI service)**
- `brave.news` - Search financial news with context
- `oanda.prices` - Real-time price feeds
- `oanda.candles` - Historical OHLC data
- `ai.recommend` - AI-powered trade recommendations

**Trading Integration**
- Complete OANDA API client with market orders, positions, pricing
- Real-time and historical data collection
- Trade execution tracking and persistence
- Support for practice and live environments

**AI & Analysis**
- News-based sentiment analysis via Brave Search
- AI recommender with Anthropic API integration (placeholder ready)
- Historical market data collection and analysis
- Context-aware trade recommendations

## Configuration
The server reads `config.yaml` and expands environment variables referenced as `${VAR}`.

Required environment variables:
- **SERVER_HOST**, **SERVER_PORT**
- **OANDA_API_KEY**, **OANDA_ACCOUNT_ID**
- Optional: **OANDA_BASE_URL** (`https://api-fxpractice.oanda.com` or `https://api-fxtrade.oanda.com`)
- **BRAVE_API_KEY**
- Optional: **BRAVE_BASE_URL** (default `https://api.search.brave.com`)
- Optional (AI): **ANTHROPIC_API_KEY** (enables non-heuristic recommender path)

Example `config.yaml` entries already present:
```yaml
server:
  port: "${SERVER_PORT}"
  host: "${SERVER_HOST}"

broker:
  oanda:
    api_key: "${OANDA_API_KEY}"
    account_id: "${OANDA_ACCOUNT_ID}"
    base_url: "${OANDA_BASE_URL}"

brave:
  api_key: "${BRAVE_API_KEY}"
  base_url: "${BRAVE_BASE_URL}"
```

## Run
### Windows PowerShell (current session)
```powershell
$env:SERVER_HOST = "0.0.0.0"
$env:SERVER_PORT = "8080"
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/go_trader?sslmode=disable"
$env:DB_HOST = "localhost"
$env:DB_PORT = "5432"
$env:DB_USER = "postgres"
$env:DB_PASSWORD = "postgres"
$env:DB_NAME = "go_trader"
$env:OANDA_API_KEY = "<your_oanda_api_key>"
$env:OANDA_ACCOUNT_ID = "<your_oanda_account_id>"
$env:BRAVE_API_KEY = "<your_brave_api_key>"
```

### Database Setup
```bash
# Run migrations (required for database functionality)
go run ./cmd/migrate --dir scripts/migrations --dsn "$DATABASE_URL"
```

### Operational Modes

#### 1. REST API Server (Primary)
```bash
go run .
# Serves on :8080 with 11 REST endpoints
# Includes database persistence and OANDA integration
```

Health checks:
```bash
curl http://localhost:8080/api/v1/health      # Service health
curl http://localhost:8080/api/v1/health/db   # Database connectivity
```

#### 2. AI Service (Integrated)
The AI service is integrated into the REST server; use the AI endpoints below.

#### 3. gRPC Server (High Performance)
```bash
# Generate protobuf code (one-time setup)
bash scripts/gen-proto.sh

# Run gRPC server
go run -tags=grpc ./cmd/grpcserver
# Serves on :9090 with 3 services and 6 RPC methods
```

#### 4. Migration Runner
```bash
go run ./cmd/migrate --dir scripts/migrations --dsn "your-postgres-dsn"
# Applies incremental SQL migrations with tracking
```

## API Usage Examples

### REST API (11 Endpoints)

#### Trading Operations
```bash
# Place market order (auto-persisted to database)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"instrument":"EUR_USD","units":100}'

# Get current positions from OANDA
curl http://localhost:8080/api/v1/positions

# List historical trades from database
curl http://localhost:8080/api/v1/trades

# Get real-time market data (auto-saved to market_data table)
curl http://localhost:8080/api/v1/market/EUR_USD
```

#### AI Recommendations with Database Persistence
```bash
# Create AI-powered recommendation (saved to database)
curl -X POST http://localhost:8080/api/v1/recommendations \
  -H "Content-Type: application/json" \
  -d '{"instrument":"EUR_USD","direction":"BUY","units":100,"rationale":"AI analysis"}'

# List all recommendations from database
curl http://localhost:8080/api/v1/recommendations

# Execute recommendation as live trade
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/accept
```

#### News & Analysis
```bash
# Search financial news via Brave API
curl http://localhost:8080/api/v1/news/eurusd

# Database health check
curl http://localhost:8080/api/v1/health/db
```

### gRPC API (3 Services, 6 Methods)
```bash
# TradeService
grpcurl -plaintext localhost:9090 gotrader.v1.TradeService/PlaceOrder
grpcurl -plaintext localhost:9090 gotrader.v1.TradeService/ListTrades

# RecommendationService  
grpcurl -plaintext localhost:9090 gotrader.v1.RecommendationService/CreateRecommendation
grpcurl -plaintext localhost:9090 gotrader.v1.RecommendationService/ListRecommendations
grpcurl -plaintext localhost:9090 gotrader.v1.RecommendationService/AcceptRecommendation

# AnalysisService
grpcurl -plaintext localhost:9090 gotrader.v1.AnalysisService/GetCandles
```

## AI Endpoints
### AI Recommendation Endpoints
- POST `/api/v1/ai/recommend` - Generate new recommendation
- GET `/api/v1/ai/status` - AI service health

## Database Schema & Persistence

### PostgreSQL Tables
- **`trades`** - Complete trade lifecycle with OANDA integration, audit trails
- **`recommendations`** - AI-generated suggestions with execution tracking
- **`market_data`** - Historical OHLC data with timeframe support
- **`audit_logs`** - Complete audit trail for all operations
- **`schema_migrations`** - Migration tracking system

### Key Features
- UUID primary keys for distributed system compatibility
- Soft deletes for data recovery and compliance
- JSONB support for flexible market conditions storage
- Proper indexing for query performance
- Foreign key relationships for data integrity
- Connection pooling with health monitoring

## AI & Analysis Components

### AI Service (`internal/ai/...`)
- **Aggregator**: Market + news + historical context assembly
- **Claude Client**: Placeholder for Anthropic API integration
- **Context-Aware**: Uses real-time news data for recommendations
- **Structured Output**: Consistent recommendation format with rationale

### Market Analysis
- **Real-time Data**: Live price feeds from OANDA with database persistence
- **Historical Analysis**: Multi-timeframe OHLC data collection and storage
- **News Integration**: Financial news search and sentiment analysis
- **Performance Tracking**: Complete audit trail of all trading decisions

## Project Structure (Production-Ready)
```
cmd/
├── grpcserver/     # gRPC server (build tag: grpc)
├── mcp/           # MCP JSON-RPC server for AI integration
├── migrate/       # Database migration runner
└── server/        # (reserved for future use)

internal/
├── api/           # REST API with 11 endpoints (Gin)
├── ai/            # AI service (aggregator + Claude client)
├── broker/        # OANDA trading client
├── config/        # Configuration with env expansion
├── database/      # PostgreSQL operations with pooling
└── news/          # Brave news client

pkg/
├── models/        # Data models with database tags
└── utils/         # Shared utilities

proto/             # Protocol buffer definitions (3 services)
scripts/           # Migration files and build scripts
```

## Implementation Status & Next Steps

### ✅ Current Implementation
- Basic REST API with Gin framework
- OANDA API integration for trading operations
- Brave Search news integration
- MCP server with JSON-RPC tools
- Configuration system with environment variables
- In-memory recommendation system

### 🚧 In Progress - See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)

**Phase 1: Database Integration (Week 1)**
- PostgreSQL schema design and migration system
- Replace in-memory storage with persistent database
- Enhanced data models and validation

**Phase 2: gRPC Server Implementation (Week 2)**
- Protocol Buffer definitions for all services
- gRPC server alongside REST endpoints
- Real-time market data streaming capabilities

**Phase 3: Enhanced MCP Tools (Week 3)**
- Database query/insert tools for AI model integration
- Advanced analysis tools for historical data
- Enhanced AI recommender with real API integration

**Phase 4: Docker Deployment (Week 3)**
- Multi-stage Dockerfile and Docker Compose setup
- Production-ready containerization
- Service networking and dependencies

**Phase 5: Testing & Documentation (Week 4)**
- Comprehensive testing suite
- API documentation and deployment guides
- Monitoring and observability setup

### 🎯 Target Architecture
```
REST API ←→ gRPC Services ←→ MCP Tools
    ↓              ↓              ↓
         PostgreSQL Database
              ↓
    Docker Containerized Deployment
```

**Key Features Being Added:**
- PostgreSQL for persistent trade data and historical analysis
- gRPC server for AI model integration
- Enhanced MCP tools for database operations
- Docker deployment for production readiness
- Comprehensive testing and monitoring

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for detailed step-by-step process and timeline.