# Phase 1 Implementation Summary

## Overview

Phase 1 of the Five-Layer Harness has been successfully implemented. This phase adds the foundational directory structure and pre-trade risk control system to Longbridge-FS.

## What's New

### 1. Directory Structure

The `fs init` command now creates a complete five-layer harness directory structure:

```
fs/
├── research/                  # L1 Research Layer
│   ├── feeds/
│   │   ├── news/
│   │   ├── topics/
│   │   └── custom/
│   ├── watchlist.json
│   └── summary.json
├── signal/                    # L2 Signal Layer
│   ├── definitions/
│   ├── output/
│   └── active.json
├── portfolio/                 # L3 Portfolio Layer
│   ├── rebalance/
│   ├── history/
│   └── current.json
├── trade/
│   ├── risk/                  # L4 Risk Control Layer
│   │   ├── policy.json
│   │   ├── pre_trade.json
│   │   ├── position_limits.json
│   │   ├── daily_limits.json
│   │   ├── status.json
│   │   └── violations.jsonl
│   └── ...
└── audit/                     # Audit logs
```

### 2. Pre-Trade Risk Control (`internal/riskgate/`)

A new risk gate system that validates orders **before** execution:

**Features:**
- **Symbol Blocklist/Allowlist**: Control which symbols can be traded
- **Order Size Limits**: Enforce maximum order sizes by value and percentage
- **Position Limits**: Prevent exceeding position count and size limits
- **Order Frequency**: Throttle orders per hour/day
- **Daily Loss Limits**: Halt trading on excessive losses (future enhancement)

**Operating Modes:**
- `ENFORCE`: Reject orders that violate rules (default)
- `WARN`: Log violations but allow orders to proceed
- `DISABLED`: Skip all checks

**Configuration Files:**

`trade/risk/policy.json` - Main risk policy:
```json
{
  "version": 1,
  "enabled": false,
  "mode": "ENFORCE",
  "pre_trade_checks": true,
  "post_trade_monitoring": true,
  "daily_loss_limit": {
    "enabled": false,
    "max_loss_pct": 0.03,
    "action": "HALT"
  },
  "order_frequency": {
    "enabled": false,
    "max_orders_per_hour": 20,
    "max_orders_per_day": 100
  }
}
```

`trade/risk/pre_trade.json` - Pre-trade rules:
```json
{
  "max_single_order_pct": 0.10,
  "max_single_order_value": 50000,
  "allowed_symbols": [],
  "blocked_symbols": [],
  "allowed_sides": ["BUY", "SELL"],
  "require_limit_price": false,
  "max_deviation_from_market_pct": 0.05
}
```

`trade/risk/position_limits.json` - Position limits:
```json
{
  "max_position_pct": 0.25,
  "max_positions_count": 15,
  "sector_limits": {},
  "per_symbol_limits": {}
}
```

### 3. Extended Order Metadata

Orders now support traceability fields for audit purposes:

```
2026-03-30 * "ORDER" "BUY AAPL.US 100"
  ; intent_id: 20260330-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; price: 180.50
  ; tif: DAY
  ; source: manual                    # NEW: manual, rebalance, risk_trigger
  ; rebalance_id: rebal-20260330-001  # NEW: links to portfolio rebalance
  ; signal_refs: sma_crossover,llm_sentiment  # NEW: triggering signals
```

### 4. Audit Logging (`internal/audit/`)

Basic audit infrastructure for tracking controller cycles:

`audit/{date}/{cycle_id}.json` - Cycle audit log:
```json
{
  "cycle_id": "cycle-20260330-081000",
  "timestamp": "2026-03-30T08:10:00Z",
  "duration_ms": 450,
  "steps": {
    "risk": {
      "orders_checked": 2,
      "orders_passed": 1,
      "orders_rejected": 1,
      "rejections": [
        { "intent_id": "20260330-005", "rule": "max_single_order_pct" }
      ]
    },
    "execution": {
      "orders_submitted": 1,
      "executions": 1,
      "rejections": 0
    }
  }
}
```

### 5. Risk Violations Log

All risk rule violations are recorded in `trade/risk/violations.jsonl`:

