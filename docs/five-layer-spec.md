# Longbridge-FS Harness 五层架构 Spec

> **版本**: v0.1.0-draft
> **日期**: 2026-03-30
> **状态**: RFC (Request for Comments)

---

## 1. 背景与动机

Longbridge-FS 目前已实现文件系统驱动的交易执行、基础风控（止损/止盈）和行情获取。对照 Harness 架构——将买方基础设施（研究、信号、组合、风控、执行）编排为完整管道并通过智能体交付——项目在前三层（研究、信号、组合构建）存在明显缺口，风控层也需要从"被动触发"升级为"前置硬约束"。

本 Spec 定义将 Longbridge-FS 扩展为完整 Harness 五层架构的设计方案，保持项目核心理念不变：**文件即接口、append-only 审计、AI 友好、人机可读**。

---

## 2. 架构总览

```
                      Harness Pipeline (fs/ 驱动)

  ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐
  │  L1 研究   │───▶│  L2 信号   │───▶│  L3 组合   │───▶│  L4 风控   │───▶│  L5 执行   │
  │  Research  │    │  Signal    │    │ Portfolio  │    │   Risk     │    │ Execution  │
  │            │    │            │    │Construction│    │  Control   │    │            │
  │ 数据、分析 │    │ 阿尔法、   │    │ 目标权重、 │    │ 前置校验、 │    │ 算法拆单、 │
  │ 情绪、事件 │    │ 因子模型   │    │ Rebalance  │    │ 硬约束     │    │ 路由       │
  └────────────┘    └────────────┘    └────────────┘    └────────────┘    └────────────┘
       │                  │                 │                 │                 │
       ▼                  ▼                 ▼                 ▼                 ▼
  fs/research/       fs/signal/       fs/portfolio/     fs/trade/risk/     fs/trade/
  (新增)             (新增)           (扩展)           (重构)             (现有,扩展)
```

每一层通过文件系统目录解耦，上游写入 → Controller 轮询消费 → 写入下游，形成单向数据管道。AI Agent 或脚本可以在任意层注入或读取，但管道的完整性由 Controller 保证。

---

## 3. 目录结构变更

### 3.1 新增目录

```
fs/
├── research/                    # L1 研究层 (NEW)
│   ├── feeds/                   # 数据源输入
│   │   ├── news/                # 新闻摘要 (Controller 从 Content API 写入)
│   │   │   └── {SYMBOL}/
│   │   │       └── latest.json
│   │   ├── topics/              # 讨论话题
│   │   │   └── {SYMBOL}/
│   │   │       └── latest.json
│   │   └── custom/              # Agent/脚本自定义研究数据
│   │       └── {name}.json
│   ├── watchlist.json           # 研究关注列表
│   └── summary.json             # 聚合摘要 (Controller 生成)
│
├── signal/                      # L2 信号层 (NEW)
│   ├── definitions/             # 信号定义 (声明式配置)
│   │   └── {signal_name}.json
│   ├── output/                  # 信号计算结果
│   │   └── {SYMBOL}/
│   │       ├── latest.json      # 最新信号快照
│   │       └── history.jsonl    # 信号历史 (append-only)
│   └── active.json              # 当前活跃信号汇总
│
├── portfolio/                   # L3 组合构建层 (NEW, 从 quote/portfolio.json 升级)
│   ├── target.json              # 目标组合权重
│   ├── current.json             # 当前持仓快照 (Controller 生成)
│   ├── diff.json                # 目标 vs 当前的差异 (Controller 生成)
│   ├── rebalance/               # Rebalance 指令队列
│   │   └── pending.json         # 待执行的 rebalance 订单列表
│   └── history/                 # 历史目标快照
│       └── {timestamp}.json
│
├── account/                     # (现有, 不变)
├── trade/                       # L5 执行层 (现有, 扩展)
│   ├── beancount.txt
│   ├── blocks/
│   ├── risk_control.json        # (保留, 向后兼容)
│   └── risk/                    # L4 风控层 (NEW, 扩展)
│       ├── policy.json          # 风控策略总配置
│       ├── pre_trade.json       # 前置校验规则
│       ├── position_limits.json # 仓位限额
│       ├── daily_limits.json    # 日度限额与状态
│       ├── violations.jsonl     # 违规记录 (append-only)
│       └── status.json          # 风控状态快照
├── quote/                       # (现有, 不变)
├── audit/                       # 审计日志 (NEW)
│   └── {date}/
│       └── {cycle_id}.json
└── .kill
```

