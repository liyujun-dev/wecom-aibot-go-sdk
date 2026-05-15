# Spec: 企业微信智能机器人 Go SDK

对标 [WecomTeam/aibot-node-sdk](https://github.com/WecomTeam/aibot-node-sdk)，基于 WebSocket 长连接通道。

## 核心能力

- **WebSocket 连接管理** — 自动认证、心跳保活、指数退避重连
- **消息收发** — 被动回复、流式回复（Markdown）、欢迎语、模板卡片、图文混排
- **模板卡片交互** — 回复卡片、流式+卡片组合、更新卡片
- **主动推送** — Markdown / 模板卡片 / 媒体消息，无需回调帧
- **媒体素材** — 三步分片上传（512KB 分片，最多 100 片 ≈ 50MB）
- **文件下载解密** — AES-256-CBC，自定义 32 字节 PKCS#7 padding
- **事件回调** — 进入会话、卡片按钮点击、用户反馈、连接断开

## 技术选型

| 项 | 选择 | 理由 |
|----|------|------|
| Go 版本 | 1.26+ | slog 内置（1.21+就有，但用户指定） |
| 模块路径 | `github.com/liyujun/wecom-aibot-go-sdk` | |
| 包结构 | 单一 `aibot` 包 | 简化导入路径 |
| WS 库 | `gorilla/websocket` | 最广泛使用的 Go WebSocket 库 |
| HTTP | `net/http` 标准库 | 只需下载文件，无额外依赖 |
| 日志 | `log/slog` 标准库 | Go 内置结构化日志 |
| 测试 | `testing` 标准库 | 零依赖测试 |
| 构建 | `mise tasks` | 用户指定的任务运行器 |

## 依赖

| 依赖 | 用途 |
|------|------|
| `gorilla/websocket` | WebSocket 长连接 |
| Go 标准库 | 其他一切（net/http、crypto/aes、crypto/cipher、log/slog、encoding/json） |

## FAQ

**Q: 是否需要 `requestTimeout` 配置项？**
A: 不需要，Go 中通过 `context.Context` 控制超时。

**Q: 是否需要导出 `WeComCrypto` 低层原语？**
A: 不需要，Go 标准库 `crypto/*` 已覆盖。只导出 `DecryptFile`。

**Q: 是否需要 `replyStreamNonBlocking`？**
A: 不需要，用户如有需求可自行基于 `HasPendingAck` 实现跳过逻辑。

**Q: 是否支持 Webhook 模式？**
A: 当前只支持 WebSocket 长连接。Webhook 可后续迭代。
