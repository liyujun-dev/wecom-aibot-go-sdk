package aibot_test

import (
	"encoding/json"
	"testing"

	aibot "github.com/liyujun/wecom-aibot-go-sdk"
)

func TestWsFrameJSON(t *testing.T) {
	input := `{"cmd":"aibot_msg_callback","headers":{"req_id":"call_123"},"body":{"msgid":"m1","aibotid":"bot1","chattype":"single","from":{"userid":"user1"},"msgtype":"text","text":{"content":"hello"}}}`

	var frame aibot.WsFrame[aibot.TextMessage]
	if err := json.Unmarshal([]byte(input), &frame); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if frame.Cmd != "aibot_msg_callback" {
		t.Errorf("Cmd = %q, want %q", frame.Cmd, "aibot_msg_callback")
	}
	if frame.Headers.ReqID != "call_123" {
		t.Errorf("Headers.ReqID = %q, want %q", frame.Headers.ReqID, "call_123")
	}
	if frame.Body.MsgID != "m1" {
		t.Errorf("Body.MsgID = %q, want %q", frame.Body.MsgID, "m1")
	}
	if frame.Body.Text.Content != "hello" {
		t.Errorf("Body.Text.Content = %q, want %q", frame.Body.Text.Content, "hello")
	}

	data, err := json.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundtrip aibot.WsFrame[aibot.TextMessage]
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("roundtrip unmarshal: %v", err)
	}
	if roundtrip.Body.Text.Content != "hello" {
		t.Errorf("roundtrip Text.Content = %q, want %q", roundtrip.Body.Text.Content, "hello")
	}
}

func TestBaseHandlerSatisfiesHandler(_ *testing.T) {
	var _ aibot.Handler = aibot.BaseHandler{}
}

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		val  aibot.MessageType
		want string
	}{
		{aibot.MessageTypeText, "text"},
		{aibot.MessageTypeImage, "image"},
		{aibot.MessageTypeMixed, "mixed"},
		{aibot.MessageTypeVoice, "voice"},
		{aibot.MessageTypeFile, "file"},
		{aibot.MessageTypeVideo, "video"},
	}
	for _, tc := range tests {
		if string(tc.val) != tc.want {
			t.Errorf("MessageType = %q, want %q", tc.val, tc.want)
		}
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		val  aibot.EventType
		want string
	}{
		{aibot.EventTypeEnterChat, "enter_chat"},
		{aibot.EventTypeTemplateCardEvent, "template_card_event"},
		{aibot.EventTypeFeedbackEvent, "feedback_event"},
		{aibot.EventTypeDisconnected, "disconnected_event"},
	}
	for _, tc := range tests {
		if string(tc.val) != tc.want {
			t.Errorf("EventType = %q, want %q", tc.val, tc.want)
		}
	}
}

func TestWsCmdConstants(t *testing.T) {
	tests := []struct {
		val  aibot.WsCmd
		want string
	}{
		{aibot.CmdSubscribe, "aibot_subscribe"},
		{aibot.CmdHeartbeat, "ping"},
		{aibot.CmdResponse, "aibot_respond_msg"},
		{aibot.CmdResponseWelcome, "aibot_respond_welcome_msg"},
		{aibot.CmdResponseUpdate, "aibot_respond_update_msg"},
		{aibot.CmdSendMsg, "aibot_send_msg"},
		{aibot.CmdUploadMediaInit, "aibot_upload_media_init"},
		{aibot.CmdUploadMediaChunk, "aibot_upload_media_chunk"},
		{aibot.CmdUploadMediaFinish, "aibot_upload_media_finish"},
		{aibot.CmdCallback, "aibot_msg_callback"},
		{aibot.CmdEventCallback, "aibot_event_callback"},
	}
	for _, tc := range tests {
		if string(tc.val) != tc.want {
			t.Errorf("WsCmd = %q, want %q", tc.val, tc.want)
		}
	}
}

func TestImageMessageJSON(t *testing.T) {
	input := `{"msgid":"m1","aibotid":"bot1","chattype":"single","from":{"userid":"u1"},"msgtype":"image","image":{"url":"https://example.com/enc","aeskey":"key123"}}`
	var msg aibot.ImageMessage
	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Image.URL != "https://example.com/enc" {
		t.Errorf("Image.URL = %q", msg.Image.URL)
	}
	if msg.Image.AesKey != "key123" {
		t.Errorf("Image.AesKey = %q", msg.Image.AesKey)
	}
}

func TestEventMessageJSON(t *testing.T) {
	input := `{"msgid":"e1","create_time":1700000000,"aibotid":"bot1","chattype":"single","from":{"userid":"u1"},"msgtype":"event","event":{"eventtype":"enter_chat"}}`
	var msg aibot.EventMessage
	if err := json.Unmarshal([]byte(input), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Event.EventType != "enter_chat" {
		t.Errorf("EventType = %q, want %q", msg.Event.EventType, "enter_chat")
	}
}

func TestTemplateCardTypeConstants(t *testing.T) {
	tests := []struct {
		val  aibot.TemplateCardType
		want string
	}{
		{aibot.TemplateCardTypeTextNotice, "text_notice"},
		{aibot.TemplateCardTypeNewsNotice, "news_notice"},
		{aibot.TemplateCardTypeButtonInteraction, "button_interaction"},
		{aibot.TemplateCardTypeVoteInteraction, "vote_interaction"},
		{aibot.TemplateCardTypeMultipleInteraction, "multiple_interaction"},
	}
	for _, tc := range tests {
		if string(tc.val) != tc.want {
			t.Errorf("TemplateCardType = %q, want %q", tc.val, tc.want)
		}
	}
}
