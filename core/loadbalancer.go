package core

import (
	"sync"

	"github.com/CrazyLionCat/sms4go/sms"
)

type weightedClient struct {
	client        sms.Client
	weight        int
	currentWeight int
}

// LoadBalancer implements the same smooth weighted round-robin strategy used by sms4j.
type LoadBalancer struct {
	mu      sync.Mutex
	clients []weightedClient
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{}
}

func (l *LoadBalancer) Add(client sms.Client, weight int) {
	if weight <= 0 {
		weight = 1
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.removeLocked(client.ConfigID())
	l.clients = append(l.clients, weightedClient{client: client, weight: weight, currentWeight: weight})
}

func (l *LoadBalancer) Remove(configID string) {
	l.mu.Lock()
	l.removeLocked(configID)
	l.mu.Unlock()
}

func (l *LoadBalancer) Next() (sms.Client, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.clients) == 0 {
		return nil, false
	}
	totalWeight := 0
	selected := -1
	for i := range l.clients {
		totalWeight += l.clients[i].weight
		l.clients[i].currentWeight += l.clients[i].weight
		if selected == -1 || l.clients[i].currentWeight > l.clients[selected].currentWeight {
			selected = i
		}
	}
	l.clients[selected].currentWeight -= totalWeight
	return l.clients[selected].client, true
}

func (l *LoadBalancer) removeLocked(configID string) {
	for i := range l.clients {
		if l.clients[i].client.ConfigID() == configID {
			l.clients = append(l.clients[:i], l.clients[i+1:]...)
			return
		}
	}
}
