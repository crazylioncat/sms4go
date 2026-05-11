package danmi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "danmi"

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
	phones := []string{phone}
	if strings.Contains(phone, ",") {
		phones = strings.Split(phone, ",")
	}
	return c.MassTexting(ctx, phones, message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: danmi does not support SendMessageWithParams")
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: danmi does not support SendTemplate")
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.doSend(ctx, phones, message, c.config.TemplateID)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: danmi does not support MassTextingTemplate")
}

func (c *Client) QueryBalance(ctx context.Context) (*sms.Response, error) {
	return c.doSend(ctx, nil, "", "")
}

func (c *Client) VoiceCode(ctx context.Context, called string, verifyCode string) (*sms.Response, error) {
	return c.doSend(ctx, []string{called}, verifyCode, "")
}

func (c *Client) VoiceNotify(ctx context.Context, called string, notifyFileID string) (*sms.Response, error) {
	return c.doSend(ctx, []string{called}, notifyFileID, "")
}

func (c *Client) VoiceTemplate(ctx context.Context, called string, templateID string, param string) (*sms.Response, error) {
	return c.doSend(ctx, []string{called}, param, templateID)
}

func (c *Client) doSend(ctx context.Context, phones []string, message string, templateID string) (*sms.Response, error) {
	body, err := c.body(phones, message, templateID)
	if err != nil {
		return nil, err
	}
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostJSON(ctx, c.config.Host+c.config.Action, body, http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["respCode"].(string); code == "00000" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) body(phones []string, message string, templateID string) (map[string]any, error) {
	body := map[string]any{"respDataType": "JSON", "accountSid": c.config.AccessKeyID}
	switch c.config.Action {
	case "distributor/sendSMS":
		if message == "" && templateID == "" {
			return nil, errors.New("sms4go: danmi message and templateId cannot both be empty")
		}
		if templateID != "" {
			body["templateid"] = templateID
		}
		if message != "" {
			body["smsContent"] = url.QueryEscape(message)
		}
		body["to"] = prefixPhones(phones)
	case "distributor/user/query":
	case "voice/voiceCode":
		body["called"] = prefix86(phones[0])
		body["verifyCode"] = message
	case "voice/voiceNotify":
		body["called"] = prefix86(phones[0])
		body["notifyFileId"] = message
	case "voice/voiceTemplate":
		body["called"] = prefix86(phones[0])
		body["templateId"] = templateID
		body["param"] = message
	default:
		return nil, errors.New("sms4go: unsupported danmi action")
	}
	timestamp := time.Now().UnixMilli()
	body["timestamp"] = timestamp
	body["sig"] = md5Hex(c.config.AccessKeyID + c.config.AccessKeySecret + strconv.FormatInt(timestamp, 10))
	return body, nil
}

func prefixPhones(phones []string) []string {
	result := make([]string, 0, len(phones))
	for _, phone := range phones {
		result = append(result, prefix86(strings.TrimSpace(phone)))
	}
	return result
}

func prefix86(phone string) string {
	if strings.HasPrefix(phone, "+86") {
		return phone
	}
	return "+86" + phone
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
	if config.Host == "" {
		config.Host = "https://openapi.danmi.com/"
	}
	if config.Action == "" {
		config.Action = "distributor/sendSMS"
	}
	return config
}
