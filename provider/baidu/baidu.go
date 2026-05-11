package baidu

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
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

const Supplier = "baidu"

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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, phone, templateID, messages, "")
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	prefixed := make([]string, 0, len(phones))
	for _, phone := range phones {
		prefixed = append(prefixed, prefix86(phone))
	}
	return c.doSend(ctx, strings.Join(prefixed, ","), templateID, messages, "")
}

func (c *Client) SendTemplateWithClientToken(ctx context.Context, phone string, templateID string, messages map[string]string, clientToken string) (*sms.Response, error) {
	if strings.TrimSpace(clientToken) == "" {
		return nil, errors.New("sms4go: baidu clientToken is required")
	}
	return c.doSend(ctx, phone, templateID, messages, clientToken)
}

func (c *Client) doSend(ctx context.Context, mobile string, templateID string, messages map[string]string, clientToken string) (*sms.Response, error) {
	body := map[string]any{
		"mobile":      mobile,
		"template":    templateID,
		"signatureId": c.config.Signature,
		"contentVar":  messages,
	}
	if c.config.Custom != "" {
		body["custom"] = c.config.Custom
	}
	if c.config.UserExtID != "" {
		body["userExtId"] = c.config.UserExtID
	}
	endpoint := c.config.Host + c.config.Action
	if clientToken != "" {
		endpoint += "?clientToken=" + url.QueryEscape(clientToken)
	}

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		headers := c.headers(clientToken)
		raw, _, err := c.http.PostJSON(ctx, endpoint, body, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); code == "1000" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) headers(clientToken string) http.Header {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	host := strings.TrimPrefix(strings.TrimPrefix(c.config.Host, "https://"), "http://")
	authPrefix := "bce-auth-v1/" + c.config.AccessKeyID + "/" + now + "/1800"
	signingKey := hmacSHA256Hex(c.config.AccessKeySecret, authPrefix)
	canonical := "POST\n" + url.QueryEscape(c.config.Action) + "\n" + canonicalQuery(clientToken) + "\nhost:" + url.QueryEscape(host)
	signature := hmacSHA256Hex(signingKey, canonical)
	headers := http.Header{}
	headers.Set("Authorization", authPrefix+"//"+signature)
	headers.Set("Host", host)
	headers.Set("x-bce-date", now)
	return headers
}

func canonicalQuery(clientToken string) string {
	if clientToken == "" {
		return ""
	}
	return "clientToken=" + url.QueryEscape(clientToken)
}

func hmacSHA256Hex(key string, text string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(text))
	return hex.EncodeToString(mac.Sum(nil))
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
	if config.Host == "" {
		config.Host = "https://smsv3.bj.baidubce.com"
	}
	if config.Action == "" {
		config.Action = "/api/v3/sendSms"
	}
	return config
}
