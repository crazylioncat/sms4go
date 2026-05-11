package provider

import (
	"fmt"
	"strings"
	"sync"

	"github.com/CrazyLionCat/sms4go/sms"
)

type Creator func(sms.BaseConfig) (sms.Client, error)

var (
	mu       sync.RWMutex
	creators = map[string]Creator{}
)

func Register(supplier string, creator Creator) {
	key := normalizeSupplier(supplier)
	if key == "" {
		panic("sms4go: supplier is empty")
	}
	if creator == nil {
		panic("sms4go: provider creator is nil")
	}
	mu.Lock()
	creators[key] = creator
	mu.Unlock()
}

func Create(config sms.BaseConfig) (sms.Client, error) {
	supplier := normalizeSupplier(config.Supplier)
	mu.RLock()
	creator, ok := creators[supplier]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sms4go: unsupported supplier %q", config.Supplier)
	}
	return creator(config)
}

func Supported() []string {
	mu.RLock()
	defer mu.RUnlock()
	suppliers := make([]string, 0, len(creators))
	for supplier := range creators {
		suppliers = append(suppliers, supplier)
	}
	return suppliers
}

func normalizeSupplier(supplier string) string {
	return strings.ToLower(strings.TrimSpace(supplier))
}
