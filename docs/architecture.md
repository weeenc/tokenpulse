# Architecture

TokenPulse 是单体 Go 服务，不使用微服务或消息队列。Handler 负责 HTTP 输入输出，Service 负责状态机和业务规则，Repository 负责数据库上下文与事务边界，GORM 负责参数化访问，MySQL 负责最终唯一性和聚合。HTTP 请求的超时 Context 会一直传播到 GORM。

## Data ownership

- User：Usage 数据所有者。
- Device：可长期保留的逻辑设备，以服务端 UUID 标识。
- Installation：一次 Agent 凭证安装实例，可在重连时轮换。
- DeviceAuthRequest：十分钟有效的一次性浏览器授权状态机。
- UsageEvent：不可重复的 Token 元数据。
- RefreshSession：只保存 Refresh Token 的 SHA-256，轮换后立即撤销旧会话；旧 Token 重放会撤销整个会话家族。

服务器从 Device Token 的 Hash 解析 `userId/deviceId/installationId`。Usage 请求体没有这些字段，也不会信任客户端提供的数据归属。

## Collection safety

每个 Collector 独立检测、版本识别、解析和标准化。JSONL 使用 byte offset 增量读取，文件截断时从头读取。Cursor 以只读模式查询 SQLite/WAL，只选择明确的 Token、模型、时间和稳定 ID 字段；没有可验证 Token 字段时输出 diagnostics，不估算聊天正文。单个 Collector 失败只产生 warning，不会中止其他 Collector。原始会话正文不会进入标准 UsageEvent。事件写入与扫描进度在本地 SQLite 中原子提交。

## Time and idempotency

API 使用 RFC3339，服务端和 MySQL 保存 UTC；统计边界根据浏览器时区偏移换算。Agent SQLite 先以 eventId 去重；MySQL 再以 `(user_id, source, event_id)` 唯一索引作最终去重。模型价格由 migration 版本化，费用是基于事件发生时生效价格的估算值。
