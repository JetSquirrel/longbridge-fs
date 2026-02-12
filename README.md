# Longbridge FS

> AI 驱动的港股/美股交易文件系统

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](go.mod)

## 简介

Longbridge FS 是一个基于文件系统的股票交易框架，通过读写文件完成行情查询与交易操作。天然适配 AI Agent 的文件操作能力。

**核心理念**：将股票交易抽象为文件系统操作，让 AI Agent 可以像操作文件一样进行交易。无需学习复杂的 API，只需读写文本文件即可完成行情查询、订单提交、风控管理等所有操作。

### 核心特性

- **文件驱动** — 通过文件读写完成所有交易操作，AI Agent 无需学习 API
- **AI 友好** — JSON 输出，天然适配 AI Agent 的文件操作能力
- **审计追踪** — 所有交易记录在 beancount 格式的 append-only 账本中
- **盈亏追踪** — 自动生成 `pnl.json` 和 `portfolio.json`，实时追踪持仓盈亏
- **风控止损** — 配置 `risk_control.json` 实现自动止损/止盈
- **容错降级** — 网络故障时自动切换到 Mock 模式，保证开发测试不中断
- **Kill Switch** — 创建 `.kill` 文件即可安全停止 controller

### 适用场景

- 使用 AI Agent 进行自动化交易
- 构建基于规则的交易系统
- 交易策略回测与模拟
- 学习股票交易 API
- 交易记录审计与分析

## 项目结构

```
longbridge-fs/
├── cmd/
│   └── longbridge-fs/
│       └── main.go              # 入口：init / controller 命令
├── internal/
│   ├── model/types.go           # 共享数据结构
│   ├── ledger/
│   │   ├── parser.go            # Beancount 解析
│   │   └── compact.go           # 已执行订单归档压缩
│   ├── credential/credential.go # 凭证加载 → config.Config
│   ├── broker/broker.go         # 订单执行（SDK / Mock）
│   ├── market/market.go         # 行情获取（overview / K线 / 分时）
│   ├── account/account.go       # 账户状态 / PnL / Portfolio
│   └── risk/risk.go             # 风控引擎（止损/止盈）
├── configs/
│   └── credential               # API 凭证 (key=value)
├── fs/                          # 运行时数据目录
│   ├── account/
│   │   ├── state.json           # 账户状态
│   │   └── pnl.json             # 持仓盈亏
│   ├── trade/
│   │   ├── beancount.txt        # 交易账本
│   │   ├── risk_control.json    # 风控规则
│   │   └── blocks/              # 归档区块
│   └── quote/
│       ├── hold/{SYMBOL}/       # 行情数据 (.txt + .json)
│       ├── track/               # 行情触发器
│       └── portfolio.json       # 组合总览
├── Makefile
├── go.mod
└── go.sum
```

## 快速开始

### 编译

```bash
make build
# 或
go build -o build/longbridge-fs ./cmd/longbridge-fs
```

### 配置凭证

创建 `configs/credential`：

```
api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN
```

### 运行

```bash
# 初始化文件系统
make init
# 或: ./build/longbridge-fs init --root ./fs

# Mock 模式（无需 API）
make dev
# 或: ./build/longbridge-fs controller --root ./fs --mock

# 真实 API
make run
# 或: ./build/longbridge-fs controller --root ./fs --credential ./configs/credential
```

### CLI 参数

```
Usage: longbridge-fs <command> [options]

Commands:
  init         初始化 FS 目录结构
  controller   启动交易控制器守护进程

Options for controller:
  --root PATH           FS 根目录 (default: .)
  --interval DURATION   轮询间隔 (default: 2s)
  --credential FILE     凭证文件路径 (default: credential)
  --mock                Mock 模式，不调用 API (default: false)
  --compact-after N     每 N 笔执行后压缩归档, 0=禁用 (default: 10)
```

## 使用示例

### 提交订单

```bash
cat >> fs/trade/beancount.txt << 'EOF'
2026-02-11 * "ORDER" "BUY 9988.HK Alibaba"
  ; intent_id: 20260211-001
  ; side: BUY
  ; symbol: 9988.HK
  ; market: HK
  ; qty: 1000
  ; type: LIMIT
  ; price: 161
  ; tif: DAY
EOF
```

Controller 会自动检测新 ORDER 并执行，结果追加为 EXECUTION 或 REJECTION。

### 查询行情

```bash
# 触发行情获取
touch fs/quote/track/AAPL.US

# Controller 获取后数据在 hold 目录，track 文件被自动删除
cat fs/quote/hold/AAPL.US/overview.json   # 实时报价 (JSON)
cat fs/quote/hold/AAPL.US/overview.txt    # 实时报价 (文本)
cat fs/quote/hold/AAPL.US/D.json          # 日K (120天, JSON)
cat fs/quote/hold/AAPL.US/W.json          # 周K (52周, JSON)
cat fs/quote/hold/AAPL.US/5D.json         # 5分钟K线 (JSON)
cat fs/quote/hold/AAPL.US/intraday.json   # 分时数据 (JSON)
```

