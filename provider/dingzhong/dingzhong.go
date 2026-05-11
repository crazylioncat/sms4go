package dingzhong

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/provider/base"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "dingzhong"

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
	values := c.baseValues(phone)
	values.Set("msg", message)
	return c.doSend(ctx, c.config.BaseAction, values)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	values := c.baseValues(phone)
	values.Set("templateId", templateID)
	data, _ := json.Marshal(messages)
	values.Set("msgParam", string(data))
	return c.doSend(ctx, c.config.TemplateAction, values)
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.SendMessage(ctx, strings.Join(prefixPhones(phones), ","), message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, strings.Join(prefixPhones(phones), ","), templateID, messages)
}

func (c *Client) baseValues(phone string) url.Values {
	values := url.Values{}
	values.Set("cdkey", c.config.AccessKeyID)
	values.Set("password", c.config.AccessKeySecret)
	values.Set("mobile", phone)
	return values
}

func (c *Client) doSend(ctx context.Context, action string, values url.Values) (*sms.Response, error) {
	headers := http.Header{}
	headers.Set("Accept", "application/json")
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.PostForm(ctx, c.config.RequestURL+"/"+action, values, headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["resCode"].(string); code == "0" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func prefixPhones(phones []string) []string {
	result := make([]string, 0, len(phones))
	for _, phone := range phones {
		if strings.HasPrefix(phone, "+86") {
			result = append(result, phone)
		} else {
			result = append(result, "+86"+phone)
		}
	}
	return result
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
		config.RequestURL = "http://demoapi.321sms.com:8201"
	}
	if config.BaseAction == "" {
		config.BaseAction = "Sms/SendSms"
	}
	if config.TemplateAction == "" {
		config.TemplateAction = "Sms/SendTemplateSms"
	}
	return config
}
