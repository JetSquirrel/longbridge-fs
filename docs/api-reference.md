# API 参考（Controller & FS）

Longbridge FS 仅依赖两个命令：`init` 用于创建目录，`controller` 用于轮询文件系统并调用 Longbridge API。

## init

初始化文件系统结构。

```bash
longbridge-fs init --root ./fs
```

| 参数       | 说明                       | 默认 |
| ---------- | -------------------------- | ---- |
| `--root`   | FS 根目录                  | `.`  |
| `-v, --verbose` | 打印创建的目录/文件 | `false` |

## controller

启动守护进程，持续：
- 解析 `trade/beancount.txt` 中的新 `ORDER`
- 执行订单（真实或 Mock）
- 刷新账户、持仓、盈亏与组合
- 处理 `quote/track` 与订阅请求
- 执行风控规则并归档账本

```bash
# 真实 API
longbridge-fs controller \
  --root ./fs \
  --credential ./configs/credential \
  --interval 2s \
  --compact-after 10

# Mock 模式（不触发真实交易/行情）
longbridge-fs controller --root ./fs --mock
```

| 参数                | 说明                                                | 默认值          |
| ------------------- | --------------------------------------------------- | --------------- |
| `--root`            | FS 根目录                                           | `.`             |
| `--credential`      | Longbridge 凭据文件路径                            | `credential`    |
| `--interval`        | 轮询间隔                                            | `2s`            |
| `--mock`            | 使用本地 Mock，不连接 Longbridge API                | `false`         |
| `--compact-after`   | 执行订单数量达到 N 后归档到 `trade/blocks/`，0 关闭 | `10`            |
| `-v, --verbose`     | 输出详细日志                                        | `false`         |

## 凭据文件

`configs/credential` 示例：
```
api_key=YOUR_APP_KEY
secret=YOUR_APP_SECRET
access_token=YOUR_ACCESS_TOKEN
```

## 退出与健康检查

- **Kill Switch**：在 FS 根目录 `touch .kill`，下一轮轮询时安全退出。
- **日志**：Controller 直接输出到 stdout，可使用 `tee`/`tail -f` 观察。

## 兼容的 CLI 子命令

仓库保留了部分 CLI（行情、账户、下单）用于兼容旧脚本，但推荐优先通过文件系统交互。全量用法可运行 `longbridge-fs --help` 查看。
