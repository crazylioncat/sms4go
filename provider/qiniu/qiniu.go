package qiniu

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "qiniu"

type Client struct {
	config sms.BaseConfig
	http   *base.HTTPClient
}

type singleRequest struct {
	Mobile     string            `json:"mobile"`
	TemplateID string            `json:"template_id"`
	Parameters map[string]string `json:"parameters"`
}

type massRequest struct {
	Mobiles    []string          `json:"mobiles"`
	TemplateID string            `json:"template_id"`
	Parameters map[string]string `json:"parameters"`
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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	payload := singleRequest{Mobile: phone, TemplateID: templateID, Parameters: messages}
	return c.doSend(ctx, c.config.BaseURL+c.config.SingleMsgURL, payload)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	payload := massRequest{Mobiles: phones, TemplateID: templateID, Parameters: messages}
	return c.doSend(ctx, c.config.BaseURL+c.config.MassMsgURL, payload)
}

func (c *Client) doSend(ctx context.Context, endpoint string, payload any) (*sms.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		headers, err := c.headers(endpoint, body)
		if err != nil {
			return nil, err
		}
		raw, _, err := c.http.PostRaw(ctx, endpoint, body, "application/json", headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if errText, _ := data["error"].(string); errText == "" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) headers(endpoint string, body []byte) (http.Header, error) {
	signDate := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	signature, err := c.signature("POST", endpoint, body, signDate)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Set("Authorization", signature)
	headers.Set("X-Qiniu-Date", signDate)
	return headers, nil
}

func (c *Client) signature(method string, endpoint string, body []byte, signDate string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	dataToSign := method + " " + parsed.Path +
		"\nHost: " + parsed.Host +
		"\nContent-Type: application/json" +
		"\nX-Qiniu-Date: " + signDate +
		"\n\n" + string(body)
	mac := hmac.New(sha1.New, []byte(c.config.AccessKeySecret))
	mac.Write([]byte(dataToSign))
	encoded := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return "Qiniu " + c.config.AccessKeyID + ":" + encoded, nil
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
		config.BaseURL = "https://sms.qiniuapi.com"
	}
	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	if config.SingleMsgURL == "" {
		config.SingleMsgURL = "/v1/message/single"
	}
	if config.MassMsgURL == "" {
		config.MassMsgURL = "/v1/message"
	}
	return config
}
