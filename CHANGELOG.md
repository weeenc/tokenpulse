# Changelog

本项目遵循 [Keep a Changelog](https://keepachangelog.com/) 的结构，并使用语义化版本。

## [Unreleased]

### Fixed

- Agent 升级后通过同步心跳刷新设备管理中的版本号，无需重新授权设备。

### Added

- TokenPulse Monorepo：Agent、Go Server、Web、Docker Compose。
- Device Authorization、Usage 去重、统计、凭证安全存储和定时同步。
- MIT License、开源社区文件和发布前质量体系。
- Cursor 只读 Schema 探测与明确 Token 字段采集，SQLite/WAL 增量状态和安全 diagnostics。
- 浏览器时区统计、自定义趋势日期、设备 Agent 版本和版本化模型价格。
- Repository/Context 数据库边界、生产配置校验及扩展的 Agent/Web/Go 测试。

## [0.1.0] - 2026-08-07

### Added

- 首个 Alpha 版本。
