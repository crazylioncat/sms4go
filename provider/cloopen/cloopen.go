package cloopen

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

const Supplier = "cloopen"

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
	return c.MassTexting(ctx, []string{phone}, message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, []string{phone}, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{"0": message})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	datas := base.ValuesBySortedKey(messages)
	payload := map[string]any{
		"to":         strings.Join(phones, ","),
		"appId":      c.config.SDKAppID,
		"templateId": templateID,
		"datas":      datas,
	}
	timestamp := time.Now().Format("20060102150405")
	sign := strings.ToUpper(md5Hex(c.config.AccessKeyID + c.config.AccessKeySecret + timestamp))
	endpoint := c.config.BaseURL + "/Accounts/" + c.config.AccessKeyID + "/SMS/TemplateSMS?sig=" + sign
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("Authorization", base64.StdEncoding.EncodeToString([]byte(c.config.AccessKeyID+":"+timestamp)))

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, endpoint, payload, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["statusCode"].(string); code == "000000" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
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
	if config.BaseURL == "" {
		config.BaseURL = "https://app.cloopen.com:8883/2013-12-26"
	}
	return config
}
