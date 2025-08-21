package ai

import (
	"context"
)

type serviceImpl struct {
	agg    Aggregator
	claude ClaudeClient
}

func NewService(agg Aggregator, claude ClaudeClient) Service {
	return &serviceImpl{agg: agg, claude: claude}
}

func (s *serviceImpl) GenerateRecommendation(ctx context.Context, request *RecommendationRequest) (*Recommendation, error) {
	market, err := s.agg.GatherMarketData(ctx, request.Instruments)
	if err != nil {
		return nil, err
	}
	news, err := s.agg.GatherNewsData(ctx, request.Instruments)
	if err != nil {
		return nil, err
	}
	hist, err := s.agg.GatherHistoricalData(ctx, request.Instruments)
	if err != nil {
		return nil, err
	}
	ctxObj := s.agg.AssembleContext(market, news, hist)
	return s.claude.GenerateRecommendation(ctx, ctxObj, request)
}

func (s *serviceImpl) ExecuteRecommendation(ctx context.Context, id string) (*Trade, error) {
	// Placeholder – execution will be wired to OANDA and DB in Phase 4
	return &Trade{ID: id, Instrument: "", Units: 0}, nil
}

func (s *serviceImpl) GetRecommendationStatus(ctx context.Context, id string) (*RecommendationStatus, error) {
	// Placeholder – status from DB in Phase 4
	return &RecommendationStatus{ID: id, Status: "PENDING"}, nil
}
