package sms

import (
	"sync"
	"time"
)

type Dao interface {
	Set(key string, value any, ttl time.Duration)
	SetDefault(key string, value any)
	Get(key string) (any, bool)
	Remove(key string) (any, bool)
	Clean()
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

type MemoryDao struct {
	mu         sync.RWMutex
	data       map[string]cacheEntry
	defaultTTL time.Duration
	stop       chan struct{}
}

func NewMemoryDao(defaultTTL time.Duration) *MemoryDao {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour
	}
	dao := &MemoryDao{
		data:       make(map[string]cacheEntry),
		defaultTTL: defaultTTL,
		stop:       make(chan struct{}),
	}
	go dao.cleanExpiredEvery(30 * time.Second)
	return dao
}

func (d *MemoryDao) Set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		ttl = d.defaultTTL
	}
	d.mu.Lock()
	d.data[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
	d.mu.Unlock()
}

func (d *MemoryDao) SetDefault(key string, value any) {
	d.Set(key, value, d.defaultTTL)
}

func (d *MemoryDao) Get(key string) (any, bool) {
	d.mu.RLock()
	entry, ok := d.data[key]
	d.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		d.Remove(key)
		return nil, false
	}
	return entry.value, true
}

func (d *MemoryDao) Remove(key string) (any, bool) {
	d.mu.Lock()
	entry, ok := d.data[key]
	if ok {
		delete(d.data, key)
	}
	d.mu.Unlock()
	return entry.value, ok
}

func (d *MemoryDao) Clean() {
	d.mu.Lock()
	d.data = make(map[string]cacheEntry)
	d.mu.Unlock()
}

func (d *MemoryDao) Close() {
	close(d.stop)
}

func (d *MemoryDao) cleanExpiredEvery(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d.cleanExpired()
		case <-d.stop:
			return
		}
	}
}

func (d *MemoryDao) cleanExpired() {
	now := time.Now()
	d.mu.Lock()
	for key, entry := range d.data {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			delete(d.data, key)
		}
	}
	d.mu.Unlock()
}
