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
	IntentID    string
	Side        string
	Symbol      string
	Qty         string
	OrderType   string
	TIF         string
	Price       string // for LIMIT orders
	Market      string // default: US
	// Phase 1: Extended traceability fields
	Source      string   // manual, rebalance, risk_trigger
	RebalanceID string   // links to portfolio rebalance
	SignalRefs  []string // triggering signals (comma-separated in beancount)
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

// RiskRule defines stop-loss/take-profit for a symbol (legacy, L5 reactive)
type RiskRule struct {
	StopLoss   float64 `json:"stop_loss,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`
	Side       string  `json:"side,omitempty"` // default: SELL (close position)
	Qty        string  `json:"qty,omitempty"`  // default: all available
}

// --- Phase 1: L4 Risk Gate types ---

// RiskPolicy is the main risk control policy configuration
type RiskPolicy struct {
	Version              int               `json:"version"`
	Enabled              bool              `json:"enabled"`
	Mode                 string            `json:"mode"` // ENFORCE, WARN, DISABLED
	PreTradeChecks       bool              `json:"pre_trade_checks"`
	PostTradeMonitoring  bool              `json:"post_trade_monitoring"`
	DailyLossLimit       DailyLossLimit    `json:"daily_loss_limit"`
	OrderFrequency       OrderFrequency    `json:"order_frequency"`
}

type DailyLossLimit struct {
	Enabled    bool    `json:"enabled"`
	MaxLossPct float64 `json:"max_loss_pct"`
	Action     string  `json:"action"` // HALT, WARN
}

type OrderFrequency struct {
	Enabled           bool `json:"enabled"`
	MaxOrdersPerHour  int  `json:"max_orders_per_hour"`
	MaxOrdersPerDay   int  `json:"max_orders_per_day"`
}

// PreTradeRules defines pre-trade validation rules
type PreTradeRules struct {
	MaxSingleOrderPct           float64  `json:"max_single_order_pct"`
	MaxSingleOrderValue         float64  `json:"max_single_order_value"`
	AllowedSymbols              []string `json:"allowed_symbols"`
	BlockedSymbols              []string `json:"blocked_symbols"`
	AllowedSides                []string `json:"allowed_sides"`
	RequireLimitPrice           bool     `json:"require_limit_price"`
	MaxDeviationFromMarketPct   float64  `json:"max_deviation_from_market_pct"`
}

// PositionLimits defines position size constraints
type PositionLimits struct {
	MaxPositionPct    float64                    `json:"max_position_pct"`
	MaxPositionsCount int                        `json:"max_positions_count"`
	SectorLimits      map[string]float64         `json:"sector_limits"`
	PerSymbolLimits   map[string]SymbolLimit     `json:"per_symbol_limits"`
}

type SymbolLimit struct {
	MaxPct float64 `json:"max_pct"`
}

// DailyLimits tracks daily risk metrics
type DailyLimits struct {
	Date            string  `json:"date"`
	StartingEquity  float64 `json:"starting_equity"`
	CurrentEquity   float64 `json:"current_equity"`
	RealizedPnL     float64 `json:"realized_pnl"`
	UnrealizedPnL   float64 `json:"unrealized_pnl"`
	TotalPnLPct     float64 `json:"total_pnl_pct"`
	OrdersThisHour  int     `json:"orders_this_hour"`
	OrdersToday     int     `json:"orders_today"`
	IsHalted        bool    `json:"is_halted"`
	HaltReason      *string `json:"halt_reason"`
}

// RiskStatus tracks real-time risk control status
type RiskStatus struct {
	UpdatedAt      string  `json:"updated_at"`
	ChecksToday    int     `json:"checks_today"`
	ChecksPassed   int     `json:"checks_passed"`
	ChecksRejected int     `json:"checks_rejected"`
	IsHalted       bool    `json:"is_halted"`
	HaltReason     *string `json:"halt_reason"`
}

// RiskViolation records a pre-trade check violation
type RiskViolation struct {
	Timestamp string `json:"timestamp"`
	Rule      string `json:"rule"`
	IntentID  string `json:"intent_id"`
	Detail    string `json:"detail"`
	Action    string `json:"action"` // REJECTED, HALTED
}

// RiskCheckResult is the result of a pre-trade check
type RiskCheckResult struct {
	Passed   bool
	Rule     string // which rule failed (if any)
	Reason   string // human-readable explanation
}

// --- Phase 1: Extended Order Metadata ---

// OrderMetadata extends ParsedOrder with Phase 1 traceability fields
type OrderMetadata struct {
	Source      string   `json:"source,omitempty"`       // manual, rebalance, risk_trigger
	RebalanceID string   `json:"rebalance_id,omitempty"` // links to portfolio rebalance
	SignalRefs  []string `json:"signal_refs,omitempty"`  // triggering signals
}