```json
{"timestamp":"2026-03-30T10:15:00Z","rule":"max_single_order_pct","intent_id":"20260330-005","detail":"Order value $15000 = 15% of equity, limit 10%","action":"REJECTED"}
{"timestamp":"2026-03-30T11:30:00Z","rule":"blocked_symbol","intent_id":"20260330-006","detail":"Symbol GME.US is blocked by risk policy","action":"REJECTED"}
```

## How to Enable Risk Control

1. Initialize a new FS or update existing:
   ```bash
   longbridge-fs init --root /path/to/fs
   ```

2. Enable risk control in `trade/risk/policy.json`:
   ```json
   {
     "enabled": true,
     "mode": "ENFORCE"
   }
   ```

3. Configure rules in `trade/risk/pre_trade.json`:
   ```json
   {
     "max_single_order_pct": 0.10,
     "blocked_symbols": ["GME.US", "AMC.US"]
   }
   ```

4. Run the controller:
   ```bash
   longbridge-fs controller --root /path/to/fs --mock
   ```

5. Orders violating risk rules will be rejected with `RISK_*` prefixed reasons:
   ```
   2026-03-30 * "REJECTION" "BUY GME.US"
     ; intent_id: 20260330-005
     ; status: REJECTED
     ; reason: RISK_BLOCKED_SYMBOL: Symbol GME.US is blocked by risk policy
   ```

## Integration with Existing Code

- **Backward Compatible**: Risk gate is disabled by default
- **Legacy Support**: Old `trade/risk_control.json` (stop-loss/take-profit) continues to work
- **Order Format**: Existing orders work unchanged; new metadata fields are optional
- **Controller**: Pre-trade checks inserted before order execution, zero impact if disabled

## What's Next (Future Phases)

Phase 1 provides the **infrastructure** for the five-layer harness. Future phases will add:

- **Phase 2**: Portfolio construction (target.json → diff.json → rebalance automation)
- **Phase 3**: Research & Signal layers (watchlist, Content API integration, builtin indicators)
- **Phase 4**: Algorithm execution (TWAP, ICEBERG order slicing)
- **Phase 5**: Agent integration enhancements (MCP tools, end-to-end demos)

## Technical Details

### New Internal Packages

- `internal/riskgate/gate.go` - Pre-trade validation engine
- `internal/audit/audit.go` - Cycle audit logging

### Modified Files

- `cmd/longbridge-fs/main.go` - Extended `fs init` with new directories
- `internal/model/types.go` - Added risk control and metadata types
- `internal/ledger/parser.go` - Parse extended ORDER metadata
- `internal/broker/broker.go` - Integrated risk gate into ProcessLedger

### Key Functions

- `riskgate.NewGate(root)` - Initialize risk gate from config
- `gate.CheckOrder(order, accountState)` - Validate order against rules
- `gate.RecordViolation(order, result)` - Log violations to JSONL
- `gate.UpdateStatus(passed)` - Update risk gate status
- `gate.IncrementOrderCount()` - Track order frequency

## Testing

All functionality tested in mock mode:

```bash
# Initialize FS
cd /tmp && mkdir test-fs && cd test-fs
longbridge-fs init --root .

# Verify directory structure
find . -type d | sort

# Verify config files
find . -type f -name "*.json" | sort

# Enable risk control and test
# Edit trade/risk/policy.json: "enabled": true
# Write test ORDER to trade/beancount.txt
# Run controller and observe risk gate behavior
```

## Migration Notes

For existing Longbridge-FS installations:

1. Run `longbridge-fs init` in your existing FS root - it will create missing directories without overwriting existing files
2. Risk gate is **disabled by default** - no impact on existing workflows
3. To enable: edit `trade/risk/policy.json` and set `"enabled": true`
4. Legacy `trade/risk_control.json` continues to work unchanged

## Resources

- Full spec: `docs/five-layer-spec.md`
- Task breakdown: `docs/five-layer-task.md`
- Source code: `internal/riskgate/`, `internal/audit/`

---

**Phase 1 Status**: ✅ Complete and tested
**Date**: 2026-03-30
**Version**: v0.3.0 (planned)
