# 文件系统结构

Longbridge FS 将交易、行情、账户、风控全部映射为文件读写。Controller 守护进程轮询这些文件夹，完成指令解析与状态更新。

## 初始化

```bash
# 创建目录和默认文件
longbridge-fs init --root ./fs
```

执行后会生成基础目录和文件：

```
fs/
├── account/
│   ├── state.json          # 账户余额、持仓、当日订单（仅真实 API 模式更新）
│   └── pnl.json            # 持仓盈亏（基于 overview 价格计算）
├── trade/
│   ├── beancount.txt       # 追加式账本，包含 ORDER/EXECUTION/REJECTION
│   ├── blocks/             # 已执行订单的归档区块
│   └── risk_control.json   # 止损/止盈配置
├── quote/
│   ├── track/              # 触发一次性行情获取的空文件
│   ├── subscribe/          # 订阅实时行情的空文件
│   ├── unsubscribe/        # 取消订阅的空文件
│   ├── hold/               # 行情输出目录，按符号分文件夹
│   ├── market/             # 预留目录
│   └── portfolio.json      # 组合汇总（positions + hold/overviews）
└── .kill                   # 可选，存在即安全退出 Controller
```

## 核心目录与文件

### account/
- `state.json`：账户余额、持仓、当日订单列表。仅在连接真实 Trade API 时刷新。
- `pnl.json`：基于 `quote/hold/*/overview.json` 计算的未实现盈亏。

### trade/
- `beancount.txt`：追加式账本。AI/脚本写入 `ORDER`，Controller 追加 `EXECUTION/REJECTION`，并按 `compact-after` 阈值归档到 `blocks/`。
- `blocks/`：被归档的历史区块文件，可拼接重建完整历史。
- `risk_control.json`：止损/止盈规则。触发时会自动写入新的 `ORDER`。

### quote/
- `track/`：创建同名空文件（如 `track/AAPL.US`）触发一次性拉取；完成后文件被删除。
- `subscribe/`：创建同名空文件触发实时订阅；数据由 WebSocket 推送到 `hold/`。
- `unsubscribe/`：创建同名空文件取消订阅。
- `hold/{SYMBOL}/`：行情输出目录，包含：
  - `overview.json` / `overview.txt`
  - `intraday.json` / `intraday.txt`
  - `D.json/W.json/M.json/Y.json/5D.json` 及对应 `.txt`
- `portfolio.json`：聚合全部 `hold/` 行情与持仓。

### 其他
- `.kill`：在 FS 根目录创建该文件，Controller 在下一轮轮询时会安全退出。

## 运行注意事项

- 轮询间隔默认 2s，可通过 `--interval` 调整。
- 归档阈值默认处理 10 笔执行后压缩，可通过 `--compact-after` 设置，设为 `0` 关闭归档。
- Mock 模式下（`--mock`）不会连接 Longbridge API，行情与账户刷新将被跳过，适合流程调试。真实行情与交易需关闭 `--mock` 并提供有效凭据。
