package mock

import (
	"context"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "mock"

type Client struct {
	Config sms.BaseConfig
}

func init() {
	provider.Register(Supplier, func(config sms.BaseConfig) (sms.Client, error) {
		return New(config), nil
	})
}

func New(config sms.BaseConfig) *Client {
	if config.Supplier == "" {
		config.Supplier = Supplier
	}
	if config.ConfigID == "" {
		config.ConfigID = config.Supplier
	}
	return &Client{Config: config}
}

func (c *Client) ConfigID() string {
	return c.Config.ID()
}

func (c *Client) Supplier() string {
	return c.Config.Supplier
}

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	return sms.Success(map[string]any{"phone": phone, "message": message}, c.ConfigID()), nil
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return sms.Success(map[string]any{"phone": phone, "messages": messages}, c.ConfigID()), nil
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return sms.Success(map[string]any{"phone": phone, "templateId": templateID, "messages": messages}, c.ConfigID()), nil
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return sms.Success(map[string]any{"phones": phones, "message": message}, c.ConfigID()), nil
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return sms.Success(map[string]any{"phones": phones, "templateId": templateID, "messages": messages}, c.ConfigID()), nil
}
