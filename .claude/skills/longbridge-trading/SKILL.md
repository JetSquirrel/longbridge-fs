---
name: longbridge-trading
description: Execute stock trading operations using Longbridge FS file-based trading system for HK/US stocks
---

# Longbridge Trading Skill

This skill enables you to perform stock trading operations through the Longbridge FS file-based trading system. All operations are performed by reading and writing files, making it natural for AI agents.

The system implements a **five-layer Harness architecture**:

```
L1 Research → L2 Signal → L3 Portfolio → L4 Risk → L5 Execution
```

Each layer communicates through files, so you can read, write, or inject data at any layer without modifying code.

## When to Use This Skill

Use this skill when the user wants to:
- Buy or sell stocks (HK/US markets)
- Check stock quotes and market data
- View account balance and positions
- Monitor portfolio performance and P&L
- Set up signal definitions (SMA_CROSS, RSI, PRICE_CHANGE)
- Configure portfolio targets and trigger rebalancing
- Set up risk control rules (stop-loss/take-profit, pre-trade limits)
- Run end-to-end pipeline: research → signal → portfolio → execution
- Query trading history

## Prerequisites

Before using this skill, verify:

1. **Controller is running**: Check if the Longbridge FS controller daemon is active
   ```bash
   ps aux | grep longbridge-fs
   ```

2. **File system is initialized**: The `fs/` directory should exist with proper structure
   ```bash
   ls -la fs/
   ```

3. **Permissions**: Ensure you have read/write access to the `fs/` directory

If the controller is not running, start it:
```bash
# Mock mode (for testing, no real API calls — enables full pipeline simulation)
./build/longbridge-fs controller --root ./fs --mock --interval 2s &

# Real mode (requires API credentials)
./build/longbridge-fs controller --root ./fs --credential ./configs/credential --interval 2s &
```

---

## L1 Research Layer

The research layer aggregates news, topics and custom data feeds for the symbols in your watchlist.

### Configure Watchlist

```bash
cat > fs/research/watchlist.json << 'EOF'
{
  "symbols": ["AAPL.US", "TSLA.US", "700.HK"],
  "refresh_interval": "5m",
  "feeds": ["news", "topics"]
}
EOF
```

Controller behavior:
- In real mode: fetches live news/topics from Content API for each symbol.
- In mock mode: generates synthetic feeds automatically to enable full pipeline testing.

### Read Research Feeds

```bash
# Latest news for a symbol
cat fs/research/feeds/news/AAPL.US/latest.json

# Community topics for a symbol
cat fs/research/feeds/topics/AAPL.US/latest.json

# Aggregated research summary across all symbols
cat fs/research/summary.json
```

### Inject Custom Research Data

AI agents can write custom research data directly:

```bash
mkdir -p fs/research/feeds/custom
cat > fs/research/feeds/custom/my_analysis.json << 'EOF'
{
  "name": "sector_rotation_analysis",
  "created_at": "2026-04-01T07:00:00Z",
  "author": "claude-agent",
  "data": {
    "recommendation": "overweight_tech",
    "confidence": 0.82
  }
}
EOF
```

---

## L2 Signal Layer

The signal layer converts market data into actionable trading signals.

### Create Signal Definitions

Signal definitions are JSON files in `fs/signal/definitions/`. The controller evaluates builtin signals every cycle.

**Built-in Signal: SMA Crossover**
```bash
cat > fs/signal/definitions/sma_cross.json << 'EOF'
{
  "name": "sma_crossover",
  "type": "builtin",
  "enabled": true,
  "symbols": ["AAPL.US", "TSLA.US"],
  "params": {
    "indicator": "SMA_CROSS",
    "fast_period": 5,
    "slow_period": 20
  }
}
EOF
```

