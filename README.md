# wecom-aibot-go-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/liyujun/wecom-aibot-go-sdk.svg)](https://pkg.go.dev/github.com/liyujun/wecom-aibot-go-sdk)
[![Go Report Card](https://goreportcard.com/badge/github.com/liyujun/wecom-aibot-go-sdk)](https://goreportcard.com/report/github.com/liyujun/wecom-aibot-go-sdk)
[![codecov](https://codecov.io/gh/liyujun/wecom-aibot-go-sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/liyujun/wecom-aibot-go-sdk)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

企业微信智能机器人 Go SDK —— 基于 WebSocket 长连接通道，提供消息收发、流式回复、模板卡片、事件回调、文件下载解密、媒体素材上传等核心能力。

## 安装

```bash
go get github.com/liyujun/wecom-aibot-go-sdk
```

## 快速开始

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	handler := &myHandler{}
	client := aibot.NewClient(
		os.Getenv("WECOM_BOT_ID"),
		os.Getenv("WECOM_BOT_SECRET"),
		aibot.WithHandler(handler),
	)
	handler.client = client

	if err := client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	<-ctx.Done()
}

type myHandler struct {
	aibot.BaseHandler
	client *aibot.Client
}

func (h *myHandler) OnAuthenticated() {
	log.Println("认证成功")
}

func (h *myHandler) OnMessage(frame *aibot.WsFrame[any]) {
	body := frame.Body.(map[string]any)
	text := body["text"].(map[string]any)
	content := text["content"].(string)

	streamID := aibot.GenerateReqID("stream")
	h.client.ReplyStream(context.Background(), &frame.Headers, streamID, content, true)
}
```

## 特性

- **WebSocket 长连接** — 基于 `wss://openws.work.weixin.qq.com` 内置默认地址
- **自动认证** — 连接建立后自动发送认证帧
- **心跳保活** — 自动维护心跳，连续未收到 ack 时判定连接异常
- **断线重连** — 指数退避重连策略（1s → 2s → 4s → ... → 30s 上限）
- **消息分发** — 自动解析消息类型并触发对应事件
- **流式回复** — 内置流式回复方法，支持 Markdown
- **模板卡片** — 支持回复模板卡片消息、流式+卡片组合回复、更新卡片
- **主动推送** — 支持向指定会话主动发送 Markdown、模板卡片或媒体消息
- **事件回调** — 支持进入会话、模板卡片按钮点击、用户反馈等事件
- **串行回复队列** — 同一 req_id 的回复消息串行发送，自动等待回执
- **文件下载解密** — 内置 AES-256-CBC 文件解密
- **媒体素材上传** — 支持分片上传临时素材

## API

### Client

```go
client := aibot.NewClient(botID, secret string, opts ...ClientOption)
```

#### 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithWsURL(url)` | 自定义 WebSocket 地址 | `wss://openws.work.weixin.qq.com` |
| `WithHandler(h)` | 消息/事件处理器 | — |
| `WithHeartbeatInterval(d)` | 心跳间隔 | 30s |
| `WithMaxReconnectAttempts(n)` | 最大重连次数 | 10 |
| `WithMaxReplyQueueSize(n)` | 回复队列最大长度 | 500 |
| `WithScene(n)` | 场景值 | — |
| `WithPlugVersion(v)` | 插件版本号 | — |

#### 生命周期

```go
client.Connect(ctx context.Context) error
client.Close()
client.IsConnected() bool
```

#### 被动回复

```go
client.Reply(ctx, frame, body)                                    // 通用回复
client.ReplyStream(ctx, frame, streamID, content, finish)         // 流式回复
client.ReplyWelcome(ctx, frame, body)                            // 欢迎语
client.ReplyTemplateCard(ctx, frame, card)                       // 模板卡片
client.ReplyStreamWithCard(ctx, frame, streamID, content, finish, card)  // 流式+卡片
client.UpdateTemplateCard(ctx, frame, card, userIDs)             // 更新卡片
client.ReplyMedia(ctx, frame, mediaType, mediaID)                // 回复媒体
```

#### 主动推送

```go
client.SendMessage(ctx, chatID, body)          // 发送 Markdown/模板卡片
client.SendMedia(ctx, chatID, mediaType, mediaID)  // 发送媒体
```

#### 媒体

```go
client.UploadMedia(ctx, data, opts) (*UploadMediaFinishResult, error)
client.DownloadFile(ctx, url, aesKey) ([]byte, string, error)
```

### Handler 接口

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
```

嵌入 `BaseHandler` 可只重写关心的方法：

```go
type myHandler struct {
	aibot.BaseHandler
}

func (h *myHandler) OnMessage(frame *aibot.WsFrame[any]) {
	// 处理消息
}
```

## 示例

```bash
# 运行基础示例
WECOM_BOT_ID=your-bot-id WECOM_BOT_SECRET=your-secret go run examples/basic/main.go
```

## 开发

```bash
mise run check   # 格式化 + lint + 测试
mise run test    # 运行测试
mise run lint    # 静态检查
mise run example # 构建示例
```

## License

MIT
