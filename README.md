## go-trader

### Overview
Production-ready AI-powered forex trading backend written in Go. Features PostgreSQL persistence, REST + optional gRPC, OANDA trading (market + brackets), Brave news, and AI-driven recommendations with risk-based sizing.

### What's implemented ✅
- Database: trades, recommendations, ai_recommendations, market_data, audit_logs
- REST API: health, market data, orders, positions, trades, news, recommendations (create/list/accept)
- AI: context-assembled recommendations with optional explicit units or risk-based sizing; persisted to DB
- OANDA: market orders with optional stop loss / take profit (brackets)
- Brave: news ingestion for context

## Configuration
The server reads `config.yaml` and expands `${VAR}`.

Required env:
- SERVER_HOST, SERVER_PORT
- OANDA_API_KEY, OANDA_ACCOUNT_ID
- BRAVE_API_KEY
- Database: either `DATABASE_URL` or discrete vars (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE)

Example `config.yaml` entries:
```yaml
server:
  port: "${SERVER_PORT}"
  host: "${SERVER_HOST}"

database:
  host: "${DB_HOST}"
  port: "${DB_PORT}"
  user: "${DB_USER}"
  password: "${DB_PASSWORD}"
  database: "${DB_NAME}"
  sslmode: "${DB_SSLMODE}" # e.g. require, verify-ca, disable

broker:
  oanda:
    api_key: "${OANDA_API_KEY}"
    account_id: "${OANDA_ACCOUNT_ID}"
    base_url: "${OANDA_BASE_URL}"

brave:
  api_key: "${BRAVE_API_KEY}"
  base_url: "${BRAVE_BASE_URL}"
```

To enforce TLS with managed Postgres:
```powershell
$env:DATABASE_URL = "postgres://USER:PASS@HOST:5432/DB?sslmode=require"
# or
$env:DB_SSLMODE = "require"
```

## Run
```powershell
$env:SERVER_HOST = "0.0.0.0"
$env:SERVER_PORT = "8080"
$env:OANDA_API_KEY = "<your_oanda_api_key>"
$env:OANDA_ACCOUNT_ID = "<your_oanda_account_id>"
$env:BRAVE_API_KEY = "<your_brave_api_key>"
# database via DATABASE_URL or discrete DB_* vars
```

Migrations:
```bash
go run ./cmd/migrate --dir scripts/migrations --dsn "$DATABASE_URL"
```

Start REST API:
```bash
go run .
```

Health:
```bash
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/health/db
```

## API Usage

### Orders (market with optional brackets)
```bash
# Simple market
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"instrument":"EUR_USD","units":100}'

# Market with SL/TP brackets
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"instrument":"EUR_USD","units":10000,"stop_loss":1.15936,"take_profit":1.16536}'
```

### AI Recommendations
Generate with explicit units:
```bash
curl -X POST http://localhost:8080/api/v1/ai/recommend \
  -H "Content-Type: application/json" \
  -d '{"instruments":["EUR_USD"],"risk_level":"medium","time_horizon":"intra_day","units":10000}'
```
Risk-based sizing (1% of NAV, 25 pip SL):
```bash
curl -X POST http://localhost:8080/api/v1/ai/recommend \
  -H "Content-Type: application/json" \
  -d '{"instruments":["EUR_USD"],"risk_percent":0.01,"stop_loss_pips":25}'
```
AI response includes `stop_loss` and `take_profit`. Accept to place a bracket order:
```bash
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/accept
```

### Market data and news
```bash
curl http://localhost:8080/api/v1/market/EUR_USD
curl http://localhost:8080/api/v1/news/eurusd
```

## Persistence
- `ai_recommendations`: full AI context; mirrored into legacy `recommendations` for compatibility
- `trades`: persisted on order or accept (includes `oanda_trade_id`)
- `market_data`: upserted on market fetch; UUID auto-generated
- `audit_logs`: auto-populated by DB layer on create/update/execute
- `ai_usage_logs` and `market_analysis_cache`: written during recommendation flow

## Notes
- MCP JSON-RPC is deprecated in favor of integrated REST AI endpoints.
- gRPC support is optional; generate protos via `scripts/gen-proto.sh` and run `./cmd/grpcserver` if needed.