# Contributing to TokenPulse

感谢你帮助改进 TokenPulse。提交代码即表示你同意贡献内容按本仓库的 MIT License 发布。

## Development

需要 Node.js 20.19+、npm、Go 1.25+ 和 MySQL 8.4+。

```bash
npm ci
npm run format:check
npm run lint
npm run build
npm test

cd tokenpulse-server
go test ./...
go vet ./...
```

MySQL 集成测试必须指向可被测试清空的一次性数据库：

```bash
MYSQL_TEST_DSN='tokenpulse:tokenpulse@tcp(127.0.0.1:3306)/tokenpulse_test?parseTime=true' go test ./internal/service
```

## Pull requests

1. 先建立 Issue 描述问题或功能目标，大改动先讨论方案。
2. 一个 PR 只处理一个主题，并为行为变化补测试和文档。
3. Collector fixture 必须脱敏，禁止提交 Prompt、回复、源码、路径或凭证。
4. 新 migration 只能新增，禁止修改已发布 migration。
5. 提交前确保 CI 中的 build、test、lint、format、Go vet 全部通过。

## Collector contributions

新增 Collector 时应保持 Detector、Parser、Normalizer 独立；无法识别格式时返回 diagnostics，禁止生成伪造 Token。Fixture 应来自公开格式说明或经过脱敏的真实记录，并说明对应工具版本。
