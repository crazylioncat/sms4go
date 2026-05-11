package yunpian

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "yunpian"

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

func (c *Client) ConfigID() string {
	return c.config.ID()
}

func (c *Client) Supplier() string {
	return Supplier
}

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	messages := map[string]string{c.config.TemplateName: message}
	return c.send(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.send(ctx, phone, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	if len(phones) > 1000 {
		return nil, errors.New("sms4go: yunpian mass texting limit is 1000 phones")
	}
	return c.SendMessage(ctx, strings.Join(phones, ","), message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	if len(phones) > 1000 {
		return nil, errors.New("sms4go: yunpian mass texting limit is 1000 phones")
	}
	return c.SendTemplate(ctx, strings.Join(phones, ","), templateID, messages)
}

func (c *Client) send(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	values := url.Values{}
	values.Set("apikey", c.config.AccessKeyID)
	values.Set("mobile", phone)
	values.Set("tpl_id", templateID)
	values.Set("tpl_value", formatTemplateValue(messages))
	if c.config.CallbackURL != "" {
		values.Set("callback_url", c.config.CallbackURL)
	}
	headers := http.Header{}
	headers.Set("Accept", "application/json; charset=utf-8")

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.BaseURL+"/sms/tpl_single_send.json", values, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, ok := numericCode(data["code"]); ok && code == 0 {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func formatTemplateValue(messages map[string]string) string {
	if len(messages) == 0 {
		return ""
	}
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, "#"+key+"#="+messages[key])
	}
	return strings.Join(parts, "&")
}

func numericCode(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
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
		config.BaseURL = "https://sms.yunpian.com/v2"
	}
	return config
}
