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

func SendMessage(ctx context.Context, configID string, phone string, message string) (*sms.Response, error) {
	return DefaultFactory.SendMessage(ctx, configID, phone, message)
}

func SendMessageAsync(ctx context.Context, configID string, phone string, message string, callback sms.Callback) {
	DefaultFactory.SendMessageAsync(ctx, configID, phone, message, callback)
}

func DelayedMessage(ctx context.Context, configID string, phone string, message string, delay time.Duration, callback sms.Callback) *time.Timer {
	return DefaultFactory.DelayedMessage(ctx, configID, phone, message, delay, callback)
}
