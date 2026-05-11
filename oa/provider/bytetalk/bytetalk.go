package bytetalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/CrazyLionCat/sms4go/oa"
	"github.com/CrazyLionCat/sms4go/oa/provider"
)

const Supplier = oa.ByteTalk

type Sender struct {
	config oa.Config
	http   *oa.HTTPClient
}

func init() {
	provider.Register(Supplier, func(config oa.Config) (oa.Sender, error) {
		return New(config), nil
	})
}

func New(config oa.Config) *Sender {
	config.Supplier = Supplier
	return &Sender{config: config, http: oa.NewHTTPClient(10 * time.Second)}
}

func (s *Sender) ConfigID() string { return s.config.ID() }

func (s *Sender) Supplier() string { return Supplier }

func (s *Sender) Send(ctx context.Context, request oa.Request, messageType oa.MessageType) (*oa.Response, error) {
	if request.Content == "" {
		return nil, oa.ErrEmptyContent
	}
	if messageType != oa.ByteTalkText {
		return nil, oa.ErrMessageType
	}
	timestamp := time.Now().Unix()
	body := map[string]any{
		"msg_type":  "text",
		"timestamp": timestamp,
		"sign":      sign(s.config.Sign, timestamp),
		"content":   map[string]any{"text": content(request)},
	}
	endpoint := "https://open.feishu.cn/open-apis/bot/v2/hook/" + s.config.TokenID
	raw, status, err := s.http.PostJSON(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	return &oa.Response{Success: status >= 200 && status < 300, Data: string(raw), ConfigID: s.ConfigID(), StatusCode: status}, nil
}

func content(request oa.Request) string {
	prefix := ""
	if request.NoticeAll {
		prefix += `<at user_id="all">所有人</at>`
	}
	for _, userID := range request.UserIDs {
		prefix += `<at user_id="` + userID + `"></at>`
	}
	if prefix != "" {
		prefix += "\n"
	}
	return prefix + request.Content
}

func sign(secret string, timestamp int64) string {
	stringToSign := strconv.FormatInt(timestamp, 10) + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	mac.Write([]byte{})
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
