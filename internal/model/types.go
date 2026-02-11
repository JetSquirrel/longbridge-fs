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
