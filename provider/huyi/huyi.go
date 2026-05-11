package huyi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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

const Supplier = "huyi"

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
	return c.sendSingle(ctx, phone, "", message)
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return nil, errors.New("sms4go: huyi does not support SendMessageWithParams")
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.sendSingle(ctx, phone, templateID, pipeValues(messages))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.sendMass(ctx, phones, "", message)
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.sendMass(ctx, phones, templateID, pipeValues(messages))
}

func (c *Client) sendSingle(ctx context.Context, phone string, templateID string, message string) (*sms.Response, error) {
	values := c.params(phone, templateID, message)
	return c.handle(ctx, c.config.BaseURL+c.config.SingleMsgURL, values)
}

func (c *Client) sendMass(ctx context.Context, phones []string, templateID string, message string) (*sms.Response, error) {
	mobiles := make([]string, 0, len(phones))
	for _, phone := range phones {
		if message != "" {
			mobiles = append(mobiles, phone+"|"+message)
		} else {
			mobiles = append(mobiles, phone)
		}
	}
	values := c.params(strings.Join(mobiles, ","), templateID, message)
	return c.handle(ctx, c.config.BaseURL+c.config.MassMsgURL, values)
}

func (c *Client) params(mobile string, templateID string, message string) url.Values {
	values := url.Values{}
	values.Set("format", "json")
	values.Set("account", c.config.AccessKeyID)
	values.Set("password", c.config.AccessKeySecret)
	values.Set("content", message)
	values.Set("mobile", mobile)
	if templateID != "" {
		values.Set("templateid", templateID)
	}
	if c.config.EnableMD5 {
		unix := time.Now().Unix()
		values.Set("time", strconvFormatInt(unix))
		values.Set("password", md5Hex(values.Get("account")+values.Get("password")+values.Get("mobile")+values.Get("content")+strconvFormatInt(unix)))
	}
	return values
}

func (c *Client) handle(ctx context.Context, endpoint string, values url.Values) (*sms.Response, error) {
	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		raw, _, err := c.http.Get(ctx, endpoint+"&"+values.Encode(), http.Header{})
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if code, _ := data["code"].(string); code == "2" {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func pipeValues(messages map[string]string) string {
	keys := make([]string, 0, len(messages))
	for key := range messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, messages[key])
	}
	return strings.Join(values, "|")
}

func md5Hex(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
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
		config.BaseURL = "https://106.ihuyi.com/webservice/sms.php"
	}
	if config.SingleMsgURL == "" {
		config.SingleMsgURL = "?method=Submit"
	}
	if config.MassMsgURL == "" {
		config.MassMsgURL = "?method=SubmitBatch"
	}
	return config
}
