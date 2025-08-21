package broker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// OANDA MT4 API Client
type OandaMT4Client struct {
	APIKey     string
	AccountID  string
	BaseURL    string
	HTTPClient *http.Client
}

// Data Structures for OANDA API Responses
type Price struct {
	Instrument string    `json:"instrument"`
	Time       time.Time `json:"time"`
	Bids       []Quote   `json:"bids"`
	Asks       []Quote   `json:"asks"`
}

type Quote struct {
	Price     string `json:"price"`
	Liquidity int    `json:"liquidity"`
}

type Candle struct {
	Complete bool      `json:"complete"`
	Volume   int       `json:"volume"`
	Time     time.Time `json:"time"`
	Mid      OHLC      `json:"mid"`
	Bid      OHLC      `json:"bid"`
	Ask      OHLC      `json:"ask"`
}

type OHLC struct {
	Open  string `json:"o"`
	High  string `json:"h"`
	Low   string `json:"l"`
	Close string `json:"c"`
}

type CandlesResponse struct {
	Instrument  string   `json:"instrument"`
	Granularity string   `json:"granularity"`
	Candles     []Candle `json:"candles"`
}

type Account struct {
	ID                string  `json:"id"`
	Currency          string  `json:"currency"`
	Balance           float64 `json:"balance,string"`
	UnrealizedPL      float64 `json:"unrealizedPL,string"`
	NAV               float64 `json:"NAV,string"`
	MarginUsed        float64 `json:"marginUsed,string"`
	MarginAvailable   float64 `json:"marginAvailable,string"`
	OpenTradeCount    int     `json:"openTradeCount"`
	OpenPositionCount int     `json:"openPositionCount"`
}

type Position struct {
	Instrument   string  `json:"instrument"`
	Long         PosSide `json:"long"`
	Short        PosSide `json:"short"`
	UnrealizedPL float64 `json:"unrealizedPL,string"`
	MarginUsed   float64 `json:"marginUsed,string"`
}

type PosSide struct {
	Units        float64  `json:"units,string"`
	AveragePrice float64  `json:"averagePrice,string"`
	UnrealizedPL float64  `json:"unrealizedPL,string"`
	TradeIDs     []string `json:"tradeIDs"`
}

type Trade struct {
	ID                    string    `json:"id"`
	Instrument            string    `json:"instrument"`
	CurrentUnits          float64   `json:"currentUnits,string"`
	Price                 float64   `json:"price,string"`
	UnrealizedPL          float64   `json:"unrealizedPL,string"`
	MarginUsed            float64   `json:"marginUsed,string"`
	OpenTime              time.Time `json:"openTime"`
	State                 string    `json:"state"`
	InitialUnits          float64   `json:"initialUnits,string"`
	InitialMarginRequired float64   `json:"initialMarginRequired,string"`
	StopLossOrder         *Order    `json:"stopLossOrder,omitempty"`
	TakeProfitOrder       *Order    `json:"takeProfitOrder,omitempty"`
	TrailingStopLossOrder *Order    `json:"trailingStopLossOrder,omitempty"`
}

type Order struct {
	ID               string    `json:"id"`
	CreateTime       time.Time `json:"createTime"`
	Type             string    `json:"type"`
	Instrument       string    `json:"instrument"`
	Units            float64   `json:"units,string"`
	Price            float64   `json:"price,string,omitempty"`
	TimeInForce      string    `json:"timeInForce"`
	State            string    `json:"state"`
	TriggerCondition string    `json:"triggerCondition,omitempty"`
}

// Order request payloads
type MarketOrderRequest struct {
	Order struct {
		Type             string  `json:"type"`
		Instrument       string  `json:"instrument"`
		Units            float64 `json:"units"`
		TimeInForce      string  `json:"timeInForce"`
		PositionFill     string  `json:"positionFill"`
		TakeProfitOnFill *struct {
			Price string `json:"price"`
		} `json:"takeProfitOnFill,omitempty"`
		StopLossOnFill *struct {
			Price string `json:"price"`
		} `json:"stopLossOnFill,omitempty"`
	} `json:"order"`
}

type OrderCreateResponse struct {
	OrderCreateTransaction struct {
		ID string `json:"id"`
	} `json:"orderCreateTransaction"`
}

