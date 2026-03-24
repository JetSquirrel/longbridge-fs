# Longbridge Terminal

> AI-native CLI for the Longbridge trading platform — real-time market data, portfolio, and trading

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](go.mod)

## Overview

**Longbridge Terminal** is an AI-native command-line interface for the Longbridge trading platform, designed for scripting, AI-agent tool-calling, and daily trading workflows from the terminal.

Covers every Longbridge OpenAPI endpoint: real-time quotes, depth, K-lines, options, and warrants for market data; account balances, stock and fund positions for portfolio management; and order submission, modification, cancellation, and execution history for trading.

### Key Features

- **AI-Native CLI** — Direct command-line access to all Longbridge OpenAPI endpoints
- **Dual Interface** — Both CLI commands and file-system based operations
- **Real-time Market Data** — Quotes, depth, K-lines, intraday data with WebSocket support
- **Portfolio Management** — Account balance, positions, order history
- **Trading Operations** — Submit, cancel, and track orders
- **Multiple Output Formats** — JSON, table, and CSV formats for easy integration
- **Authentication** — OAuth 2.0 support via Longbridge SDK
- **Mock Mode** — Test without real API calls
- **Audit Trail** — All trades recorded in Beancount format
- **Risk Control** — Automatic stop-loss/take-profit

### Use Cases

- AI agents performing automated trading
- Daily trading workflows from the terminal
- Portfolio monitoring and analysis
- Building rule-based trading systems
- Backtesting and simulation
- Learning the Longbridge API

## Quick Start

### Installation

```bash
# Build from source
make build
# or
go build -o build/longbridge-fs ./cmd/longbridge-fs
```

### Configuration

Create a credential file at `configs/credential`:

```
api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN
```

### Authentication

```bash
# Verify credentials and API connectivity
longbridge-fs check

# Authenticate (currently verifies existing credentials)
longbridge-fs login

# Clear credentials
longbridge-fs logout
```

## CLI Usage

### Market Data Commands

#### Real-time Quotes

```bash
# Get quotes for multiple symbols (table format)
longbridge-fs quote TSLA.US NVDA.US

# JSON output
longbridge-fs quote AAPL.US --format json

# Static table format (matches problem statement example)
longbridge-fs static NVDA.US TSLA.US
```

Example output:
```
| Symbol  | Last    | Prev Close | Open    | High    | Low     | Volume    | Turnover        | Status |
|---------|---------|------------|---------|---------|---------|-----------|-----------------|--------|
| TSLA.US | 395.560 | 391.200    | 396.220 | 403.730 | 394.420 | 58068343  | 23138752546.000 | Normal |
| NVDA.US | 183.220 | 180.250    | 182.970 | 188.880 | 181.410 | 217307380 | 40023702698.000 | Normal |
```

#### Market Depth

```bash
# Get order book depth
longbridge-fs depth AAPL.US

# JSON format
longbridge-fs depth 700.HK --format json
```

#### K-Line Data

```bash
# Get daily K-lines (default: 30 bars)
longbridge-fs klines AAPL.US

# Get weekly K-lines
longbridge-fs klines 700.HK --period week --count 52

# Other periods: day, week, month, year, 1m, 5m, 15m, 30m, 60m
longbridge-fs klines TSLA.US --period 5m --count 100 --format json
```

#### Intraday Data

```bash
# Get minute-by-minute intraday data
longbridge-fs intraday AAPL.US

# JSON output
longbridge-fs intraday 9988.HK --format json
```

### Account & Portfolio Commands

#### Account Balance

```bash
# Get account balance
longbridge-fs account balance

# JSON format
longbridge-fs account balance --format json
```

Example output:
```
Currency        Total Cash      Available       Frozen          Settling        Withdrawable
--------        ----------      ---------       ------          --------        ------------
USD             100000.00       95000.00        2000.00         3000.00         90000.00
HKD             500000.00       480000.00       10000.00        10000.00        470000.00
```

#### Stock Positions

```bash
# Get current positions
longbridge-fs account positions

# JSON format
longbridge-fs account positions --format json
```

Example output:
```
Symbol          Quantity        Available       Cost Price      Currency        Market
------          --------        ---------       ----------      --------        ------
AAPL.US         100             100             180.500         USD             US
700.HK          1000            1000            320.000         HKD             HK
```

