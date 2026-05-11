package mas

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "mas"

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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	data, _ := json.Marshal(base.ValuesBySortedKey(messages))
	return c.doSend(ctx, phone, string(data), templateID)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.doSend(ctx, strings.Join(phones, ","), message, c.config.TemplateID)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	data, _ := json.Marshal(base.ValuesBySortedKey(messages))
	return c.doSend(ctx, strings.Join(phones, ","), string(data), templateID)
}

func (c *Client) doSend(ctx context.Context, phone string, message string, templateID string) (*sms.Response, error) {
	encoded, err := c.encodedBody(phone, message, templateID)
	if err != nil {
		return nil, err
	}
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostRaw(ctx, c.config.RequestURL+c.config.Action, []byte(encoded), "application/json", http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if rsp, _ := data["rspcod"].(string); rsp == "success" {
			if ok, _ := data["success"].(bool); ok {
				last.Success = true
				return last, nil
			}
		}
	}
	return last, lastErr
}

func (c *Client) encodedBody(phone string, message string, templateID string) (string, error) {
	body := map[string]string{}
	var signBuilder strings.Builder
	add := func(key string, value string, includeEmpty bool) {
		if value != "" || includeEmpty {
			body[key] = value
		}
		if value != "" {
			signBuilder.WriteString(value)
		}
	}
	add("ecName", strings.TrimSpace(c.config.ECName), false)
	add("apId", strings.TrimSpace(c.config.SDKAppID), false)
	add("secretKey", strings.TrimSpace(c.config.AccessKeySecret), false)
	if c.config.Action == "tmpsubmit" {
		add("templateId", strings.TrimSpace(templateID), false)
		add("mobiles", strings.TrimSpace(phone), false)
		if strings.TrimSpace(message) == "" {
			body["params"] = `[""]`
			signBuilder.WriteString(`[""]`)
		} else {
			add("params", strings.TrimSpace(message), false)
		}
	} else {
		add("mobiles", strings.TrimSpace(phone), false)
		add("content", strings.TrimSpace(message), false)
	}
	add("sign", strings.TrimSpace(c.config.Signature), false)
	if c.config.AddSerial != "" {
		add("addSerial", strings.TrimSpace(c.config.AddSerial), false)
	} else {
		body["addSerial"] = ""
	}
	body["mac"] = md5Hex(signBuilder.String())
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
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

func withDefaults(config sms.BaseConfig) sms.BaseConfig {
	config.Supplier = Supplier
	if config.RequestURL == "" {
		config.RequestURL = "http://112.35.1.155:1992/sms/"
	}
	if config.Action == "" {
		config.Action = "tmpsubmit"
	}
	return config
}
