package aliyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
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

const Supplier = "alibaba"

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

func (c *Client) ConfigID() string {
	return c.config.ID()
}

func (c *Client) Supplier() string {
	return Supplier
}

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	messages := map[string]string{c.config.TemplateName: message}
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, phone, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	messages := map[string]string{c.config.TemplateName: message}
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, messages)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, strings.Join(phones, ","), templateID, messages)
}

func (c *Client) doSend(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	templateParam, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	body := map[string]string{
		"PhoneNumbers":  phone,
		"SignName":      c.config.Signature,
		"TemplateParam": string(templateParam),
		"TemplateCode":  templateID,
	}
	endpoint, form, err := c.signedRequest(body)
	if err != nil {
		return nil, err
	}
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, endpoint, form, http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["Code"].(string); code == "OK" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) signedRequest(body map[string]string) (string, url.Values, error) {
	common := map[string]string{
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   time.Now().UTC().Format("20060102150405.000000000"),
		"AccessKeyId":      c.config.AccessKeyID,
		"SignatureVersion": "1.0",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Format":           "JSON",
		"Action":           c.extra("action", "SendSms"),
		"Version":          c.extra("version", "2017-05-25"),
		"RegionId":         c.extra("regionId", "cn-hangzhou"),
	}
	signed := make(map[string]string, len(common)+len(body))
	for key, value := range common {
		signed[key] = value
	}
	for key, value := range body {
		signed[key] = value
	}
	canonical := canonicalQuery(signed)
	stringToSign := "POST&" + specialURLEncode("/") + "&" + specialURLEncode(canonical)
	signature := sign(c.config.AccessKeySecret+"&", stringToSign)

	query := url.Values{}
	query.Set("Signature", signature)
	for key, value := range common {
		query.Set(key, value)
	}
	form := url.Values{}
	for key, value := range body {
		form.Set(key, value)
	}
	endpoint := "https://" + c.config.RequestURL + "/?" + query.Encode()
	return endpoint, form, nil
}

func canonicalQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(specialURLEncode(key))
		builder.WriteByte('=')
		builder.WriteString(specialURLEncode(values[key]))
	}
	return builder.String()
}

func specialURLEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func sign(secret string, text string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(text))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *Client) extra(key string, fallback string) string {
	if c.config.Extra == nil || c.config.Extra[key] == "" {
		return fallback
	}
	return c.config.Extra[key]
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
		config.RequestURL = "dysmsapi.aliyuncs.com"
	}
	if config.TemplateName == "" {
		config.TemplateName = "code"
	}
	return config
}