**Built-in Signal: RSI**
```bash
cat > fs/signal/definitions/rsi.json << 'EOF'
{
  "name": "rsi_signal",
  "type": "builtin",
  "enabled": true,
  "symbols": ["AAPL.US"],
  "params": {
    "indicator": "RSI",
    "period": 14,
    "overbought": 70,
    "oversold": 30
  }
}
EOF
```

**Built-in Signal: Price Change**
```bash
cat > fs/signal/definitions/price_change.json << 'EOF'
{
  "name": "price_momentum",
  "type": "builtin",
  "enabled": true,
  "symbols": ["TSLA.US"],
  "params": {
    "indicator": "PRICE_CHANGE",
    "threshold_pct": 5.0,
    "window": 5
  }
}
EOF
```

**External Signal (Agent-computed)**
```bash
# Agent writes signal output directly
mkdir -p fs/signal/output/AAPL.US
cat > fs/signal/output/AAPL.US/latest.json << 'EOF'
{
  "symbol": "AAPL.US",
  "updated_at": "2026-04-01T08:00:00Z",
  "signals": [
    {
      "name": "llm_sentiment",
      "value": "BULLISH",
      "strength": 0.78,
      "detail": "Positive earnings sentiment detected",
      "computed_at": "2026-04-01T08:00:00Z"
    }
  ]
}
EOF
```

### Read Signal Output

```bash
# All active signals across all symbols
cat fs/signal/active.json

# Per-symbol signal output
cat fs/signal/output/AAPL.US/latest.json

# Signal history (append-only JSONL)
cat fs/signal/output/AAPL.US/history.jsonl
```

**active.json example:**
```json
{
  "updated_at": "2026-04-01T08:05:00Z",
  "signals": [
    { "symbol": "AAPL.US", "name": "sma_crossover", "value": "BULLISH", "strength": 0.72 },
    { "symbol": "AAPL.US", "name": "rsi_signal",    "value": "NEUTRAL", "strength": 0.45 },
    { "symbol": "TSLA.US", "name": "sma_crossover", "value": "BEARISH", "strength": 0.61 }
  ]
}
```

Signal values: `BULLISH`, `BEARISH`, `NEUTRAL`, `OVERBOUGHT`, `OVERSOLD`, `SURGE`, `DROP`

---

## L3 Portfolio Layer

The portfolio layer manages target allocations and rebalancing.

### Set Portfolio Target

```bash
cat > fs/portfolio/target.json << 'EOF'
{
  "version": 1,
  "updated_at": "2026-04-01T00:00:00Z",
  "total_capital_pct": 0.90,
  "cash_reserve_pct": 0.10,
  "positions": {
    "AAPL.US": { "weight": 0.40 },
    "TSLA.US": { "weight": 0.35 },
    "700.HK":  { "weight": 0.15 },
    "NVDA.US": { "weight": 0.10 }
  }
}
EOF
```

### Read Portfolio State

```bash
# Current portfolio positions and weights
cat fs/portfolio/current.json

# Target vs current comparison
cat fs/portfolio/diff.json

# Historical snapshots
ls fs/portfolio/history/
```

**diff.json example:**
```json
{
  "updated_at": "2026-04-01T08:10:00Z",
  "target_version": 1,
  "requires_rebalance": true,
  "adjustments": [
    {
      "symbol": "AAPL.US",
      "current_weight": 0.28,
      "target_weight": 0.40,
      "drift": -0.12,
      "action": "BUY",
      "estimated_value": 12000
    }
  ]
}
```

### Trigger Rebalance

**Manual rebalance** (write pending orders):
```bash
cat > fs/portfolio/rebalance/pending.json << 'EOF'
{
  "rebalance_id": "rebal-20260401-001",
  "created_at": "2026-04-01T08:10:00Z",
  "orders": [
    {
      "symbol": "AAPL.US",
      "side": "BUY",
      "qty": 50,
      "order_type": "MARKET",
      "tif": "DAY"
    }
  ]
}
EOF
```

**Auto-rebalance mode** (controller creates pending orders automatically when drift exceeds threshold):
```bash
./build/longbridge-fs controller --root ./fs --mock --auto-rebalance &
```

