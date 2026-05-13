package starter

import (
	"github.com/CrazyLionCat/sms4go/config"
	"github.com/CrazyLionCat/sms4go/core"
	"github.com/CrazyLionCat/sms4go/sms"
)

type Config = config.FileConfig
type SMSConfig = config.SMSConfig
type SmsBlend = sms.SmsBlend

type Option func(*bootstrap) error

type bootstrap struct {
	factory        *core.Factory
	factoryOptions []core.Option
	configs        []*Config
}

func New(options ...Option) (*core.Factory, error) {
	var boot bootstrap
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&boot); err != nil {
			return nil, err
		}
	}
	factory := boot.factory
	if factory == nil {
		factory = core.NewFactory(boot.factoryOptions...)
	}
	for _, cfg := range boot.configs {
		if err := Register(factory, cfg); err != nil {
			return nil, err
		}
	}
	return factory, nil
}

func Init(options ...Option) error {
	_, err := New(append([]Option{WithFactory(core.DefaultFactory)}, options...)...)
	return err
}

func WithFactory(factory *core.Factory) Option {
	return func(boot *bootstrap) error {
		if factory != nil {
			boot.factory = factory
		}
		return nil
	}
}

func WithFactoryOptions(options ...core.Option) Option {
	return func(boot *bootstrap) error {
		boot.factoryOptions = append(boot.factoryOptions, options...)
		return nil
	}
}

func WithConfig(cfg *Config) Option {
	return func(boot *bootstrap) error {
		if cfg != nil {
			boot.configs = append(boot.configs, cfg)
		}
		return nil
	}
}

func WithYAML(path string) Option {
	return func(boot *bootstrap) error {
		cfg, err := LoadYAML(path)
		if err != nil {
			return err
		}
		boot.configs = append(boot.configs, cfg)
		return nil
	}
}

func WithJSON(path string) Option {
	return func(boot *bootstrap) error {
		cfg, err := LoadJSON(path)
		if err != nil {
			return err
		}
		boot.configs = append(boot.configs, cfg)
		return nil
	}
}

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
	return New(WithFactoryOptions(options...), WithYAML(path))
}

func NewFactoryFromJSON(path string, options ...core.Option) (*core.Factory, error) {
	return New(WithFactoryOptions(options...), WithJSON(path))
}

func Get(configID string) (SmsBlend, error) {
	return core.SmsBlend(configID)
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