### 3.2 现有目录兼容

`quote/portfolio.json` 保留为只读视图，内容由 `portfolio/current.json` 软链或复制生成，确保现有 Agent 脚本不受影响。`trade/risk_control.json`（止损/止盈触发规则）继续工作，作为 L4 风控的一个子集。

---

## 4. L1 研究层 (Research)

### 4.1 目标

为信号层提供结构化数据输入，包括行情数据（已有）、新闻/话题（Content API）、以及 Agent 自定义的研究结果。

### 4.2 文件格式

#### `research/watchlist.json`

```json
{
  "symbols": ["AAPL.US", "TSLA.US", "700.HK"],
  "refresh_interval": "5m",
  "feeds": ["news", "topics"]
}
```

Controller 行为：根据 watchlist 定期调用 Content API，写入 `feeds/news/{SYMBOL}/latest.json` 和 `feeds/topics/{SYMBOL}/latest.json`。同时自动 `touch fs/quote/track/{SYMBOL}` 确保行情可用。

#### `research/feeds/news/{SYMBOL}/latest.json`

```json
{
  "symbol": "AAPL.US",
  "fetched_at": "2026-03-30T08:00:00Z",
  "items": [
    {
      "id": "news-001",
      "title": "Apple Q2 earnings beat expectations",
      "source": "reuters",
      "published_at": "2026-03-30T06:30:00Z",
      "summary": "..."
    }
  ]
}
```

#### `research/feeds/custom/{name}.json`

AI Agent 或外部脚本可以自由写入自定义研究数据，Controller 不解析此目录但会将其包含在 `summary.json` 的索引中。格式约定：

```json
{
  "name": "sector_rotation_analysis",
  "created_at": "2026-03-30T07:00:00Z",
  "author": "claude-agent",
  "data": { ... }
}
```

#### `research/summary.json`（Controller 生成）

```json
{
  "updated_at": "2026-03-30T08:05:00Z",
  "symbols": {
    "AAPL.US": {
      "has_quote": true,
      "has_news": true,
      "has_topics": true,
      "custom_feeds": ["sector_rotation_analysis"]
    }
  }
}
```

### 4.3 Controller 逻辑

在主轮询循环中新增 `refreshResearch()` 步骤，位于行情刷新之后：

1. 读取 `watchlist.json`
2. 对每个 symbol，根据 `refresh_interval` 判断是否需要刷新
3. 调用 Content API 的 news/topics 端点
4. 写入 `feeds/` 对应文件
5. 重新生成 `summary.json`

### 4.4 Go 包结构

```
internal/
  research/
    research.go       // RefreshFeeds(root, symbols, client) error
    watchlist.go      // ParseWatchlist / WriteWatchlist
    summary.go        // GenerateSummary
```

---

## 5. L2 信号层 (Signal)

### 5.1 目标

将研究数据和行情数据转化为可操作的交易信号。信号层采用**声明式 + 外部计算**的混合模式——简单技术指标由 Controller 内置计算，复杂信号由 Agent 外部写入。

### 5.2 信号定义

#### `signal/definitions/{signal_name}.json`

**内置信号（Controller 计算）**：

```json
{
  "name": "sma_crossover",
  "type": "builtin",
  "version": 1,
  "params": {
    "indicator": "SMA_CROSS",
    "fast_period": 5,
    "slow_period": 20,
    "data_source": "D"
  },
  "symbols": ["AAPL.US", "TSLA.US"],
  "enabled": true
}
```

