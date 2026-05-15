//nolint:revive
package aibot

import "time"

// =============================================================================
// WebSocket Frame Types
// =============================================================================

type WsFrameHeaders struct {
	ReqID string `json:"req_id"`
}

type WsFrame[T any] struct {
	Cmd     string         `json:"cmd,omitempty"`
	Headers WsFrameHeaders `json:"headers"`
	Body    T              `json:"body,omitempty"`
	Errcode int            `json:"errcode,omitempty"`
	Errmsg  string         `json:"errmsg,omitempty"`
}

// =============================================================================
// Message Type Constants
// =============================================================================

type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeMixed MessageType = "mixed"
	MessageTypeVoice MessageType = "voice"
	MessageTypeFile  MessageType = "file"
	MessageTypeVideo MessageType = "video"
)

// =============================================================================
// Event Type Constants
// =============================================================================

type EventType string

const (
	EventTypeEnterChat         EventType = "enter_chat"
	EventTypeTemplateCardEvent EventType = "template_card_event"
	EventTypeFeedbackEvent     EventType = "feedback_event"
	EventTypeDisconnected      EventType = "disconnected_event"
)

// =============================================================================
// WebSocket Command Constants
// =============================================================================

type WsCmd string

const (
	CmdSubscribe         WsCmd = "aibot_subscribe"
	CmdHeartbeat         WsCmd = "ping"
	CmdResponse          WsCmd = "aibot_respond_msg"
	CmdResponseWelcome   WsCmd = "aibot_respond_welcome_msg"
	CmdResponseUpdate    WsCmd = "aibot_respond_update_msg"
	CmdSendMsg           WsCmd = "aibot_send_msg"
	CmdUploadMediaInit   WsCmd = "aibot_upload_media_init"
	CmdUploadMediaChunk  WsCmd = "aibot_upload_media_chunk"
	CmdUploadMediaFinish WsCmd = "aibot_upload_media_finish"
	CmdCallback          WsCmd = "aibot_msg_callback"
	CmdEventCallback     WsCmd = "aibot_event_callback"
)

// =============================================================================
// Template Card Type Constants
// =============================================================================

type TemplateCardType string

const (
	TemplateCardTypeTextNotice          TemplateCardType = "text_notice"
	TemplateCardTypeNewsNotice          TemplateCardType = "news_notice"
	TemplateCardTypeButtonInteraction   TemplateCardType = "button_interaction"
	TemplateCardTypeVoteInteraction     TemplateCardType = "vote_interaction"
	TemplateCardTypeMultipleInteraction TemplateCardType = "multiple_interaction"
)

// =============================================================================
// Message Sender Info
// =============================================================================

type MessageFrom struct {
	UserID string `json:"userid"`
}

type EventFrom struct {
	UserID string `json:"userid"`
	CorpID string `json:"corpid,omitempty"`
}

// =============================================================================
// Message Content Types
// =============================================================================

type TextContent struct {
	Content string `json:"content"`
}

type ImageContent struct {
	URL    string `json:"url"`
	AesKey string `json:"aeskey,omitempty"`
}

type VoiceContent struct {
	Content string `json:"content"`
}

type FileContent struct {
	URL    string `json:"url"`
	AesKey string `json:"aeskey,omitempty"`
}

type VideoContent struct {
	URL    string `json:"url"`
	AesKey string `json:"aeskey,omitempty"`
}

type MixedMsgItem struct {
	MsgType string        `json:"msgtype"`
	Text    *TextContent  `json:"text,omitempty"`
	Image   *ImageContent `json:"image,omitempty"`
}

type MixedContent struct {
	MsgItem []MixedMsgItem `json:"msg_item"`
}

type QuoteContent struct {
	MsgType string        `json:"msgtype"`
	Text    *TextContent  `json:"text,omitempty"`
	Image   *ImageContent `json:"image,omitempty"`
	Mixed   *MixedContent `json:"mixed,omitempty"`
	Voice   *VoiceContent `json:"voice,omitempty"`
	File    *FileContent  `json:"file,omitempty"`
}

