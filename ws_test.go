package aibot_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

func wsTestServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestWsConnectionManagerConnect(t *testing.T) {
	var connected, authenticated, disconnected bool
	var mu sync.Mutex

	srv := wsTestServer(t, func(conn *websocket.Conn) {
		msg, err := readFrame(conn)
		if err != nil {
			return
		}
		resp := aibot.WsFrame[any]{
			Headers: aibot.WsFrameHeaders{ReqID: msg.Headers.ReqID},
			Errcode: 0,
			Errmsg:  "ok",
		}
		data, _ := json.Marshal(resp)
		_ = conn.WriteMessage(websocket.TextMessage, data)
	})

	wsManager := aibot.NewWsConnectionManager()
	wsManager.SetCredentials("bot123", "secret123", nil)

	wsManager.OnConnected = func() {
		mu.Lock()
		connected = true
		mu.Unlock()
	}
	wsManager.OnAuthenticated = func() {
		mu.Lock()
		authenticated = true
		mu.Unlock()
	}
	wsManager.OnDisconnected = func(_ string) {
		mu.Lock()
		disconnected = true
		mu.Unlock()
	}

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsManager.ConnectTo(wsURL)
	defer wsManager.Disconnect()

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !connected {
		t.Error("OnConnected was not called")
	}
	if !authenticated {
		t.Error("OnAuthenticated was not called")
	}
	if disconnected {
		t.Error("OnDisconnected was unexpectedly called")
	}
}

func TestWsConnectionManagerReplyAndAck(t *testing.T) {
	srv := wsTestServer(t, func(conn *websocket.Conn) {
		msg, err := readFrame(conn)
		if err != nil {
			return
		}
		resp := aibot.WsFrame[any]{
			Headers: aibot.WsFrameHeaders{ReqID: msg.Headers.ReqID},
			Errcode: 0,
			Errmsg:  "ok",
		}
		writeFrame(conn, resp)

		for {
			frame, err := readFrame(conn)
			if err != nil {
				return
			}
			ack := aibot.WsFrame[any]{
				Headers: aibot.WsFrameHeaders{ReqID: frame.Headers.ReqID},
				Errcode: 0,
				Errmsg:  "ok",
			}
			writeFrame(conn, ack)
		}
	})

	wsManager := aibot.NewWsConnectionManager()
	wsManager.SetCredentials("bot1", "secret1", nil)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsManager.ConnectTo(wsURL)
	defer wsManager.Disconnect()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ackFrame, err := wsManager.SendReply(ctx, "req_001",
		map[string]string{"msgtype": "text", "text": `{"content":"hello"}`},
		aibot.CmdResponse)
	if err != nil {
		t.Fatalf("SendReply: %v", err)
	}
	if ackFrame.Errcode != 0 {
		t.Errorf("ack errcode = %d, want 0", ackFrame.Errcode)
	}
}

func TestWsConnectionManagerDisconnect(t *testing.T) {
	srv := wsTestServer(t, func(conn *websocket.Conn) {
		msg, err := readFrame(conn)
		if err != nil {
			return
		}
		resp := aibot.WsFrame[any]{
			Headers: aibot.WsFrameHeaders{ReqID: msg.Headers.ReqID},
			Errcode: 0,
			Errmsg:  "ok",
		}
		writeFrame(conn, resp)
	})

	wsManager := aibot.NewWsConnectionManager()
	wsManager.SetCredentials("bot1", "secret1", nil)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsManager.ConnectTo(wsURL)

	time.Sleep(100 * time.Millisecond)

	if !wsManager.IsConnected() {
		t.Error("expected connected after ConnectTo")
	}

	wsManager.Disconnect()

	if wsManager.IsConnected() {
		t.Error("expected disconnected after Disconnect")
	}
}

func TestWsConnectionManagerAuthFailure(t *testing.T) {
	authAttempts := 0
	srv := wsTestServer(t, func(conn *websocket.Conn) {
		msg, err := readFrame(conn)
		if err != nil {
			return
		}
		authAttempts++

		resp := aibot.WsFrame[any]{
			Headers: aibot.WsFrameHeaders{ReqID: msg.Headers.ReqID},
			Errcode: 40001,
			Errmsg:  "invalid secret",
		}
		writeFrame(conn, resp)
	})

	var errReceived error
	wsManager := aibot.NewWsConnectionManager(
		aibot.WithWsLogger(slog.New(slog.DiscardHandler)),
	)
	wsManager.SetCredentials("bad-bot", "bad-secret", nil)
	wsManager.OnError = func(err error) {
		errReceived = err
	}

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsManager.ConnectTo(wsURL)
	defer wsManager.Disconnect()

	time.Sleep(300 * time.Millisecond)

	if errReceived == nil {
		t.Error("expected error from auth failure")
	}
	if authAttempts < 1 {
		t.Error("expected at least 1 auth attempt")
	}
}

func readFrame(conn *websocket.Conn) (aibot.WsFrame[any], error) {
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return aibot.WsFrame[any]{}, err
	}
	var frame aibot.WsFrame[any]
	if err := json.Unmarshal(msg, &frame); err != nil {
		return aibot.WsFrame[any]{}, err
	}
	return frame, nil
}

func writeFrame(conn *websocket.Conn, frame aibot.WsFrame[any]) {
	data, _ := json.Marshal(frame)
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

var _ = slog.New(slog.DiscardHandler)