---

## L4 Risk Control Layer

The risk layer enforces pre-trade checks and monitors trading limits.

### Configure Risk Policy

```bash
cat > fs/trade/risk/policy.json << 'EOF'
{
  "version": 1,
  "enabled": true,
  "mode": "ENFORCE",
  "pre_trade_checks": true,
  "post_trade_monitoring": true,
  "daily_loss_limit": {
    "enabled": true,
    "max_loss_pct": 0.03,
    "action": "HALT"
  },
  "order_frequency": {
    "enabled": true,
    "max_orders_per_hour": 20,
    "max_orders_per_day": 100
  }
}
EOF
```

Risk modes:
- `ENFORCE` (default): reject orders that violate rules
- `WARN`: log violations but allow orders through
- `DISABLED`: skip all pre-trade checks

### Configure Pre-Trade Rules

```bash
cat > fs/trade/risk/pre_trade.json << 'EOF'
{
  "max_single_order_pct": 0.10,
  "max_single_order_value": 50000,
  "allowed_symbols": [],
  "blocked_symbols": ["MEME.US"],
  "allowed_sides": ["BUY", "SELL"],
  "require_limit_price": false,
  "max_deviation_from_market_pct": 0.05
}
EOF
```

### Configure Position Limits

```bash
cat > fs/trade/risk/position_limits.json << 'EOF'
{
  "max_position_pct": 0.25,
  "max_positions_count": 15,
  "sector_limits": {},
  "per_symbol_limits": {
    "TSLA.US": { "max_pct": 0.10 }
  }
}
EOF
```

### Configure Stop-Loss / Take-Profit (Legacy Risk Control)

```bash
cat > fs/trade/risk_control.json << 'EOF'
{
  "AAPL.US": {
    "stop_loss": 170.00,
    "take_profit": 200.00,
    "qty": "100"
  },
  "TSLA.US": {
    "stop_loss": 200.00,
    "take_profit": 350.00
  }
}
EOF
```

### Read Risk Status

```bash
# Current risk state and counters
cat fs/trade/risk/status.json

# Today's order/loss counters
cat fs/trade/risk/daily_limits.json

# Violations log (append-only)
cat fs/trade/risk/violations.jsonl
```

---

## L5 Execution Layer

### Submit Standard Orders

Append ORDER entries to `fs/trade/beancount.txt`:

**Market Order:**
```bash
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260401-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

EOF
```

**Limit Order with traceability:**
```bash
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US from signal"
  ; intent_id: 20260401-002
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; price: 180.50
  ; tif: DAY
  ; source: rebalance
  ; rebalance_id: rebal-20260401-001
  ; signal_refs: sma_crossover,rsi_signal

EOF
```

### Submit Algorithmic Orders

**TWAP (Time-Weighted Average Price):**
```bash
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US via TWAP"
  ; intent_id: 20260401-003
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 500
  ; type: LIMIT
  ; price: 182.00
  ; tif: DAY
  ; algo: TWAP
  ; algo_duration: 30m
  ; algo_slices: 5

EOF
```

**ICEBERG (hidden quantity):**
```bash
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US via ICEBERG"
  ; intent_id: 20260401-004
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 1000
  ; type: LIMIT
  ; price: 182.00
  ; tif: GTC
  ; algo: ICEBERG
  ; algo_slices: 10

EOF
```

### Check Order Results

```bash
# Wait for controller to process
sleep 3

# Check for EXECUTION or REJECTION
grep -A 10 "intent_id: 20260401-001" fs/trade/beancount.txt
```

**EXECUTION example:**
```
2026-04-01 * "EXECUTION" "BUY AAPL.US @ 180.25"
  ; intent_id: 20260401-001
  ; order_id: 1234567890
  ; side: BUY
  ; symbol: AAPL.US
  ; filled_qty: 100
  ; avg_price: 180.25
  ; status: FILLED
  ; executed_at: 2026-04-01T10:30:15Z
```

