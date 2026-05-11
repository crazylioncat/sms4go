package lianlu

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "lianlu"

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

func (c *Client) SendMessage(ctx context.Context, phone string, templateParam string) (*sms.Response, error) {
	return c.MassTexting(ctx, []string{phone}, templateParam)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, []string{phone}, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, templateParam string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{"0": templateParam})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	body := c.requestBody("3")
	body["PhoneNumberSet"] = phones
	body["TemplateId"] = templateID
	body["TemplateParamSet"] = base.ValuesBySortedKey(messages)
	return c.doSend(ctx, "/template/send", body)
}

func (c *Client) SendNormalMessage(ctx context.Context, phones []string, message string, signName string) (*sms.Response, error) {
	body := c.requestBody("1")
	body["PhoneNumberSet"] = phones
	body["SessionContext"] = message
	body["SignName"] = signName
	return c.doSend(ctx, "/normal/send", body)
}

func (c *Client) requestBody(msgType string) map[string]any {
	return map[string]any{
		"MchId":     c.config.MchID,
		"Type":      msgType,
		"AppId":     c.config.AppID,
		"Version":   valueOr(c.config.Version, "1.1.0"),
		"SignType":  valueOr(c.config.SignType, "MD5"),
		"TimeStamp": time.Now().UnixMilli(),
		"SignName":  c.config.Signature,
	}
}

func (c *Client) doSend(ctx context.Context, path string, body map[string]any) (*sms.Response, error) {
	body["Signature"] = c.signature(body)
	headers := http.Header{}
	headers.Set("Accept", "application/json")

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, c.config.RequestURL+path, body, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if status, _ := data["status"].(string); status == "00" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) signature(body map[string]any) string {
	ignored := map[string]bool{
		"Signature": true, "PhoneList": true, "phoneSet": true, "PhoneNumberSet": true,
		"TemplateParamSet": true, "SessionContext": true, "SessionContextSet": true, "ContextParamSet": true,
	}
	keys := make([]string, 0, len(body))
	for key, value := range body {
		if ignored[key] || value == nil || strings.TrimSpace(stringify(value)) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		parts = append(parts, key+"="+stringify(body[key]))
	}
	text := strings.Join(parts, "&") + "&key=" + c.config.AppKey
	if strings.EqualFold(c.config.SignType, "HMACSHA256") {
		mac := hmac.New(sha256.New, []byte(c.config.AppKey))
		mac.Write([]byte(text))
		return strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	}
	sum := md5.Sum([]byte(text))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		data, _ := json.Marshal(typed)
		return string(data)
	}
}

func valueOr(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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
		config.RequestURL = "https://apis.shlianlu.com/sms/trade"
	}
	if config.SignType == "" {
		config.SignType = "MD5"
	}
	if config.Version == "" {
		config.Version = "1.1.0"
	}
	return config
}
