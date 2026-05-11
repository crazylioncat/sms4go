package core

import (
	"context"
	"strings"

	"github.com/CrazyLionCat/sms4go/sms"
)

type Operation string

const (
	OperationSendMessage         Operation = "SendMessage"
	OperationSendMessageParams   Operation = "SendMessageWithParams"
	OperationSendTemplate        Operation = "SendTemplate"
	OperationMassTexting         Operation = "MassTexting"
	OperationMassTextingTemplate Operation = "MassTextingTemplate"
)

type Request struct {
	Operation  Operation
	Phone      string
	Phones     []string
	Message    string
	TemplateID string
	Messages   map[string]string
}

type Handler func(context.Context, sms.Client, Request) (*sms.Response, error)

type Middleware func(Handler) Handler

func Chain(final Handler, middleware ...Middleware) Handler {
	handler := final
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

func ValidationMiddleware(verifier sms.PhoneVerifier) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, client sms.Client, req Request) (*sms.Response, error) {
			switch req.Operation {
			case OperationSendMessage:
				if err := validatePhone(req.Phone, verifier); err != nil {
					return nil, err
				}
				if strings.TrimSpace(req.Message) == "" {
					return nil, sms.ErrEmptyMessage
				}
			case OperationSendMessageParams:
				if err := validatePhone(req.Phone, verifier); err != nil {
					return nil, err
				}
				if len(req.Messages) == 0 {
					return nil, sms.ErrEmptyMessage
				}
			case OperationSendTemplate:
				if err := validatePhone(req.Phone, verifier); err != nil {
					return nil, err
				}
				if strings.TrimSpace(req.TemplateID) == "" {
					return nil, sms.ErrEmptyTemplate
				}
			case OperationMassTexting:
				if err := validatePhones(req.Phones, verifier); err != nil {
					return nil, err
				}
				if strings.TrimSpace(req.Message) == "" {
					return nil, sms.ErrEmptyMessage
				}
			case OperationMassTextingTemplate:
				if err := validatePhones(req.Phones, verifier); err != nil {
					return nil, err
				}
				if strings.TrimSpace(req.TemplateID) == "" {
					return nil, sms.ErrEmptyTemplate
				}
			}
			return next(ctx, client, req)
		}
	}
}

func validatePhone(phone string, verifier sms.PhoneVerifier) error {
	if strings.TrimSpace(phone) == "" {
		return sms.ErrInvalidPhone
	}
	if verifier != nil && !verifier.VerifyPhone(phone) {
		return sms.ErrInvalidPhone
	}
	return nil
}

func validatePhones(phones []string, verifier sms.PhoneVerifier) error {
	if len(phones) == 0 {
		return sms.ErrInvalidPhone
	}
	for _, phone := range phones {
		if err := validatePhone(phone, verifier); err != nil {
			return err
		}
	}
	return nil
}

func BlacklistMiddleware(blacklist *Blacklist) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, client sms.Client, req Request) (*sms.Response, error) {
			if blacklist == nil {
				return next(ctx, client, req)
			}
			if req.Phone != "" && blacklist.Contains(req.Phone) {
				return nil, sms.ErrBlacklisted
			}
			for _, phone := range req.Phones {
				if blacklist.Contains(phone) {
					return nil, sms.ErrBlacklisted
				}
			}
			return next(ctx, client, req)
		}
	}
}
