# tokenpulse-agent

```bash
npm install
npm run build
npm install -g .
tokenpulse login
tokenpulse status
tokenpulse collect --verbose
tokenpulse sync
tokenpulse autosubmit enable --interval 1h
```

需要 Node.js 20+。macOS 凭证进入 Keychain；Windows 凭证经当前用户 DPAPI 加密。普通配置和 SQLite 分别位于 `~/.tokenpulse` 或 `%APPDATA%\tokenpulse-agent`。

Agent 只读取 Token 元数据。Codex/Claude Code 使用已验证 JSONL；Cursor 只读取 SQLite 中明确的 Token 字段，格式不可验证时输出 diagnostics，不读取或上传聊天正文。
