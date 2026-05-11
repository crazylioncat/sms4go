package huawei

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "huawei"

const requestPath = "/sms/batchSendSms/v1"

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

func (c *Client) ConfigID() string {
	return c.config.ID()
}

func (c *Client) Supplier() string {
	return Supplier
}

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, map[string]string{"0": message})
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, phone, templateID, base.ValuesBySortedKey(messages))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.SendMessage(ctx, strings.Join(phones, ","), message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, strings.Join(phones, ","), templateID, messages)
}

func (c *Client) doSend(ctx context.Context, receiver string, templateID string, params []string) (*sms.Response, error) {
	values := url.Values{}
	values.Set("from", c.config.Sender)
	values.Set("to", receiver)
	values.Set("templateId", templateID)
	if len(params) > 0 {
		templateParas, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		values.Set("templateParas", string(templateParas))
	}
	if c.config.StatusCallBack != "" {
		values.Set("statusCallback", c.config.StatusCallBack)
	}
	if c.config.Signature != "" {
		values.Set("signature", c.config.Signature)
	}

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		headers, err := c.headers()
		if err != nil {
			return nil, err
		}
		raw, _, err := c.http.PostForm(ctx, c.endpoint(), values, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); code == "000000" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) headers() (http.Header, error) {
	wsse, err := buildWSSEHeader(c.config.AccessKeyID, c.config.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Set("Authorization", "WSSE realm=\"SDP\",profile=\"UsernameToken\",type=\"Appkey\"")
	headers.Set("X-WSSE", wsse)
	return headers, nil
}

func (c *Client) endpoint() string {
	return strings.TrimRight(c.config.URL, "/") + requestPath
}

func buildWSSEHeader(appKey string, appSecret string) (string, error) {
	nonce, err := nonceHex(16)
	if err != nil {
		return "", err
	}
	created := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	hash := sha256.Sum256([]byte(nonce + created + appSecret))
	digest := base64.StdEncoding.EncodeToString(hash[:])
	return "UsernameToken Username=\"" + appKey + "\",PasswordDigest=\"" + digest + "\",Nonce=\"" + nonce + "\",Created=\"" + created + "\"", nil
}

func nonceHex(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (c *Client) retryInterval() time.Duration {
	if c.config.RetryInterval <= 0 {
		return 5 * time.Second
	}
	return c.config.RetryInterval
}
