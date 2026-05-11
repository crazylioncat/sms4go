package emay

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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

const Supplier = "emay"

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
	config.Supplier = Supplier
	httpClient, err := base.NewHTTPClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{config: config, http: httpClient}, nil
}

func (c *Client) ConfigID() string { return c.config.ID() }

func (c *Client) Supplier() string { return Supplier }

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	return c.doSend(ctx, phone, message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, phone, jsonArrayString(base.ValuesBySortedKey(messages)))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	if len(phones) > 500 {
		return nil, errors.New("sms4go: emay mass texting limit is 500 phones")
	}
	return c.doSend(ctx, strings.Join(phones, ","), message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	if len(phones) > 500 {
		return nil, errors.New("sms4go: emay mass texting limit is 500 phones")
	}
	return c.doSend(ctx, strings.Join(phones, ","), jsonArrayString(base.ValuesBySortedKey(messages)))
}

func (c *Client) doSend(ctx context.Context, phone string, message string) (*sms.Response, error) {
	timestamp := time.Now().Format("20060102150405")
	values := url.Values{}
	values.Set("appId", c.config.AccessKeyID)
	values.Set("timestamp", timestamp)
	values.Set("sign", md5Hex(c.config.AccessKeyID+c.config.AccessKeySecret+timestamp))
	values.Set("mobiles", phone)
	values.Set("content", url.QueryEscape(message))

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
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); strings.EqualFold(code, "success") {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func jsonArrayString(values []string) string {
	data, _ := json.Marshal(values)
	return string(data)
}

func md5Hex(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}
