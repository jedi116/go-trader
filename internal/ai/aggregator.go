package ai

import (
	"context"
	"time"
)

type Aggregator interface {
	GatherMarketData(ctx context.Context, instruments []string) (*MarketContext, error)
	GatherNewsData(ctx context.Context, instruments []string) ([]NewsItem, error)
	GatherHistoricalData(ctx context.Context, instruments []string) (*HistoricalContext, error)
	AssembleContext(market *MarketContext, news []NewsItem, historical *HistoricalContext) *TradingContext
}

type aggregatorImpl struct {
	marketFetcher func(ctx context.Context, instruments []string) (*MarketContext, error)
	newsFetcher   func(ctx context.Context, instruments []string) ([]NewsItem, error)
	histFetcher   func(ctx context.Context, instruments []string) (*HistoricalContext, error)
}

func NewAggregator(
	marketFetcher func(ctx context.Context, instruments []string) (*MarketContext, error),
	newsFetcher func(ctx context.Context, instruments []string) ([]NewsItem, error),
	histFetcher func(ctx context.Context, instruments []string) (*HistoricalContext, error),
) Aggregator {
	return &aggregatorImpl{marketFetcher: marketFetcher, newsFetcher: newsFetcher, histFetcher: histFetcher}
}

func (a *aggregatorImpl) GatherMarketData(ctx context.Context, instruments []string) (*MarketContext, error) {
	return a.marketFetcher(ctx, instruments)
}

func (a *aggregatorImpl) GatherNewsData(ctx context.Context, instruments []string) ([]NewsItem, error) {
	return a.newsFetcher(ctx, instruments)
}

func (a *aggregatorImpl) GatherHistoricalData(ctx context.Context, instruments []string) (*HistoricalContext, error) {
	return a.histFetcher(ctx, instruments)
}

func (a *aggregatorImpl) AssembleContext(market *MarketContext, news []NewsItem, historical *HistoricalContext) *TradingContext {
	return &TradingContext{
		Timestamp:    time.Now(),
		MarketData:   market,
		NewsAnalysis: news,
		Historical:   historical,
	}
}
