// Package aibot provides a Go SDK for WeChat Work (WeCom) AI Bot WebSocket API.
package aibot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const defaultWsURL = "wss://openws.work.weixin.qq.com"

var (
	// ErrNotConnected is returned when attempting to send on a disconnected connection.
	ErrNotConnected = errors.New("not connected")
	// ErrAuthExhausted is returned when authentication retries are exhausted.
	ErrAuthExhausted = errors.New("auth failure attempts exhausted")
	// ErrReconnectExhausted is returned when reconnection retries are exhausted.
	ErrReconnectExhausted = errors.New("reconnect attempts exhausted")
	// ErrQueueFull is returned when the reply queue for a req_id exceeds maxReplyQueueSize.
	ErrQueueFull = errors.New("reply queue full")
)

type replyQueueItem struct {
	frame   WsFrame[any]
	resolve func(*WsFrame[any])
	reject  func(error)
}

type pendingAck struct {
	resolve func(*WsFrame[any])
	reject  func(error)
	timer   *time.Timer
	seq     int
}

// WsConnectionManager manages the WebSocket lifecycle including authentication,
// heartbeat, reconnection, and reply queue serialization.
type WsConnectionManager struct {
	logger *slog.Logger

	botID     string
	botSecret string
	extraAuth map[string]any

	wsURL  string
	dialer *websocket.Dialer

	heartbeatInterval      time.Duration
	reconnectBaseDelay     time.Duration
	maxReconnectAttempts   int
	maxAuthFailureAttempts int
	maxReplyQueueSize      int
	replyAckTimeout        time.Duration

	conn   *websocket.Conn
	connMu sync.Mutex

	isManualClose bool
	started       bool

	reconnectAttempts       int
	authFailureAttempts     int
	lastCloseWasAuthFailure bool

	missedPongCount int
	maxMissedPong   int

	reconnectTimer *time.Timer
	heartbeatTimer *time.Ticker
	authDone       chan error
	heartbeatStop  chan struct{}
	readStop       chan struct{}
	writeStop      chan struct{}

	writeCh chan []byte

	replyQueues   map[string][]replyQueueItem
	replyQueuesMu sync.Mutex
	pendingAcks   map[string]*pendingAck
	pendingAcksMu sync.Mutex
	pendingAckSeq int

	OnConnected        func()
	OnAuthenticated    func()
	OnDisconnected     func(reason string)
	OnReconnecting     func(attempt int)
	OnError            func(err error)
	OnMessage          func(frame *WsFrame[any])
	OnServerDisconnect func(reason string)
}

// WsConnectionOption configures a WsConnectionManager.
type WsConnectionOption func(*WsConnectionManager)

// WithWsLogger sets the slog.Logger for the connection manager.
func WithWsLogger(logger *slog.Logger) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.logger = logger
	}
}

// WithWsDialer sets the gorilla/websocket Dialer for the connection manager.
func WithWsDialer(d *websocket.Dialer) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.dialer = d
	}
}

func withWsURL(url string) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.wsURL = url
	}
}

// WithWsHeartbeatInterval sets the heartbeat interval.
func WithWsHeartbeatInterval(d time.Duration) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.heartbeatInterval = d
	}
}

// WithWsReconnectBaseDelay sets the base delay for exponential backoff reconnection.
func WithWsReconnectBaseDelay(d time.Duration) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.reconnectBaseDelay = d
	}
}

// WithWsMaxReconnectAttempts sets the maximum reconnection attempts (-1 for unlimited).
func WithWsMaxReconnectAttempts(n int) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.maxReconnectAttempts = n
	}
}

// WithWsMaxAuthFailureAttempts sets the maximum auth failure retries (-1 for unlimited).
func WithWsMaxAuthFailureAttempts(n int) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.maxAuthFailureAttempts = n
	}
}

// WithWsMaxReplyQueueSize sets the maximum reply queue length per req_id.
func WithWsMaxReplyQueueSize(n int) WsConnectionOption {
	return func(w *WsConnectionManager) {
		w.maxReplyQueueSize = n
	}
}

