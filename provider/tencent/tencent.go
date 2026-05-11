package tencent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
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

const Supplier = "tencent"

type Client struct {
	config sms.BaseConfig
	http   *base.HTTPClient
}

type sendRequest struct {
	PhoneNumberSet   []string `json:"PhoneNumberSet"`
	SmsSdkAppId      string   `json:"SmsSdkAppId"`
	SignName         string   `json:"SignName"`
	TemplateId       string   `json:"TemplateId"`
	TemplateParamSet []string `json:"TemplateParamSet"`
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
	return c.SendTemplate(ctx, phone, c.config.TemplateID, base.ParamsByAmpersand(message))
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return c.SendTemplate(ctx, phone, c.config.TemplateID, messages)
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return c.doSend(ctx, []string{normalizePhone(phone)}, templateID, base.ValuesBySortedKey(messages))
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return c.MassTextingTemplate(ctx, phones, c.config.TemplateID, base.ParamsByAmpersand(message))
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	normalized := make([]string, 0, len(phones))
	for _, phone := range phones {
		normalized = append(normalized, normalizePhone(phone))
	}
	return c.doSend(ctx, normalized, templateID, base.ValuesBySortedKey(messages))
}

func (c *Client) doSend(ctx context.Context, phones []string, templateID string, params []string) (*sms.Response, error) {
	body := sendRequest{
		PhoneNumberSet:   phones,
		SmsSdkAppId:      c.config.SDKAppID,
		SignName:         c.config.Signature,
		TemplateId:       templateID,
		TemplateParamSet: params,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var last *sms.Response
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryInterval())
		}
		timestamp := time.Now().Unix()
		headers := c.headers(timestamp, payload)
		raw, _, err := c.http.PostRaw(ctx, "https://"+c.config.RequestURL, payload, "application/json; charset=utf-8", headers)
		if err != nil {
			lastErr = err
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
		last = sms.Failure(data, c.ConfigID())
		if tencentSuccess(data) {
			last.Success = true
			return last, nil
		}
	}
	return last, lastErr
}

func (c *Client) headers(timestamp int64, payload []byte) http.Header {
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	canonicalHeaders := "content-type:application/json; charset=utf-8\nhost:" + c.config.RequestURL + "\n"
	signedHeaders := "content-type;host"
	hashedPayload := sha256Hex(payload)
	canonicalRequest := strings.Join([]string{
		"POST",
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	}, "\n")
	credentialScope := date + "/" + c.config.Service + "/tc3_request"
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		strconvFormatInt(timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	secretDate := hmacSHA256([]byte("TC3"+c.config.AccessKeySecret), date)
	secretService := hmacSHA256(secretDate, c.config.Service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := "TC3-HMAC-SHA256 Credential=" + c.config.AccessKeyID + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	headers := http.Header{}
	headers.Set("Authorization", authorization)
	headers.Set("Host", c.config.RequestURL)
	headers.Set("X-TC-Action", c.config.Action)
	headers.Set("X-TC-Timestamp", strconvFormatInt(timestamp))
	headers.Set("X-TC-Version", c.config.Version)
	headers.Set("X-TC-Region", c.config.Territory)
	return headers
}

func normalizePhone(phone string) string {
	if strings.Contains(phone, "-") {
		return strings.ReplaceAll(phone, "-", "")
	}
	if strings.HasPrefix(phone, "+86") {
		return phone
	}
	return "+86" + phone
}

func tencentSuccess(data map[string]any) bool {
	response, _ := data["Response"].(map[string]any)
	if response == nil {
		return false
	}
	if _, hasError := response["Error"]; hasError {
		return false
	}
	statusSet, _ := response["SendStatusSet"].([]any)
	for _, item := range statusSet {
		status, _ := item.(map[string]any)
		if code, _ := status["Code"].(string); code != "" && code != "Ok" {
			return false
		}
	}
	return true
}

func hmacSHA256(key []byte, text string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(text))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
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
	if config.RequestURL == "" {
		config.RequestURL = "sms.tencentcloudapi.com"
	}
	if config.Action == "" {
		config.Action = "SendSms"
	}
	if config.Version == "" {
		config.Version = "2021-01-11"
	}
	if config.Territory == "" {
		config.Territory = "ap-guangzhou"
	}
	if config.Service == "" {
		config.Service = "sms"
	}
	return config
}