**REJECTION example:**
```
2026-04-01 * "REJECTION" "BUY AAPL.US"
  ; intent_id: 20260401-001
  ; reason: max_single_order_pct exceeded
```

---

## Common Workflows

### Workflow 1: Full Harness Pipeline (Research → Signal → Portfolio → Execution)

```bash
# Step 1: Initialize FS
./build/longbridge-fs init --root ./fs

# Step 2: Configure watchlist (L1)
cat > fs/research/watchlist.json << 'EOF'
{"symbols": ["AAPL.US", "TSLA.US"], "refresh_interval": "5m", "feeds": ["news", "topics"]}
EOF

# Step 3: Define signals (L2)
cat > fs/signal/definitions/sma.json << 'EOF'
{"name": "sma_crossover", "type": "builtin", "enabled": true,
 "symbols": ["AAPL.US"], "params": {"indicator": "SMA_CROSS", "fast_period": 5, "slow_period": 20}}
EOF

# Step 4: Set portfolio target (L3)
cat > fs/portfolio/target.json << 'EOF'
{"version": 1, "total_capital_pct": 0.90, "cash_reserve_pct": 0.10,
 "positions": {"AAPL.US": {"weight": 0.40}}}
EOF

# Step 5: Start controller in mock mode (enables full pipeline simulation)
./build/longbridge-fs controller --root ./fs --mock --interval 2s &

# Step 6: Wait and inspect pipeline output
sleep 5
cat fs/research/summary.json
cat fs/signal/active.json
cat fs/portfolio/diff.json

# Step 7: Submit orders (L5)
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260401-100
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 50
  ; type: MARKET
  ; tif: DAY

EOF
sleep 3
tail -20 fs/trade/beancount.txt

# Step 8: Stop controller
touch fs/.kill
```

### Workflow 2: Signal-Driven Order Submission

```bash
# Read active signals and submit orders for BULLISH signals
SIGNALS=$(cat fs/signal/active.json)
echo "$SIGNALS" | python3 -c "
import json, sys
active = json.load(sys.stdin)
for s in active.get('signals', []):
    if s['value'] == 'BULLISH' and s['strength'] > 0.6:
        print(f\"Buy signal: {s['symbol']} ({s['name']}, strength={s['strength']:.2f})\")
"
```

### Workflow 3: Buy Stock at Market Price

```bash
# Step 1: Check current price
touch fs/quote/track/AAPL.US
sleep 3
cat fs/quote/hold/AAPL.US/overview.json

# Step 2: Submit market buy order
cat >> fs/trade/beancount.txt << 'EOF'
2026-04-01 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260401-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

EOF

# Step 3: Wait and check result
sleep 3
tail -20 fs/trade/beancount.txt

# Step 4: Verify position
cat fs/account/pnl.json
```

### Workflow 4: Set Stop-Loss for Existing Position

```bash
# Set stop-loss at 5% below current price for AAPL.US
touch fs/quote/track/AAPL.US
sleep 3
CURRENT_PRICE=$(jq -r '.last' fs/quote/hold/AAPL.US/overview.json)
STOP_PRICE=$(echo "$CURRENT_PRICE * 0.95" | bc)

jq --arg symbol "AAPL.US" --argjson stop "$STOP_PRICE" \
  '.[$symbol] = {"stop_loss": $stop}' \
  fs/trade/risk_control.json > /tmp/risk.json && \
  mv /tmp/risk.json fs/trade/risk_control.json
```

---

## Quick Reference: All Layer Files