支持的内置指标（Phase 1）：

| indicator     | 说明             | 参数                         |
|---------------|----------------|------------------------------|
| SMA_CROSS     | 均线交叉         | fast_period, slow_period     |
| RSI           | RSI 超买超卖     | period, overbought, oversold |
| PRICE_CHANGE  | 价格变动百分比   | threshold_pct, window        |

**外部信号（Agent 写入）**：

```json
{
  "name": "llm_sentiment",
  "type": "external",
  "version": 1,
  "description": "LLM-based news sentiment scoring",
  "symbols": ["AAPL.US"],
  "enabled": true
}
```

外部信号的 `type: "external"` 表示 Controller 不计算，仅读取 Agent 写入的 `output/` 结果。

### 5.3 信号输出

#### `signal/output/{SYMBOL}/latest.json`

```json
{
  "symbol": "AAPL.US",
  "updated_at": "2026-03-30T08:10:00Z",
  "signals": [
    {
      "name": "sma_crossover",
      "value": "BULLISH",
      "strength": 0.72,
      "detail": "SMA5 (182.3) crossed above SMA20 (179.8)",
      "computed_at": "2026-03-30T08:10:00Z"
    },
    {
      "name": "llm_sentiment",
      "value": "POSITIVE",
      "strength": 0.85,
      "detail": "Earnings beat + positive guidance",
      "computed_at": "2026-03-30T08:05:00Z"
    }
  ]
}
```

**`value` 枚举**: `BULLISH`, `BEARISH`, `NEUTRAL`, `POSITIVE`, `NEGATIVE`

**`strength`**: 0.0 ~ 1.0，信号强度

#### `signal/active.json`（Controller 生成）

```json
{
  "updated_at": "2026-03-30T08:10:00Z",
  "signals": [
    {
      "symbol": "AAPL.US",
      "name": "sma_crossover",
      "value": "BULLISH",
      "strength": 0.72
    }
  ]
}
```

### 5.4 Controller 逻辑

在 `refreshResearch()` 之后新增 `computeSignals()` 步骤：

1. 扫描 `signal/definitions/` 下所有 `enabled: true` 的定义
2. 对 `type: "builtin"` 的信号，从 `quote/hold/{SYMBOL}/` 读取 K 线数据，计算指标
3. 写入 `signal/output/{SYMBOL}/latest.json`，追加 `history.jsonl`
4. 对 `type: "external"` 的信号，仅检查 `output/` 是否存在更新
5. 重新生成 `active.json`

### 5.5 Go 包结构

```
internal/
  signal/
    signal.go         // ComputeAll(root, definitions) error
    indicator.go      // SMA, RSI, PriceChange 等内置计算
    definitions.go    // ParseDefinition / ListDefinitions
    output.go         // WriteOutput / ReadLatest / AppendHistory
```

---

## 6. L3 组合构建层 (Portfolio Construction)

### 6.1 目标

将信号转化为目标持仓权重，自动计算 rebalance 订单，替代手动逐条写 ORDER。

### 6.2 目标组合定义

#### `portfolio/target.json`

```json
{
  "version": 1,
  "updated_at": "2026-03-30T09:00:00Z",
  "updated_by": "claude-agent",
  "strategy": "signal_weighted",
  "total_capital_pct": 0.80,
  "positions": {
    "AAPL.US": {
      "weight": 0.30,
      "reason": "SMA bullish + positive sentiment",
      "signal_refs": ["sma_crossover", "llm_sentiment"]
    },
    "TSLA.US": {
      "weight": 0.20,
      "reason": "RSI oversold bounce",
      "signal_refs": ["rsi_oversold"]
    },
    "700.HK": {
      "weight": 0.30,
      "reason": "Sector rotation into CN tech",
      "signal_refs": ["sector_rotation"]
    }
  },
  "cash_reserve_pct": 0.20
}
```

