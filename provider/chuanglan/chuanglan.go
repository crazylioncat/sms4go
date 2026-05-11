package chuanglan

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "chuanglan"

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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, base.ParamsByAmpersand(message))
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	payload := c.basePayload(templateID)
	payload["params"] = phone + "," + joinValues(messages)
	return c.doPost(ctx, payload)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, base.ParamsByAmpersand(message))
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	message := joinValues(messages)
	var builder strings.Builder
	for _, phone := range phones {
		builder.WriteString(phone)
		builder.WriteByte(',')
		builder.WriteString(message)
		builder.WriteByte(';')
	}
	payload := c.basePayload(templateID)
	payload["params"] = builder.String()
	return c.doPost(ctx, payload)
}

func (c *Client) basePayload(templateID string) map[string]any {
	return map[string]any{
		"account":  c.config.AccessKeyID,
		"password": c.config.AccessKeySecret,
		"msg":      templateID,
	}
}

func (c *Client) doPost(ctx context.Context, payload map[string]any) (*sms.Response, error) {
	endpoint := c.config.BaseURL + c.config.SingleMsgURL
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, endpoint, payload, http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); code == "0" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func joinValues(messages map[string]string) string {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, messages[key])
	}
	return strings.Join(values, ",")
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}

func withDefaults(config sms.BaseConfig) sms.BaseConfig {
	config.Supplier = Supplier
	if config.BaseURL == "" {
		config.BaseURL = "https://smssh1.253.com/msg"
	}
	if config.SingleMsgURL == "" {
		config.SingleMsgURL = "/variable/json"
	}
	return config
}
