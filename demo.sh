#!/bin/bash
set -e

BIN="go run ./cmd/longbridge-fs"

echo "==> 1. 初始化文件系统"
$BIN init --root ./fs

echo ""
echo "==> 2. 查看初始账户状态"
cat fs/account/state.json

echo ""
echo "==> 3. 查看初始 beancount 文件"
cat fs/trade/beancount.txt

echo ""
echo "==> 4. 启动 controller（mock 模式，后台运行）"
$BIN controller --root ./fs --interval 1s --mock --compact-after 4 &
CONTROLLER_PID=$!
sleep 1

echo ""
echo "==> 5. 写入第一个交易指令"
cat >> fs/trade/beancount.txt << 'EOF'
2026-02-11 * "ORDER" "BUY NVDA"
  ; intent_id: 20260211-0001
  ; side: BUY
  ; symbol: NVDA
  ; qty: 1
  ; type: MARKET
  ; tif: DAY

EOF

echo "等待 controller 执行..."
sleep 3

echo ""
echo "==> 6. 查看执行结果"
tail -n 20 fs/trade/beancount.txt

echo ""
echo "==> 7. 写入第二个交易指令"
cat >> fs/trade/beancount.txt << 'EOF'
2026-02-11 * "ORDER" "SELL AAPL"
  ; intent_id: 20260211-0002
  ; side: SELL
  ; symbol: AAPL
  ; qty: 1
  ; type: LIMIT
  ; price: 250.00
  ; tif: GTC

EOF

echo "等待 controller 执行..."
sleep 3

echo ""
echo "==> 8. 查看最终账本"
cat fs/trade/beancount.txt

echo ""
echo "==> 9. 测试 track 订阅文件"
touch fs/quote/track/NVDA.US
touch fs/quote/track/AAPL.US
echo "已创建 track 文件: $(ls fs/quote/track/)"


echo ""
echo "==> Demo 完成！"
