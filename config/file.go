package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/CrazyLionCat/sms4go/sms"
)

type FileConfig struct {
	SMS SMSConfig `json:"sms" yaml:"sms"`
}

type SMSConfig struct {
	Blends map[string]sms.BaseConfig `json:"blends" yaml:"blends"`
}

type Decoder func([]byte) (*FileConfig, error)

func Load(path string, decode Decoder) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decode(data)
}

func LoadJSON(path string) (*FileConfig, error) {
	return Load(path, DecodeJSON)
}

func DecodeJSON(data []byte) (*FileConfig, error) {
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	normalizeConfigIDs(&cfg)
	return &cfg, nil
}

func LoadYAML(path string) (*FileConfig, error) {
	return Load(path, DecodeYAML)
}

func DecodeYAML(data []byte) (*FileConfig, error) {
	var raw struct {
		SMS struct {
			Blends map[string]map[string]any `yaml:"blends"`
		} `yaml:"sms"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	cfg := FileConfig{SMS: SMSConfig{Blends: make(map[string]sms.BaseConfig, len(raw.SMS.Blends))}}
	for id, rawBlend := range raw.SMS.Blends {
		data, err := json.Marshal(normalizeYAMLConfigMap(rawBlend))
		if err != nil {
			return nil, err
		}
		var blend sms.BaseConfig
		if err := json.Unmarshal(data, &blend); err != nil {
			return nil, err
		}
		cfg.SMS.Blends[id] = blend
	}
	normalizeConfigIDs(&cfg)
	return &cfg, nil
}

func normalizeConfigIDs(cfg *FileConfig) {
	if cfg == nil || cfg.SMS.Blends == nil {
		return
	}
	for id, blend := range cfg.SMS.Blends {
		if blend.ConfigID == "" {
			blend.ConfigID = id
		}
		if blend.Supplier == "" {
			blend.Supplier = id
		}
		cfg.SMS.Blends[id] = blend
	}
}

func normalizeYAMLConfigMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		normalizedKey := normalizeYAMLKey(key)
		if normalizedKey == "extra" {
			output[normalizedKey] = value
			continue
		}
		output[normalizedKey] = normalizeYAMLValue(normalizedKey, value)
	}
	return output
}

func normalizeYAMLValue(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeYAMLConfigMap(typed)
	case []any:
		values := make([]any, len(typed))
		for i, item := range typed {
			values[i] = normalizeYAMLValue("", item)
		}
		return values
	}
	switch key {
	case "timeout":
		return normalizeDurationValue(value, time.Millisecond)
	case "retryInterval":
		return normalizeDurationValue(value, time.Second)
	default:
		return value
	}
}

func normalizeYAMLKey(key string) string {
	if !strings.ContainsAny(key, "-_") {
		return key
	}
	parts := strings.FieldsFunc(key, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return key
	}
	var builder strings.Builder
	builder.WriteString(strings.ToLower(parts[0]))
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}
	return builder.String()
}

func normalizeDurationValue(value any, unit time.Duration) any {
	switch typed := value.(type) {
	case int:
		return int64(typed) * int64(unit)
	case int64:
		return typed * int64(unit)
	case int32:
		return int64(typed) * int64(unit)
	case uint:
		return int64(typed) * int64(unit)
	case uint64:
		return int64(typed) * int64(unit)
	case uint32:
		return int64(typed) * int64(unit)
	case float64:
		return int64(typed * float64(unit))
	case float32:
		return int64(float64(typed) * float64(unit))
	case string:
		if duration, err := time.ParseDuration(typed); err == nil {
			return int64(duration)
		}
		if number, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return number * int64(unit)
		}
	}
	return value
}
