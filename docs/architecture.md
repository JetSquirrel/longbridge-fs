# 架构设计

## 设计理念

Longbridge FS 的核心设计理念是**将股票交易抽象为文件系统操作**，使得 AI Agent 和自动化脚本可以通过简单的文件读写完成复杂的交易任务。

### 为什么选择文件系统？

1. **通用性**：所有编程语言和工具都支持文件操作
2. **可审计性**：所有操作都留有文件记录，便于审计和回溯
3. **AI 友好**：AI Agent 擅长文件操作，无需学习复杂的 API
4. **人机可读**：文本格式既可被程序解析，也可被人类直接阅读
5. **解耦合**：交易逻辑与执行引擎分离，便于测试和维护

## 系统架构

```
┌─────────────┐
│  AI Agent   │
│  / Script   │
└──────┬──────┘
       │ Read/Write Files
       ▼
┌─────────────────────────────────────┐
│         File System (fs/)           │
│  ┌─────────┬─────────┬───────────┐ │
│  │ account │  trade  │   quote   │ │
│  └─────────┴─────────┴───────────┘ │
└──────────────┬──────────────────────┘
               │ Watch & Process
               ▼
┌───────────────────────────────────────┐
│         Controller (守护进程)         │
│  ┌──────────────────────────────┐   │
│  │  Polling Loop (每 2 秒)       │   │
│  │  1. 解析新 ORDER              │   │
│  │  2. 执行订单 (SDK/Mock)       │   │
│  │  3. 追加 EXECUTION/REJECTION  │   │
│  │  4. 更新账户状态              │   │
│  │  5. 获取行情数据              │   │
│  │  6. 检查风控规则              │   │
│  │  7. 检查 kill switch          │   │
│  └──────────────────────────────┘   │
└──────────┬────────────────────────────┘
           │ API Calls
           ▼
┌───────────────────────┐
│   Longbridge API      │
│   (或 Mock 模式)       │
└───────────────────────┘
```

## 核心组件

### 1. Controller (cmd/longbridge-fs/main.go)

守护进程，负责监听文件系统变化并执行相应操作。

**主要职责**：
- 轮询检测新的 ORDER 指令
- 调用 Broker 执行订单
- 更新账户状态和持仓
- 获取行情数据
- 执行风控规则
- 账本归档压缩

**工作流程**：
```
初始化 → 轮询循环 → 解析订单 → 执行订单 → 追加结果 → 更新状态 → 检查风控 → 检查 kill switch
            ↑                                                                    │
            └────────────────────────────────────────────────────────────────────┘
```

### 2. Ledger (internal/ledger/)

账本管理模块，负责解析和管理 Beancount 格式的交易记录。

**核心功能**：
- **parser.go**：解析 beancount.txt，识别 ORDER/EXECUTION/REJECTION
- **compact.go**：归档已执行订单到 blocks/ 目录

**账本格式**：
- 采用 Beancount 复式记账格式
- Append-only，所有记录不可修改
- 使用注释字段（`;`）存储交易元数据

### 3. Broker (internal/broker/)

订单执行引擎，负责与 Longbridge API 交互。

**执行模式**：
- **真实模式**：调用 Longbridge SDK 执行真实订单
- **Mock 模式**：模拟订单执行，用于开发和测试

**订单类型支持**：
- MARKET：市价单
- LIMIT：限价单
- 其他类型可扩展

### 4. Market (internal/market/)

行情数据获取模块。

**数据类型**：
- Overview：实时报价
- Kline：K线数据（日K、周K、5分钟K等）
- Intraday：分时数据

**输出格式**：
- JSON：程序解析
- TXT：人类阅读

### 5. Account (internal/account/)

账户管理模块，负责计算账户状态和盈亏。

**输出文件**：
- `state.json`：账户余额、可用资金
- `pnl.json`：持仓盈亏（未实现）

### 6. Risk (internal/risk/)

风控引擎，负责监控价格并触发止损/止盈。

**工作原理**：
1. 读取 `risk_control.json` 配置
2. 获取实时价格
3. 判断是否触及阈值
4. 自动生成 SELL ORDER
5. 移除已触发的规则

## 数据流

### 订单执行流程

```
1. AI/用户写入 ORDER 到 beancount.txt
          ↓
2. Controller 解析新 ORDER
          ↓
3. Broker 执行订单 (调用 API 或 Mock)
          ↓
4. 追加 EXECUTION 或 REJECTION 到 beancount.txt
          ↓
5. 更新 account/state.json 和 pnl.json
          ↓
6. 达到归档阈值时，压缩到 blocks/ 目录
```

### 行情获取流程

```
1. AI/用户创建 quote/track/SYMBOL 文件
          ↓
2. Controller 检测到 track 文件
          ↓
3. Market 模块获取行情数据
          ↓
4. 写入 quote/hold/SYMBOL/{overview,kline,...}.json
          ↓
5. 删除 track 文件
```

### 风控触发流程

```
1. 用户配置 risk_control.json
          ↓
2. Controller 定期获取持仓价格
          ↓
3. Risk 模块检查是否触及阈值
          ↓
4. 触发时自动追加 SELL ORDER
          ↓
5. 移除已触发规则，避免重复
```

## 扩展性

### 添加新的订单类型

1. 在 `internal/model/types.go` 添加新类型定义
2. 在 `internal/ledger/parser.go` 添加解析逻辑
3. 在 `internal/broker/broker.go` 添加执行逻辑

### 添加新的行情数据源

1. 在 `internal/market/market.go` 添加新的获取函数
2. 定义新的文件命名规则
3. 更新 Controller 轮询逻辑

### 集成其他 Broker

替换 `internal/broker/broker.go` 的 SDK 调用即可，接口保持不变。

## 容错与降级

- **网络故障**：自动切换到 Mock 模式（如果启用）
- **API 限流**：返回 REJECTION，不影响后续订单
- **文件损坏**：Controller 记录错误日志，继续运行
- **Kill Switch**：创建 `.kill` 文件安全退出，不影响已提交订单

## 性能考虑

- **轮询间隔**：默认 2 秒，可通过 `--interval` 调整
- **账本压缩**：定期归档减少主文件大小，加快解析速度
- **行情缓存**：重复请求相同 Symbol 时可复用数据（未实现）
- **并发控制**：单个 Controller 进程，避免订单重复执行