字段说明：

- **total_capital_pct**: 可投入总资本的百分比，剩余保留现金
- **positions.weight**: 该标的占 total_capital_pct 部分的目标权重，所有 weight 之和应为 1.0
- **cash_reserve_pct**: 显式现金储备比例（= 1 - total_capital_pct）
- **signal_refs**: 触发该配置的信号名称，用于审计追踪

Agent 工作流：读取 `signal/active.json` → 决策 → 写入 `portfolio/target.json`。

### 6.3 Controller 生成的文件

#### `portfolio/current.json`

```json
{
  "updated_at": "2026-03-30T09:01:00Z",
  "total_equity": 100000.00,
  "positions": {
    "AAPL.US": {
      "qty": 200,
      "market_value": 36100.00,
      "weight": 0.361,
      "avg_cost": 175.50
    }
  },
  "cash": 28000.00,
  "cash_pct": 0.28
}
```

数据来源：`account/state.json`（持仓）+ `quote/hold/*/overview.json`（市值计算）。

#### `portfolio/diff.json`

```json
{
  "computed_at": "2026-03-30T09:01:00Z",
  "target_version": 1,
  "adjustments": [
    {
      "symbol": "AAPL.US",
      "current_weight": 0.361,
      "target_weight": 0.240,
      "action": "REDUCE",
      "delta_qty": -67,
      "delta_value": -12117.00,
      "estimated_side": "SELL",
      "estimated_qty": 67
    },
    {
      "symbol": "700.HK",
      "current_weight": 0.0,
      "target_weight": 0.240,
      "action": "ADD",
      "delta_qty": 80,
      "delta_value": 24000.00,
      "estimated_side": "BUY",
      "estimated_qty": 80
    }
  ],
  "requires_rebalance": true
}
```

### 6.4 Rebalance 执行

#### `portfolio/rebalance/pending.json`

当 `diff.json` 中 `requires_rebalance: true` 且用户或 Agent 确认执行时，将调整动作写入此文件：

```json
{
  "rebalance_id": "rebal-20260330-001",
  "created_at": "2026-03-30T09:05:00Z",
  "created_by": "claude-agent",
  "auto_execute": false,
  "orders": [
    {
      "symbol": "AAPL.US",
      "side": "SELL",
      "qty": 67,
      "type": "LIMIT",
      "price": 180.50,
      "tif": "DAY"
    },
    {
      "symbol": "700.HK",
      "side": "BUY",
      "qty": 80,
      "type": "LIMIT",
      "price": 300.00,
      "tif": "DAY"
    }
  ]
}
```

**执行模式**：

- `auto_execute: false`（默认）：Controller 生成 `diff.json` 供 Agent 审核，Agent 确认后写入 `pending.json`，Controller 将其转化为 ORDER 写入 `trade/beancount.txt`
- `auto_execute: true`：Controller 检测到 `diff.json` 变化后直接写入 ORDER（需在 `portfolio/target.json` 中显式声明）

所有 rebalance 生成的 ORDER 带有 `; source: rebalance` 和 `; rebalance_id: rebal-xxx` 元数据字段。

### 6.5 Controller 逻辑

在 `computeSignals()` 之后新增 `syncPortfolio()` 步骤：

1. 读取 `account/state.json` + `quote/hold/*/overview.json` → 生成 `current.json`
2. 如果 `target.json` 存在且版本 > 上次处理版本 → 计算 `diff.json`
3. 如果 `rebalance/pending.json` 存在且未处理 → 将 orders 逐条写入 `trade/beancount.txt`，然后将 `pending.json` 移动到 `portfolio/history/`

### 6.6 Go 包结构

```
internal/
  portfolio/
    portfolio.go     // SyncCurrent / ComputeDiff / ExecuteRebalance
    target.go        // ParseTarget / ValidateTarget
    diff.go          // Diff calculation logic
    rebalance.go     // PendingToOrders conversion
```

---

