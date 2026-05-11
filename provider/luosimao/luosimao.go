package luosimao

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "luosimao"

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
	return c.send(ctx, []string{phone}, message, nil, c.config.Action)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: luosimao does not support SendMessageWithParams")
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: luosimao does not support SendTemplate")
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.send(ctx, phones, message, nil, "send_batch.json")
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: luosimao does not support MassTextingTemplate")
}

func (c *Client) MassTextingOnTime(ctx context.Context, phones []string, message string, sendAt time.Time) (*sms.Response, error) {
	return c.send(ctx, phones, message, &sendAt, "send_batch.json")
}

func (c *Client) QueryAccountBalance(ctx context.Context) (*sms.Response, error) {
	headers := c.headers()
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.Get(ctx, c.config.Host+"status.json", headers)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := c.response(raw)
		if err != nil {
			return nil, err
		}
		if resp.Success {
			return resp, nil
		}
		last = resp
	}
	return last, lastErr
}

func (c *Client) send(ctx context.Context, phones []string, message string, sendAt *time.Time, action string) (*sms.Response, error) {
	if len(phones) == 0 {
		return nil, sms.ErrInvalidPhone
	}
	if strings.TrimSpace(message) == "" {
		return nil, sms.ErrEmptyMessage
	}
	values := url.Values{}
	values.Set("mobile", strings.Join(phones, ","))
	values.Set("message", message)
	if sendAt != nil {
		values.Set("time", sendAt.In(time.FixedZone("GMT+8", 8*60*60)).Format("2006-01-02 15:04:05"))
	}
	headers := c.headers()

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.Host+action, values, headers)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := c.response(raw)
		if err != nil {
			return nil, err
		}
		if resp.Success {
			return resp, nil
		}
		last = resp
	}
	return last, lastErr
}

func (c *Client) response(raw []byte) (*sms.Response, error) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	resp := sms.Failure(data, c.ConfigID())
	switch value := data["error"].(type) {
	case float64:
		resp.Success = value == 0
	case int:
		resp.Success = value == 0
	}
	return resp, nil
}

func (c *Client) headers() http.Header {
	token := base64.StdEncoding.EncodeToString([]byte("api:key-" + c.config.AccessKeyID))
	headers := http.Header{}
	headers.Set("Authorization", "Basic "+token)
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return headers
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}

func withDefaults(config sms.BaseConfig) sms.BaseConfig {
	config.Supplier = Supplier
	if config.Host == "" {
		config.Host = "https://sms-api.luosimao.com/v1/"
	}
	if !strings.HasSuffix(config.Host, "/") {
		config.Host += "/"
	}
	if config.Action == "" {
		config.Action = "send.json"
	}
	return config
}
