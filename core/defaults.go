package core

import (
	"context"
	"time"

	"github.com/CrazyLionCat/sms4go/sms"
)

var DefaultFactory = NewFactory()

func Register(client sms.Client, weight int) error {
	return DefaultFactory.Register(client, weight)
}

func RegisterIfAbsent(client sms.Client, weight int) (bool, error) {
	return DefaultFactory.RegisterIfAbsent(client, weight)
}

func Unregister(configID string) bool {
	return DefaultFactory.Unregister(configID)
}

func Get(configID string) (sms.Client, bool) {
	return DefaultFactory.Get(configID)
}

func GetBySupplier(supplier string) (sms.Client, bool) {
	return DefaultFactory.GetBySupplier(supplier)
}

func ListBySupplier(supplier string) []sms.Client {
	return DefaultFactory.ListBySupplier(supplier)
}

func GetAll() []sms.Client {
	return DefaultFactory.All()
}

func Next() (sms.Client, error) {
	return DefaultFactory.Next()
}

func SmsBlend(configID string) (sms.SmsBlend, error) {
	if configID == "" {
		return DefaultFactory.Next()
	}
	client, ok := DefaultFactory.Get(configID)
	if !ok {
		return nil, sms.ErrClientNotFound
	}
	return client, nil
}

func Use(middleware ...Middleware) {
	DefaultFactory.Use(middleware...)
}

func Block(phone string) {
	DefaultFactory.Block(phone)
}

func Unblock(phone string) {
	DefaultFactory.Unblock(phone)
}

func BlockAll(phones []string) {
	DefaultFactory.BlockAll(phones)
}

func UnblockAll(phones []string) {
	DefaultFactory.UnblockAll(phones)
}

func SendMessage(ctx context.Context, configID string, phone string, message string) (*sms.Response, error) {
	return DefaultFactory.SendMessage(ctx, configID, phone, message)
}

func SendMessageWithParams(ctx context.Context, configID string, phone string, messages map[string]string) (*sms.Response, error) {
	return DefaultFactory.SendMessageWithParams(ctx, configID, phone, messages)
}

func SendTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	return DefaultFactory.SendTemplate(ctx, configID, phone, templateID, messages)
}

func MassTexting(ctx context.Context, configID string, phones []string, message string) (*sms.Response, error) {
	return DefaultFactory.MassTexting(ctx, configID, phones, message)
}

func MassTextingTemplate(ctx context.Context, configID string, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	return DefaultFactory.MassTextingTemplate(ctx, configID, phones, templateID, messages)
}

func SendMessageAsync(ctx context.Context, configID string, phone string, message string, callback sms.Callback) {
	DefaultFactory.SendMessageAsync(ctx, configID, phone, message, callback)
}

func SendMessageChan(ctx context.Context, configID string, phone string, message string) <-chan sms.Result {
	return DefaultFactory.SendMessageChan(ctx, configID, phone, message)
}

func SendTemplateAsync(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, callback sms.Callback) {
	DefaultFactory.SendTemplateAsync(ctx, configID, phone, templateID, messages, callback)
}

func SendTemplateChan(ctx context.Context, configID string, phone string, templateID string, messages map[string]string) <-chan sms.Result {
	return DefaultFactory.SendTemplateChan(ctx, configID, phone, templateID, messages)
}

func DelayedMessage(ctx context.Context, configID string, phone string, message string, delay time.Duration, callback sms.Callback) *time.Timer {
	return DefaultFactory.DelayedMessage(ctx, configID, phone, message, delay, callback)
}

func DelayMessage(ctx context.Context, configID string, phone string, message string, delay time.Duration) <-chan sms.Result {
	return DefaultFactory.DelayMessage(ctx, configID, phone, message, delay)
}

func DelayedTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, delay time.Duration, callback sms.Callback) *time.Timer {
	return DefaultFactory.DelayedTemplate(ctx, configID, phone, templateID, messages, delay, callback)
}

func DelayTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, delay time.Duration) <-chan sms.Result {
	return DefaultFactory.DelayTemplate(ctx, configID, phone, templateID, messages, delay)
}
