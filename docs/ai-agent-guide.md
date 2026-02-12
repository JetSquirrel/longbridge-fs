# AI Agent 使用指南

本指南面向 AI Agent（如 Claude、ChatGPT 等）和自动化脚本开发者，介绍如何使用 Longbridge FS 进行股票交易。

## 核心概念

Longbridge FS 将股票交易抽象为**文件操作**：
- **写入文件** = 提交订单/配置
- **读取文件** = 查询状态/行情
- **创建文件** = 触发操作
- **删除文件** = 取消/停止

## 基本工作流

### 1. 初始化环境

确保 Controller 正在运行：

```bash
# 检查 Controller 进程
ps aux | grep longbridge-fs

# 如果未运行，启动 Mock 模式
./build/longbridge-fs controller --root ./fs --mock --interval 2s &
```

### 2. 提交订单

向 `fs/trade/beancount.txt` 追加订单记录：

```python
# Python 示例
import datetime

def submit_order(symbol, side, qty, order_type="MARKET", price=None):
    intent_id = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")

    order = f"""
{datetime.date.today()} * "ORDER" "{side} {symbol}"
  ; intent_id: {intent_id}
  ; side: {side}
  ; symbol: {symbol}
  ; qty: {qty}
  ; type: {order_type}
  ; tif: DAY
"""

    if order_type == "LIMIT" and price:
        order += f"  ; price: {price}\n"

    with open("fs/trade/beancount.txt", "a") as f:
        f.write(order)

    return intent_id

# 提交买入订单
intent_id = submit_order("AAPL.US", "BUY", 100, "LIMIT", 180.00)
print(f"Order submitted: {intent_id}")
```

**订单字段说明**：
- `intent_id`：唯一标识符，建议使用时间戳
- `side`：BUY 或 SELL
- `symbol`：股票代码（如 AAPL.US, 9988.HK）
- `qty`：数量
- `type`：MARKET（市价）或 LIMIT（限价）
- `tif`：Time in Force，DAY（当日有效）或 GTC（撤单前有效）
- `price`：限价单必须，市价单忽略

### 3. 检查订单结果

等待 Controller 处理（默认 2 秒轮询），然后读取账本：

```python
def get_order_result(intent_id):
    with open("fs/trade/beancount.txt", "r") as f:
        lines = f.readlines()

    # 查找包含 intent_id 的 EXECUTION 或 REJECTION
    for i, line in enumerate(lines):
        if intent_id in line:
            # 读取后续行直到找到状态
            for j in range(i, min(i+20, len(lines))):
                if "EXECUTION" in lines[j]:
                    return "FILLED", extract_execution_details(lines[j:j+10])
                elif "REJECTION" in lines[j]:
                    return "REJECTED", extract_rejection_reason(lines[j:j+10])

    return "PENDING", None

# 轮询等待结果
import time
for _ in range(10):  # 最多等待 20 秒
    status, details = get_order_result(intent_id)
    if status != "PENDING":
        print(f"Order {status}: {details}")
        break
    time.sleep(2)
```

### 4. 查询行情

创建 track 文件触发行情获取：

```python
def get_quote(symbol):
    # 创建 track 文件
    track_file = f"fs/quote/track/{symbol}"
    open(track_file, "w").close()

    # 等待 Controller 处理
    time.sleep(3)

    # 读取行情数据
    import json
    quote_file = f"fs/quote/hold/{symbol}/overview.json"

    if os.path.exists(quote_file):
        with open(quote_file, "r") as f:
            return json.load(f)
    else:
        return None

# 获取 AAPL 行情
quote = get_quote("AAPL.US")
if quote:
    print(f"AAPL Price: ${quote['last_done']}")
    print(f"Change: {quote['last_done'] - quote['prev_close']}")
```

### 5. 查看持仓和盈亏

