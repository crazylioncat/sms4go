package core

import (
	"context"
	"sync"
	"time"

	"github.com/CrazyLionCat/sms4go/sms"
)

type Factory struct {
	mu          sync.RWMutex
	clients     map[string]sms.Client
	load        *LoadBalancer
	middleware  []Middleware
	handler     Handler
	blacklist   *Blacklist
	phoneVerify sms.PhoneVerifier
}

func NewFactory(options ...Option) *Factory {
	f := &Factory{
		clients: make(map[string]sms.Client),
		load:    NewLoadBalancer(),
	}
	for _, option := range options {
		option(f)
	}
	f.rebuildHandler()
	return f
}

type Option func(*Factory)

func WithPhoneVerifier(verifier sms.PhoneVerifier) Option {
	return func(f *Factory) {
		f.phoneVerify = verifier
	}
}

func WithBlacklist(blacklist *Blacklist) Option {
	return func(f *Factory) {
		f.blacklist = blacklist
	}
}

func WithMiddleware(middleware ...Middleware) Option {
	return func(f *Factory) {
		f.middleware = append(f.middleware, middleware...)
	}
}

func (f *Factory) Register(client sms.Client, weight int) error {
	if client == nil {
		return sms.ErrNilClient
	}
	f.mu.Lock()
	f.clients[client.ConfigID()] = client
	f.load.Add(client, weight)
	f.mu.Unlock()
	return nil
}

func (f *Factory) RegisterIfAbsent(client sms.Client, weight int) (bool, error) {
	if client == nil {
		return false, sms.ErrNilClient
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.clients[client.ConfigID()]; ok {
		return false, nil
	}
	f.clients[client.ConfigID()] = client
	f.load.Add(client, weight)
	return true, nil
}

func (f *Factory) Unregister(configID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.clients[configID]; !ok {
		return false
	}
	delete(f.clients, configID)
	f.load.Remove(configID)
	return true
}

func (f *Factory) Get(configID string) (sms.Client, bool) {
	f.mu.RLock()
	client, ok := f.clients[configID]
	f.mu.RUnlock()
	return client, ok
}

func (f *Factory) GetBySupplier(supplier string) (sms.Client, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, client := range f.clients {
		if client.Supplier() == supplier {
			return client, true
		}
	}
	return nil, false
}

func (f *Factory) ListBySupplier(supplier string) []sms.Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var clients []sms.Client
	for _, client := range f.clients {
		if client.Supplier() == supplier {
			clients = append(clients, client)
		}
	}
	return clients
}

func (f *Factory) All() []sms.Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	clients := make([]sms.Client, 0, len(f.clients))
	for _, client := range f.clients {
		clients = append(clients, client)
	}
	return clients
}

func (f *Factory) Next() (sms.Client, error) {
	client, ok := f.load.Next()
	if !ok {
		return nil, sms.ErrNoClients
	}
	return client, nil
}

func (f *Factory) Use(middleware ...Middleware) {
	f.mu.Lock()
	f.middleware = append(f.middleware, middleware...)
	f.rebuildHandler()
	f.mu.Unlock()
}

func (f *Factory) Block(phone string) {
	if f.blacklist != nil {
		f.blacklist.Join(phone)
	}
}

func (f *Factory) Unblock(phone string) {
	if f.blacklist != nil {
		f.blacklist.Remove(phone)
	}
}

func (f *Factory) BlockAll(phones []string) {
	if f.blacklist != nil {
		f.blacklist.BatchJoin(phones)
	}
}

func (f *Factory) UnblockAll(phones []string) {
	if f.blacklist != nil {
		f.blacklist.BatchRemove(phones)
	}
}

func (f *Factory) JoinInBlacklist(phone string) {
	f.Block(phone)
}

func (f *Factory) RemoveFromBlacklist(phone string) {
	f.Unblock(phone)
}

func (f *Factory) BatchJoinBlacklist(phones []string) {
	f.BlockAll(phones)
}

func (f *Factory) BatchRemovalFromBlacklist(phones []string) {
	f.UnblockAll(phones)
}

func (f *Factory) SendMessage(ctx context.Context, configID string, phone string, message string) (*sms.Response, error) {
	client, err := f.clientFor(configID)
	if err != nil {
		return nil, err
	}
	return f.call(ctx, client, Request{Operation: OperationSendMessage, Phone: phone, Message: message})
}

func (f *Factory) SendMessageWithParams(ctx context.Context, configID string, phone string, messages map[string]string) (*sms.Response, error) {
	client, err := f.clientFor(configID)
	if err != nil {
		return nil, err
	}
	return f.call(ctx, client, Request{Operation: OperationSendMessageParams, Phone: phone, Messages: messages})
}

func (f *Factory) SendTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string) (*sms.Response, error) {
	client, err := f.clientFor(configID)
	if err != nil {
		return nil, err
	}
	return f.call(ctx, client, Request{Operation: OperationSendTemplate, Phone: phone, TemplateID: templateID, Messages: messages})
}

