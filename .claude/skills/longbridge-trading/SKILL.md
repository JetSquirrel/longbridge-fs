---
name: longbridge-trading
description: Execute stock trading operations using Longbridge FS file-based trading system for HK/US stocks
---

# Longbridge Trading Skill

This skill enables you to perform stock trading operations through the Longbridge FS file-based trading system. All operations are performed by reading and writing files, making it natural for AI agents.

## When to Use This Skill

Use this skill when the user wants to:
- Buy or sell stocks (HK/US markets)
- Check stock quotes and market data
- View account balance and positions
- Monitor portfolio performance and P&L
- Set up risk control rules (stop-loss/take-profit)
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
# Mock mode (for testing, no real API calls)
./build/longbridge-fs controller --root ./fs --mock --interval 2s &

# Real mode (requires API credentials)
./build/longbridge-fs controller --root ./fs --credential ./configs/credential --interval 2s &
```

## Core Operations

### 1. Submit Buy/Sell Orders

To submit an order, append a new ORDER entry to `fs/trade/beancount.txt`:

**Market Order Format:**
```
2026-02-12 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260212-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

```

**Limit Order Format:**
```
2026-02-12 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260212-002
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; price: 180.50
  ; tif: DAY

```

**Required Fields:**
- `intent_id`: Unique identifier (use timestamp format: YYYYMMDD-HHMMSS or YYYYMMDD-NNN)
- `side`: BUY or SELL
- `symbol`: Stock symbol with market suffix (e.g., AAPL.US, 9988.HK, 700.HK)
- `qty`: Quantity to trade
- `type`: MARKET or LIMIT
- `tif`: DAY (expires end of day) or GTC (good till canceled)
- `price`: Required for LIMIT orders only

**Important Notes:**
- Always APPEND to the file, never overwrite
- Include an empty line at the end
- Controller polls every 2 seconds (default), so wait 2-3 seconds after submission
- Each field MUST start with `  ;` (two spaces + semicolon)
- Use proper date format: YYYY-MM-DD

### 2. Check Order Results

After submitting an order, wait 2-3 seconds and read `fs/trade/beancount.txt` to check results:

```bash
# Read the entire beancount file
cat fs/trade/beancount.txt

# Search for specific intent_id
grep "intent_id: 20260212-001" fs/trade/beancount.txt -A 10
```

The controller will append either:
- **EXECUTION** record if order was filled
- **REJECTION** record if order was rejected

**EXECUTION Example:**
```
2026-02-12 * "EXECUTION" "BUY AAPL.US @ 180.25"
  ; intent_id: 20260212-001
  ; order_id: 1234567890
  ; side: BUY
  ; symbol: AAPL.US
  ; filled_qty: 100
  ; avg_price: 180.25
  ; status: FILLED
  ; executed_at: 2026-02-12T10:30:15Z
```

**REJECTION Example:**
```
2026-02-12 * "REJECTION" "BUY AAPL.US"
  ; intent_id: 20260212-001
  ; reason: Insufficient funds
```

### 3. Get Stock Quotes

To get real-time market data, create a track file:

```bash
# Request quote for AAPL.US
touch fs/quote/track/AAPL.US

# Wait 2-3 seconds for controller to process
sleep 3

# Read quote data
cat fs/quote/hold/AAPL.US/overview.json
cat fs/quote/hold/AAPL.US/overview.txt
```

**Available Quote Files:**
- `overview.json` / `overview.txt`: Real-time price, volume, change
- `D.json`: Daily K-line (120 days)
- `W.json`: Weekly K-line (52 weeks)
- `5D.json`: 5-minute K-line
- `intraday.json`: Intraday tick data

**Quote JSON Format:**
```json
{
  "symbol": "AAPL.US",
  "last_done": 180.50,
  "prev_close": 179.00,
  "open": 179.50,
  "high": 181.00,
  "low": 178.50,
  "volume": 45000000,
  "turnover": 8100000000,
  "timestamp": "2026-02-12T16:00:00Z"
}
```

### 4. Check Account Status

```bash
# View account balance and summary
cat fs/account/state.json
```

**state.json Format:**
```json
{
  "cash": 10000.00,
  "market_value": 18050.00,
  "total_value": 28050.00,
  "available": 10000.00,
  "updated_at": "2026-02-12T10:30:00Z"
}
```

### 5. View Positions and P&L

```bash
# View position-level P&L
cat fs/account/pnl.json

# View portfolio with current quotes
cat fs/quote/portfolio.json
```

**pnl.json Format:**
```json
{
  "positions": [
    {
      "symbol": "AAPL.US",
      "qty": 100,
      "avg_cost": 175.50,
      "current_price": 180.50,
      "market_value": 18050.00,
      "cost_basis": 17550.00,
      "unrealized_pnl": 500.00,
      "unrealized_pnl_percent": 2.85
    }
  ],
  "total_unrealized_pnl": 500.00,
  "updated_at": "2026-02-12T10:30:00Z"
}
```

### 6. Set Risk Control Rules

Configure automatic stop-loss and take-profit by editing `fs/trade/risk_control.json`:

```bash
# Create or update risk control configuration
cat > fs/trade/risk_control.json << 'EOF'
{
  "AAPL.US": {
    "stop_loss": 170.00,
    "take_profit": 200.00,
    "qty": "100"
  },
  "9988.HK": {
    "stop_loss": 150.00,
    "take_profit": 180.00
  }
}
EOF
```

**How it works:**
- Controller monitors prices for configured symbols
- When price hits `stop_loss`, automatically submits SELL order
- When price hits `take_profit`, automatically submits SELL order
- Rule is removed after triggering to prevent duplicate orders
- If `qty` is specified, sells that quantity; otherwise sells entire position

### 7. Stop Controller Safely

To safely stop the controller daemon:

```bash
touch fs/.kill
```

The controller will detect this file in the next polling cycle and exit gracefully without affecting pending orders.

## Common Workflows

### Workflow 1: Buy Stock at Market Price

```bash
# Step 1: Check current price
touch fs/quote/track/AAPL.US
sleep 3
cat fs/quote/hold/AAPL.US/overview.json

