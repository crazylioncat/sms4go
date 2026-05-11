package wetalk

import (
	"context"
	"time"

	"github.com/CrazyLionCat/sms4go/oa"
	"github.com/CrazyLionCat/sms4go/oa/provider"
)

const Supplier = oa.WeTalk

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
	body, err := message(request, messageType)
	if err != nil {
		return nil, err
	}
	endpoint := "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + s.config.TokenID
	raw, status, err := s.http.PostJSON(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	return &oa.Response{Success: status >= 200 && status < 300, Data: string(raw), ConfigID: s.ConfigID(), StatusCode: status}, nil
}

func message(request oa.Request, messageType oa.MessageType) (map[string]any, error) {
	switch messageType {
	case oa.WeTalkText:
		if request.Content == "" {
			return nil, oa.ErrEmptyContent
		}
		text := map[string]any{"content": request.Content}
		if request.NoticeAll && len(request.UserIDs) == 0 && len(request.Phones) == 0 {
			text["mentioned_list"] = []string{"@all"}
		}
		if len(request.UserIDs) > 0 {
			users := append([]string{}, request.UserIDs...)
			if request.NoticeAll {
				users = append(users, "@all")
			}
			text["mentioned_list"] = users
		}
		if len(request.Phones) > 0 {
			text["mentioned_mobile_list"] = request.Phones
		}
		return map[string]any{"msgtype": "text", "text": text}, nil
	case oa.WeTalkMarkdown:
		if request.Content == "" {
			return nil, oa.ErrEmptyContent
		}
		content := ""
		for _, userID := range request.UserIDs {
			content += "<@" + userID + ">"
		}
		if content != "" {
			content += "\n"
		}
		content += request.Content
		return map[string]any{"msgtype": "markdown", "markdown": map[string]any{"content": content, "title": request.Title}}, nil
	case oa.WeTalkNews:
		return map[string]any{"msgtype": "news", "news": map[string]any{"articles": request.Articles}}, nil
	default:
		return nil, oa.ErrMessageType
	}
}