func (f *Factory) MassTexting(ctx context.Context, configID string, phones []string, message string) (*sms.Response, error) {
	client, err := f.clientFor(configID)
	if err != nil {
		return nil, err
	}
	return f.call(ctx, client, Request{Operation: OperationMassTexting, Phones: phones, Message: message})
}

func (f *Factory) MassTextingTemplate(ctx context.Context, configID string, phones []string, templateID string, messages map[string]string) (*sms.Response, error) {
	client, err := f.clientFor(configID)
	if err != nil {
		return nil, err
	}
	return f.call(ctx, client, Request{Operation: OperationMassTextingTemplate, Phones: phones, TemplateID: templateID, Messages: messages})
}

func (f *Factory) SendMessageAsync(ctx context.Context, configID string, phone string, message string, callback sms.Callback) {
	go func() {
		resp, err := f.SendMessage(ctx, configID, phone, message)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

func (f *Factory) SendMessageChan(ctx context.Context, configID string, phone string, message string) <-chan sms.Result {
	return f.run(func() (*sms.Response, error) {
		return f.SendMessage(ctx, configID, phone, message)
	})
}

func (f *Factory) SendTemplateAsync(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, callback sms.Callback) {
	go func() {
		resp, err := f.SendTemplate(ctx, configID, phone, templateID, messages)
		if callback != nil {
			callback(resp, err)
		}
	}()
}

func (f *Factory) SendTemplateChan(ctx context.Context, configID string, phone string, templateID string, messages map[string]string) <-chan sms.Result {
	return f.run(func() (*sms.Response, error) {
		return f.SendTemplate(ctx, configID, phone, templateID, messages)
	})
}

func (f *Factory) DelayedMessage(ctx context.Context, configID string, phone string, message string, delay time.Duration, callback sms.Callback) *time.Timer {
	return time.AfterFunc(delay, func() {
		resp, err := f.SendMessage(ctx, configID, phone, message)
		if callback != nil {
			callback(resp, err)
		}
	})
}

func (f *Factory) DelayMessage(ctx context.Context, configID string, phone string, message string, delay time.Duration) <-chan sms.Result {
	ch := make(chan sms.Result, 1)
	time.AfterFunc(delay, func() {
		resp, err := f.SendMessage(ctx, configID, phone, message)
		ch <- sms.Result{Response: resp, Err: err}
		close(ch)
	})
	return ch
}

func (f *Factory) DelayedTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, delay time.Duration, callback sms.Callback) *time.Timer {
	return time.AfterFunc(delay, func() {
		resp, err := f.SendTemplate(ctx, configID, phone, templateID, messages)
		if callback != nil {
			callback(resp, err)
		}
	})
}

func (f *Factory) DelayTemplate(ctx context.Context, configID string, phone string, templateID string, messages map[string]string, delay time.Duration) <-chan sms.Result {
	ch := make(chan sms.Result, 1)
	time.AfterFunc(delay, func() {
		resp, err := f.SendTemplate(ctx, configID, phone, templateID, messages)
		ch <- sms.Result{Response: resp, Err: err}
		close(ch)
	})
	return ch
}

func (f *Factory) run(send func() (*sms.Response, error)) <-chan sms.Result {
	ch := make(chan sms.Result, 1)
	go func() {
		resp, err := send()
		ch <- sms.Result{Response: resp, Err: err}
		close(ch)
	}()
	return ch
}

func (f *Factory) clientFor(configID string) (sms.Client, error) {
	if configID == "" {
		return f.Next()
	}
	client, ok := f.Get(configID)
	if !ok {
		return nil, sms.ErrClientNotFound
	}
	return client, nil
}

func (f *Factory) call(ctx context.Context, client sms.Client, req Request) (*sms.Response, error) {
	f.mu.RLock()
	handler := f.handler
	f.mu.RUnlock()
	return handler(ctx, client, req)
}

func (f *Factory) rebuildHandler() {
	middleware := []Middleware{ValidationMiddleware(f.phoneVerify)}
	if f.blacklist != nil {
		middleware = append(middleware, BlacklistMiddleware(f.blacklist))
	}
	middleware = append(middleware, f.middleware...)
	f.handler = Chain(dispatch, middleware...)
}

func dispatch(ctx context.Context, client sms.Client, req Request) (*sms.Response, error) {
	switch req.Operation {
	case OperationSendMessage:
		return client.SendMessage(ctx, req.Phone, req.Message)
	case OperationSendMessageParams:
		return client.SendMessageWithParams(ctx, req.Phone, req.Messages)
	case OperationSendTemplate:
		return client.SendTemplate(ctx, req.Phone, req.TemplateID, req.Messages)
	case OperationMassTexting:
		return client.MassTexting(ctx, req.Phones, req.Message)
	case OperationMassTextingTemplate:
		return client.MassTextingTemplate(ctx, req.Phones, req.TemplateID, req.Messages)
	default:
		return nil, sms.ErrClientNotFound
	}
}
