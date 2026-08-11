# Device Authorization

1. Agent 生成新的 installation UUID，并提交设备名称、平台、架构和版本。
2. Server 只向 CLI 返回随机 device code，并给用户展示不含易混字符的 user code。
3. CLI 打开 Web 完整验证地址，每五秒轮询 token endpoint。
4. Web 用户可以添加为新 Device，或重连自己账户下的已有 Device。
5. Approve 仅改变授权请求状态；CLI 首次消费后才创建 Installation 和 256-bit Device Token。
6. Server 仅保存 SHA-256 token hash，明文只返回一次。
7. 重连时原 Device UUID 不变，所有旧 ACTIVE Credential 在同一事务中撤销。

状态只允许 `PENDING → APPROVED → CONSUMED` 或 `PENDING → DENIED/EXPIRED`。目标 Device 必须属于批准授权的同一 User。