#### Order History

```bash
# Get today's orders
longbridge-fs account orders

# JSON format
longbridge-fs account orders --format json
```

### Trading Commands

#### Submit Orders

```bash
# Market order
longbridge-fs order submit AAPL.US BUY 100

# Limit order
longbridge-fs order submit TSLA.US BUY 50 --type LIMIT --price 180.50

# With time in force
longbridge-fs order submit 700.HK SELL 1000 --type LIMIT --price 350.00 --tif GTC

# JSON output
longbridge-fs order submit NVDA.US BUY 25 --type LIMIT --price 183.00 --format json
```

Order parameters:
- **Side**: BUY or SELL
- **Type**: MARKET (default), LIMIT, ELO, ALO
- **TIF** (Time In Force): DAY (default), GTC (Good Till Canceled), GTD (Good Till Date)

#### Cancel Orders

```bash
# Cancel an order by ID
longbridge-fs order cancel 1234567890

# JSON output
longbridge-fs order cancel 9876543210 --format json
```

### Global Flags

All commands support these global flags:

```bash
--format string       Output format: table (default), json, csv
--credential string   Credential file path (default: "credential")
--verbose, -v         Enable verbose output
```

## File-System Interface

In addition to CLI commands, Longbridge Terminal supports a file-system based interface for AI agents and automated workflows.

### Initialize File System

```bash
# Initialize directory structure
longbridge-fs init --root ./fs

# Start the controller daemon
longbridge-fs controller --root ./fs --credential ./configs/credential

# Or run in mock mode (no real API calls)
longbridge-fs controller --root ./fs --mock
```

### File-Based Operations

#### Submit Orders via Beancount

Append orders to `fs/trade/beancount.txt`:

```
2026-03-24 * "ORDER" "BUY 9988.HK Alibaba"
  ; intent_id: 20260324-001
  ; side: BUY
  ; symbol: 9988.HK
  ; market: HK
  ; qty: 1000
  ; type: LIMIT
  ; price: 161
  ; tif: DAY
```

The controller automatically processes and appends execution results.

#### Query Quotes via File Triggers

```bash
# WebSocket subscription (real-time updates)
touch fs/quote/subscribe/AAPL.US
# Data continuously updated in fs/quote/hold/AAPL.US/overview.json

# One-shot fetch
touch fs/quote/track/TSLA.US
# Data written to fs/quote/hold/TSLA.US/ and track file removed
```

#### View Account & PnL

```bash
# Account state
cat fs/account/state.json | jq

# Position P&L
cat fs/account/pnl.json | jq

# Portfolio summary
cat fs/quote/portfolio.json | jq
```

#### Risk Control

Configure automatic stop-loss/take-profit in `fs/trade/risk_control.json`:

```json
{
  "700.HK": {
    "stop_loss": 280.0,
    "take_profit": 350.0
  },
  "AAPL.US": {
    "stop_loss": 150.0,
    "take_profit": 210.0,
    "qty": "10"
  }
}
```

#### Kill Switch

```bash
# Safely stop the controller
touch fs/.kill
```

## Architecture

```
longbridge-terminal/
├── cmd/longbridge-fs/
│   ├── main.go              # CLI entry point
│   ├── cmd_auth.go          # Authentication commands
│   ├── cmd_quote.go         # Market data commands
│   ├── cmd_account.go       # Account/portfolio commands
│   └── cmd_order.go         # Trading commands
├── internal/
│   ├── model/types.go       # Data structures
│   ├── ledger/              # Beancount parsing & archival
│   ├── credential/          # API credential loading
│   ├── broker/              # Order execution (real/mock)
│   ├── market/              # Quote fetching & WebSocket
│   ├── account/             # Account state & P&L
│   └── risk/                # Risk control engine
├── configs/
│   └── credential           # API credentials
└── fs/                      # File-system interface (when using controller)
    ├── account/             # Account state & P&L
    ├── trade/               # Orders & risk control
    └── quote/               # Market data
```

## Development

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Clean build artifacts
make clean

# Download dependencies
make deps
```

## Examples

### AI Agent Integration

```python
import subprocess
import json

