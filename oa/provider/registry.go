package provider

import (
	"fmt"
	"strings"
	"sync"

	"github.com/CrazyLionCat/sms4go/oa"
)

type Creator func(oa.Config) (oa.Sender, error)

var (
	mu       sync.RWMutex
	creators = map[string]Creator{}
)

func Register(supplier string, creator Creator) {
	mu.Lock()
	creators[strings.ToLower(strings.TrimSpace(supplier))] = creator
	mu.Unlock()
}

func Create(config oa.Config) (oa.Sender, error) {
	mu.RLock()
	creator, ok := creators[strings.ToLower(strings.TrimSpace(config.Supplier))]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sms4go/oa: unsupported supplier %q", config.Supplier)
	}
	return creator(config)
}
