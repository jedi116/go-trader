//go:build grpc

package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jedi116/go-trader/internal/broker"
	"github.com/jedi116/go-trader/internal/config"
	"github.com/jedi116/go-trader/internal/database"
	"github.com/jedi116/go-trader/pkg/models"
	v1 "github.com/jedi116/go-trader/proto/gotrader/v1"
)

type tradeServer struct {
	v1.UnimplementedTradeServiceServer
	oanda *broker.OandaMT4Client
	db    *database.Postgres
}

type recServer struct {
	v1.UnimplementedRecommendationServiceServer
	db    *database.Postgres
	oanda *broker.OandaMT4Client
}

type analysisServer struct {
	v1.UnimplementedAnalysisServiceServer
	oanda *broker.OandaMT4Client
}

func (s *tradeServer) PlaceOrder(ctx context.Context, req *v1.PlaceOrderRequest) (*v1.PlaceOrderResponse, error) {
	resp, err := s.oanda.PlaceMarketOrder(req.Instrument, req.Units)
	if err != nil {
		return nil, err
	}
	if s.db != nil && resp != nil {
		tr := structToModelTrade(resp.OrderCreateTransaction.ID, req.Instrument, req.Units)
		_ = s.db.CreateTrade(ctx, &tr)
	}
	return &v1.PlaceOrderResponse{Trade: &v1.Trade{Id: resp.OrderCreateTransaction.ID, Instrument: req.Instrument, Units: req.Units}}, nil
}

func (s *tradeServer) ListTrades(ctx context.Context, req *v1.ListTradesRequest) (*v1.ListTradesResponse, error) {
	var limit int = int(req.Limit)
	if limit == 0 {
		limit = 200
	}
	trs, err := s.db.ListTrades(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Trade, 0, len(trs))
	for _, t := range trs {
		out = append(out, &v1.Trade{Id: t.ID, Instrument: t.Instrument, Units: t.Units})
	}
	return &v1.ListTradesResponse{Trades: out}, nil
}

func (s *recServer) CreateRecommendation(ctx context.Context, req *v1.CreateRecommendationRequest) (*v1.CreateRecommendationResponse, error) {
	// Simplified
	rec := recReqToModel(req)
	id, err := s.db.CreateRecommendation(ctx, &rec)
	if err != nil {
		return nil, err
	}
	return &v1.CreateRecommendationResponse{Recommendation: &v1.Recommendation{Id: id, Instrument: req.Instrument, Units: req.Units, Rationale: req.Rationale}}, nil
}

func (s *recServer) ListRecommendations(ctx context.Context, req *v1.ListRecommendationsRequest) (*v1.ListRecommendationsResponse, error) {
	list, err := s.db.ListRecommendations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*v1.Recommendation, 0, len(list))
	for _, r := range list {
		var rationale string
		if r.Rationale != nil {
			rationale = *r.Rationale
		}
		out = append(out, &v1.Recommendation{Id: r.ID, Instrument: r.Instrument, Units: r.Units, Rationale: rationale})
	}
	return &v1.ListRecommendationsResponse{Recommendations: out}, nil
}

func (s *recServer) AcceptRecommendation(ctx context.Context, req *v1.AcceptRecommendationRequest) (*v1.AcceptRecommendationResponse, error) {
	list, err := s.db.ListRecommendations(ctx)
	if err != nil {
		return nil, err
	}
	var found *v1.Recommendation
	var instr string
	var units float64
	for _, r := range list {
		if r.ID == req.Id {
			instr = r.Instrument
			units = r.Units
			found = &v1.Recommendation{Id: r.ID, Instrument: r.Instrument, Units: r.Units}
			break
		}
	}
	if found == nil {
		return nil, status.Errorf(codes.NotFound, "not found")
	}
	ord, err := s.oanda.PlaceMarketOrder(instr, units)
	if err != nil {
		return nil, err
	}
	_ = s.db.MarkRecommendationExecuted(ctx, req.Id, ord.OrderCreateTransaction.ID)
	return &v1.AcceptRecommendationResponse{Trade: &v1.Trade{Id: ord.OrderCreateTransaction.ID, Instrument: instr, Units: units}, Recommendation: found}, nil
}

func (s *analysisServer) GetCandles(ctx context.Context, req *v1.GetCandlesRequest) (*v1.GetCandlesResponse, error) {
	data, err := s.oanda.GetCandles(req.Instrument, req.Granularity, int(req.Count), nil, nil)
	if err != nil {
		return nil, err
	}
	out := &v1.GetCandlesResponse{Instrument: data.Instrument, Granularity: data.Granularity}
	for _, c := range data.Candles {
		out.Candles = append(out.Candles, &v1.Candle{Time: c.Time.Format("2006-01-02T15:04:05Z07:00"), Open: parseFloat(c.Mid.Open), High: parseFloat(c.Mid.High), Low: parseFloat(c.Mid.Low), Close: parseFloat(c.Mid.Close)})
	}
	return out, nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	oanda := broker.NewOandaMT4Client(os.Getenv("OANDA_API_KEY"), os.Getenv("OANDA_ACCOUNT_ID"), false)
	db, _ := database.NewPostgres(cfg)

	s := grpc.NewServer()
	v1.RegisterTradeServiceServer(s, &tradeServer{oanda: oanda, db: db})
	v1.RegisterRecommendationServiceServer(s, &recServer{oanda: oanda, db: db})
	v1.RegisterAnalysisServiceServer(s, &analysisServer{oanda: oanda})

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("gRPC listening on :9090 (build tag grpc)")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func structToModelTrade(id string, instrument string, units float64) models.Trade {
	dir := "BUY"
	if units < 0 {
		dir = "SELL"
	}
	return models.Trade{ID: id, Instrument: instrument, Direction: dir, Units: units, Status: models.TradeStatusOpen}
}

func recReqToModel(req *v1.CreateRecommendationRequest) models.Recommendation {
	dir := "BUY"
	if req.Direction == v1.Direction_DIRECTION_SELL {
		dir = "SELL"
	}
	var rationale *string
	if req.Rationale != "" {
		r := req.Rationale
		rationale = &r
	}
	return models.Recommendation{Instrument: req.Instrument, Direction: dir, Units: req.Units, Rationale: rationale, Status: models.RecommendationStatusPending}
}

func parseFloat(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