def get_quote(symbol):
    """Get real-time quote for a symbol"""
    result = subprocess.run(
        ['longbridge-fs', 'quote', symbol, '--format', 'json'],
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

def submit_market_order(symbol, side, quantity):
    """Submit a market order"""
    result = subprocess.run(
        ['longbridge-fs', 'order', 'submit', symbol, side, str(quantity), '--format', 'json'],
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

def get_positions():
    """Get current positions"""
    result = subprocess.run(
        ['longbridge-fs', 'account', 'positions', '--format', 'json'],
        capture_output=True,
        text=True
    )
    return json.loads(result.stdout)

# Example usage
quote = get_quote('AAPL.US')
print(f"AAPL price: ${quote['last']}")

# Check positions
positions = get_positions()
for channel in positions:
    for pos in channel['positions']:
        print(f"{pos['symbol']}: {pos['quantity']} shares @ {pos['cost_price']}")

# Submit order
order = submit_market_order('TSLA.US', 'BUY', 10)
print(f"Order submitted: {order['order_id']}")
```

### Scripting Workflows

```bash
#!/bin/bash
# Daily portfolio check

echo "=== Account Balance ==="
longbridge-fs account balance

echo -e "\n=== Current Positions ==="
longbridge-fs account positions

echo -e "\n=== Today's Orders ==="
longbridge-fs account orders

echo -e "\n=== Key Holdings ==="
longbridge-fs quote AAPL.US TSLA.US NVDA.US
```

## Authentication

The CLI uses OAuth 2.0 via the Longbridge SDK. Token management is handled automatically by the SDK.

```bash
# Verify credentials and test connectivity
longbridge-fs check

# Output:
# Checking Longbridge API connectivity...
#
# Credential file: credential
# Status: ✓ Found
#
# Loading credentials... ✓ Success
# Detecting region... ✓ Global (CN auto-detection not yet implemented)
# Testing Trade API... ✓ Connected
# Testing Quote API... ✓ Connected
#
# ✓ All checks passed. API is ready to use.
```

Note: OAuth browser flow and China Mainland auto-detection are planned future enhancements.

## Configuration

### Credential File Format

The credential file uses key=value format:

```
api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN
```

### Environment Variables

You can also set credentials via environment variables (future enhancement):

```bash
export LONGBRIDGE_API_KEY=YOUR_APP_KEY
export LONGBRIDGE_SECRET=YOUR_APP_SECRET
export LONGBRIDGE_ACCESS_TOKEN=YOUR_ACCESS_TOKEN
```

## Troubleshooting

### Authentication Errors

```bash
# Verify credentials
longbridge-fs check

# Check credential file format
cat configs/credential
```

### API Connectivity Issues

```bash
# Test with verbose output
longbridge-fs quote AAPL.US --verbose

# Use mock mode for testing
longbridge-fs controller --root ./fs --mock
```

### Order Submission Failures

- Check account balance: `longbridge-fs account balance`
- Verify market hours
- Check symbol format (e.g., `AAPL.US`, `700.HK`)
- Review order type and parameters

## FAQ

### Q: What markets are supported?

A: All markets supported by Longbridge API: Hong Kong (HK), US (US), China A-shares (CN), etc.

### Q: Can I use this in production?

A: Yes, but use proper risk management. Start with small positions and use the `--mock` mode for testing.

### Q: How do I get API credentials?

A: Sign up at the [Longbridge Developer Portal](https://open.longportapp.com) to obtain API keys.

### Q: Can I run multiple commands in parallel?

A: Yes, CLI commands can be run in parallel. However, avoid running multiple `controller` instances on the same file system.

### Q: How is this different from the official Longbridge Terminal?

A: This is an independent implementation focused on AI-agent integration and file-system based workflows. It provides both CLI commands and a file-based interface.

## Contributing

Contributions are welcome! Please submit issues and pull requests on GitHub.

## Related Links

- [Longbridge OpenAPI Documentation](https://open.longportapp.com/docs)
- [Longbridge OpenAPI Go SDK](https://github.com/longportapp/openapi-go)
- [Beancount Documentation](https://beancount.github.io/docs/)

## License

MIT License - see [LICENSE](LICENSE) file for details