// --- Phase 2: Portfolio Construction types ---

// TargetPortfolio is the JSON structure for /portfolio/target.json
type TargetPortfolio struct {
	Version         int                       `json:"version"`
	UpdatedAt       string                    `json:"updated_at"`
	UpdatedBy       string                    `json:"updated_by"`
	Strategy        string                    `json:"strategy"`
	TotalCapitalPct float64                   `json:"total_capital_pct"`
	Positions       map[string]TargetPosition `json:"positions"`
	CashReservePct  float64                   `json:"cash_reserve_pct"`
}

// TargetPosition defines target allocation for a symbol
type TargetPosition struct {
	Weight     float64  `json:"weight"`
	Reason     string   `json:"reason"`
	SignalRefs []string `json:"signal_refs"`
}

// CurrentPortfolio is the JSON structure for /portfolio/current.json
type CurrentPortfolio struct {
	UpdatedAt    string                        `json:"updated_at"`
	TotalEquity  float64                       `json:"total_equity"`
	Positions    map[string]CurrentPosition    `json:"positions"`
	Cash         float64                       `json:"cash"`
	CashPct      float64                       `json:"cash_pct"`
}

// CurrentPosition represents current holding with market value
type CurrentPosition struct {
	Qty         float64 `json:"qty"`
	MarketValue float64 `json:"market_value"`
	Weight      float64 `json:"weight"`
	AvgCost     float64 `json:"avg_cost"`
}

// PortfolioDiff is the JSON structure for /portfolio/diff.json
type PortfolioDiff struct {
	ComputedAt        string                `json:"computed_at"`
	TargetVersion     int                   `json:"target_version"`
	Adjustments       []Adjustment          `json:"adjustments"`
	RequiresRebalance bool                  `json:"requires_rebalance"`
}

// Adjustment describes a required portfolio adjustment
type Adjustment struct {
	Symbol        string  `json:"symbol"`
	CurrentWeight float64 `json:"current_weight"`
	TargetWeight  float64 `json:"target_weight"`
	Action        string  `json:"action"` // ADD, REDUCE, CLOSE, HOLD
	DeltaQty      int64   `json:"delta_qty"`
	DeltaValue    float64 `json:"delta_value"`
	EstimatedSide string  `json:"estimated_side"` // BUY, SELL
	EstimatedQty  int64   `json:"estimated_qty"`
}

// RebalancePending is the JSON structure for /portfolio/rebalance/pending.json
type RebalancePending struct {
	RebalanceID string              `json:"rebalance_id"`
	CreatedAt   string              `json:"created_at"`
	CreatedBy   string              `json:"created_by"`
	AutoExecute bool                `json:"auto_execute"`
	Orders      []RebalanceOrder    `json:"orders"`
}

// RebalanceOrder defines an order in a rebalance operation
type RebalanceOrder struct {
	Symbol string  `json:"symbol"`
	Side   string  `json:"side"`
	Qty    int64   `json:"qty"`
	Type   string  `json:"type"`
	Price  float64 `json:"price,omitempty"`
	TIF    string  `json:"tif"`
}

// --- Phase 3: Research & Signal types ---

// Watchlist is the JSON structure for /research/watchlist.json
type Watchlist struct {
	Symbols         []string `json:"symbols"`
	RefreshInterval string   `json:"refresh_interval"` // e.g., "5m"
	Feeds           []string `json:"feeds"`            // e.g., ["news", "topics"]
}

// NewsFeed is the JSON structure for /research/feeds/news/{SYMBOL}/latest.json
type NewsFeed struct {
	Symbol    string     `json:"symbol"`
	FetchedAt string     `json:"fetched_at"`
	Items     []NewsItem `json:"items"`
}

// NewsItem represents a single news article
type NewsItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Summary     string `json:"summary"`
	URL         string `json:"url,omitempty"`
}

// TopicsFeed is the JSON structure for /research/feeds/topics/{SYMBOL}/latest.json
type TopicsFeed struct {
	Symbol    string      `json:"symbol"`
	FetchedAt string      `json:"fetched_at"`
	Items     []TopicItem `json:"items"`
}

// TopicItem represents a discussion topic
type TopicItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Summary     string `json:"summary"`
	URL         string `json:"url,omitempty"`
}

// ResearchSummary is the JSON structure for /research/summary.json
type ResearchSummary struct {
	UpdatedAt string                       `json:"updated_at"`
	Symbols   map[string]SymbolResearch    `json:"symbols"`
}

// SymbolResearch tracks available research data for a symbol
type SymbolResearch struct {
	HasQuote      bool     `json:"has_quote"`
	HasNews       bool     `json:"has_news"`
	HasTopics     bool     `json:"has_topics"`
	CustomFeeds   []string `json:"custom_feeds"`
}

