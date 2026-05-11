package config

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/CrazyLionCat/sms4go/sms"
)

var ErrYAMLUnsupported = errors.New("sms4go: YAML loading requires a YAML decoder; use LoadJSON in the dependency-free core")

type FileConfig struct {
	SMS SMSConfig `json:"sms" yaml:"sms"`
}

type SMSConfig struct {
	Blends map[string]sms.BaseConfig `json:"blends" yaml:"blends"`
}

func LoadJSON(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	normalizeConfigIDs(&cfg)
	return &cfg, nil
}

func LoadYAML(path string) (*FileConfig, error) {
	return nil, ErrYAMLUnsupported
}

func normalizeConfigIDs(cfg *FileConfig) {
	for id, blend := range cfg.SMS.Blends {
		if blend.ConfigID == "" {
			blend.ConfigID = id
		}
		cfg.SMS.Blends[id] = blend
	}
}