```python
def get_account_state():
    with open("fs/account/state.json", "r") as f:
        return json.load(f)

def get_pnl():
    with open("fs/account/pnl.json", "r") as f:
        return json.load(f)

def get_portfolio():
    with open("fs/quote/portfolio.json", "r") as f:
        return json.load(f)

# 查看账户
state = get_account_state()
print(f"Cash: ${state['cash']}")
print(f"Total Value: ${state['total_value']}")

# 查看盈亏
pnl = get_pnl()
for position in pnl['positions']:
    print(f"{position['symbol']}: {position['unrealized_pnl']:+.2f}")

# 查看组合
portfolio = get_portfolio()
for item in portfolio:
    print(f"{item['symbol']}: {item['qty']} shares @ ${item['current_price']}")
```

### 6. 配置风控

```python
def set_risk_control(symbol, stop_loss=None, take_profit=None, qty=None):
    import json

    # 读取现有配置
    risk_file = "fs/trade/risk_control.json"
    try:
        with open(risk_file, "r") as f:
            risk_config = json.load(f)
    except FileNotFoundError:
        risk_config = {}

    # 更新配置
    risk_config[symbol] = {}
    if stop_loss:
        risk_config[symbol]["stop_loss"] = stop_loss
    if take_profit:
        risk_config[symbol]["take_profit"] = take_profit
    if qty:
        risk_config[symbol]["qty"] = str(qty)

    # 写回文件
    with open(risk_file, "w") as f:
        json.dump(risk_config, f, indent=2)

# 设置 AAPL 止损止盈
set_risk_control("AAPL.US", stop_loss=170.0, take_profit=200.0, qty=100)
```

## 高级用法

### 批量订单

```python
def submit_batch_orders(orders):
    """
    orders: [{"symbol": "AAPL.US", "side": "BUY", "qty": 100, ...}, ...]
    """
    batch_text = ""
    intent_ids = []

    for order in orders:
        intent_id = datetime.datetime.now().strftime("%Y%m%d-%H%M%S%f")
        intent_ids.append(intent_id)

        batch_text += f"""
{datetime.date.today()} * "ORDER" "{order['side']} {order['symbol']}"
  ; intent_id: {intent_id}
  ; side: {order['side']}
  ; symbol: {order['symbol']}
  ; qty: {order['qty']}
  ; type: {order.get('type', 'MARKET')}
  ; tif: {order.get('tif', 'DAY')}
"""
        if 'price' in order:
            batch_text += f"  ; price: {order['price']}\n"

    with open("fs/trade/beancount.txt", "a") as f:
        f.write(batch_text)

    return intent_ids
```

### 监控价格并触发交易

```python
def monitor_price_and_trade(symbol, target_price, side, qty):
    """
    监控价格，达到目标时自动下单
    """
    while True:
        quote = get_quote(symbol)
        if quote and quote['last_done'] >= target_price:
            intent_id = submit_order(symbol, side, qty)
            print(f"Price reached ${target_price}, order submitted: {intent_id}")
            break
        time.sleep(5)

# 当 AAPL 达到 185 时买入
monitor_price_and_trade("AAPL.US", 185.0, "BUY", 100)
```

### 自动止损策略

```python
def auto_stop_loss(symbol, loss_percent=0.05):
    """
    自动为所有持仓设置止损（基于当前价格）
    """
    pnl = get_pnl()

    for position in pnl['positions']:
        if position['symbol'] == symbol:
            current_price = position['current_price']
            stop_price = current_price * (1 - loss_percent)

            set_risk_control(
                symbol=symbol,
                stop_loss=stop_price,
                qty=position['qty']
            )

            print(f"Set stop loss for {symbol} at ${stop_price:.2f}")
            return

    print(f"No position found for {symbol}")

# 为 AAPL 持仓设置 5% 止损
auto_stop_loss("AAPL.US", 0.05)
```

## 完整示例：AI 交易助手

