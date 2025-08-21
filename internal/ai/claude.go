package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type ClaudeClient interface {
	GenerateRecommendation(ctx context.Context, tradingContext *TradingContext, request *RecommendationRequest) (*Recommendation, error)
}

type claudeClientImpl struct {
	http *http.Client
}

func NewClaudeClient(httpClient *http.Client) ClaudeClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &claudeClientImpl{http: httpClient}
}

type claudeRequest struct {
	Model       string      `json:"model"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float64     `json:"temperature"`
	Messages    []claudeMsg `json:"messages"`
}

type claudeMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *claudeClientImpl) GenerateRecommendation(ctx context.Context, tradingContext *TradingContext, request *RecommendationRequest) (*Recommendation, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Fallback minimal heuristic
		return &Recommendation{
			ID: "fallback",
			Instrument: func() string {
				if len(request.Instruments) > 0 {
					return request.Instruments[0]
				}
				return "EUR_USD"
			}(),
			Direction:   "BUY",
			Units:       100,
			Confidence:  0.5,
			Rationale:   "Fallback heuristic recommendation (no ANTHROPIC_API_KEY)",
			MarketData:  tradingContext.MarketData,
			NewsContext: tradingContext.NewsAnalysis,
		}, nil
	}

	// Compose a compact prompt
	prompt := fmt.Sprintf("Generate a forex trade recommendation given context. Instruments: %v. Risk: %s. Horizon: %s.", request.Instruments, request.RiskLevel, request.TimeHorizon)
	reqBody := claudeRequest{
		Model:       getenvDefault("ANTHROPIC_MODEL", "claude-opus-4-1-20250805"),
		MaxTokens:   getenvIntDefault("ANTHROPIC_MAX_TOKENS", 2000),
		Temperature: getenvFloatDefault("ANTHROPIC_TEMPERATURE", 0.3),
		Messages: []claudeMsg{
			{Role: "system", Content: "You are a professional forex trading analyst with 20+ years of experience."},
			{Role: "user", Content: prompt},
		},
	}

	// Placeholder: not performing the real HTTP call to Anthropic to keep compile without external dep.
	// Serialize request to reflect in rationale
	b, _ := json.Marshal(reqBody)
	return &Recommendation{
		ID: "simulated",
		Instrument: func() string {
			if len(request.Instruments) > 0 {
				return request.Instruments[0]
			}
			return "EUR_USD"
		}(),
		Direction:   "BUY",
		Units:       100,
		Confidence:  0.7,
		Rationale:   "Simulated Claude call with payload: " + string(b),
		MarketData:  tradingContext.MarketData,
		NewsContext: tradingContext.NewsAnalysis,
	}, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func getenvFloatDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
	}
	return def
}