// =============================================================================
// Concrete Message Types
// =============================================================================

type BaseMessage struct {
	MsgID       string        `json:"msgid"`
	AIBotID     string        `json:"aibotid"`
	ChatID      string        `json:"chatid,omitempty"`
	ChatType    string        `json:"chattype"`
	From        MessageFrom   `json:"from"`
	CreateTime  int64         `json:"create_time,omitempty"`
	ResponseURL string        `json:"response_url,omitempty"`
	MsgType     string        `json:"msgtype"`
	Quote       *QuoteContent `json:"quote,omitempty"`
}

type TextMessage struct {
	BaseMessage
	Text TextContent `json:"text"`
}

type ImageMessage struct {
	BaseMessage
	Image ImageContent `json:"image"`
}

type MixedMessage struct {
	BaseMessage
	Mixed MixedContent `json:"mixed"`
}

type VoiceMessage struct {
	BaseMessage
	Voice VoiceContent `json:"voice"`
}

type FileMessage struct {
	BaseMessage
	File FileContent `json:"file"`
}

type VideoMessage struct {
	BaseMessage
	Video VideoContent `json:"video"`
}

// =============================================================================
// Event Content Types
// =============================================================================

type EventContent struct {
	EventType string `json:"eventtype"`
	EventKey  string `json:"event_key,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
}

type EventMessage struct {
	MsgID      string       `json:"msgid"`
	CreateTime int64        `json:"create_time"`
	AIBotID    string       `json:"aibotid"`
	ChatID     string       `json:"chatid,omitempty"`
	ChatType   string       `json:"chattype,omitempty"`
	From       EventFrom    `json:"from"`
	MsgType    string       `json:"msgtype"`
	Event      EventContent `json:"event"`
}

// =============================================================================
// Handler Interface
// =============================================================================

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

func (BaseHandler) OnConnected()            {}
func (BaseHandler) OnAuthenticated()        {}
func (BaseHandler) OnDisconnected(string)   {}
func (BaseHandler) OnReconnecting(int)      {}
func (BaseHandler) OnError(error)           {}
func (BaseHandler) OnMessage(*WsFrame[any]) {}
func (BaseHandler) OnEvent(*WsFrame[any])   {}

// =============================================================================
// Template Card Types
// =============================================================================

type TemplateCardSource struct {
	IconURL   string `json:"icon_url,omitempty"`
	Desc      string `json:"desc,omitempty"`
	DescColor int    `json:"desc_color,omitempty"`
}

type TemplateCardActionMenuItem struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}

type TemplateCardActionMenu struct {
	Desc       string                       `json:"desc"`
	ActionList []TemplateCardActionMenuItem `json:"action_list"`
}

type TemplateCardMainTitle struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

type TemplateCardEmphasisContent struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

type TemplateCardQuoteArea struct {
	Type      int    `json:"type,omitempty"`
	URL       string `json:"url,omitempty"`
	AppID     string `json:"appid,omitempty"`
	PagePath  string `json:"pagepath,omitempty"`
	Title     string `json:"title,omitempty"`
	QuoteText string `json:"quote_text,omitempty"`
}

type TemplateCardHorizontalContent struct {
	Type    int    `json:"type,omitempty"`
	KeyName string `json:"keyname"`
	Value   string `json:"value,omitempty"`
	URL     string `json:"url,omitempty"`
	UserID  string `json:"userid,omitempty"`
}

type TemplateCardJumpAction struct {
	Type     int    `json:"type,omitempty"`
	Title    string `json:"title"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Question string `json:"question,omitempty"`
}