## 7. L4 风控层 (Risk Control)

### 7.1 目标

从当前的"被动止损/止盈触发"升级为**前置硬约束 + 被动触发**双引擎，确保所有 ORDER（无论来源）在执行前必须通过风控校验。

### 7.2 风控策略配置

#### `trade/risk/policy.json`

```json
{
  "version": 1,
  "enabled": true,
  "mode": "ENFORCE",
  "pre_trade_checks": true,
  "post_trade_monitoring": true,
  "daily_loss_limit": {
    "enabled": true,
    "max_loss_pct": 0.03,
    "action": "HALT"
  },
  "order_frequency": {
    "enabled": true,
    "max_orders_per_hour": 20,
    "max_orders_per_day": 100
  }
}
```

**`mode` 枚举**：

- `ENFORCE`：风控违规直接 REJECT，硬约束
- `WARN`：风控违规写入 `violations.jsonl` 但仍执行，软约束
- `DISABLED`：跳过风控

#### `trade/risk/pre_trade.json`

每笔 ORDER 执行前的校验规则：

```json
{
  "max_single_order_pct": 0.10,
  "max_single_order_value": 50000,
  "allowed_symbols": [],
  "blocked_symbols": ["GME.US", "AMC.US"],
  "allowed_sides": ["BUY", "SELL"],
  "require_limit_price": false,
  "max_deviation_from_market_pct": 0.05
}
```

| 规则                        | 说明                                        |
|-----------------------------|-------------------------------------------|
| max_single_order_pct        | 单笔订单不超过总资产 N%                      |
| max_single_order_value      | 单笔订单金额上限                            |
| blocked_symbols             | 禁止交易的标的                              |
| max_deviation_from_market_pct| 限价单偏离市场价不超过 N%                     |

#### `trade/risk/position_limits.json`

```json
{
  "max_position_pct": 0.25,
  "max_positions_count": 15,
  "sector_limits": {
    "US_TECH": 0.50,
    "HK_TECH": 0.30
  },
  "per_symbol_limits": {
    "TSLA.US": { "max_pct": 0.10 }
  }
}
```

#### `trade/risk/daily_limits.json`（Controller 维护）

```json
{
  "date": "2026-03-30",
  "starting_equity": 100000.00,
  "current_equity": 97500.00,
  "realized_pnl": -1200.00,
  "unrealized_pnl": -1300.00,
  "total_pnl_pct": -0.025,
  "orders_this_hour": 3,
  "orders_today": 12,
  "is_halted": false,
  "halt_reason": null
}
```

当 `total_pnl_pct` 超过 `policy.json` 中的 `max_loss_pct` 时，`is_halted` 设为 `true`，后续所有 ORDER 直接 REJECT，直到次日重置或手动解除。

### 7.3 违规记录

#### `trade/risk/violations.jsonl`（append-only）

```json
{"timestamp":"2026-03-30T10:15:00Z","rule":"max_single_order_pct","intent_id":"20260330-005","detail":"Order value $15000 = 15% of equity, limit 10%","action":"REJECTED"}
{"timestamp":"2026-03-30T11:30:00Z","rule":"daily_loss_limit","intent_id":"N/A","detail":"Daily loss -3.1% exceeded limit -3.0%","action":"HALTED"}
```

### 7.4 Controller 逻辑变更

**关键变更：风控校验插入 ORDER 执行之前**

当前 Controller 的订单处理流程：

```
解析 ORDER → 调用 Broker 执行 → 追加 EXECUTION/REJECTION
```

变更为：

```
解析 ORDER → 前置风控校验 → (通过) 调用 Broker 执行 → 追加 EXECUTION/REJECTION
                            → (拒绝) 追加 REJECTION (reason: RISK_...)
```

风控校验步骤 `preTradeCheck(order)`:

