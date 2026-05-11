package core

import (
	"strings"
	"time"

	"github.com/CrazyLionCat/sms4go/sms"
)

const blacklistPrefix = "sms4j:blacklist:"

type Blacklist struct {
	dao sms.Dao
	ttl time.Duration
}

func NewBlacklist(dao sms.Dao, ttl time.Duration) *Blacklist {
	if dao == nil {
		dao = sms.NewMemoryDao(24 * time.Hour)
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Blacklist{dao: dao, ttl: ttl}
}

func (b *Blacklist) Join(phone string) {
	b.dao.Set(b.key(phone), true, b.ttl)
}

func (b *Blacklist) Remove(phone string) {
	b.dao.Remove(b.key(phone))
}

func (b *Blacklist) BatchJoin(phones []string) {
	for _, phone := range phones {
		b.Join(phone)
	}
}

func (b *Blacklist) BatchRemove(phones []string) {
	for _, phone := range phones {
		b.Remove(phone)
	}
}

func (b *Blacklist) Contains(phone string) bool {
	_, ok := b.dao.Get(b.key(phone))
	return ok
}

func (b *Blacklist) key(phone string) string {
	return blacklistPrefix + strings.TrimSpace(phone)
}
