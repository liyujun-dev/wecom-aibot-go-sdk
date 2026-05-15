package aibot_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

type testHandler struct {
	aibot.BaseHandler
	mu            sync.Mutex
	connected     bool
	authenticated bool
	messages      []*aibot.WsFrame[any]
	events        []*aibot.WsFrame[any]
}

func (h *testHandler) OnConnected() {
	h.mu.Lock()
	h.connected = true
	h.mu.Unlock()
}

func (h *testHandler) OnAuthenticated() {
	h.mu.Lock()
	h.authenticated = true
	h.mu.Unlock()
}

func (h *testHandler) OnMessage(frame *aibot.WsFrame[any]) {
	h.mu.Lock()
	h.messages = append(h.messages, frame)
	h.mu.Unlock()
}

func (h *testHandler) OnEvent(frame *aibot.WsFrame[any]) {
	h.mu.Lock()
	h.events = append(h.events, frame)
	h.mu.Unlock()
}

func TestClientConnect(t *testing.T) {
	handler := &testHandler{}

	srv := wsTestServer(t, func(conn *websocket.Conn) {
		frame, err := readFrame(conn)
		if err != nil {
			return
		}
		if frame.Cmd == string(aibot.CmdSubscribe) {
			resp := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: frame.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, resp)
		}
		for {
			f, err := readFrame(conn)
			if err != nil {
				return
			}
			ack := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: f.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, ack)
		}
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := aibot.NewClient("bot1", "secret1",
		aibot.WithWsURL(wsURL),
		aibot.WithHandler(handler),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	if !handler.connected {
		t.Error("handler.OnConnected not called")
	}
	if !handler.authenticated {
		t.Error("handler.OnAuthenticated not called")
	}
	if !client.IsConnected() {
		t.Error("client not connected")
	}
}

func TestClientReply(t *testing.T) {
	handler := &testHandler{}

	srv := wsTestServer(t, func(conn *websocket.Conn) {
		frame, err := readFrame(conn)
		if err != nil {
			return
		}
		if frame.Cmd == string(aibot.CmdSubscribe) {
			resp := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: frame.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, resp)
		}
		for {
			f, err := readFrame(conn)
			if err != nil {
				return
			}
			ack := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: f.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, ack)
		}
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := aibot.NewClient("bot1", "secret1",
		aibot.WithWsURL(wsURL),
		aibot.WithHandler(handler),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	frame := &aibot.WsFrameHeaders{ReqID: "msg_001"}
	body := map[string]string{"msgtype": "text", "text": `{"content":"hello"}`}

	ack, err := client.Reply(ctx, frame, body)
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if ack.Errcode != 0 {
		t.Errorf("ack errcode = %d, want 0", ack.Errcode)
	}
}

func TestClientReceiveMessage(t *testing.T) {
	handler := &testHandler{}

	var connMu sync.Mutex
	var clientConn *websocket.Conn

	srv := wsTestServer(t, func(conn *websocket.Conn) {
		connMu.Lock()
		clientConn = conn
		connMu.Unlock()

		frame, err := readFrame(conn)
		if err != nil {
			return
		}
		if frame.Cmd == string(aibot.CmdSubscribe) {
			resp := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: frame.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, resp)
		}
		for {
			f, err := readFrame(conn)
			if err != nil {
				return
			}
			ack := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: f.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, ack)
		}
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := aibot.NewClient("bot1", "secret1",
		aibot.WithWsURL(wsURL),
		aibot.WithHandler(handler),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	msgFrame := aibot.WsFrame[any]{
		Cmd: string(aibot.CmdCallback),
		Headers: aibot.WsFrameHeaders{
			ReqID: "cb_001",
		},
		Body: map[string]any{
			"msgid":    "m1",
			"aibotid":  "bot1",
			"chattype": "single",
			"from":     map[string]any{"userid": "user1"},
			"msgtype":  "text",
			"text":     map[string]any{"content": "hello"},
		},
	}

	data, _ := json.Marshal(msgFrame)
	connMu.Lock()
	if clientConn != nil {
		_ = clientConn.WriteMessage(websocket.TextMessage, data)
	}
	connMu.Unlock()

	time.Sleep(100 * time.Millisecond)

	handler.mu.Lock()
	msgCount := len(handler.messages)
	handler.mu.Unlock()

	if msgCount < 1 {
		t.Error("expected at least 1 message received")
	}
}

func TestClientReplyStream(t *testing.T) {
	handler := &testHandler{}

	srv := wsTestServer(t, func(conn *websocket.Conn) {
		frame, err := readFrame(conn)
		if err != nil {
			return
		}
		if frame.Cmd == string(aibot.CmdSubscribe) {
			resp := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: frame.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, resp)
		}
		for {
			f, err := readFrame(conn)
			if err != nil {
				return
			}
			ack := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: f.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, ack)
		}
	})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client := aibot.NewClient("bot1", "secret1",
		aibot.WithWsURL(wsURL),
		aibot.WithHandler(handler),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	frame := &aibot.WsFrameHeaders{ReqID: "msg_001"}

	streamID := aibot.GenerateReqID("stream")
	ack, err := client.ReplyStream(ctx, frame, streamID, "processing...", false)
	if err != nil {
		t.Fatalf("ReplyStream (non-finish): %v", err)
	}
	if ack.Errcode != 0 {
		t.Errorf("non-finish ack errcode = %d", ack.Errcode)
	}

	ack, err = client.ReplyStream(ctx, frame, streamID, "done!", true)
	if err != nil {
		t.Fatalf("ReplyStream (finish): %v", err)
	}
	if ack.Errcode != 0 {
		t.Errorf("finish ack errcode = %d", ack.Errcode)
	}
}
