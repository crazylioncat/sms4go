package yixintong

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "yixintong"

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
	return c.doSend(ctx, phone, message, c.config.TemplateID)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: yixintong does not support SendMessageWithParams")
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: yixintong does not support SendTemplate")
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.doSend(ctx, strings.Join(phones, ","), message, c.config.TemplateID)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: yixintong does not support MassTextingTemplate")
}

func (c *Client) doSend(ctx context.Context, phone string, message string, templateID string) (*sms.Response, error) {
	values := url.Values{}
	values.Set("SpCode", c.config.SpCode)
	values.Set("LoginName", c.config.AccessKeyID)
	values.Set("Password", c.config.AccessKeySecret)
	values.Set("MessageContent", message)
	values.Set("UserNumber", phone)
	values.Set("templateId", templateID)
	values.Set("SerialNumber", randomDigits(20))
	values.Set("ScheduleTime", "")
	values.Set("f", c.config.F)
	values.Set("signCode", c.config.SignCode)

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.RequestURL, values, http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		body := string(raw)
		last = sms.Failure(body, c.ConfigID())
		if strings.Contains(body, "result=0&") {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func randomDigits(length int) string {
	var builder strings.Builder
	for builder.Len() < length {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			builder.WriteByte('0')
			continue
		}
		builder.WriteString(n.String())
	}
	return builder.String()
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
		config.RequestURL = "https://api.ums86.com:9600/sms/Api/Send.do"
	}
	if config.F == "" {
		config.F = "1"
	}
	return config
}