type TemplateCardAction struct {
	Type     int    `json:"type"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

type TemplateCardVerticalContent struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

type TemplateCardImage struct {
	URL         string  `json:"url"`
	AspectRatio float64 `json:"aspect_ratio,omitempty"`
}

type TemplateCardImageTextArea struct {
	Type     int    `json:"type,omitempty"`
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Title    string `json:"title,omitempty"`
	Desc     string `json:"desc,omitempty"`
	ImageURL string `json:"image_url"`
}

type TemplateCardSubmitButton struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}

type TemplateCardSelectionOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type TemplateCardSelectionItem struct {
	QuestionKey string                        `json:"question_key"`
	Title       string                        `json:"title,omitempty"`
	Disable     bool                          `json:"disable,omitempty"`
	SelectedID  string                        `json:"selected_id,omitempty"`
	OptionList  []TemplateCardSelectionOption `json:"option_list"`
}

type TemplateCardButton struct {
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
	Key   string `json:"key"`
}

type TemplateCardCheckboxOption struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsChecked bool   `json:"is_checked,omitempty"`
}

type TemplateCardCheckbox struct {
	QuestionKey string                       `json:"question_key"`
	Disable     bool                         `json:"disable,omitempty"`
	Mode        int                          `json:"mode,omitempty"`
	OptionList  []TemplateCardCheckboxOption `json:"option_list"`
}

type TemplateCard struct {
	CardType              string                          `json:"card_type"`
	Source                *TemplateCardSource             `json:"source,omitempty"`
	ActionMenu            *TemplateCardActionMenu         `json:"action_menu,omitempty"`
	MainTitle             *TemplateCardMainTitle          `json:"main_title,omitempty"`
	EmphasisContent       *TemplateCardEmphasisContent    `json:"emphasis_content,omitempty"`
	QuoteArea             *TemplateCardQuoteArea          `json:"quote_area,omitempty"`
	SubTitleText          string                          `json:"sub_title_text,omitempty"`
	HorizontalContentList []TemplateCardHorizontalContent `json:"horizontal_content_list,omitempty"`
	JumpList              []TemplateCardJumpAction        `json:"jump_list,omitempty"`
	CardAction            *TemplateCardAction             `json:"card_action,omitempty"`
	CardImage             *TemplateCardImage              `json:"card_image,omitempty"`
	ImageTextArea         *TemplateCardImageTextArea      `json:"image_text_area,omitempty"`
	VerticalContentList   []TemplateCardVerticalContent   `json:"vertical_content_list,omitempty"`
	ButtonSelection       *TemplateCardSelectionItem      `json:"button_selection,omitempty"`
	ButtonList            []TemplateCardButton            `json:"button_list,omitempty"`
	Checkbox              *TemplateCardCheckbox           `json:"checkbox,omitempty"`
	SelectList            []TemplateCardSelectionItem     `json:"select_list,omitempty"`
	SubmitButton          *TemplateCardSubmitButton       `json:"submit_button,omitempty"`
	TaskID                string                          `json:"task_id,omitempty"`
	Feedback              *ReplyFeedback                  `json:"feedback,omitempty"`
}

// =============================================================================
// Reply Types
// =============================================================================

type ReplyMsgItem struct {
	MsgType string         `json:"msgtype"`
	Image   *ReplyMsgImage `json:"image,omitempty"`
}

type ReplyMsgImage struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

type ReplyFeedback struct {
	ID string `json:"id"`
}

type StreamReplyBody struct {
	MsgType string             `json:"msgtype"`
	Stream  StreamReplyContent `json:"stream"`
}

type StreamReplyContent struct {
	ID       string         `json:"id"`
	Finish   bool           `json:"finish,omitempty"`
	Content  string         `json:"content,omitempty"`
	MsgItem  []ReplyMsgItem `json:"msg_item,omitempty"`
	Feedback *ReplyFeedback `json:"feedback,omitempty"`
}

// =============================================================================
// Welcome Reply Types
// =============================================================================

type WelcomeTextReplyBody struct {
	MsgType string      `json:"msgtype"`
	Text    TextContent `json:"text"`
}

type WelcomeTemplateCardReplyBody struct {
	MsgType      string       `json:"msgtype"`
	TemplateCard TemplateCard `json:"template_card"`
}

// =============================================================================
// Send Message Types
// =============================================================================

type SendMarkdownMsgBody struct {
	MsgType  string          `json:"msgtype"`
	Markdown MarkdownContent `json:"markdown"`
}

type MarkdownContent struct {
	Content string `json:"content"`
}

type SendTemplateCardMsgBody struct {
	MsgType      string       `json:"msgtype"`
	TemplateCard TemplateCard `json:"template_card"`
}

type SendMediaMsgBody struct {
	MsgType string             `json:"msgtype"`
	File    *MediaContent      `json:"file,omitempty"`
	Image   *MediaContent      `json:"image,omitempty"`
	Voice   *MediaContent      `json:"voice,omitempty"`
	Video   *VideoMediaContent `json:"video,omitempty"`
}

type MediaContent struct {
	MediaID string `json:"media_id"`
}

type VideoMediaContent struct {
	MediaID     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// =============================================================================
// Media Upload Types
// =============================================================================

type WeComMediaType string

const (
	MediaTypeFile  WeComMediaType = "file"
	MediaTypeImage WeComMediaType = "image"
	MediaTypeVoice WeComMediaType = "voice"
	MediaTypeVideo WeComMediaType = "video"
)

type UploadMediaOptions struct {
	Type     WeComMediaType
	Filename string
}

type UploadMediaFinishResult struct {
	Type      WeComMediaType `json:"type"`
	MediaID   string         `json:"media_id"`
	CreatedAt string         `json:"created_at"`
}

// =============================================================================
// Stream with Template Card Reply
// =============================================================================

type StreamWithTemplateCardReplyBody struct {
	MsgType      string             `json:"msgtype"`
	Stream       StreamReplyContent `json:"stream"`
	TemplateCard *TemplateCard      `json:"template_card,omitempty"`
}

// =============================================================================
// Update Template Card Body
// =============================================================================

type UpdateTemplateCardBody struct {
	ResponseType string       `json:"response_type"`
	UserIDs      []string     `json:"userids,omitempty"`
	TemplateCard TemplateCard `json:"template_card"`
}

// =============================================================================
// Template Card Reply Body
// =============================================================================

type TemplateCardReplyBody struct {
	MsgType      string       `json:"msgtype"`
	TemplateCard TemplateCard `json:"template_card"`
}

// =============================================================================
// Client Options
// =============================================================================

type Config struct {
	BotID                  string
	Secret                 string
	Scene                  int
	PlugVersion            string
	ReconnectInterval      time.Duration
	MaxReconnectAttempts   int
	MaxAuthFailureAttempts int
	HeartbeatInterval      time.Duration
	RequestTimeout         time.Duration
	WsURL                  string
	MaxReplyQueueSize      int
	Handler                Handler
}

type ClientOption func(*Config)

func WithWsURL(url string) ClientOption {
	return func(c *Config) {
		c.WsURL = url
	}
}

func WithHeartbeatInterval(d time.Duration) ClientOption {
	return func(c *Config) {
		c.HeartbeatInterval = d
	}
}

func WithMaxReconnectAttempts(n int) ClientOption {
	return func(c *Config) {
		c.MaxReconnectAttempts = n
	}
}

func WithMaxReplyQueueSize(n int) ClientOption {
	return func(c *Config) {
		c.MaxReplyQueueSize = n
	}
}

func WithScene(n int) ClientOption {
	return func(c *Config) {
		c.Scene = n
	}
}

func WithPlugVersion(v string) ClientOption {
	return func(c *Config) {
		c.PlugVersion = v
	}
}

func WithMaxAuthFailureAttempts(n int) ClientOption {
	return func(c *Config) {
		c.MaxAuthFailureAttempts = n
	}
}

func WithReconnectInterval(d time.Duration) ClientOption {
	return func(c *Config) {
		c.ReconnectInterval = d
	}
}

func WithRequestTimeout(d time.Duration) ClientOption {
	return func(c *Config) {
		c.RequestTimeout = d
	}
}

func WithHandler(h Handler) ClientOption {
	return func(c *Config) {
		c.Handler = h
	}
}