```
fs/
├── research/                         # L1 Research
│   ├── watchlist.json                #   <- WRITE: symbols to track
│   ├── summary.json                  #   -> READ:  aggregated feed status
│   └── feeds/
│       ├── news/{SYMBOL}/latest.json #   -> READ:  news articles
│       ├── topics/{SYMBOL}/latest.json#  -> READ:  community topics
│       └── custom/{name}.json        #   <- WRITE: agent custom data
│
├── signal/                           # L2 Signal
│   ├── definitions/{name}.json       #   <- WRITE: signal configs
│   ├── active.json                   #   -> READ:  current signals
│   └── output/{SYMBOL}/
│       ├── latest.json               #   -> READ:  per-symbol output
│       └── history.jsonl             #   -> READ:  signal history
│
├── portfolio/                        # L3 Portfolio
│   ├── target.json                   #   <- WRITE: target weights
│   ├── current.json                  #   -> READ:  actual weights
│   ├── diff.json                     #   -> READ:  drift / actions
│   └── rebalance/pending.json        #   <- WRITE: pending orders
│
├── account/
│   ├── state.json                    #   -> READ:  balances and orders
│   └── pnl.json                      #   -> READ:  per-position P&L
│
├── trade/
│   ├── beancount.txt                 #   <- WRITE ORDER / -> READ EXECUTION
│   ├── risk_control.json             #   <- WRITE: stop-loss/take-profit
│   ├── blocks/                       #   -> READ:  archived orders
│   └── risk/                         # L4 Risk
│       ├── policy.json               #   <- WRITE: risk policy
│       ├── pre_trade.json            #   <- WRITE: order limits
│       ├── position_limits.json      #   <- WRITE: position caps
│       ├── daily_limits.json         #   -> READ:  daily counters
│       ├── status.json               #   -> READ:  risk gate status
│       └── violations.jsonl          #   -> READ:  violation log
│
└── quote/
    ├── track/                        #   <- CREATE: request a quote
    ├── hold/{SYMBOL}/
    │   ├── overview.json             #   -> READ:  current price
    │   ├── D.json                    #   -> READ:  daily kline (120d)
    │   └── intraday.json             #   -> READ:  intraday ticks
    └── portfolio.json                #   -> READ:  portfolio with quotes
```

## Controller Options

| Flag                | Default      | Description                              |
|---------------------|--------------|------------------------------------------|
| `--root`            | `.`          | FS root directory                        |
| `--interval`        | `2s`         | Poll interval                            |
| `--mock`            | `false`      | Mock mode — no API calls, full pipeline  |
| `--auto-rebalance`  | `false`      | Auto-create rebalance orders on drift    |
| `--compact-after`   | `10`         | Compact ledger after N executions        |
| `--credential`      | `credential` | Credential file (real mode)              |

In **mock mode** (`--mock`):
- All five layers run without API calls
- Research feeds are populated with synthetic data
- Kline data is generated automatically for signal computation
- Orders are simulated with realistic mock fills
- Fully self-contained for testing and development

## Stock Symbol Format

**US Stocks:** `AAPL.US`, `MSFT.US`, `TSLA.US`, `NVDA.US`
**HK Stocks:** `700.HK`, `9988.HK`, `0001.HK`
**CN Stocks:** `600519.SH`, `000001.SZ`

## Tips for AI Agents

1. **Always wait after operations**: Controller needs 2-3 seconds per cycle to process files
2. **Mock mode for development**: Use `--mock` to test the full pipeline without credentials
3. **Read before acting**: Check `signal/active.json` and `portfolio/diff.json` before submitting orders
4. **Append, don't overwrite**: Always append to `beancount.txt`, never overwrite
5. **Use unique intent_ids**: Use timestamp-based IDs to avoid conflicts
6. **Signal to order traceability**: Always include `signal_refs` in orders for audit trail
7. **Stop gracefully**: Use `touch fs/.kill` to stop the controller cleanly

## Additional Resources

- [README](../README.md) - Project overview and quick start
- [End-to-End Demo](../demo_e2e.sh) - Full pipeline demo script (mock mode)
- [Basic Demo](../demo.sh) - Basic execution demo
- [Spec](../spec.md) - Full architecture specification
