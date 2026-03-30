# 风控配置

`trade/risk_control.json` 用于配置止损/止盈规则。Controller 每个轮询周期都会读取并检查是否触发。

## 配置格式

```json
{
  "700.HK": {
    "stop_loss": 280.0,
    "take_profit": 350.0
  },
  "AAPL.US": {
    "stop_loss": 150.0,
    "take_profit": 210.0,
    "qty": "10",
    "side": "SELL"
  }
}
```

- `stop_loss`：价格 <= 该值触发
- `take_profit`：价格 >= 该值触发
- `qty`：可选，字符串格式。默认 `ALL`（卖出全部）。
- `side`：可选，默认为 `SELL`。若设置，可使用 `BUY/SELL`。

> 价格来源为 `quote/hold/{SYMBOL}/overview.json`，请确保行情已被拉取或订阅。

## 触发行为

1. 符合阈值时，生成一条新的 `ORDER` 追加到 `trade/beancount.txt`，默认市价单、当日有效：
   ```
   2026-03-24 ORDER
     intent_id: risk-700-HK-...
     side: SELL
     symbol: 700.HK
     qty: ALL
     order_type: MARKET
     tif: DAY
     reason: STOP_LOSS triggered: ...
   ```
2. 触发的规则会从 `risk_control.json` 中移除，避免重复下单。
3. 后续成交或拒单记录同样会写回账本（由 Broker/Controller 处理）。

## 使用建议

- 与 `quote/track` 或 `subscribe` 配合，保证 `overview.json` 中有最新价格。
- 将 `qty` 设置为具体数值可部分止盈/止损，省略则默认全部持仓。
- 模拟环境可用 `--mock` 先验证写入流程；真实交易需关闭 `--mock` 并提供凭据。
- 如需临时停止风控，可先 `touch fs/.kill` 结束 Controller，再修改配置。
