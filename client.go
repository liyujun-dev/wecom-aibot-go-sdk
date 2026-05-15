// Package aibot provides a Go SDK for WeChat Work (WeCom) AI Bot WebSocket API.
package aibot

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Client is the main entry point for the WeCom AIBot SDK.
// It wraps a WsConnectionManager and dispatches WebSocket frames to the Handler.
type Client struct {
	botID     string
	botSecret string
	handler   Handler
	ws        *WsConnectionManager
	api       *WeComAPIClient
	logger    *slog.Logger
	config    Config
}

// NewClient creates a new Client with the given bot credentials and options.
func NewClient(botID, botSecret string, opts ...ClientOption) *Client {
	cfg := Config{
		ReconnectInterval:      time.Second,
		MaxReconnectAttempts:   10,
		MaxAuthFailureAttempts: 5,
		HeartbeatInterval:      30 * time.Second,
		MaxReplyQueueSize:      500,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	logger := slog.Default()

	if cfg.WsURL == "" {
		cfg.WsURL = defaultWsURL
	}

	c := &Client{
		botID:     botID,
		botSecret: botSecret,
		handler:   cfg.Handler,
		config:    cfg,
		logger:    logger,
	}

	c.api = &WeComAPIClient{
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		logger: logger,
	}

	extraAuth := map[string]any{}
	if cfg.Scene != 0 {
		extraAuth["scene"] = cfg.Scene
	}
	if cfg.PlugVersion != "" {
		extraAuth["plug_version"] = cfg.PlugVersion
	}

	wsOpts := []WsConnectionOption{
		WithWsLogger(logger),
		withWsURL(cfg.WsURL),
		WithWsHeartbeatInterval(cfg.HeartbeatInterval),
		WithWsReconnectBaseDelay(cfg.ReconnectInterval),
		WithWsMaxReconnectAttempts(cfg.MaxReconnectAttempts),
		WithWsMaxAuthFailureAttempts(cfg.MaxAuthFailureAttempts),
		WithWsMaxReplyQueueSize(cfg.MaxReplyQueueSize),
	}

	c.ws = NewWsConnectionManager(wsOpts...)
	c.ws.SetCredentials(botID, botSecret, extraAuth)

	c.setupWsCallbacks()

	return c
}

func (c *Client) setupWsCallbacks() {
	c.ws.OnConnected = func() {
		if c.handler != nil {
			c.handler.OnConnected()
		}
	}

	c.ws.OnAuthenticated = func() {
		if c.handler != nil {
			c.handler.OnAuthenticated()
		}
	}

	c.ws.OnDisconnected = func(reason string) {
		if c.handler != nil {
			c.handler.OnDisconnected(reason)
		}
	}

	c.ws.OnReconnecting = func(attempt int) {
		if c.handler != nil {
			c.handler.OnReconnecting(attempt)
		}
	}

	c.ws.OnError = func(err error) {
		if c.handler != nil {
			c.handler.OnError(err)
		}
	}

	c.ws.OnServerDisconnect = func(reason string) {
		if c.handler != nil {
			c.handler.OnDisconnected(reason)
		}
	}

	c.ws.OnMessage = func(frame *WsFrame[any]) {
		if c.handler == nil {
			return
		}
		if frame.Cmd == string(CmdCallback) {
			c.handler.OnMessage(frame)
		} else if frame.Cmd == string(CmdEventCallback) {
			c.handler.OnEvent(frame)
		}
	}
}

// Connect establishes the WebSocket connection and blocks until authentication completes.
func (c *Client) Connect(ctx context.Context) error {
	return c.ws.Connect(ctx)
}

// Close disconnects the WebSocket and stops reconnection.
func (c *Client) Close() {
	c.ws.Disconnect()
}

// IsConnected reports whether the client is connected.
func (c *Client) IsConnected() bool {
	return c.ws.IsConnected()
}

// Reply sends a generic reply message through the WebSocket channel.
func (c *Client) Reply(ctx context.Context, frame *WsFrameHeaders, body any) (*WsFrame[any], error) {
	reqID := ""
	if frame != nil {
		reqID = frame.ReqID
	}
	return c.ws.SendReply(ctx, reqID, body, CmdResponse)
}

// ReplyStream sends a streaming reply message. Set finish to true on the last message.
func (c *Client) ReplyStream(ctx context.Context, frame *WsFrameHeaders, streamID, content string, finish bool) (*WsFrame[any], error) {
	stream := StreamReplyContent{
		ID:      streamID,
		Finish:  finish,
		Content: content,
	}
	return c.Reply(ctx, frame, StreamReplyBody{
		MsgType: "stream",
		Stream:  stream,
	})
}

// ReplyWelcome sends a welcome message (must be called within 5 seconds of enter_chat event).
func (c *Client) ReplyWelcome(ctx context.Context, frame *WsFrameHeaders, body any) (*WsFrame[any], error) {
	reqID := ""
	if frame != nil {
		reqID = frame.ReqID
	}
	return c.ws.SendReply(ctx, reqID, body, CmdResponseWelcome)
}

// ReplyTemplateCard sends a template card reply message.
func (c *Client) ReplyTemplateCard(ctx context.Context, frame *WsFrameHeaders, card *TemplateCard) (*WsFrame[any], error) {
	return c.Reply(ctx, frame, TemplateCardReplyBody{
		MsgType:      "template_card",
		TemplateCard: *card,
	})
}

// ReplyStreamWithCard sends a streaming reply with an optional template card.
func (c *Client) ReplyStreamWithCard(ctx context.Context, frame *WsFrameHeaders, streamID, content string, finish bool, card *TemplateCard) (*WsFrame[any], error) {
	stream := StreamReplyContent{
		ID:      streamID,
		Finish:  finish,
		Content: content,
	}
	body := StreamWithTemplateCardReplyBody{
		MsgType: "stream_with_template_card",
		Stream:  stream,
	}
	if card != nil {
		body.TemplateCard = card
	}
	return c.Reply(ctx, frame, body)
}

// UpdateTemplateCard updates a previously sent template card (must be called within 5 seconds of card event).
func (c *Client) UpdateTemplateCard(ctx context.Context, frame *WsFrameHeaders, card *TemplateCard, userIDs []string) (*WsFrame[any], error) {
	reqID := ""
	if frame != nil {
		reqID = frame.ReqID
	}
	body := UpdateTemplateCardBody{
		ResponseType: "update_template_card",
		TemplateCard: *card,
		UserIDs:      userIDs,
	}
	return c.ws.SendReply(ctx, reqID, body, CmdResponseUpdate)
}

// SendMessage proactively sends a message to a chat (single or group).
func (c *Client) SendMessage(ctx context.Context, chatID string, body any) (*WsFrame[any], error) {
	reqID := GenerateReqID(string(CmdSendMsg))

	fullBody := map[string]any{"chatid": chatID}
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		var bodyMap map[string]any
		if err := json.Unmarshal(bodyJSON, &bodyMap); err != nil {
			return nil, err
		}
		for k, v := range bodyMap {
			fullBody[k] = v
		}
	}

	return c.ws.SendReply(ctx, reqID, fullBody, CmdSendMsg)
}

