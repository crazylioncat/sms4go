package sms

import "context"

// Client is the Go equivalent of sms4j's SmsBlend.
type Client interface {
	ConfigID() string
	Supplier() string
	SendMessage(ctx context.Context, phone string, message string) (*Response, error)
	SendMessageWithParams(ctx context.Context, phone string, messages map[string]string) (*Response, error)
	SendTemplate(ctx context.Context, phone string, templateID string, messages map[string]string) (*Response, error)
	MassTexting(ctx context.Context, phones []string, message string) (*Response, error)
	MassTextingTemplate(ctx context.Context, phones []string, templateID string, messages map[string]string) (*Response, error)
}

type Callback func(*Response, error)

type PhoneVerifier interface {
	VerifyPhone(phone string) bool
}

type PhoneVerifierFunc func(phone string) bool

func (f PhoneVerifierFunc) VerifyPhone(phone string) bool {
	return f(phone)
}
