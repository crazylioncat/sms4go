package unisms

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "unisms"

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
	return c.doSend(ctx, []string{phone}, templateID, messages)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, map[string]string{c.config.TemplateName: message})
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	if len(phones) > 1000 {
		return nil, errors.New("sms4go: unisms mass texting limit is 1000 phones")
	}
	return c.doSend(ctx, phones, templateID, messages)
}

func (c *Client) doSend(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	payload := map[string]any{
		"to":           phones,
		"signature":    c.config.Signature,
		"templateId":   templateID,
		"templateData": messages,
	}
	endpoint := c.endpoint("sms.message.send")
	headers := http.Header{}
	headers.Set("User-Agent", "uni-java-sdk/0.0.4")
	headers.Set("Accept", "application/json")

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
		if code, _ := data["code"].(string); code == "0" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) endpoint(action string) string {
	query := map[string]string{"action": action, "accessKeyId": c.config.AccessKeyID}
	if !c.config.IsSimple && c.config.AccessKeySecret != "" {
		query["algorithm"] = "hmac-sha256"
		query["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
		query["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 36)
		query["signature"] = c.signature(query)
	}
	values := url.Values{}
	for key, value := range query {
		values.Set(key, value)
	}
	return c.config.BaseURL + "?" + values.Encode()
}

func (c *Client) signature(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	mac := hmac.New(sha256.New, []byte(c.config.AccessKeySecret))
	mac.Write([]byte(strings.Join(parts, "&")))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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
		config.BaseURL = "https://uni.apistd.com"
	}
	if config.AccessKeySecret == "" {
		config.IsSimple = true
	}
	return config
}
