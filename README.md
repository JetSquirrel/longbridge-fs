# Longbridge FS

> File-system native interface for Longbridge — AI/脚本通过读写文件完成行情、下单、风控与账户管理。

## 概览

- **文件即接口**：追加 `ORDER`、创建触发文件、读取 JSON/文本即可驱动全流程。
- **Controller 守护进程**：轮询 `fs/`，执行订单、刷新行情与账户、生成盈亏/组合、触发风控、归档账本。
- **双模式**：真实 API 与 Mock 之间一键切换，便于联调和回归。
- **可审计**：所有交易事件都写入 Beancount 账本，可追溯、可归档。
- **兼容 CLI**：保留 CLI 子命令用于旧脚本；推荐优先使用文件系统工作流。
- **Content API**：支持查询讨论话题、新闻资讯、公司文件等内容数据（OpenAPI v0.22.0 新增功能）。

## 快速开始（FS 工作流）

1) 编译
```bash
make build
# 或
go build -o build/longbridge-fs ./cmd/longbridge-fs
```

2) 准备凭据 `configs/credential`
```
api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN
```

3) 初始化文件系统
```bash
./build/longbridge-fs init --root ./fs
```

4) 启动 Controller
```bash
# 真实 API
./build/longbridge-fs controller --root ./fs --credential ./configs/credential

# Mock（不连 API，便于流程验证）
./build/longbridge-fs controller --root ./fs --mock
```

5) 目录一览
```
fs/
├── account/              # state.json, pnl.json
├── trade/                # beancount.txt, blocks/, risk_control.json
├── quote/                # track/, subscribe/, hold/, portfolio.json
└── .kill                 # 触发安全退出
```

## 文件系统操作速览

- **提交订单**：向 `fs/trade/beancount.txt` 追加一条 `ORDER`
  ```
  2026-03-24 * "ORDER" "BUY AAPL.US 100"
    ; intent_id: 20260324-001
    ; side: BUY
    ; symbol: AAPL.US
    ; qty: 100
    ; type: LIMIT
    ; price: 180.50
    ; tif: DAY
  ```
  Controller 执行后追加 `EXECUTION` 或 `REJECTION`，并在达到 `--compact-after` 阈值时归档到 `trade/blocks/`。

- **拉取行情（一次性）**：`touch fs/quote/track/TSLA.US`，轮询后数据写入 `fs/quote/hold/TSLA.US/`（overview、intraday、D/W/M/Y/5D）。

- **订阅实时行情**：`touch fs/quote/subscribe/TSLA.US`，推送数据持续刷新 `hold/TSLA.US/overview.{json,txt}`；取消订阅 `touch fs/quote/unsubscribe/TSLA.US`。

- **账户与组合**：`fs/account/state.json`（余额/持仓/当日订单，需真实 API），`fs/account/pnl.json`（基于 overview 计算），`fs/quote/portfolio.json`（行情+持仓聚合）。

- **风控**：在 `fs/trade/risk_control.json` 配置止损/止盈，例如：
  ```json
  {
    "AAPL.US": { "stop_loss": 150, "take_profit": 210, "qty": "50" }
  }
  ```
  触发后自动写入卖出 `ORDER` 并移除规则。

- **Kill Switch**：`touch fs/.kill`，下一轮轮询安全退出。

## Controller 主要参数

| 参数 | 作用 | 默认 |
| --- | --- | --- |
| `--root` | FS 根目录 | `.` |
| `--credential` | Longbridge 凭据文件 | `credential` |
| `--interval` | 轮询间隔 | `2s` |
| `--mock` | 不连接 API，使用本地 Mock | `false` |
| `--compact-after` | 执行订单数达到 N 后归档，0 关闭 | `10` |
| `-v, --verbose` | 输出详细日志 | `false` |

完整说明见 [docs/api-reference.md](docs/api-reference.md)。

## CLI 子命令（可选）

除文件系统工作流外，longbridge-fs 还提供了丰富的 CLI 子命令用于直接调用 API：

```bash
# 行情数据
./build/longbridge-fs quote AAPL.US TSLA.US          # 实时报价
./build/longbridge-fs klines AAPL.US --period day --count 30  # K线数据
./build/longbridge-fs filings AAPL.US                # 公司文件

# 内容数据（新增）
./build/longbridge-fs content topics AAPL.US         # 讨论话题
./build/longbridge-fs content news AAPL.US           # 新闻资讯

# 账户与交易
./build/longbridge-fs account balance                # 账户余额
./build/longbridge-fs account positions              # 持仓信息
./build/longbridge-fs order submit AAPL.US BUY 100 --type LIMIT --price 180.50

# 身份验证
./build/longbridge-fs login                          # 验证凭据
./build/longbridge-fs check                          # 检查 API 连接
```

完整命令列表：`./build/longbridge-fs --help`

## 文档

- [文件系统结构](docs/filesystem.md)
- [订单格式](docs/order-format.md)
- [行情数据](docs/market-data.md)
- [风控配置](docs/risk-control.md)
- [AI Agent 使用指南](docs/ai-agent-guide.md)
- [架构设计](docs/architecture.md)
- **[Phase 1: 五层架构实现](docs/phase1-implementation.md)** ⭐ NEW
- [五层架构完整规范](docs/five-layer-spec.md)
- [五层架构任务分解](docs/five-layer-task.md)
- [更多条目索引](docs/README.md)

## 开发

```bash
make fmt     # 格式化
make lint    # 静态检查
make test    # 运行测试
make clean   # 清理
```