```python
class LongbridgeAgent:
    def __init__(self, fs_root="./fs"):
        self.fs_root = fs_root

    def buy_stock(self, symbol, qty, price=None):
        """买入股票"""
        order_type = "LIMIT" if price else "MARKET"
        intent_id = submit_order(symbol, "BUY", qty, order_type, price)

        # 等待结果
        for _ in range(10):
            status, details = get_order_result(intent_id)
            if status != "PENDING":
                return {"status": status, "details": details}
            time.sleep(2)

        return {"status": "TIMEOUT"}

    def sell_stock(self, symbol, qty, price=None):
        """卖出股票"""
        order_type = "LIMIT" if price else "MARKET"
        intent_id = submit_order(symbol, "SELL", qty, order_type, price)

        for _ in range(10):
            status, details = get_order_result(intent_id)
            if status != "PENDING":
                return {"status": status, "details": details}
            time.sleep(2)

        return {"status": "TIMEOUT"}

    def get_price(self, symbol):
        """获取实时价格"""
        quote = get_quote(symbol)
        return quote['last_done'] if quote else None

    def get_positions(self):
        """获取所有持仓"""
        pnl = get_pnl()
        return pnl['positions']

    def set_stop_loss(self, symbol, price, qty=None):
        """设置止损"""
        set_risk_control(symbol, stop_loss=price, qty=qty)

    def stop_controller(self):
        """停止 Controller"""
        open(f"{self.fs_root}/.kill", "w").close()

# 使用示例
agent = LongbridgeAgent()

# 查询价格
price = agent.get_price("AAPL.US")
print(f"AAPL: ${price}")

# 买入
result = agent.buy_stock("AAPL.US", 100, price=180.0)
print(f"Buy result: {result}")

# 设置止损
agent.set_stop_loss("AAPL.US", 170.0, qty=100)

# 查看持仓
positions = agent.get_positions()
for pos in positions:
    print(f"{pos['symbol']}: {pos['qty']} shares, PnL: ${pos['unrealized_pnl']:.2f}")
```

## 注意事项

1. **等待处理时间**：订单提交后需等待 Controller 轮询处理（默认 2 秒）
2. **intent_id 唯一性**：确保每个订单的 intent_id 唯一，避免冲突
3. **文件追加**：订单必须追加到 beancount.txt，不要覆盖
4. **格式正确**：注意 Beancount 格式的缩进和分号
5. **错误处理**：订单可能被拒绝（资金不足、市场关闭等），需检查 REJECTION
6. **并发控制**：避免同时运行多个写入进程
7. **Mock 模式**：开发时使用 `--mock` 模式，避免真实交易

## 调试技巧

### 查看 Controller 日志

Controller 会输出详细日志，帮助诊断问题：

```bash
# 查看实时日志
tail -f /path/to/controller.log
```

### 手动检查账本

```bash
# 查看最近的订单
tail -50 fs/trade/beancount.txt

# 搜索特定订单
grep "intent_id: 20260211-001" fs/trade/beancount.txt -A 10
```

### 验证 JSON 格式

```bash
# 验证 JSON 文件格式
jq . fs/account/state.json
jq . fs/trade/risk_control.json
```

## 常见错误

### 订单格式错误

```
# 错误：缺少必需字段
2026-02-11 * "ORDER" "BUY AAPL"
  ; intent_id: 001
  ; side: BUY
  # 缺少 symbol, qty, type, tif

# 正确：包含所有必需字段
2026-02-11 * "ORDER" "BUY AAPL"
  ; intent_id: 20260211-001
  ; side: BUY
  ; symbol: AAPL.US
  ; qty: 100
  ; type: MARKET
  ; tif: DAY
```

### 文件权限错误

确保 Controller 和脚本都有读写权限：

```bash
chmod -R 755 fs/
```

### Controller 未运行

提交订单前确保 Controller 在运行：

```bash
ps aux | grep longbridge-fs | grep -v grep
```

## 下一步

- 阅读[订单格式文档](./order-format.md)了解完整的字段定义
- 阅读[风控配置文档](./risk-control.md)了解高级风控策略
- 查看[架构设计文档](./architecture.md)了解系统内部原理