### 查看盈亏

```bash
# PnL 报告 (持仓 + 当前价格 → 未实现盈亏)
cat fs/account/pnl.json

# 组合总览 (所有持仓 + 行情)
cat fs/quote/portfolio.json
```

### 风控止损

```bash
# 配置止损/止盈规则
cat > fs/trade/risk_control.json << 'EOF'
{
  "700.HK":  { "stop_loss": 280.0, "take_profit": 350.0 },
  "AAPL.US": { "stop_loss": 150.0, "take_profit": 210.0, "qty": "10" }
}
EOF
# 当价格触及阈值时，controller 自动追加 SELL ORDER 到 beancount.txt
# 触发后规则自动移除，避免重复下单
```

### Kill Switch

```bash
touch fs/.kill    # controller 下一轮检测到后安全退出
```

## 开发

```bash
make fmt      # 格式化
make lint     # 静态检查
make test     # 运行测试
make clean    # 清理构建
make deps     # 下载依赖
```

## 进阶功能

### 批量订单处理

Controller 会批量处理所有未执行的 ORDER，每个订单处理完成后追加结果：

```bash
# 一次性提交多个订单
cat >> fs/trade/beancount.txt << 'EOF'
2026-02-11 * "ORDER" "BUY AAPL"
  ; intent_id: 20260211-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; price: 180.00
  ; tif: DAY

2026-02-11 * "ORDER" "BUY MSFT"
  ; intent_id: 20260211-002
  ; side: BUY
  ; symbol: MSFT.US
  ; qty: 50
  ; type: MARKET
  ; tif: DAY
EOF
```

### 订单状态追踪

每个订单执行后，controller 会追加 EXECUTION 或 REJECTION 记录：

```
2026-02-11 * "EXECUTION" "BUY AAPL.US @ 179.50"
  ; intent_id: 20260211-001
  ; order_id: 1234567890
  ; side: BUY
  ; symbol: AAPL.US
  ; filled_qty: 100
  ; avg_price: 179.50
  ; status: FILLED
  ; executed_at: 2026-02-11T10:30:00Z
```

### 账本归档与压缩

当执行订单数量达到 `--compact-after` 阈值时，controller 会自动将已执行订单归档到 `fs/trade/blocks/` 目录：

```
fs/trade/blocks/
├── block_0001.txt    # 前 10 笔订单
├── block_0002.txt    # 第 11-20 笔订单
└── ...
```

主账本文件 `beancount.txt` 只保留未执行的订单，保持文件精简。

### 行情数据格式

行情数据同时提供 JSON 和文本两种格式：

**JSON 格式**（适合程序解析）：
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
  "timestamp": "2026-02-11T16:00:00Z"
}
```

**文本格式**（适合人类阅读）：
```
Symbol: AAPL.US
Price: $180.50  Change: +1.50 (+0.84%)
Open: $179.50  High: $181.00  Low: $178.50
Volume: 45,000,000  Turnover: $8.1B
Updated: 2026-02-11 16:00:00 EST
```

## 故障排查

### Controller 无法启动

1. 检查凭证文件是否存在且格式正确
2. 使用 `--mock` 模式测试基本功能
3. 查看日志输出，检查具体错误信息

### 订单未执行

1. 检查 beancount.txt 格式是否正确（每个字段必须以 `;` 开头）
2. 确认 controller 正在运行（`ps aux | grep longbridge-fs`）
3. 查看是否有 REJECTION 记录，了解拒绝原因

### 行情数据未更新

1. 确认 track 文件已创建
2. 等待下一个轮询周期（默认 2 秒）
3. 检查网络连接和 API 配额

## 安全建议

- 不要将 `configs/credential` 文件提交到版本控制系统
- 使用 Mock 模式进行开发和测试
- 在生产环境设置合理的风控规则
- 定期备份 beancount.txt 账本文件
- 使用只读 API token 进行行情查询

## 常见问题

### Q: 为什么使用 Beancount 格式？

A: Beancount 是一种成熟的复式记账格式，具有良好的可读性和可审计性。每笔交易都是 append-only，便于追溯历史记录。

### Q: 如何实现定时交易？

A: 可以配合 cron 或其他调度工具，在指定时间写入 ORDER 到 beancount.txt 文件。

### Q: 支持哪些市场？

A: 支持所有 Longbridge API 支持的市场，包括港股（HK）、美股（US）、A股（CN）等。

### Q: 可以同时运行多个 controller 吗？

A: 不建议。多个 controller 可能导致订单重复执行。如需分布式部署，请使用文件锁机制。

## 相关链接

- [Longbridge OpenAPI Go SDK](https://github.com/longportapp/openapi-go)
- [Beancount 文档](https://beancount.github.io/docs/)
- [项目文档](./docs/)

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## License

MIT License - 详见 [LICENSE](LICENSE) 文件
