package jiguang

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "jiguang"

type Client struct {
	config sms.BaseConfig
	http   *base.HTTPClient
}

func init() {
	provider.Register(Supplier, func(config sms.BaseConfig) (sms.Client, error) {
		return New(config)
	})
}

func New(config sms.BaseConfig) (*Client, error) {
	config = withDefaults(config)
	httpClient, err := base.NewHTTPClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{config: config, http: httpClient}, nil
}

func (c *Client) ConfigID() string { return c.config.ID() }

func (c *Client) Supplier() string { return Supplier }

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	messages := map[string]string{}
	if c.config.TemplateName != "" && message != "" {
		messages[c.config.TemplateName] = message
	}
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	body, err := c.body(phone, templateID, messages, "", "")
	if err != nil {
		return nil, err
	}
	return c.doPost(ctx, c.url(""), body, jsonKey(c.config.Action))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	messages := map[string]string{}
	if c.config.TemplateName != "" && message != "" {
		messages[c.config.TemplateName] = message
	}
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, messages)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	body, err := c.body(strings.Join(prefixPhones(phones), ","), templateID, messages, "", "")
	if err != nil {
		return nil, err
	}
	return c.doPost(ctx, c.url(""), body, jsonKey(c.config.Action))
}

func (c *Client) SendVoiceCode(ctx context.Context, phone string, code string) (*sms.Response, error) {
	body, err := c.body(phone, "", nil, code, "")
	if err != nil {
		return nil, err
	}
	return c.doPost(ctx, c.url(""), body, jsonKey(c.config.Action))
}

func (c *Client) VerifyCode(ctx context.Context, code string, msgID string) (*sms.Response, error) {
	body, err := c.body("", "", nil, code, msgID)
	if err != nil {
		return nil, err
	}
	return c.doPost(ctx, c.url(msgID), body, "is_valid")
}

func (c *Client) body(phone string, templateID string, messages map[string]string, code string, msgID string) (map[string]any, error) {
	switch c.config.Action {
	case "codes":
		if phone == "" || templateID == "" {
			return nil, errors.New("sms4go: jiguang phone and template id are required")
		}
		body := map[string]any{"mobile": phone, "temp_id": templateID}
		if c.config.SignID != "" {
			body["sign_id"] = c.config.SignID
		}
		return body, nil
	case "voice_codes":
		if phone == "" {
			return nil, sms.ErrInvalidPhone
		}
		ttl := c.config.TTL
		if ttl <= 0 {
			ttl = 60
		}
		body := map[string]any{"mobile": phone, "ttl": ttl}
		if code != "" {
			body["code"] = code
		}
		if c.config.Voice != "" {
			body["voice_lang"] = c.config.Voice
		}
		return body, nil
	case "valid":
		if code == "" || msgID == "" {
			return nil, errors.New("sms4go: jiguang code and msg id are required")
		}
		return map[string]any{"code": code}, nil
	case "messages/batch":
		recipients := make([]map[string]any, 0)
		for _, mobile := range strings.Split(phone, ",") {
			if strings.TrimSpace(mobile) != "" {
				recipients = append(recipients, map[string]any{"mobile": prefix86(strings.TrimSpace(mobile)), "temp_para": messages})
			}
		}
		body := map[string]any{"temp_id": templateID, "recipients": recipients}
		if c.config.SignID != "" {
			body["sign_id"] = c.config.SignID
		}
		if c.config.Tag != "" {
			body["tag"] = c.config.Tag
		}
		return body, nil
	case "messages":
		body := map[string]any{"mobile": phone, "temp_id": templateID, "temp_para": messages}
		if c.config.SignID != "" {
			body["sign_id"] = c.config.SignID
		}
		return body, nil
	default:
		return nil, errors.New("sms4go: unsupported jiguang action")
	}
}

func (c *Client) doPost(ctx context.Context, endpoint string, payload map[string]any, successKey string) (*sms.Response, error) {
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, endpoint, payload, c.headers())
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if _, ok := data[successKey]; ok {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) url(msgID string) string {
	if c.config.Action == "valid" {
		return c.config.RequestURL + "codes/" + msgID + "/valid"
	}
	return c.config.RequestURL + c.config.Action
}

func (c *Client) headers() http.Header {
	token := base64.StdEncoding.EncodeToString([]byte(c.config.AccessKeyID + ":" + c.config.AccessKeySecret))
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("Authorization", "Basic "+token)
	return headers
}

func jsonKey(action string) string {
	switch action {
	case "valid":
		return "is_valid"
	case "messages/batch":
		return "success_count"
	default:
		return "msg_id"
	}
}

func prefixPhones(phones []string) []string {
	result := make([]string, 0, len(phones))
	for _, phone := range phones {
		result = append(result, prefix86(phone))
	}
	return result
}

func prefix86(phone string) string {
	if strings.HasPrefix(phone, "+86") {
		return phone
	}
	return "+86" + phone
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}

func withDefaults(config sms.BaseConfig) sms.BaseConfig {
	config.Supplier = Supplier
	if config.RequestURL == "" {
		config.RequestURL = "https://api.sms.jpush.cn/v1/"
	}
	if config.Action == "" {
		config.Action = "messages"
	}
	if config.TTL <= 0 {
		config.TTL = 60
	}
	return config
}
