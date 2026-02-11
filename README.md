# Longbridge FS

> AI 驱动的港股/美股交易文件系统

## 简介

Longbridge FS 是一个基于文件系统的股票交易框架，通过读写文件完成行情查询与交易操作。天然适配 AI Agent 的文件操作能力。

### 核心特性

- **文件驱动** — 通过文件读写完成所有交易操作
- **AI 友好** — JSON 输出，天然适配 AI Agent 的文件操作能力
- **审计追踪** — 所有交易记录在 beancount 格式的 append-only 账本中
- **盈亏追踪** — 自动生成 `pnl.json` 和 `portfolio.json`
- **风控止损** — 配置 `risk_control.json` 实现自动止损/止盈
- **容错降级** — 网络故障时自动切换到 Mock 模式
- **Kill Switch** — 创建 `.kill` 文件即可安全停止 controller

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

## 相关链接

- [Longbridge OpenAPI Go SDK](https://github.com/longportapp/openapi-go)
- [Beancount](https://beancount.github.io/docs/)

