package dingtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strconv"
	"time"

	"github.com/CrazyLionCat/sms4go/oa"
	"github.com/CrazyLionCat/sms4go/oa/provider"
)

const Supplier = oa.DingTalk

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
	body, err := message(request, messageType)
	if err != nil {
		return nil, err
	}
	endpoint := "https://oapi.dingtalk.com/robot/send?access_token=" + s.config.TokenID
	if s.config.Sign != "" {
		endpoint += signQuery(s.config.Sign)
	}
	raw, status, err := s.http.PostJSON(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	return &oa.Response{Success: status >= 200 && status < 300, Data: string(raw), ConfigID: s.ConfigID(), StatusCode: status}, nil
}

func message(request oa.Request, messageType oa.MessageType) (map[string]any, error) {
	body := map[string]any{}
	switch messageType {
	case oa.DingTalkText:
		body["msgtype"] = "text"
		body["text"] = map[string]any{"content": request.Content}
	case oa.DingTalkMarkdown:
		body["msgtype"] = "markdown"
		body["markdown"] = map[string]any{"text": request.Content, "title": request.Title}
	case oa.DingTalkLink:
		body["msgtype"] = "link"
		body["link"] = map[string]any{"text": request.Content, "title": request.Title, "picUrl": request.PicURL, "messageUrl": request.MessageURL}
	default:
		return nil, oa.ErrMessageType
	}
	at := map[string]any{"isAtAll": request.NoticeAll}
	if len(request.Phones) > 0 {
		at["atMobiles"] = request.Phones
	}
	if len(request.UserIDs) > 0 {
		at["atUserIds"] = request.UserIDs
	}
	body["at"] = at
	return body, nil
}

func signQuery(secret string) string {
	timestamp := time.Now().UnixMilli()
	text := strconv.FormatInt(timestamp, 10) + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(text))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	return "&timestamp=" + strconv.FormatInt(timestamp, 10) + "&sign=" + sign
}
