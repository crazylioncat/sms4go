package oa

import (
	"container/heap"
	"context"
	"sync"
)

type Factory struct {
	mu      sync.RWMutex
	senders map[string]Sender
	queue   priorityQueue
	wake    chan struct{}
}

func NewFactory() *Factory {
	f := &Factory{
		senders: make(map[string]Sender),
		wake:    make(chan struct{}, 1),
	}
	heap.Init(&f.queue)
	go f.dispatchPriority()
	return f
}

func (f *Factory) Register(sender Sender) {
	f.mu.Lock()
	f.senders[sender.ConfigID()] = sender
	f.mu.Unlock()
}

func (f *Factory) Get(configID string) (Sender, bool) {
	f.mu.RLock()
	sender, ok := f.senders[configID]
	f.mu.RUnlock()
	return sender, ok
}

func (f *Factory) Send(ctx context.Context, configID string, request Request, messageType MessageType) (*Response, error) {
	sender, ok := f.Get(configID)
	if !ok {
		return nil, ErrSenderNotFound
	}
	return sender.Send(ctx, request, messageType)
}

func (f *Factory) SendAsync(ctx context.Context, configID string, request Request, messageType MessageType, callback Callback) {
	go func() {
		resp, err := f.Send(ctx, configID, request, messageType)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

func (f *Factory) SendByPriority(configID string, request Request, messageType MessageType) {
	request.MessageType = messageType
	f.mu.Lock()
	heap.Push(&f.queue, priorityItem{configID: configID, request: request})
	f.mu.Unlock()
	select {
	case f.wake <- struct{}{}:
	default:
	}
}

func (f *Factory) dispatchPriority() {
	for range f.wake {
		for {
			f.mu.Lock()
			if f.queue.Len() == 0 {
				f.mu.Unlock()
				break
			}
			item := heap.Pop(&f.queue).(priorityItem)
			f.mu.Unlock()
			go f.Send(context.Background(), item.configID, item.request, item.request.MessageType)
		}
	}
}

var DefaultFactory = NewFactory()

func Register(sender Sender) {
	DefaultFactory.Register(sender)
}

type priorityItem struct {
	configID string
	request  Request
}

type priorityQueue []priorityItem

func (q priorityQueue) Len() int { return len(q) }

func (q priorityQueue) Less(i, j int) bool {
	return q[i].request.Priority > q[j].request.Priority
}

func (q priorityQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *priorityQueue) Push(x any) {
	*q = append(*q, x.(priorityItem))
}

func (q *priorityQueue) Pop() any {
	old := *q
	item := old[len(old)-1]
	*q = old[:len(old)-1]
	return item
}