# Step 2: Submit market buy order
cat >> fs/trade/beancount.txt << 'EOF'
2026-02-12 * "ORDER" "BUY AAPL.US"
  ; intent_id: 20260212-001
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

### Workflow 2: Set Stop-Loss for Existing Position

```bash
# Step 1: Check current positions
cat fs/account/pnl.json

# Step 2: Get current price
touch fs/quote/track/AAPL.US
sleep 3
CURRENT_PRICE=$(jq -r '.last_done' fs/quote/hold/AAPL.US/overview.json)

# Step 3: Set stop-loss at 5% below current price
STOP_PRICE=$(echo "$CURRENT_PRICE * 0.95" | bc)
jq --arg symbol "AAPL.US" --arg stop "$STOP_PRICE" \
  '.[$symbol] = {"stop_loss": ($stop | tonumber)}' \
  fs/trade/risk_control.json > /tmp/risk.json && \
  mv /tmp/risk.json fs/trade/risk_control.json
```

### Workflow 3: Monitor and Trade Based on Price

```bash
# Monitor AAPL, buy when price drops below 175
while true; do
  touch fs/quote/track/AAPL.US
  sleep 3
  PRICE=$(jq -r '.last_done' fs/quote/hold/AAPL.US/overview.json)
  echo "Current price: $PRICE"

  if (( $(echo "$PRICE < 175" | bc -l) )); then
    # Submit buy order
    cat >> fs/trade/beancount.txt << EOF
2026-02-12 * "ORDER" "BUY AAPL.US"
  ; intent_id: $(date +%Y%m%d-%H%M%S)
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

EOF
    echo "Buy order submitted at $PRICE"
    break
  fi

  sleep 10
done
```

## Stock Symbol Format

Always use the correct symbol format with market suffix:

**US Stocks:**
- AAPL.US (Apple)
- MSFT.US (Microsoft)
- TSLA.US (Tesla)
- NVDA.US (NVIDIA)

**HK Stocks:**
- 700.HK (Tencent)
- 9988.HK (Alibaba)
- 0001.HK (CKH Holdings)

**CN Stocks:**
- 600519.SH (Kweichow Moutai - Shanghai)
- 000001.SZ (Ping An Bank - Shenzhen)

## Error Handling

### Order Rejected

If you see a REJECTION record, common reasons include:
- Insufficient funds
- Market closed
- Invalid symbol
- Invalid price (outside allowable range)
- Invalid quantity (less than minimum lot size)

**Solution:** Check the rejection reason and fix the order parameters.

### Controller Not Responding

If orders are not being processed:
1. Check if controller is running: `ps aux | grep longbridge-fs`
2. Check controller logs for errors
3. Restart controller if needed
4. Use `--mock` mode for testing

### File Format Errors

If controller logs show parsing errors:
- Verify Beancount format (indentation with 2 spaces, semicolon prefix)
- Check date format (YYYY-MM-DD)
- Ensure all required fields are present
- Verify no extra characters or wrong encoding

## Tips for AI Agents

1. **Always wait after operations**: File operations need 2-3 seconds for controller to process
2. **Use unique intent_ids**: Use timestamp-based IDs to avoid conflicts
3. **Append, don't overwrite**: Always append to beancount.txt, never overwrite
4. **Check results**: Always verify order execution by reading the beancount file
5. **Handle errors gracefully**: Orders can be rejected; check for REJECTION records
6. **Use Mock mode for testing**: Start controller with `--mock` flag during development
7. **Read before acting**: Check current state (positions, prices) before submitting orders

## File System Reference

```
fs/
├── account/
│   ├── state.json           # Account balance and summary
│   └── pnl.json             # Position-level P&L
├── trade/
│   ├── beancount.txt        # Main order ledger (read/write)
│   ├── risk_control.json    # Risk control rules (read/write)
│   └── blocks/              # Archived orders (read-only)
│       └── block_NNNN.txt
└── quote/
    ├── track/               # Create files here to request quotes
    ├── hold/                # Quote data stored here
    │   └── SYMBOL/
    │       ├── overview.json
    │       ├── overview.txt
    │       ├── D.json
    │       └── intraday.json
    └── portfolio.json       # Full portfolio with quotes
```

## Additional Resources

- [README](../README.md) - Project overview and quick start
- [AI Agent Guide](../docs/ai-agent-guide.md) - Detailed programming guide
- [Architecture](../docs/architecture.md) - System design and internals
- [Longbridge API](https://github.com/longportapp/openapi-go) - Official SDK documentation

## Summary

This skill allows you to trade stocks through simple file operations:
- **Write** to `fs/trade/beancount.txt` to submit orders
- **Create** files in `fs/quote/track/` to request quotes
- **Read** from `fs/account/` and `fs/quote/hold/` to check status
- **Edit** `fs/trade/risk_control.json` to configure risk rules
- **Create** `fs/.kill` to stop controller

All operations are file-based, making them natural for AI agents and easy to audit.
