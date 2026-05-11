package budingv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "buding_v2"

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
	return c.send(ctx, phone, message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.sendMany(ctx, []string{phone}, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.SendMessageWithParams(ctx, phone, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	failed := 0
	for _, phone := range phones {
		resp, err := c.send(ctx, phone, message)
		if err != nil || resp == nil || !resp.Success {
			failed++
		}
	}
	return sms.Success(map[string]any{"failed": failed}, c.ConfigID()), nil
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.sendMany(ctx, phones, messages)
}

func (c *Client) sendMany(ctx context.Context, phones []string, messages map[string]string) (*sms.Response, error) {
	responses := make([]*sms.Response, 0)
	for _, phone := range phones {
		for _, message := range messages {
			resp, err := c.send(ctx, phone, message)
			if err != nil {
				return nil, err
			}
			responses = append(responses, resp)
		}
	}
	return sms.Success(responses, c.ConfigID()), nil
}

func (c *Client) send(ctx context.Context, phone string, message string) (*sms.Response, error) {
	if c.config.SignKey == "" && c.config.Signature == "" {
		return nil, errors.New("sms4go: buding_v2 signature key is empty")
	}
	values := url.Values{}
	values.Set("key", c.config.AccessKeyID)
	values.Set("to", phone)
	values.Set("content", message)
	if c.config.SignKey == "" {
		values.Set("sign", c.config.Signature)
	}
	headers := http.Header{}
	headers.Set("Accept", "application/json; charset=utf-8")

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.BaseURL+"/Api/Sent", values, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if ok, _ := data["bool"].(bool); ok {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
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
		config.BaseURL = "https://smsapi.idcbdy.com"
	}
	return config
}
