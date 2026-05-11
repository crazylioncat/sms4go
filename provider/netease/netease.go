package netease

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "netease"

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
	return c.doSend(ctx, []string{phone}, c.config.TemplateID, message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	message := messages["params"]
	if message == "" {
		message = messages["message"]
	}
	return c.doSend(ctx, []string{phone}, templateID, message)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	if len(phones) > 100 {
		return nil, errors.New("sms4go: netease mass texting limit is 100 phones")
	}
	return c.doSend(ctx, phones, c.config.TemplateID, message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	if len(phones) > 100 {
		return nil, errors.New("sms4go: netease mass texting limit is 100 phones")
	}
	message := messages["message"]
	if message == "" {
		message = messages["params"]
	}
	return c.doSend(ctx, phones, templateID, message)
}

func (c *Client) doSend(ctx context.Context, phones []string, templateID string, message string) (*sms.Response, error) {
	mobiles, _ := json.Marshal(phones)
	params := message
	if !strings.HasPrefix(strings.TrimSpace(message), "[") {
		data, _ := json.Marshal([]string{message})
		params = string(data)
	}
	values := url.Values{}
	values.Set("templateid", templateID)
	values.Set("mobiles", string(mobiles))
	values.Set("params", params)
	values.Set("needUp", strconv.FormatBool(c.config.NeedUp))

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.TemplateURL, values, c.headers())
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, ok := numericCode(data["code"]); ok && code <= 200 {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) headers() http.Header {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 36)
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	headers := http.Header{}
	headers.Set("AppKey", c.config.AccessKeyID)
	headers.Set("Nonce", nonce)
	headers.Set("CurTime", curTime)
	headers.Set("CheckSum", sha1Hex(c.config.AccessKeySecret+nonce+curTime))
	return headers
}

func sha1Hex(text string) string {
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
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
	if config.TemplateURL == "" {
		config.TemplateURL = "https://api.netease.im/sms/sendtemplate.action"
	}
	if config.CodeURL == "" {
		config.CodeURL = "https://api.netease.im/sms/sendcode.action"
	}
	if config.VerifyURL == "" {
		config.VerifyURL = "https://api.netease.im/sms/verifycode.action"
	}
	return config
}
