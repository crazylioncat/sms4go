package jdcloud

import (
	"context"
	"errors"

	"github.com/CrazyLionCat/sms4go/provider"
	"github.com/CrazyLionCat/sms4go/sms"
)

const Supplier = "jdcloud"

var ErrSDKRequired = errors.New("sms4go: jdcloud provider requires JDCloud Go SDK/OpenAPI signing integration")

type Client struct {
	config sms.BaseConfig
}

func init() {
	provider.Register(Supplier, func(config sms.BaseConfig) (sms.Client, error) {
		return New(config), nil
	})
}

func New(config sms.BaseConfig) *Client {
	config.Supplier = Supplier
	if config.Region == "" {
		config.Region = "cn-north-1"
	}
	return &Client{config: config}
}

func (c *Client) ConfigID() string { return c.config.ID() }

func (c *Client) Supplier() string { return Supplier }

func (c *Client) SendMessage(ctx context.Context, phone string, message string) (*sms.Response, error) {
	return nil, ErrSDKRequired
}

func (c *Client) SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*sms.Response, error) {
	return nil, ErrSDKRequired
}

func (c *Client) SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, ErrSDKRequired
}

func (c *Client) MassTexting(ctx context.Context, phones []string, message string) (*sms.Response, error) {
	return nil, ErrSDKRequired
}

func (c *Client) MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return nil, ErrSDKRequired
}
