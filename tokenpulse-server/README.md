# tokenpulse-server

```bash
go run ./cmd/server
go test ./...
```

服务需要 MySQL 8+，启动时从 `MIGRATIONS_DIR` 应用版本化 SQL。生产环境必须提供 `JWT_SECRET`、数据库变量、`WEB_BASE_URL` 和 `CORS_ALLOWED_ORIGINS`。

全新数据库也可以一次性导入完整脚本：

```bash
mysql -u tokenpulse -p token_usage < database.sql
```

`database.sql` 已包含当前全部表结构、字段中文注释、初始模型价格数据以及迁移版本标记，仅用于空数据库初始化。已有数据库请继续使用 `migrations/` 中的版本化脚本升级；`000005_database_comments` 会为已有表补齐中文注释。
