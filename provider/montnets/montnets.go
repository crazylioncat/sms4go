package montnets

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "montnets"

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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, map[string]string{c.config.TemplateParam: message})
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, phone, templateID, formatMessages(messages))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{c.config.TemplateParam: message})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	prefixed := make([]string, 0, len(phones))
	for _, phone := range phones {
		if strings.HasPrefix(phone, "+86") {
			prefixed = append(prefixed, phone)
		} else {
			prefixed = append(prefixed, "+86"+phone)
		}
	}
	return c.doSend(ctx, strings.Join(prefixed, ","), templateID, formatMessages(messages))
}

func (c *Client) doSend(ctx context.Context, phone string, templateID string, content string) (*sms.Response, error) {
	payload, err := json.Marshal(c.body(phone, templateID, content))
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json; charset=utf-8")

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostRaw(ctx, c.config.URL+c.config.API, payload, "application/json; charset=utf-8", headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		desc, _ := data["desc"].(string)
		if decoded, err := url.QueryUnescape(desc); err == nil {
			desc = decoded
		}
		last = sms.Failure(desc, c.ConfigID())
		if result, _ := data["result"].(string); result == "0" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) body(phone string, templateID string, content string) map[string]string {
	timestamp := time.Now().In(time.FixedZone("GMT+8", 8*60*60)).Format("0102150405")
	userID := c.config.AccessKeyID
	return map[string]string{
		"userid":    userID,
		"pwd":       md5Lower(strings.ToUpper(userID) + "00000000" + c.config.AccessKeySecret + timestamp),
		"timestamp": timestamp,
		"mobile":    phone,
		"content":   content,
		"tmplid":    templateID,
	}
}

func formatMessages(messages map[string]string) string {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+url.QueryEscape(messages[key]))
	}
	return url.QueryEscape(strings.Join(parts, "&"))
}

func md5Lower(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}