// SendMedia proactively sends a media message to a chat.
func (c *Client) SendMedia(ctx context.Context, chatID string, mediaType WeComMediaType, mediaID string) (*WsFrame[any], error) {
	body := SendMediaMsgBody{
		MsgType: string(mediaType),
	}
	switch mediaType {
	case MediaTypeFile:
		body.File = &MediaContent{MediaID: mediaID}
	case MediaTypeImage:
		body.Image = &MediaContent{MediaID: mediaID}
	case MediaTypeVoice:
		body.Voice = &MediaContent{MediaID: mediaID}
	case MediaTypeVideo:
		body.Video = &VideoMediaContent{MediaID: mediaID}
	}
	return c.SendMessage(ctx, chatID, body)
}

// ReplyMedia replies with a media message (file/image/voice/video).
func (c *Client) ReplyMedia(ctx context.Context, frame *WsFrameHeaders, mediaType WeComMediaType, mediaID string) (*WsFrame[any], error) {
	body := SendMediaMsgBody{
		MsgType: string(mediaType),
	}
	switch mediaType {
	case MediaTypeFile:
		body.File = &MediaContent{MediaID: mediaID}
	case MediaTypeImage:
		body.Image = &MediaContent{MediaID: mediaID}
	case MediaTypeVoice:
		body.Voice = &MediaContent{MediaID: mediaID}
	case MediaTypeVideo:
		body.Video = &VideoMediaContent{MediaID: mediaID}
	}
	return c.Reply(ctx, frame, body)
}

// HasPendingAck reports whether a reply for the given frame's req_id is waiting for ack.
func (c *Client) HasPendingAck(frame *WsFrameHeaders) bool {
	if frame == nil {
		return false
	}
	return c.ws.HasPendingAck(frame.ReqID)
}

// =============================================================================
// WeComApiClient (HTTP file download)
// =============================================================================

// WeComAPIClient handles HTTP requests for file downloads.
type WeComAPIClient struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// DownloadFileRaw downloads a file from the given URL and returns the raw bytes and filename.
func (api *WeComAPIClient) DownloadFileRaw(url string) ([]byte, string, error) {
	api.logger.Info("downloading file", "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	buf := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return nil, "", err
	}

	cd := resp.Header.Get("Content-Disposition")
	filename := parseFilename(cd)

	return buf, filename, nil
}

func parseFilename(contentDisposition string) string {
	if contentDisposition == "" {
		return ""
	}
	for _, part := range splitHeader(contentDisposition, ";") {
		part = trimSpace(part)
		if len(part) > 9 && part[:9] == "filename=" {
			name := part[9:]
			if len(name) >= 2 && name[0] == '"' && name[len(name)-1] == '"' {
				name = name[1 : len(name)-1]
			}
			return name
		}
	}
	return ""
}

func splitHeader(s, sep string) []string {
	var result []string
	for {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