// NewWsConnectionManager creates a new WebSocket connection manager.
func NewWsConnectionManager(opts ...WsConnectionOption) *WsConnectionManager {
	w := &WsConnectionManager{
		logger:                 slog.Default(),
		wsURL:                  defaultWsURL,
		heartbeatInterval:      30 * time.Second,
		reconnectBaseDelay:     time.Second,
		maxReconnectAttempts:   10,
		maxAuthFailureAttempts: 5,
		maxReplyQueueSize:      500,
		replyAckTimeout:        5 * time.Second,
		maxMissedPong:          2,
		replyQueues:            make(map[string][]replyQueueItem),
		pendingAcks:            make(map[string]*pendingAck),
		dialer:                 websocket.DefaultDialer,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// SetCredentials sets the bot authentication credentials and optional extra auth params.
func (w *WsConnectionManager) SetCredentials(botID, botSecret string, extra map[string]any) {
	w.botID = botID
	w.botSecret = botSecret
	w.extraAuth = extra
}

// Connect establishes a WebSocket connection and blocks until authentication completes.
func (w *WsConnectionManager) Connect(ctx context.Context) error {
	return w.connect(ctx, w.wsURL)
}

// ConnectTo establishes a WebSocket connection to a specific URL (non-blocking).
func (w *WsConnectionManager) ConnectTo(wsURL string) {
	ctx := context.Background()
	_ = w.connect(ctx, wsURL)
}

func (w *WsConnectionManager) connect(ctx context.Context, wsURL string) error {
	w.connMu.Lock()
	if w.started {
		w.connMu.Unlock()
		return nil
	}
	w.isManualClose = false
	w.started = true
	w.connMu.Unlock()

	if w.reconnectTimer != nil {
		w.reconnectTimer.Stop()
		w.reconnectTimer = nil
	}

	w.closeConnection()

	conn, _, err := w.dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		w.started = false
		w.logger.Error("failed to dial", "url", wsURL, "err", err)
		if w.OnError != nil {
			w.OnError(err)
		}
		w.scheduleReconnect()
		return err
	}

	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()

	w.missedPongCount = 0
	w.lastCloseWasAuthFailure = false

	w.writeCh = make(chan []byte, 256)
	w.heartbeatStop = make(chan struct{})
	w.readStop = make(chan struct{})
	w.writeStop = make(chan struct{})
	w.authDone = make(chan error, 1)

	if w.OnConnected != nil {
		w.OnConnected()
	}

	go w.readLoop()
	go w.writeLoop()

	w.sendAuth()

	select {
	case err := <-w.authDone:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Disconnect closes the WebSocket connection and stops reconnection.
func (w *WsConnectionManager) Disconnect() {
	w.connMu.Lock()
	w.isManualClose = true
	w.started = false
	w.connMu.Unlock()

	if w.reconnectTimer != nil {
		w.reconnectTimer.Stop()
		w.reconnectTimer = nil
	}

	w.clearPendingMessages("connection manually closed")
	w.closeConnection()
}

// IsConnected reports whether the WebSocket connection is established.
func (w *WsConnectionManager) IsConnected() bool {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	return w.conn != nil
}

func (w *WsConnectionManager) closeConnection() {
	w.connMu.Lock()
	conn := w.conn
	w.conn = nil
	w.connMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	w.stopHeartbeat()
	w.signalStop(w.readStop)
	w.signalStop(w.writeStop)
	w.signalStop(w.heartbeatStop)

	if w.writeCh != nil {
		select {
		case <-w.writeCh:
		default:
		}
	}
}

func (w *WsConnectionManager) signalStop(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (w *WsConnectionManager) sendAuth() {
	body := map[string]any{
		"bot_id": w.botID,
		"secret": w.botSecret,
	}
	for k, v := range w.extraAuth {
		body[k] = v
	}
	w.sendFrame(WsFrame[any]{
		Cmd:     string(CmdSubscribe),
		Headers: WsFrameHeaders{ReqID: generateReqID(string(CmdSubscribe))},
		Body:    body,
	})
}

func (w *WsConnectionManager) startHeartbeat() {
	w.stopHeartbeat()
	w.heartbeatStop = make(chan struct{})
	w.heartbeatTimer = time.NewTicker(w.heartbeatInterval)
	go func() {
		for {
			select {
			case <-w.heartbeatTimer.C:
				w.sendHeartbeat()
			case <-w.heartbeatStop:
				w.heartbeatTimer.Stop()
				return
			}
		}
	}()
}

func (w *WsConnectionManager) stopHeartbeat() {
	if w.heartbeatStop != nil {
		select {
		case <-w.heartbeatStop:
		default:
			close(w.heartbeatStop)
		}
	}
}

func (w *WsConnectionManager) sendHeartbeat() {
	if w.missedPongCount >= w.maxMissedPong {
		w.logger.Warn("heartbeat missed too many times, closing connection")
		w.stopHeartbeat()
		w.connMu.Lock()
		conn := w.conn
		w.connMu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
		return
	}

	w.missedPongCount++
	w.sendFrame(WsFrame[any]{
		Cmd:     string(CmdHeartbeat),
		Headers: WsFrameHeaders{ReqID: generateReqID(string(CmdHeartbeat))},
	})
}

func (w *WsConnectionManager) sendFrame(frame WsFrame[any]) {
	data, err := json.Marshal(frame)
	if err != nil {
		w.logger.Error("failed to marshal frame", "err", err)
		return
	}
	select {
	case w.writeCh <- data:
	default:
		w.logger.Warn("write channel full, dropping frame")
	}
}

func (w *WsConnectionManager) readLoop() {
	defer func() {
		w.handleDisconnect("read loop ended")
	}()

	for {
		select {
		case <-w.readStop:
			return
		default:
		}

		w.connMu.Lock()
		conn := w.conn
		w.connMu.Unlock()
		if conn == nil {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if !w.isManualClose {
				w.logger.Warn("read error", "err", err)
			}
			return
		}

		var frame WsFrame[any]
		if err := json.Unmarshal(msg, &frame); err != nil {
			w.logger.Error("failed to parse frame", "err", err)
			continue
		}

		w.handleFrame(&frame)
	}
}

func (w *WsConnectionManager) writeLoop() {
	for {
		select {
		case <-w.writeStop:
			return
		case data, ok := <-w.writeCh:
			if !ok {
				return
			}
			w.connMu.Lock()
			conn := w.conn
			w.connMu.Unlock()
			if conn == nil {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				w.logger.Error("write error", "err", err)
				return
			}
		}
	}
}

func (w *WsConnectionManager) handleFrame(frame *WsFrame[any]) {
	reqID := frame.Headers.ReqID

	if frame.Cmd == string(CmdCallback) || frame.Cmd == string(CmdEventCallback) {
		if frame.Cmd == string(CmdEventCallback) {
			bodyJSON, _ := json.Marshal(frame.Body)
			var bodyMap map[string]any
			_ = json.Unmarshal(bodyJSON, &bodyMap)
			if bodyMap != nil {
				if event, ok := bodyMap["event"].(map[string]any); ok {
					if event["eventtype"] == "disconnected_event" {
						w.logger.Warn("received disconnected_event, server closing connection")
						w.stopHeartbeat()
						w.clearPendingMessages("server disconnected")
						w.connMu.Lock()
						w.isManualClose = true
						conn := w.conn
						w.conn = nil
						w.connMu.Unlock()
						if conn != nil {
							_ = conn.Close()
						}
						if w.OnServerDisconnect != nil {
							w.OnServerDisconnect("new connection established")
						}
						return
					}
				}
			}
		}
		if w.OnMessage != nil {
			w.OnMessage(frame)
		}
		return
	}

	if stringsHasPrefix(reqID, string(CmdSubscribe)) {
		if frame.Errcode != 0 {
			w.logger.Error("auth failed", "errcode", frame.Errcode, "errmsg", frame.Errmsg)
			authErr := fmt.Errorf("auth failed: %s (code: %d)", frame.Errmsg, frame.Errcode)
			select {
			case w.authDone <- authErr:
			default:
			}
			if w.OnError != nil {
				w.OnError(authErr)
			}
			w.lastCloseWasAuthFailure = true
			w.connMu.Lock()
			conn := w.conn
			w.connMu.Unlock()
			if conn != nil {
				_ = conn.Close()
			}
			return
		}
		w.logger.Info("auth successful")
		select {
		case w.authDone <- nil:
		default:
		}
		w.reconnectAttempts = 0
		w.authFailureAttempts = 0
		w.startHeartbeat()
		if w.OnAuthenticated != nil {
			w.OnAuthenticated()
		}
		return
	}

	if stringsHasPrefix(reqID, string(CmdHeartbeat)) {
		if frame.Errcode == 0 {
			w.missedPongCount = 0
		}
		return
	}

	w.handleReplyAck(reqID, frame)
}

func (w *WsConnectionManager) handleDisconnect(reason string) {
	w.stopHeartbeat()
	w.clearPendingMessages(reason)

	w.connMu.Lock()
	isManual := w.isManualClose
	w.connMu.Unlock()

	if w.OnDisconnected != nil {
		w.OnDisconnected(reason)
	}

	if !isManual {
		w.scheduleReconnect()
	}
}

func (w *WsConnectionManager) scheduleReconnect() {
	if w.lastCloseWasAuthFailure {
		if w.maxAuthFailureAttempts > 0 && w.authFailureAttempts >= w.maxAuthFailureAttempts {
			w.logger.Error("max auth failure attempts reached")
			w.connMu.Lock()
			w.started = false
			w.connMu.Unlock()
			if w.OnError != nil {
				w.OnError(ErrAuthExhausted)
			}
			return
		}
		w.authFailureAttempts++
		delay := time.Duration(math.Min(
			float64(w.reconnectBaseDelay)*math.Pow(2, float64(w.authFailureAttempts-1)),
			float64(30*time.Second),
		))
		w.logger.Info("auth failed, reconnecting", "delay", delay, "attempt", w.authFailureAttempts)
		if w.OnReconnecting != nil {
			w.OnReconnecting(w.authFailureAttempts)
		}
		w.reconnectTimer = time.AfterFunc(delay, func() {
			_ = w.connect(context.Background(), w.wsURL)
		})
	} else {
		if w.maxReconnectAttempts > 0 && w.reconnectAttempts >= w.maxReconnectAttempts {
			w.logger.Error("max reconnect attempts reached")
			w.connMu.Lock()
			w.started = false
			w.connMu.Unlock()
			if w.OnError != nil {
				w.OnError(ErrReconnectExhausted)
			}
			return
		}
		w.reconnectAttempts++
		delay := time.Duration(math.Min(
			float64(w.reconnectBaseDelay)*math.Pow(2, float64(w.reconnectAttempts-1)),
			float64(30*time.Second),
		))
		w.logger.Info("reconnecting", "delay", delay, "attempt", w.reconnectAttempts)
		if w.OnReconnecting != nil {
			w.OnReconnecting(w.reconnectAttempts)
		}
		w.reconnectTimer = time.AfterFunc(delay, func() {
			_ = w.connect(context.Background(), w.wsURL)
		})
	}
}

// SendReply sends a message through the reply queue and waits for the server ack.
func (w *WsConnectionManager) SendReply(ctx context.Context, reqID string, body any, cmd WsCmd) (*WsFrame[any], error) {
	return w.sendReplyInternal(ctx, reqID, string(cmd), body)
}

func (w *WsConnectionManager) sendReplyInternal(ctx context.Context, reqID string, cmd string, body any) (*WsFrame[any], error) {
	resultCh := make(chan struct {
		frame *WsFrame[any]
		err   error
	}, 1)

	frame := WsFrame[any]{
		Cmd:     cmd,
		Headers: WsFrameHeaders{ReqID: reqID},
		Body:    body,
	}

	item := replyQueueItem{
		frame: frame,
		resolve: func(f *WsFrame[any]) {
			resultCh <- struct {
				frame *WsFrame[any]
				err   error
			}{frame: f, err: nil}
		},
		reject: func(err error) {
			resultCh <- struct {
				frame *WsFrame[any]
				err   error
			}{frame: nil, err: err}
		},
	}

	w.replyQueuesMu.Lock()
	queue := w.replyQueues[reqID]
	if len(queue) >= w.maxReplyQueueSize {
		w.replyQueuesMu.Unlock()
		return nil, ErrQueueFull
	}
	w.replyQueues[reqID] = append(queue, item)
	wasEmpty := len(queue) == 0
	w.replyQueuesMu.Unlock()

	if wasEmpty {
		w.processReplyQueue(reqID)
	}

	select {
	case result := <-resultCh:
		return result.frame, result.err
	case <-ctx.Done():
		w.replyQueuesMu.Lock()
		queue := w.replyQueues[reqID]
		for i, qItem := range queue {
			if qItem.resolve != nil {
				queue[i].resolve = nil
				queue[i].reject = nil
			}
		}
		w.replyQueuesMu.Unlock()
		return nil, ctx.Err()
	}
}

func (w *WsConnectionManager) processReplyQueue(reqID string) {
	w.replyQueuesMu.Lock()
	queue := w.replyQueues[reqID]
	if len(queue) == 0 {
		delete(w.replyQueues, reqID)
		w.replyQueuesMu.Unlock()
		return
	}
	item := queue[0]
	w.replyQueuesMu.Unlock()

	w.connMu.Lock()
	conn := w.conn
	w.connMu.Unlock()
	if conn == nil {
		w.replyQueuesMu.Lock()
		queue = w.replyQueues[reqID]
		if len(queue) > 0 {
			queue = queue[1:]
			w.replyQueues[reqID] = queue
		}
		w.replyQueuesMu.Unlock()
		if item.reject != nil {
			item.reject(ErrNotConnected)
		}
		go w.processReplyQueue(reqID)
		return
	}

	w.sendFrame(item.frame)

	w.pendingAcksMu.Lock()
	w.pendingAckSeq++
	seq := w.pendingAckSeq
	pending := &pendingAck{
		resolve: item.resolve,
		reject:  item.reject,
		seq:     seq,
	}
	pending.timer = time.AfterFunc(w.replyAckTimeout, func() {
		w.pendingAcksMu.Lock()
		current := w.pendingAcks[reqID]
		if current != nil && current.seq == seq {
			delete(w.pendingAcks, reqID)
			w.pendingAcksMu.Unlock()

			w.replyQueuesMu.Lock()
			q := w.replyQueues[reqID]
			if len(q) > 0 {
				q = q[1:]
				w.replyQueues[reqID] = q
			}
			w.replyQueuesMu.Unlock()

			if pending.reject != nil {
				pending.reject(fmt.Errorf("reply ack timeout for reqID: %s", reqID))
			}
			w.processReplyQueue(reqID)
		} else {
			w.pendingAcksMu.Unlock()
		}
	})
	w.pendingAcks[reqID] = pending
	w.pendingAcksMu.Unlock()
}

func (w *WsConnectionManager) handleReplyAck(reqID string, frame *WsFrame[any]) {
	w.pendingAcksMu.Lock()
	pending := w.pendingAcks[reqID]
	if pending == nil {
		w.pendingAcksMu.Unlock()
		return
	}
	if pending.timer != nil {
		pending.timer.Stop()
	}
	delete(w.pendingAcks, reqID)
	w.pendingAcksMu.Unlock()

	w.replyQueuesMu.Lock()
	queue := w.replyQueues[reqID]
	if len(queue) > 0 {
		queue = queue[1:]
		w.replyQueues[reqID] = queue
	}
	w.replyQueuesMu.Unlock()

	if frame.Errcode != 0 {
		if pending.reject != nil {
			pending.reject(fmt.Errorf("reply error: %s (code: %d)", frame.Errmsg, frame.Errcode))
		}
	} else {
		if pending.resolve != nil {
			pending.resolve(frame)
		}
	}

	w.processReplyQueue(reqID)
}

// HasPendingAck reports whether the given req_id is waiting for a server ack.
func (w *WsConnectionManager) HasPendingAck(reqID string) bool {
	w.pendingAcksMu.Lock()
	defer w.pendingAcksMu.Unlock()
	_, ok := w.pendingAcks[reqID]
	return ok
}

func (w *WsConnectionManager) clearPendingMessages(reason string) {
	w.pendingAcksMu.Lock()
	for reqID, pending := range w.pendingAcks {
		if pending.timer != nil {
			pending.timer.Stop()
		}
		if pending.reject != nil {
			pending.reject(fmt.Errorf("%s: %s", reason, reqID))
		}
		delete(w.pendingAcks, reqID)
	}
	w.pendingAcksMu.Unlock()

	w.replyQueuesMu.Lock()
	for reqID, queue := range w.replyQueues {
		for _, item := range queue {
			if item.reject != nil {
				item.reject(fmt.Errorf("%s: %s", reason, reqID))
			}
		}
		delete(w.replyQueues, reqID)
	}
	w.replyQueuesMu.Unlock()
}
