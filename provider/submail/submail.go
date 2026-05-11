package submail

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "mysubmail"

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

func (c *Client) SendMessage(ctx context.Context, phone string, content string) (*sms.Response, error) {
	return c.send(ctx, []string{phone}, content, c.config.TemplateID, nil)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, vars map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, vars)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, vars map[string]string) (*sms.Response, error) {
	content, copied := splitContent(vars)
	return c.send(ctx, []string{phone}, content, templateID, copied)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, content string) (*sms.Response, error) {
	return c.send(ctx, phones, content, c.config.TemplateID, nil)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, vars map[string]string) (*sms.Response, error) {
	content, copied := splitContent(vars)
	return c.send(ctx, phones, content, templateID, copied)
}

func (c *Client) send(ctx context.Context, phones []string, content string, templateID string, vars map[string]string) (*sms.Response, error) {
	if len(phones) == 0 {
		return nil, sms.ErrInvalidPhone
	}
	body, err := c.buildBody(ctx, phones, content, templateID, vars)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	headers.Set("Content-Type", "application/json")

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, c.config.Host+c.config.Action, body, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if status, _ := data["status"].(string); status == "success" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) buildBody(ctx context.Context, phones []string, content string, templateID string, vars map[string]string) (map[string]any, error) {
	switch c.config.Action {
	case "send.json":
		if strings.TrimSpace(content) == "" {
			return nil, sms.ErrEmptyMessage
		}
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "to": prefix86(phones[0]), "content": c.content(content)})
	case "xsend.json":
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "to": prefix86(phones[0]), "project": templateID, "vars": mustJSON(vars)}, "vars")
	case "multisend.json":
		if strings.TrimSpace(content) == "" || len(vars) == 0 {
			return nil, errors.New("sms4go: submail content and vars are required")
		}
		multi := buildMulti(limit(phones, 50), vars)
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "content": c.content(content), "multi": mustJSON(multi)}, "multi", "content")
	case "multixsend.json":
		if len(vars) == 0 {
			return nil, errors.New("sms4go: submail vars are required")
		}
		multi := buildMulti(limit(phones, 200), vars)
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "project": templateID, "multi": mustJSON(multi)}, "multi", "content")
	case "batchsend.json":
		if strings.TrimSpace(content) == "" {
			return nil, sms.ErrEmptyMessage
		}
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "to": prefixPhones(limit(phones, 10000)), "content": c.content(content)}, "content")
	case "batchxsend.json":
		if len(vars) == 0 {
			return nil, errors.New("sms4go: submail vars are required")
		}
		return c.sign(ctx, map[string]any{"appid": c.config.AccessKeyID, "to": prefixPhones(limit(phones, 10000)), "project": templateID, "vars": mustJSON(vars)}, "vars")
	default:
		return nil, errors.New("sms4go: unsupported submail action")
	}
}

func (c *Client) sign(ctx context.Context, body map[string]any, excludes ...string) (map[string]any, error) {
	timestamp, err := c.timestamp(ctx)
	if err != nil {
		return nil, err
	}
	body["timestamp"] = timestamp
	body["sign_type"] = c.config.SignType
	if c.config.SignVersion != "" {
		body["sign_version"] = c.config.SignVersion
	}
	body["signature"] = signature(body, c.config.SignType, c.config.AccessKeyID, c.config.AccessKeySecret, c.config.SignVersion == "2", excludes...)
	return body, nil
}

func (c *Client) content(content string) string {
	if len(content) > 1000 {
		content = content[:1000]
	}
	if c.config.Signature != "" && !strings.HasPrefix(content, "【 "+c.config.Signature+"】") {
		content = "【 " + c.config.Signature + "】" + content
	}
	return content
}

func (c *Client) timestamp(ctx context.Context) (string, error) {
	raw, _, err := c.http.Get(ctx, "https://api-v4.mysubmail.com/service/timestamp", http.Header{})
	if err != nil {
		return "", err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", err
	}
	if timestamp, _ := data["timestamp"].(string); timestamp != "" {
		return timestamp, nil
	}
	return "", errors.New("sms4go: submail timestamp response missing timestamp")
}

func signature(body map[string]any, signType string, appID string, appKey string, excludeEnabled bool, excludes ...string) string {
	upper := strings.ToUpper(signType)
	if upper != "MD5" && upper != "SHA1" && upper != "SHA-1" {
		return appKey
	}
	text := appID + appKey + sortedParams(body, excludeEnabled, excludes...) + appID + appKey
	if upper == "MD5" {
		sum := md5.Sum([]byte(text))
		return hex.EncodeToString(sum[:])
	}
	sum := sha1.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func sortedParams(body map[string]any, excludeEnabled bool, excludes ...string) string {
	excluded := map[string]bool{}
	if excludeEnabled {
		for _, key := range excludes {
			excluded[key] = true
		}
	}
	keys := make([]string, 0, len(body))
	for key := range body {
		if !excluded[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+stringify(body[key]))
	}
	return strings.Join(parts, "&")
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []string:
		return strings.Join(typed, ",")
	default:
		data, _ := json.Marshal(typed)
		return string(data)
	}
}

func splitContent(vars map[string]string) (string, map[string]string) {
	copied := map[string]string{}
	content := ""
	for key, value := range vars {
		if key == "content" {
			content = value
			continue
		}
		copied[key] = value
	}
	return content, copied
}

func buildMulti(phones []string, vars map[string]string) []map[string]any {
	multi := make([]map[string]any, 0, len(phones))
	for _, phone := range phones {
		multi = append(multi, map[string]any{"to": prefix86(phone), "vars": vars})
	}
	return multi
}

func prefix86(phone string) string {
	if strings.HasPrefix(phone, "+86") {
		return phone
	}
	return "+86" + phone
}

func prefixPhones(phones []string) []string {
	result := make([]string, 0, len(phones))
	for _, phone := range phones {
		result = append(result, prefix86(phone))
	}
	return result
}

func limit[T any](items []T, max int) []T {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func mustJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
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
		config.Host = "https://api-v4.mysubmail.com/sms/"
	}
	if !strings.HasSuffix(config.Host, "/") {
		config.Host += "/"
	}
	if config.Action == "" {
		config.Action = "send.json"
	}
	if config.SignType == "" {
		config.SignType = "MD5"
	}
	return config
}
