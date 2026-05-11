package oa

import "context"

type MessageType string

const (
	DingTalkText     MessageType = "DING_TALK_TEXT"
	DingTalkMarkdown MessageType = "DING_TALK_MARKDOWN"
	DingTalkLink     MessageType = "DING_TALK_LINK"
	ByteTalkText     MessageType = "BYTE_TALK_TEXT"
	WeTalkText       MessageType = "WE_TALK_TEXT"
	WeTalkMarkdown   MessageType = "WE_TALK_MARKDOWN"
	WeTalkNews       MessageType = "WE_TALK_NEWS"
)

const (
	DingTalk = "ding_ding"
	WeTalk   = "we_talk"
	ByteTalk = "byte_talk"
)

type Article struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	PicURL      string `json:"picurl,omitempty"`
}

type Request struct {
	Title       string
	Content     string
	PicURL      string
	MessageURL  string
	Articles    []Article
	Phones      []string
	UserIDs     []string
	UserNames   []string
	NoticeAll   bool
	OAType      string
	Priority    int
	MessageType MessageType
}

type Response struct {
	Success    bool   `json:"success"`
	Data       any    `json:"data,omitempty"`
	ConfigID   string `json:"oaConfigId,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
}

type Config struct {
	ConfigID string `json:"configId" yaml:"configId"`
	Supplier string `json:"supplier" yaml:"supplier"`
	TokenID  string `json:"tokenId" yaml:"tokenId"`
	Sign     string `json:"sign" yaml:"sign"`
	Enable   bool   `json:"enable" yaml:"enable"`
}

func (c Config) ID() string {
	if c.ConfigID != "" {
		return c.ConfigID
	}
	return c.Supplier
}

type Sender interface {
	ConfigID() string
	Supplier() string
	Send(ctx context.Context, request Request, messageType MessageType) (*Response, error)
}

type Callback func(*Response, error)
