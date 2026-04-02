#!/bin/bash
# demo_e2e.sh — End-to-end Harness Pipeline Demo
# Demonstrates the full five-layer pipeline in mock mode:
#   L1 Research → L2 Signal → L3 Portfolio → L4 Risk → L5 Execution
#
# Usage:
#   ./demo_e2e.sh           # run full demo
#   ./demo_e2e.sh --clean   # remove fs/ directory first, then run
set -e

BIN="go run ./cmd/longbridge-fs"
FS_ROOT="./fs_e2e_demo"

# Optional cleanup
if [ "${1}" = "--clean" ]; then
  echo "==> Removing previous demo FS..."
  rm -rf "${FS_ROOT}"
fi

echo "================================================================"
echo "  Longbridge-FS  |  End-to-End Harness Pipeline Demo (Mock)"
echo "================================================================"
echo ""

# ------------------------------------------------------------------ #
# STEP 1: Initialize the five-layer FS
# ------------------------------------------------------------------ #
echo "==> [1/9] Initialize file system (five-layer structure)"
$BIN init --root "${FS_ROOT}"
echo ""

# ------------------------------------------------------------------ #
# STEP 2: L1 Research — configure watchlist
# ------------------------------------------------------------------ #
echo "==> [2/9] L1 Research: configure watchlist"
cat > "${FS_ROOT}/research/watchlist.json" << 'EOF'
{
  "symbols": ["AAPL.US", "TSLA.US", "700.HK"],
  "refresh_interval": "5m",
  "feeds": ["news", "topics"]
}
EOF
echo "    watchlist.json written with symbols: AAPL.US, TSLA.US, 700.HK"
echo ""

# ------------------------------------------------------------------ #
# STEP 3: L2 Signal — configure signal definitions
# ------------------------------------------------------------------ #
echo "==> [3/9] L2 Signal: create signal definitions"

cat > "${FS_ROOT}/signal/definitions/sma_cross.json" << 'EOF'
{
  "name": "sma_crossover",
  "type": "builtin",
  "enabled": true,
  "symbols": ["AAPL.US", "TSLA.US", "700.HK"],
  "params": {
    "indicator": "SMA_CROSS",
    "fast_period": 5,
    "slow_period": 20
  }
}
EOF

cat > "${FS_ROOT}/signal/definitions/rsi.json" << 'EOF'
{
  "name": "rsi_signal",
  "type": "builtin",
  "enabled": true,
  "symbols": ["AAPL.US", "TSLA.US"],
  "params": {
    "indicator": "RSI",
    "period": 14,
    "overbought": 70,
    "oversold": 30
  }
}
EOF

echo "    Signal definitions created: sma_crossover, rsi_signal"
echo ""

# ------------------------------------------------------------------ #
# STEP 4: L3 Portfolio — configure target allocation
# ------------------------------------------------------------------ #
echo "==> [4/9] L3 Portfolio: set target allocation"
cat > "${FS_ROOT}/portfolio/target.json" << 'EOF'
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
echo "    portfolio/target.json written (AAPL 40%, TSLA 35%, 700.HK 15%, NVDA 10%)"
echo ""

# ------------------------------------------------------------------ #
# STEP 5: L4 Risk — enable risk gate
# ------------------------------------------------------------------ #
echo "==> [5/9] L4 Risk: configure risk policy"
cat > "${FS_ROOT}/trade/risk/policy.json" << 'EOF'
{
  "version": 1,
  "enabled": true,
  "mode": "WARN",
  "pre_trade_checks": true,
  "post_trade_monitoring": false,
  "daily_loss_limit": {
    "enabled": false,
    "max_loss_pct": 0.05,
    "action": "HALT"
  },
  "order_frequency": {
    "enabled": false,
    "max_orders_per_hour": 50,
    "max_orders_per_day": 200
  }
}
EOF
cat > "${FS_ROOT}/trade/risk/pre_trade.json" << 'EOF'
{
  "max_single_order_pct": 0.20,
  "max_single_order_value": 100000,
  "allowed_symbols": [],
  "blocked_symbols": [],
  "allowed_sides": ["BUY", "SELL"],
  "require_limit_price": false,
  "max_deviation_from_market_pct": 0.10
}
EOF
echo "    Risk policy set to WARN mode with pre-trade checks enabled"
echo ""

