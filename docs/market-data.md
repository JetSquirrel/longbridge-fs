# 行情数据

Controller 通过文件触发拉取或订阅行情，并将结果写入 `quote/hold/`。确保 Controller 已运行。

> ⚠️ Mock 模式下不会连接 Longbridge Quote API，`track/` 和 `subscribe/` 将不产出实时数据。

## 一次性拉取（track）

1. 创建触发文件：
   ```bash
   touch fs/quote/track/AAPL.US
   ```
2. 等待 1-2 个轮询周期（默认 2s），Controller 将：
   - 获取实时报价、分时、K 线（D/W/M/Y/5D）
   - 写入 `fs/quote/hold/AAPL.US/`
   - 删除触发文件

生成文件示例：
```
fs/quote/hold/AAPL.US/
├── overview.json  # 实时报价（含涨跌幅）
├── overview.txt   # 便于人读的文本版
├── intraday.json
├── intraday.txt
├── D.json / D.txt
├── W.json / W.txt
├── M.json / M.txt
├── Y.json / Y.txt
└── 5D.json / 5D.txt
```

重新拉取：再次 `touch fs/quote/track/AAPL.US` 即可。

## 实时订阅（subscribe）

1. 创建订阅文件：
   ```bash
   touch fs/quote/subscribe/TSLA.US
   ```
2. Controller 调用 WebSocket 订阅，随后每次推送都会刷新：
   - `fs/quote/hold/TSLA.US/overview.json`
   - `fs/quote/hold/TSLA.US/overview.txt`
3. 取消订阅：
   ```bash
   touch fs/quote/unsubscribe/TSLA.US
   ```

> 订阅文件在处理后会被移除，当前订阅状态由 Controller 内部维护。

## 组合视图

- `fs/quote/portfolio.json`：由 `hold/` 行情和 `account/state.json` 聚合生成，包含每个符号的最新价格、持仓数量、成本、估值与盈亏。
- 更新时机：每个轮询周期运行，无需触发文件。
