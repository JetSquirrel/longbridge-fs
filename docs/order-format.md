# 订单格式详解

本文档详细说明 Longbridge FS 的 Beancount 订单格式。

## Beancount 基础

Longbridge FS 使用 [Beancount](https://beancount.github.io/) 复式记账格式记录所有交易。每笔交易都是不可修改的（append-only），确保审计追踪。

## 订单类型

### ORDER - 交易指令

用户提交的交易意图，等待 Controller 执行。

**基本格式：**
```
YYYY-MM-DD * "ORDER" "描述文本"
  ; field1: value1
  ; field2: value2
  ...
```

**完整示例：**
```
2026-02-12 * "ORDER" "BUY AAPL.US 100 shares"
  ; intent_id: 20260212-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; price: 180.50
  ; tif: DAY
```

### EXECUTION - 执行记录

订单成功执行后，Controller 追加的执行记录。

**格式：**
```
YYYY-MM-DD * "EXECUTION" "执行描述"
  ; intent_id: 原订单ID
  ; order_id: 交易所订单ID
  ; side: BUY/SELL
  ; symbol: 股票代码
  ; filled_qty: 成交数量
  ; avg_price: 成交均价
  ; status: FILLED
  ; executed_at: 执行时间戳
```

**示例：**
```
2026-02-12 * "EXECUTION" "BUY AAPL.US @ 180.25"
  ; intent_id: 20260212-001
  ; order_id: 9876543210
  ; side: BUY
  ; symbol: AAPL.US
  ; filled_qty: 100
  ; avg_price: 180.25
  ; status: FILLED
  ; executed_at: 2026-02-12T10:30:15Z
```

### REJECTION - 拒绝记录

订单被拒绝时，Controller 追加的拒绝记录。

**格式：**
```
YYYY-MM-DD * "REJECTION" "拒绝描述"
  ; intent_id: 原订单ID
  ; reason: 拒绝原因
  ; rejected_at: 拒绝时间戳
```

**示例：**
```
2026-02-12 * "REJECTION" "BUY AAPL.US - Insufficient funds"
  ; intent_id: 20260212-001
  ; reason: Insufficient funds. Available: $5000, Required: $18050
  ; rejected_at: 2026-02-12T10:30:15Z
```

## ORDER 字段详解

### 必需字段

#### intent_id
- **类型**：字符串
- **说明**：订单唯一标识符，用于追踪订单状态
- **格式建议**：YYYYMMDD-NNN 或 YYYYMMDD-HHMMSS
- **示例**：`20260212-001`, `20260212-103015`

#### side
- **类型**：枚举
- **值**：BUY 或 SELL
- **说明**：交易方向

#### symbol
- **类型**：字符串
- **格式**：SYMBOL.MARKET
- **示例**：
  - 美股：AAPL.US, MSFT.US, TSLA.US
  - 港股：700.HK, 9988.HK, 0001.HK
  - A股：600519.SH, 000001.SZ

#### qty
- **类型**：整数
- **说明**：交易数量
- **注意**：必须符合交易所最小交易单位（如港股 1 手 = 100 股）

#### type
- **类型**：枚举
- **值**：MARKET 或 LIMIT
- **说明**：
  - MARKET：市价单，以当前市场价格立即成交
  - LIMIT：限价单，以指定价格或更好价格成交

#### tif
- **类型**：枚举
- **值**：DAY 或 GTC
- **说明**：
  - DAY：当日有效，收盘前未成交则自动取消
  - GTC：撤单前有效，除非手动取消

### 条件字段

#### price
- **类型**：浮点数
- **必需条件**：type = LIMIT 时必须提供
- **说明**：限价单的目标价格
- **示例**：`180.50`, `9988.00`

### 可选字段

#### market (未实现)
- **类型**：枚举
- **值**：US, HK, CN
- **说明**：市场代码，可从 symbol 推断，因此可选

## 格式规范

### 缩进规则

1. 日期行顶格
2. 字段行必须以两个空格 + 分号开头：`  ;`
3. 字段行示例：`  ; field: value`

**正确：**
```
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
```

**错误：**
```
2026-02-12 * "ORDER" "BUY AAPL"
; intent_id: 001       # 缺少缩进
 ; side: BUY           # 只有一个空格
    ; qty: 100         # 缩进过多
```

### 空行规范

每个订单记录之后必须有一个空行，以便 Parser 正确分隔。

**正确：**
```
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

2026-02-12 * "ORDER" "BUY MSFT"
  ; intent_id: 002
  ; side: BUY
```

**错误（缺少空行）：**
```
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
2026-02-12 * "ORDER" "BUY MSFT"  # 两个订单粘在一起
  ; intent_id: 002
```

### 日期格式

- **格式**：YYYY-MM-DD
- **示例**：`2026-02-12`, `2026-12-31`
- **不要使用**：`2026/02/12`, `02-12-2026`, `20260212`

### 字段值格式

- **字符串**：直接写，不需要引号（除非描述文本）
- **数字**：整数或小数，不要加单位或逗号
- **布尔值**：true 或 false
- **时间戳**：ISO 8601 格式（YYYY-MM-DDTHH:MM:SSZ）

**正确：**
```
  ; qty: 100
  ; price: 180.50
  ; executed_at: 2026-02-12T10:30:15Z
```

**错误：**
```
  ; qty: "100"              # 数字不需要引号
  ; price: $180.50          # 不要加货币符号
  ; price: 180.50 USD       # 不要加单位
  ; qty: 1,000              # 不要加逗号分隔符
```

## 完整示例

### 市价单买入

```
2026-02-12 * "ORDER" "BUY 100 AAPL at market price"
  ; intent_id: 20260212-100001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

```

### 限价单卖出

```
2026-02-12 * "ORDER" "SELL 50 MSFT at $420.00"
  ; intent_id: 20260212-100002
  ; side: SELL
  ; symbol: MSFT.US
  ; qty: 50
  ; type: LIMIT
  ; price: 420.00
  ; tif: GTC

```

### 港股交易

```
2026-02-12 * "ORDER" "BUY 1000 Tencent"
  ; intent_id: 20260212-100003
  ; side: BUY
  ; symbol: 700.HK
  ; qty: 1000
  ; type: LIMIT
  ; price: 350.00
  ; tif: DAY

```

### 执行成功案例

```
2026-02-12 * "ORDER" "BUY 100 NVDA"
  ; intent_id: 20260212-100004
  ; side: BUY
  ; symbol: NVDA.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY

2026-02-12 * "EXECUTION" "BUY NVDA.US @ 875.50"
  ; intent_id: 20260212-100004
  ; order_id: 123456789
  ; side: BUY
  ; symbol: NVDA.US
  ; filled_qty: 100
  ; avg_price: 875.50
  ; status: FILLED
  ; executed_at: 2026-02-12T10:31:22Z

```

### 执行失败案例

```
2026-02-12 * "ORDER" "BUY 10000 AAPL"
  ; intent_id: 20260212-100005
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 10000
  ; type: MARKET
  ; tif: DAY

2026-02-12 * "REJECTION" "BUY AAPL.US - Insufficient funds"
  ; intent_id: 20260212-100005
  ; reason: Insufficient funds. Available: $50000, Required: $1805000
  ; rejected_at: 2026-02-12T10:31:25Z

```

## 常见错误

### 1. 缺少必需字段

```
# 错误：缺少 type 和 tif
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100

```

**解决**：补全所有必需字段。

### 2. 限价单缺少 price

```
# 错误：LIMIT 类型必须有 price
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: LIMIT
  ; tif: DAY

```

**解决**：添加 price 字段。

### 3. intent_id 重复

如果两个订单使用相同的 intent_id，后续查询时无法区分。

**解决**：确保每个订单的 intent_id 唯一，建议使用时间戳。

### 4. 符号格式错误

```
# 错误：缺少市场后缀
  ; symbol: AAPL

# 错误：使用错误的分隔符
  ; symbol: AAPL:US
  ; symbol: AAPL-US
```

**解决**：使用正确格式 `SYMBOL.MARKET`，如 `AAPL.US`。

## 扩展字段

系统支持自定义扩展字段，但不会被 Controller 使用。可用于记录额外信息：

```
2026-02-12 * "ORDER" "BUY AAPL"
  ; intent_id: 20260212-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY
  ; strategy: momentum_buying
  ; note: Technical breakout at $180
  ; trader: alice

```

这些扩展字段会被保留在账本中，但不影响订单执行。

## 账本归档

当执行订单数量达到阈值（默认 10 笔），Controller 会自动归档已执行订单到 `fs/trade/blocks/` 目录，主账本只保留待执行订单。

归档后的区块文件格式相同，可以串联所有区块重建完整交易历史：

```bash
cat fs/trade/blocks/*.txt fs/trade/beancount.txt > full_history.txt
```

## 参考资料

- [Beancount 官方文档](https://beancount.github.io/docs/)
- [Beancount 语法](https://beancount.github.io/docs/beancount_language_syntax.html)
- [示例脚本](../demo.sh)