# ------------------------------------------------------------------ #
# STEP 6: Start controller in mock + harness mode
# ------------------------------------------------------------------ #
echo "==> [6/9] Start controller in mock mode (background)"
$BIN controller --root "${FS_ROOT}" --interval 2s --mock --compact-after 10 &
CONTROLLER_PID=$!
echo "    Controller PID: ${CONTROLLER_PID}"
echo "    Waiting for initial cycle (research feeds + signals)..."
sleep 5
echo ""

# ------------------------------------------------------------------ #
# STEP 7: Inspect pipeline output (L1-L3)
# ------------------------------------------------------------------ #
echo "==> [7/9] Inspect pipeline state"

echo ""
echo "  --- L1 Research Summary ---"
if [ -f "${FS_ROOT}/research/summary.json" ]; then
  cat "${FS_ROOT}/research/summary.json"
else
  echo "  (no summary yet)"
fi

echo ""
echo "  --- L2 Active Signals ---"
if [ -f "${FS_ROOT}/signal/active.json" ]; then
  cat "${FS_ROOT}/signal/active.json"
else
  echo "  (no signals yet)"
fi

echo ""
echo "  --- L3 Portfolio Diff ---"
if [ -f "${FS_ROOT}/portfolio/diff.json" ]; then
  cat "${FS_ROOT}/portfolio/diff.json"
else
  echo "  (no diff yet)"
fi
echo ""

# ------------------------------------------------------------------ #
# STEP 8: L5 Execution — inject orders into ledger
# ------------------------------------------------------------------ #
echo "==> [8/9] L5 Execution: submit orders through beancount ledger"

TODAY=$(date +%Y-%m-%d)
TS=$(date +%Y%m%d-%H%M%S)

cat >> "${FS_ROOT}/trade/beancount.txt" << EOF
${TODAY} * "ORDER" "BUY AAPL.US via harness demo"
  ; intent_id: ${TS}-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 10
  ; type: MARKET
  ; tif: DAY
  ; source: rebalance

EOF

cat >> "${FS_ROOT}/trade/beancount.txt" << EOF
${TODAY} * "ORDER" "BUY TSLA.US via harness demo"
  ; intent_id: ${TS}-002
  ; side: BUY
  ; symbol: TSLA.US
  ; qty: 5
  ; type: MARKET
  ; tif: DAY
  ; source: rebalance

EOF

echo "    Orders appended. Waiting for controller to execute..."
sleep 4

echo ""
echo "  --- Beancount Ledger (last 40 lines) ---"
tail -n 40 "${FS_ROOT}/trade/beancount.txt"
echo ""

# ------------------------------------------------------------------ #
# STEP 9: Stop controller and print summary
# ------------------------------------------------------------------ #
echo "==> [9/9] Stop controller"
touch "${FS_ROOT}/.kill"
sleep 3
# Fallback: kill by PID if still running
if kill -0 "${CONTROLLER_PID}" 2>/dev/null; then
  kill "${CONTROLLER_PID}" 2>/dev/null || true
fi

echo ""
echo "================================================================"
echo "  Pipeline Summary"
echo "================================================================"
echo ""
echo "  File System Root : ${FS_ROOT}/"
echo ""
echo "  Layer       | Output File"
echo "  ------------|---------------------------------------------------"
echo "  L1 Research | ${FS_ROOT}/research/summary.json"
echo "              | ${FS_ROOT}/research/feeds/news/AAPL.US/latest.json"
echo "  L2 Signal   | ${FS_ROOT}/signal/active.json"
echo "              | ${FS_ROOT}/signal/output/AAPL.US/latest.json"
echo "  L3 Portfolio| ${FS_ROOT}/portfolio/current.json"
echo "              | ${FS_ROOT}/portfolio/diff.json"
echo "  L4 Risk     | ${FS_ROOT}/trade/risk/status.json"
echo "  L5 Execution| ${FS_ROOT}/trade/beancount.txt"
echo ""
echo "  To inspect any layer:"
echo "    cat ${FS_ROOT}/signal/active.json"
echo "    cat ${FS_ROOT}/portfolio/diff.json"
echo "    cat ${FS_ROOT}/trade/beancount.txt"
echo ""
echo "================================================================"
echo "  Demo complete! Run './demo_e2e.sh --clean' to start fresh."
echo "================================================================"
