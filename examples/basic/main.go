// Basic example: echo bot that replies with the received message content.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

func main() {
	botID := os.Getenv("WECOM_BOT_ID")
	botSecret := os.Getenv("WECOM_BOT_SECRET")

	if botID == "" || botSecret == "" {
		log.Fatal("WECOM_BOT_ID and WECOM_BOT_SECRET environment variables are required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	handler := &botHandler{}
	client := aibot.NewClient(botID, botSecret,
		aibot.WithHandler(handler),
	)
	handler.client = client

	log.Println("Connecting to WeCom AIBot...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	log.Println("Connected and authenticated")

	<-ctx.Done()
	log.Println("Shutting down...")
	client.Close()
	log.Println("Disconnected")
}

type botHandler struct {
	aibot.BaseHandler
	client *aibot.Client
}

func (h *botHandler) OnAuthenticated() {
	log.Println("[event] authenticated")
}

func (h *botHandler) OnDisconnected(reason string) {
	log.Printf("[event] disconnected: %s", reason)
}

func (h *botHandler) OnReconnecting(attempt int) {
	log.Printf("[event] reconnecting (attempt %d)", attempt)
}

func (h *botHandler) OnError(err error) {
	log.Printf("[event] error: %v", err)
}

func (h *botHandler) OnMessage(frame *aibot.WsFrame[any]) {
	body, ok := frame.Body.(map[string]any)
	if !ok {
		return
	}

	msgType, _ := body["msgtype"].(string)
	from, _ := body["from"].(map[string]any)
	userID, _ := from["userid"].(string)

	log.Printf("[msg] type=%s from=%s", msgType, userID)

	ctx := context.Background()
	headers := &frame.Headers

	switch msgType {
	case "text":
		text, _ := body["text"].(map[string]any)
		content, _ := text["content"].(string)
		log.Printf("[text] content=%s", content)
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, content, true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	case "image":
		log.Println("[image] received")
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, "收到图片消息", true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	case "voice":
		voice, _ := body["voice"].(map[string]any)
		content, _ := voice["content"].(string)
		log.Printf("[voice] content=%s", content)
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, content, true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	case "file":
		log.Println("[file] received")
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, "收到文件消息", true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	case "mixed":
		mixed, _ := body["mixed"].(map[string]any)
		msgItems, _ := mixed["msg_item"].([]any)
		var sb string
		for _, item := range msgItems {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType, _ := m["msgtype"].(string)
			switch itemType {
			case "text":
				text, _ := m["text"].(map[string]any)
				content, _ := text["content"].(string)
				sb += content
			case "image":
				sb += "[图片]"
			}
		}
		log.Printf("[mixed] content=%s", sb)
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, sb, true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	case "video":
		log.Println("[video] received")
		streamID := aibot.GenerateReqID("stream")
		if _, err := h.client.ReplyStream(ctx, headers, streamID, "收到视频消息", true); err != nil {
			log.Printf("[reply] error: %v", err)
		}

	default:
		log.Printf("[msg] unhandled type: %s", msgType)
	}
}

func (h *botHandler) OnEvent(frame *aibot.WsFrame[any]) {
	body, ok := frame.Body.(map[string]any)
	if !ok {
		return
	}

	event, _ := body["event"].(map[string]any)
	eventType, _ := event["eventtype"].(string)
	log.Printf("[event] type=%s", eventType)
}