1. 检查 `policy.json` 是否启用
2. 检查 `daily_limits.json` 是否已 halt
3. 遍历 `pre_trade.json` 规则
4. 检查 `position_limits.json` 仓位约束
5. 检查 `order_frequency` 频率限制
6. 全部通过返回 `PASS`，否则返回 `REJECT` + 原因

### 7.5 向后兼容

`trade/risk_control.json`（现有止损/止盈配置）继续由 `internal/risk/` 包处理，逻辑不变。新增的前置风控是独立的 `internal/riskgate/` 包。两者并行运行，互不干扰。

### 7.6 Go 包结构

```
internal/
  riskgate/                  # 新增: 前置风控引擎
    gate.go                  // PreTradeCheck(order, state) (Pass|Reject, reason)
    policy.go                // ParsePolicy
    pretrade.go              // 单笔校验规则
    position.go              // 仓位限额校验
    daily.go                 // 日度限额维护与校验
    violations.go            // AppendViolation
  risk/                      # 现有: 止损/止盈触发 (不变)
    risk.go
```

---

## 8. L5 执行层 (Execution) 扩展

### 8.1 目标

在现有直接下单基础上，增加算法拆单能力，减少大单的市场冲击。

### 8.2 ORDER 格式扩展

在现有 ORDER 的扩展字段中新增以下可选字段：

```
2026-03-30 * "ORDER" "BUY AAPL.US 500 via TWAP"
  ; intent_id: 20260330-010
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 500
  ; type: LIMIT
  ; price: 182.00
  ; tif: DAY
  ; algo: TWAP
  ; algo_duration: 30m
  ; algo_slices: 5
  ; source: rebalance
  ; rebalance_id: rebal-20260330-001
  ; signal_refs: sma_crossover,llm_sentiment
```

新增字段说明：

| 字段          | 类型   | 说明                                     |
|---------------|--------|----------------------------------------|
| algo          | 枚举   | NONE (默认), TWAP, ICEBERG              |
| algo_duration | 时长   | 算法执行时间窗口                         |
| algo_slices   | 整数   | 拆分份数                                 |
| source        | 字符串 | 订单来源: manual, rebalance, risk_trigger |
| rebalance_id  | 字符串 | 关联的 rebalance ID                      |
| signal_refs   | 字符串 | 触发该订单的信号列表 (逗号分隔)           |

### 8.3 算法执行

#### TWAP (Time-Weighted Average Price)

将大单拆分为 N 份（`algo_slices`），在 `algo_duration` 内均匀提交。

Controller 行为：

1. 解析到 `algo: TWAP` 的 ORDER
2. 计算每份数量：`slice_qty = qty / algo_slices`
3. 计算间隔：`interval = algo_duration / algo_slices`
4. 在内存中创建 `AlgoTask`，由独立 goroutine 按间隔提交子单
5. 每个子单追加为独立的 EXECUTION，关联原始 `intent_id`：

```
2026-03-30 * "EXECUTION" "TWAP slice 1/5 BUY AAPL.US"
  ; intent_id: 20260330-010
  ; slice: 1/5
  ; order_id: 111111
  ; side: BUY
  ; symbol: AAPL.US
  ; filled_qty: 100
  ; avg_price: 181.80
  ; status: FILLED
  ; executed_at: 2026-03-30T09:10:00Z
```

#### ICEBERG

显示部分数量（`visible_qty = qty / algo_slices`），成交后自动补单直到总量完成。实现方式类似 TWAP，但触发条件是上一份成交而非时间间隔。

### 8.4 Go 包结构扩展

```
internal/
  broker/
    broker.go          // 现有, 扩展 SubmitOrder 支持 algo 字段
    algo.go            // NEW: AlgoTask / TWAP / Iceberg scheduler
    algo_twap.go       // TWAP 实现
    algo_iceberg.go    // Iceberg 实现
```

---

## 9. 审计追踪 (Audit Trail)

### 9.1 目标

实现"研究 → 信号 → 组合决策 → 风控校验 → 执行"全链路可追溯。

### 9.2 审计日志

