package model

// AccountState is the JSON structure for /account/state.json
type AccountState struct {
	UpdatedAt string       `json:"updated_at"`
	Cash      []CashEntry  `json:"cash"`
	Positions []PositionEx `json:"positions"`
	Orders    []OrderRef   `json:"orders"`
}

// CashEntry represents cash balance for a currency
type CashEntry struct {
	Currency  string  `json:"currency"`
	Available float64 `json:"available"`
	Frozen    float64 `json:"frozen"`
	Settling  float64 `json:"settling"`
	Withdraw  float64 `json:"withdraw"`
}

// PositionEx represents a stock position with extended info
type PositionEx struct {
	Symbol    string  `json:"symbol"`
	Quantity  string  `json:"quantity"`
	Available string  `json:"available"`
	CostPrice float64 `json:"cost_price"`
	Currency  string  `json:"currency"`
	Market    string  `json:"market"`
}

// OrderRef tracks the mapping between beancount intent_id and API order_id
type OrderRef struct {
	IntentID string `json:"intent_id"`
	OrderID  string `json:"order_id"`
	Status   string `json:"status"`
}

// Entry is a parsed beancount entry (ORDER, EXECUTION, etc.)
type Entry struct {
	Type     string
	Meta     map[string]string
	RawLines []string // original text lines for compaction
}

// ParsedOrder is a trade order extracted from a beancount ORDER entry
type ParsedOrder struct {
	IntentID  string
	Side      string
	Symbol    string
	Qty       string
	OrderType string
	TIF       string
	Price     string // for LIMIT orders
	Market    string // default: US
}

// --- Quote JSON types ---

// QuoteOverview is the JSON structure for /quote/hold/{SYMBOL}/overview.json
type QuoteOverview struct {
	Symbol     string  `json:"symbol"`
	Last       float64 `json:"last"`
	Open       float64 `json:"open"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	PrevClose  float64 `json:"prev_close"`
	Volume     int64   `json:"volume"`
	Turnover   float64 `json:"turnover"`
	PreMarket  float64 `json:"pre_market,omitempty"`
	PostMarket float64 `json:"post_market,omitempty"`
	Change     float64 `json:"change"`
	ChangePct  float64 `json:"change_pct"`
	UpdatedAt  string  `json:"updated_at"`
}

// Candlestick is one K-line bar in JSON output
type Candlestick struct {
	Date     string  `json:"date"`
	Open     float64 `json:"open"`
	Close    float64 `json:"close"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Volume   int64   `json:"volume"`
	Turnover float64 `json:"turnover"`
}

// IntradayPoint is one minute-bar in JSON output
type IntradayPoint struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`
	Volume   int64   `json:"volume"`
	AvgPrice float64 `json:"avg_price"`
}

// --- PnL types ---

// PnLReport is the JSON structure for /account/pnl.json
type PnLReport struct {
	UpdatedAt  string        `json:"updated_at"`
	Positions  []PositionPnL `json:"positions"`
	TotalCost  float64       `json:"total_cost"`
	TotalValue float64       `json:"total_value"`
	TotalPnL   float64       `json:"total_pnl"`
	TotalPct   float64       `json:"total_pnl_pct"`
}

// PositionPnL is per-symbol P&L
type PositionPnL struct {
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	CostPrice float64 `json:"cost_price"`
	LastPrice float64 `json:"last_price"`
	Currency  string  `json:"currency"`
	Cost      float64 `json:"cost"`
	Value     float64 `json:"value"`
	PnL       float64 `json:"pnl"`
	PnLPct    float64 `json:"pnl_pct"`
}

// --- Portfolio types ---

// Portfolio is the JSON structure for /quote/portfolio.json
type Portfolio struct {
	UpdatedAt string          `json:"updated_at"`
	Holdings  []PortfolioItem `json:"holdings"`
}

// PortfolioItem is one holding with live quote
type PortfolioItem struct {
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	CostPrice float64 `json:"cost_price"`
	Last      float64 `json:"last"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"change_pct"`
	PnL       float64 `json:"pnl"`
	PnLPct    float64 `json:"pnl_pct"`
	Currency  string  `json:"currency"`
	Market    string  `json:"market"`
}

// --- Risk Control types ---

// RiskRule defines stop-loss/take-profit for a symbol
type RiskRule struct {
	StopLoss   float64 `json:"stop_loss,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`
	Side       string  `json:"side,omitempty"` // default: SELL (close position)
	Qty        string  `json:"qty,omitempty"`  // default: all available
}

