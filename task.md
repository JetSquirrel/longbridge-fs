# Task Progress for longbridge-fs

**Project**: longbridge-fs
**Status**: Imported
**Last Update**: 2026-03-31 07:35:38 UTC
**Current Iteration**: 0
**Source**: https://github.com/JetSquirrel/longbridge-fs

### Phase 1: 基础管道 (v0.3.0)

目标：跑通五层目录结构 + 前置风控

- [x] `fs init` 创建新增目录
- [x] `internal/riskgate/` 实现前置风控引擎
- [x] Controller 插入 `preTradeCheck()` 步骤
- [x] `trade/risk/policy.json` + `pre_trade.json` + `position_limits.json`
- [x] ORDER 扩展字段 (`source`, `signal_refs`, `rebalance_id`)
- [x] 审计日志骨架 (`audit/`)
- [x] 文档更新

### Phase 2: 组合构建 (v0.4.0)

目标：信号到 rebalance 的完整链路

- [ ] `portfolio/target.json` → `current.json` → `diff.json` 计算
- [ ] `rebalance/pending.json` → ORDER 转化
- [ ] `portfolio/history/` 快照归档
- [ ] `--auto-rebalance` 模式

### Phase 3: 研究与信号 (v0.5.0)

目标：数据管道自动化

- [ ] `research/watchlist.json` + Content API 集成
- [ ] `signal/definitions/` 声明式配置
- [ ] 内置指标计算 (SMA_CROSS, RSI, PRICE_CHANGE)
- [ ] `signal/active.json` 聚合

### Phase 4: 算法执行 (v0.6.0)

目标：大单拆分执行

- [ ] TWAP 实现 (goroutine scheduler)
- [ ] ICEBERG 实现
- [ ] 算法子单关联与审计

### Phase 5: Agent 集成增强 (v0.7.0)

目标：降低 Agent 接入门槛

- [ ] `.claude/skills/` 更新，覆盖五层操作
- [ ] 端到端 demo 脚本 (研究 → 信号 → 组合 → 执行)
- [ ] Mock 模式支持全管道模拟