#### `audit/{date}/{cycle_id}.json`

Controller 每个轮询周期生成一条审计记录：

```json
{
  "cycle_id": "cycle-20260330-081000",
  "timestamp": "2026-03-30T08:10:00Z",
  "duration_ms": 450,
  "steps": {
    "research": {
      "feeds_refreshed": ["AAPL.US/news"],
      "errors": []
    },
    "signal": {
      "computed": [
        { "symbol": "AAPL.US", "name": "sma_crossover", "value": "BULLISH" }
      ]
    },
    "portfolio": {
      "diff_computed": true,
      "rebalance_pending": false
    },
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
      "rejections": 0,
      "algo_tasks_active": 0
    }
  }
}
```

### 9.3 ORDER 审计链

通过 ORDER 中的元数据字段，可以反向追踪完整决策链：

```
signal_refs: sma_crossover,llm_sentiment    → 哪些信号触发
source: rebalance                           → 通过什么途径
rebalance_id: rebal-20260330-001            → 哪次 rebalance
intent_id: 20260330-010                     → 唯一订单 ID
```

结合 `signal/output/*/history.jsonl`、`portfolio/history/`、`trade/risk/violations.jsonl`、`trade/beancount.txt`，可完整重建任意订单的决策路径。

---

## 10. Controller 轮询循环（变更后）

```
初始化
  │
  ▼
轮询循环 ─────────────────────────────────────────────────────────┐
  │                                                               │
  ├─ 1. refreshResearch()       # L1: 刷新研究数据                │
  ├─ 2. computeSignals()        # L2: 计算/读取信号               │
  ├─ 3. syncPortfolio()         # L3: 同步组合、计算 diff         │
  ├─ 4. processRebalance()      # L3→L5: 将 pending rebalance     │
  │                             #        转为 ORDER                │
  ├─ 5. parseNewOrders()        # 解析 beancount.txt 新 ORDER     │
  ├─ 6. preTradeCheck(orders)   # L4: 前置风控校验                 │
  ├─ 7. executeOrders(passed)   # L5: 执行通过风控的订单           │
  ├─ 8. appendResults()         # 写回 EXECUTION/REJECTION        │
  ├─ 9. refreshAccount()        # 更新账户状态                     │
  ├─ 10. refreshQuotes()        # 更新行情                        │
  ├─ 11. checkRiskTriggers()    # L4: 止损/止盈触发 (现有逻辑)    │
  ├─ 12. updateDailyLimits()    # L4: 更新日度风控状态             │
  ├─ 13. writeAuditLog()        # 写入审计日志                     │
  ├─ 14. compactLedger()        # 账本归档 (现有逻辑)              │
  ├─ 15. checkKillSwitch()      # 检查 .kill                      │
  │                                                               │
  └───────────────────────── sleep(interval) ─────────────────────┘
```

### 启动参数扩展

| 参数                  | 作用                           | 默认    |
|-----------------------|-------------------------------|---------|
| `--harness`           | 启用 Harness 五层管道          | `false` |
| `--research-interval` | 研究数据刷新间隔               | `5m`    |
| `--risk-mode`         | 风控模式: enforce/warn/disabled| `enforce`|
| `--auto-rebalance`    | 允许自动执行 rebalance         | `false` |

当 `--harness=false` 时，Controller 行为与当前版本完全一致，仅运行步骤 5-11, 14-15。

---

## 11. 设计原则回顾

1. **文件即接口**：所有新增功能通过 JSON/JSONL 文件交互，无新增 API 端点
2. **Append-only 审计**：`violations.jsonl`、`signal/history.jsonl`、`beancount.txt` 均为追加写入
3. **渐进式启用**：`--harness` 开关控制新功能，默认关闭，完全向后兼容
4. **Agent 友好**：所有文件格式人机可读，Agent 只需 read/write 文件即可驱动全流程
5. **硬约束优先**：风控层默认 `ENFORCE` 模式，宁可错杀不可放过
