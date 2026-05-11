package config

import (
	"github.com/CrazyLionCat/sms4go/core"
	"github.com/CrazyLionCat/sms4go/provider"
)

func RegisterBlends(factory *core.Factory, cfg *FileConfig) error {
	if factory == nil {
		factory = core.DefaultFactory
	}
	if cfg == nil {
		return nil
	}
	for _, blend := range cfg.SMS.Blends {
		client, err := provider.Create(blend)
		if err != nil {
			return err
		}
		if err := factory.Register(client, blend.LoadWeight()); err != nil {
			return err
		}
	}
	return nil
}
