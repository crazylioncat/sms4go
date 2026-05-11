package starter

import (
	"github.com/CrazyLionCat/sms4go/config"
	"github.com/CrazyLionCat/sms4go/core"
	"github.com/CrazyLionCat/sms4go/sms"
)

type Config = config.FileConfig
type SMSConfig = config.SMSConfig
type SmsBlend = sms.SmsBlend

func LoadYAML(path string) (*Config, error) {
	return config.LoadYAML(path)
}

func LoadJSON(path string) (*Config, error) {
	return config.LoadJSON(path)
}

func Register(factory *core.Factory, cfg *Config) error {
	return config.RegisterBlends(factory, cfg)
}

func RegisterYAML(factory *core.Factory, path string) error {
	cfg, err := LoadYAML(path)
	if err != nil {
		return err
	}
	return Register(factory, cfg)
}

func RegisterJSON(factory *core.Factory, path string) error {
	cfg, err := LoadJSON(path)
	if err != nil {
		return err
	}
	return Register(factory, cfg)
}

func NewFactoryFromYAML(path string, options ...core.Option) (*core.Factory, error) {
	factory := core.NewFactory(options...)
	if err := RegisterYAML(factory, path); err != nil {
		return nil, err
	}
	return factory, nil
}

func NewFactoryFromJSON(path string, options ...core.Option) (*core.Factory, error) {
	factory := core.NewFactory(options...)
	if err := RegisterJSON(factory, path); err != nil {
		return nil, err
	}
	return factory, nil
}

func GetSmsBlend(factory *core.Factory, configID string) (SmsBlend, error) {
	if factory == nil {
		factory = core.DefaultFactory
	}
	if configID == "" {
		return factory.Next()
	}
	client, ok := factory.Get(configID)
	if !ok {
		return nil, sms.ErrClientNotFound
	}
	return client, nil
}
