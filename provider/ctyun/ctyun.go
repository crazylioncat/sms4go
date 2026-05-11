package ctyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "ctyun"

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
	return c.doSend(ctx, phone, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
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
	return c.doSend(ctx, strings.Join(prefixed, ","), templateID, messages)
}

func (c *Client) doSend(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	templateParam, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	payload := map[string]string{
		"action":        c.config.Action,
		"phoneNumber":   phone,
		"signName":      c.config.Signature,
		"templateParam": string(templateParam),
		"templateCode":  templateID,
	}
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
		raw, _, err := c.http.PostRaw(ctx, c.config.RequestURL, body, "application/json; charset=utf-8", c.signHeader(body))
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); code == "OK" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) signHeader(body []byte) http.Header {
	now := time.Now().In(time.FixedZone("GMT+8", 8*60*60))
	signatureDate := now.Format("20060102")
	signatureTime := now.UTC().Format("20060102T150405Z")
	requestID := strconv.FormatInt(now.UnixNano(), 36)
	contentHash := sha256Hex(body)

	kTime := hmacSHA256([]byte(c.config.AccessKeySecret), signatureTime)
	kAk := hmacSHA256(kTime, c.config.AccessKeyID)
	kDate := hmacSHA256(kAk, signatureDate)
	strToSign := "ctyun-eop-request-id:" + requestID + "\neop-date:" + signatureTime + "\n\n\n" + contentHash
	signature := base64.StdEncoding.EncodeToString(hmacSHA256(kDate, strToSign))
	headers := http.Header{}
	headers.Set("ctyun-eop-request-id", requestID)
	headers.Set("Eop-date", signatureTime)
	headers.Set("Eop-Authorization", c.config.AccessKeyID+" Headers=ctyun-eop-request-id;eop-date Signature="+signature)
	return headers
}

func hmacSHA256(key []byte, text string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(text))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
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
		config.RequestURL = "https://sms-global.ctapi.ctyun.cn/sms/api/v1"
	}
	if config.Action == "" {
		config.Action = "SendSms"
	}
	return config
}
