# Longbridge FS 文档

欢迎使用 Longbridge FS 文档！

## 核心文档

### 用户指南
- [架构设计](./architecture.md) - 核心组件与数据流
- [文件系统结构](./filesystem.md) - 目录布局与关键文件
- [订单格式](./order-format.md) - Beancount 账本写入规范
- [行情数据](./market-data.md) - `track` 拉取与 `subscribe` 推送
- [风控配置](./risk-control.md) - 止损/止盈规则与触发行为
- [API 参考](./api-reference.md) - `init` 与 `controller` 参数说明
- [AI Agent 使用指南](./ai-agent-guide.md) - 面向 Agent 的端到端示例

### 五层架构 (Harness)
- **[五层架构完整规范](./five-layer-spec.md)** - 详细设计文档
- [Phase 1 实现总结](./phase1-implementation.md) - 基础管道与风控层实现
- [实现进度日志](./implementation-log.md) - Phase 1-5 完成状态

## 快速链接

- [主 README](../README.md)
- [端到端示例脚本](../demo_e2e.sh)
- [配置示例](../configs/)

## 获取帮助

如有问题，请提交 [GitHub Issue](https://github.com/JetSquirrel/longbridge-fs/issues)。
