# Plan: 企业微信智能机器人 Go SDK

## 项目布局

```
wecom-aibot-go-sdk/
├── client.go       # Client 结构体、构造函数、Connect/Close、reply/stream/send 等方法
├── ws.go           # WsConnectionManager（连接生命周期、心跳、重连、回复队列）
├── types.go        # 全部类型定义 + Handler 接口 + BaseHandler + 枚举常量
├── media.go        # UploadMedia、DownloadFile、DecryptFile、ReplyMedia、SendMedia
├── go.mod
├── go.sum
├── .mise.toml      # mise 任务配置
├── docs/
│   ├── specs/
│   └── plans/
└── examples/
    ├── basic/       # 基础连接+收发消息
    ├── stream/      # 流式回复
    ├── media/       # 文件上传下载
    └── template_card/ # 模板卡片交互
```

## Client API

```go
// 构造函数（函数式选项模式）
client := aibot.NewClient(botId, secret string,
    aibot.WithWsURL("wss://..."),
    aibot.WithDialer(myDialer),
    aibot.WithLogger(mySlog),
    aibot.WithHeartbeatInterval(30*time.Second),
    aibot.WithMaxReconnectAttempts(10),
    aibot.WithMaxReplyQueueSize(500),
    aibot.WithScene(1),
    aibot.WithPlugVersion("1.0"),
)

// 生命周期
client.Connect(ctx context.Context) error
client.Close()
client.IsConnected() bool

// 被动回复
client.Reply(ctx, frame, body)
client.ReplyStream(ctx, frame, streamId, content, finish)
client.ReplyWelcome(ctx, frame, body)
client.ReplyTemplateCard(ctx, frame, card)
client.ReplyStreamWithCard(ctx, frame, streamId, content, finish, opts...)
client.UpdateTemplateCard(ctx, frame, card, userIds)

// 主动推送
client.SendMessage(ctx, chatId, body)
client.SendMedia(ctx, chatId, mediaType, mediaId)

// 媒体
client.UploadMedia(ctx, data, opts) (*UploadResult, error)
client.ReplyMedia(ctx, frame, mediaType, mediaId)
client.DownloadFile(ctx, url, aesKey) ([]byte, string, error)

// 工具
client.HasPendingAck(frame *WsFrameHeaders) bool
```

## Handler 接口（7 方法 + BaseHandler 嵌入体）

```go
type Handler interface {
    OnConnected()
    OnAuthenticated()
    OnDisconnected(reason string)
    OnReconnecting(attempt int)
    OnError(err error)
    OnMessage(frame *WsFrame[any])
    OnEvent(frame *WsFrame[any])
}

type BaseHandler struct{}

// 用户使用模式：
type MyHandler struct { aibot.BaseHandler }
func (h *MyHandler) OnMessage(frame *aibot.WsFrame[any]) {
    body := frame.Body.(aibot.BaseMessage)
    switch body.MsgType {
    case aibot.MessageTypeText:
        // 处理文本
    }
}
```

## 类型系统

- `WsFrame[T any]` — 泛型帧结构，Body 为 T
- 消息体使用扁平结构体 + `json:"...,omitempty"`
- `MessageType`、`EventType`、`TemplateCardType` 为 string 常量
- `WsCmd` 为 string 常量组

## 并发模型

```
读 goroutine ──→ 解析帧 ──→ handler.OnMessage / handler.OnEvent（同步回调）
写 goroutine ──→ 从 channel 取帧写入 WebSocket
心跳       ──→ time.Ticker，认证成功后启动
回复队列   ──→ 按 req_id 分组，同组串行，sync.Mutex 保护
                      ↓
              回执等待 map[req_id]chan WsFrame（5s 超时）
```

## 错误处理

| 错误 | 含义 |
|------|------|
| `ErrAuthExhausted` | 认证失败次数用尽（botId/secret 配置错误） |
| `ErrReconnectExhausted` | 重连次数用尽（网络/服务端持续不可用） |
| `ErrNotConnected` | 连接未建立时调用发送方法 |
| `ErrQueueFull` | 回复队列超过最大长度 |

## 实现顺序

1. `types.go` — 类型、常量、接口（无依赖，优先完成）
2. `media.go` — `DecryptFile`（纯函数，独立可测）
3. `ws.go` — `WsConnectionManager`（WebSocket 生命周期、心跳、重连、队列）
4. `client.go` — `Client` 构造、生命周期、所有 reply/send 方法
5. `examples/basic/` — 第一个可运行示例验证端到端流程
