# TokenPulse

TokenPulse 是一个可自托管的跨平台 AI Coding Token 使用量平台。项目采用 Monorepo，但三个可独立构建和发布的应用均使用提示词中指定的项目名称：

```text
TokenPulse/
├── tokenpulse-agent/    # macOS / Windows 本地 CLI 客户端
├── tokenpulse-server/   # Go API 服务端
├── tokenpulse-web/      # Vue 服务端前端
├── deploy/              # Docker Compose 与 Nginx
└── docs/                # 架构、认证和 API 文档
```

## Architecture

```text
tokenpulse-agent (TypeScript + SQLite)
        │ HTTPS + opaque Device Token
        ▼
Nginx ──► tokenpulse-server (Go/Gin) ──► MySQL 8
  │                 ▲
  └── tokenpulse-web┘  HttpOnly User JWT Cookie
```

核心数据关系为 `User 1:N Device`、`Device 1:N Installation`。Web JWT 与 Device Token 完全分离；Usage Event 通过 `(user_id, source, event_id)` 在服务端最终去重。

## Local Development

要求 Node.js 20.19+、npm、Go 1.25+ 和 MySQL 8.4+。首次安装请在仓库根目录运行 `npm ci`，以根锁文件保证 Agent 与 Web 依赖一致。

启动 MySQL：

```bash
cp deploy/.env.example deploy/.env
cd deploy
docker compose up -d mysql
```

启动 Go 服务端。服务启动时自动应用 `tokenpulse-server/migrations/*.up.sql`，生产环境不使用 GORM AutoMigrate：

```bash
cd tokenpulse-server
MYSQL_HOST=127.0.0.1 \
MYSQL_USERNAME=tokenpulse \
MYSQL_PASSWORD=change-this-database-password \
JWT_SECRET=development-secret \
go run ./cmd/server
```

启动服务端前端：

```bash
cd tokenpulse-web
npm ci --workspace tokenpulse-web --include-workspace-root
npm run dev
```

## Build Agent

```bash
cd tokenpulse-agent
npm ci --workspace tokenpulse-agent --include-workspace-root
npm run build
npm test
```

## Install Agent

```bash
cd tokenpulse-agent
npm install -g .
tokenpulse --help
```

公网服务地址可通过环境变量覆盖：

```bash
TOKENPULSE_SERVER_URL=https://tokenpulse.example.com tokenpulse login
```

## Device Login Flow

```bash
tokenpulse login
# 无桌面环境：tokenpulse login --no-browser
```

CLI 生成 Installation UUID、申请一次性代码并打开 `/device?code=...`。用户可添加为新 Device，或重连自己账户中的已有 Device。重连会保留 Device UUID 和历史 Usage，同时撤销旧 Installation Credential。

## Collection and Sync

```bash
tokenpulse collect
tokenpulse sync
tokenpulse sync --verbose
tokenpulse status
tokenpulse doctor
tokenpulse logout
```

Agent 默认只上传 Token Usage Metadata，不上传 Prompt、AI Response、源码、文件内容、Git diff 或仓库内容。

Codex 与 Claude Code 使用已验证的 JSONL Token 字段。Cursor 以只读方式查询
`cursorDiskKV`，只选择明确的 `tokenCount`、模型、时间和稳定 ID；当前 Cursor
版本若未在本地公开非零 Token 字段，Agent 会给出 diagnostics，不会根据聊天正文估算或上传内容。

## Auto Submit

```bash
tokenpulse autosubmit enable --interval 1h
tokenpulse autosubmit status
tokenpulse autosubmit run
tokenpulse autosubmit disable
```

macOS 使用 launchd，Windows 使用 Task Scheduler；两端都执行短生命周期的 `tokenpulse sync`。

## Server Deployment

填写 `deploy/.env`，配置 `deploy/nginx.conf` 中的域名，并将 TLS 证书放入 `deploy/certs/`：

```bash
cd deploy
docker compose up -d --build
docker compose ps
```

## MySQL

数据库使用 InnoDB、utf8mb4 和 UTC。服务通过 `golang-migrate` 与 `tokenpulse_schema_migrations` 管理数据库版本、迁移锁和 dirty state。
模型价格同样由 migration 版本化；初始价格基线来自 OpenAI 和 Anthropic 官方公开价格，费用按事件发生时间选择当时生效的价格，因此展示值是可追溯的估算值。

可选 MySQL 集成测试只能指向一次性测试数据库：

```bash
cd tokenpulse-server
MYSQL_TEST_DSN='tokenpulse:tokenpulse@tcp(127.0.0.1:3306)/tokenpulse_test?parseTime=true' go test ./internal/service
```

## Troubleshooting

- 返回 `401`：运行 `tokenpulse login` 并重新连接已有设备。
- `config.json` 丢失但 Keychain/DPAPI 凭证仍在：运行 `tokenpulse status` 自动恢复。
- Collector 格式无法识别：运行 `tokenpulse collect --verbose` 查看 diagnostics。
- Cursor 已检测但为 0：这表示当前本地数据库没有可验证的非零 Token 字段；TokenPulse 不会用正文长度伪造使用量。
- Server 无法启动：检查 MySQL 环境变量、migration 日志和 `/health`。

## Security

- Device Token 使用至少 256-bit 随机数据，服务端只保存 SHA-256。
- macOS 使用 Keychain；Windows 使用当前用户范围 DPAPI。
- 密码使用 bcrypt。
- Refresh Token 是服务端保存 Hash 的不透明会话，使用后立即轮换，退出时撤销；Cookie 使用 HttpOnly 与 SameSite。
- 浏览器写请求使用 CSRF Cookie/Header 双提交校验，并验证 `Origin` allow list。
- 生产环境必须启用 HTTPS、强随机 Secret 和精确 CORS allow list。
- 生产配置会拒绝短 JWT Secret、HTTP 公网地址和缺失的数据库密码。

## Open source and releases

代码采用 [MIT License](LICENSE)。贡献规范、安全报告、行为准则和变更记录分别见 [CONTRIBUTING.md](CONTRIBUTING.md)、[SECURITY.md](SECURITY.md)、[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) 与 [CHANGELOG.md](CHANGELOG.md)。

- 普通 PR 自动执行格式、Lint、Node/Web 测试、Go race/vet、MySQL 集成测试、跨平台 Agent 测试、容器构建和 CodeQL。
- `agent-vX.Y.Z` 标签通过 npm Trusted Publishing 发布 Agent。
- `vX.Y.Z` 标签向 GHCR 发布 Server/Web 的 amd64 与 arm64 镜像，并生成 provenance 与 SBOM。

首次发布前需在 npm 包设置中绑定本仓库为 Trusted Publisher，并在 GitHub 启用私密漏洞报告及 `main` 分支 ruleset。

更多信息见 [架构](docs/architecture.md)、[设备授权](docs/device-auth.md)、[API](docs/api.md) 和 [OpenAPI](docs/openapi.yaml)。