type Instrument struct {
	Name                        string  `json:"name"`
	Type                        string  `json:"type"`
	DisplayName                 string  `json:"displayName"`
	PipLocation                 int     `json:"pipLocation"`
	DisplayPrecision            int     `json:"displayPrecision"`
	TradeUnitsPrecision         int     `json:"tradeUnitsPrecision"`
	MinimumTradeSize            float64 `json:"minimumTradeSize,string"`
	MaximumTrailingStopDistance float64 `json:"maximumTrailingStopDistance,string"`
	MinimumTrailingStopDistance float64 `json:"minimumTrailingStopDistance,string"`
	MaximumPositionSize         float64 `json:"maximumPositionSize,string"`
	MaximumOrderUnits           float64 `json:"maximumOrderUnits,string"`
	MarginRate                  float64 `json:"marginRate,string"`
}

// Constructor
func NewOandaMT4Client(apiKey, accountID string, live bool) *OandaMT4Client {
	baseURL := "https://api-fxpractice.oanda.com"
	if live {
		baseURL = "https://api-fxtrade.oanda.com"
	}

	return &OandaMT4Client{
		APIKey:    apiKey,
		AccountID: accountID,
		BaseURL:   baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HTTP Request Helper
func (c *OandaMT4Client) makeRequest(method, endpoint string, params url.Values, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	fullURL := c.BaseURL + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept-Datetime-Format", "RFC3339")

	// Only set Content-Type if we have a body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.HTTPClient.Do(req)
}

// 1. Get Real-time Prices
func (c *OandaMT4Client) GetPrices(instruments []string) ([]Price, error) {
	params := url.Values{}
	params.Set("instruments", strings.Join(instruments, ","))

	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s/pricing", c.AccountID), params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Prices []Price `json:"prices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Prices, nil
}

// 2. Get Historical Candles
func (c *OandaMT4Client) GetCandles(instrument, granularity string, count int, from, to *time.Time) (*CandlesResponse, error) {
	params := url.Values{}
	params.Set("granularity", granularity)

	if count > 0 {
		params.Set("count", strconv.Itoa(count))
	}

	if from != nil {
		params.Set("from", from.Format(time.RFC3339))
	}

	if to != nil {
		params.Set("to", to.Format(time.RFC3339))
	}

	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/instruments/%s/candles", instrument), params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result CandlesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// 3. Get Account Information
func (c *OandaMT4Client) GetAccount() (*Account, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s", c.AccountID), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Account Account `json:"account"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.Account, nil
}

// 4. Get Positions
func (c *OandaMT4Client) GetPositions() ([]Position, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s/positions", c.AccountID), nil, nil)
	log.Print(resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error reading body: %v", err)
		return nil, err
	}

	var result struct {
		Positions []Position `json:"positions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("error reading body: %v", err)
		return nil, err
	}

	return result.Positions, nil
}

// 5. Get Open Trades
func (c *OandaMT4Client) GetTrades() ([]Trade, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s/trades", c.AccountID), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Trades []Trade `json:"trades"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Trades, nil
}

// 6. Get Pending Orders
func (c *OandaMT4Client) GetOrders() ([]Order, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s/orders", c.AccountID), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Orders []Order `json:"orders"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Orders, nil
}

// 7. Get Available Instruments
func (c *OandaMT4Client) GetInstruments() ([]Instrument, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/accounts/%s/instruments", c.AccountID), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Instruments []Instrument `json:"instruments"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Instruments, nil
}

// 8. Get Order Book (Market Depth)
func (c *OandaMT4Client) GetOrderBook(instrument string) (map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/instruments/%s/orderBook", instrument), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// 9. Get Position Book (Client Sentiment)
func (c *OandaMT4Client) GetPositionBook(instrument string) (map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/v3/instruments/%s/positionBook", instrument), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// 10. Get Account Summary with Calculated Metrics
func (c *OandaMT4Client) GetAccountSummary() (map[string]interface{}, error) {
	account, err := c.GetAccount()
	if err != nil {
		return nil, err
	}

	positions, err := c.GetPositions()
	if err != nil {
		return nil, err
	}

	trades, err := c.GetTrades()
	if err != nil {
		return nil, err
	}

	// Calculate additional metrics
	summary := map[string]interface{}{
		"account_id":       account.ID,
		"currency":         account.Currency,
		"balance":          account.Balance,
		"nav":              account.NAV,
		"unrealized_pl":    account.UnrealizedPL,
		"margin_used":      account.MarginUsed,
		"margin_available": account.MarginAvailable,
		"margin_rate":      (account.MarginUsed / account.NAV) * 100,
		"equity":           account.Balance + account.UnrealizedPL,
		"open_trades":      len(trades),
		"open_positions":   len(positions),
		"free_margin":      account.NAV - account.MarginUsed,
		"margin_level":     (account.NAV / account.MarginUsed) * 100,
	}

	// Add position details
	var totalProfit, totalLoss float64
	for _, pos := range positions {
		if pos.UnrealizedPL > 0 {
			totalProfit += pos.UnrealizedPL
		} else {
			totalLoss += pos.UnrealizedPL
		}
	}

	summary["total_profit"] = totalProfit
	summary["total_loss"] = totalLoss
	summary["net_exposure"] = totalProfit + totalLoss

	return summary, nil
}

// 11. Get Multi-Timeframe Price Data
func (c *OandaMT4Client) GetMultiTimeframeData(instrument string, timeframes []string, count int) (map[string]*CandlesResponse, error) {
	result := make(map[string]*CandlesResponse)

	for _, tf := range timeframes {
		candles, err := c.GetCandles(instrument, tf, count, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s data for %s: %w", tf, instrument, err)
		}
		result[tf] = candles
	}

	return result, nil
}

// 12. Get Market Status and Trading Hours
func (c *OandaMT4Client) GetMarketStatus(instruments []string) (map[string]interface{}, error) {
	prices, err := c.GetPrices(instruments)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"timestamp":   time.Now(),
		"market_open": len(prices) > 0,
		"instruments": make(map[string]interface{}),
	}

	for _, price := range prices {
		status["instruments"].(map[string]interface{})[price.Instrument] = map[string]interface{}{
			"tradeable":   len(price.Bids) > 0 && len(price.Asks) > 0,
			"last_update": price.Time,
			"bid_liquidity": func() int {
				if len(price.Bids) > 0 {
					return price.Bids[0].Liquidity
				}
				return 0
			}(),
			"ask_liquidity": func() int {
				if len(price.Asks) > 0 {
					return price.Asks[0].Liquidity
				}
				return 0
			}(),
		}
	}

	return status, nil
}

// 13. Place Market Order (Buy +Units, Sell -Units)
func (c *OandaMT4Client) PlaceMarketOrder(instrument string, units float64) (*OrderCreateResponse, error) {
	var payload MarketOrderRequest
	payload.Order.Type = "MARKET"
	payload.Order.Instrument = instrument
	payload.Order.Units = units
	payload.Order.TimeInForce = "FOK"
	payload.Order.PositionFill = "DEFAULT"

	resp, err := c.makeRequest("POST",
		fmt.Sprintf("/v3/accounts/%s/orders", c.AccountID),
		nil, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("order failed status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result OrderCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// 13b. Place Market Order with optional SL/TP
func (c *OandaMT4Client) PlaceMarketOrderWithBrackets(instrument string, units float64, stopLoss, takeProfit *float64) (*OrderCreateResponse, error) {
	var payload MarketOrderRequest
	payload.Order.Type = "MARKET"
	payload.Order.Instrument = instrument
	payload.Order.Units = units
	payload.Order.TimeInForce = "FOK"
	payload.Order.PositionFill = "DEFAULT"
	if takeProfit != nil && *takeProfit > 0 {
		price := fmt.Sprintf("%.5f", *takeProfit)
		payload.Order.TakeProfitOnFill = &struct {
			Price string `json:"price"`
		}{Price: price}
	}
	if stopLoss != nil && *stopLoss > 0 {
		price := fmt.Sprintf("%.5f", *stopLoss)
		payload.Order.StopLossOnFill = &struct {
			Price string `json:"price"`
		}{Price: price}
	}

	resp, err := c.makeRequest("POST",
		fmt.Sprintf("/v3/accounts/%s/orders", c.AccountID),
		nil, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("order failed status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result OrderCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
