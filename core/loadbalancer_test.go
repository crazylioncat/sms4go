package core

import (
	"context"
	"testing"
	"time"

	"github.com/CrazyLionCat/sms4go/provider/mock"
	"github.com/CrazyLionCat/sms4go/sms"
)

func TestLoadBalancerSmoothWeight(t *testing.T) {
	load := NewLoadBalancer()
	load.Add(mock.New(sms.BaseConfig{ConfigID: "a", Supplier: mock.Supplier}), 5)
	load.Add(mock.New(sms.BaseConfig{ConfigID: "b", Supplier: mock.Supplier}), 1)

	counts := map[string]int{}
	for i := 0; i < 12; i++ {
		client, ok := load.Next()
		if !ok {
			t.Fatal("expected client")
		}
		counts[client.ConfigID()]++
	}

	if counts["a"] != 10 || counts["b"] != 2 {
		t.Fatalf("unexpected distribution: %#v", counts)
	}
}

func TestFactorySendMessage(t *testing.T) {
	factory := NewFactory()
	client := mock.New(sms.BaseConfig{ConfigID: "mock-1", Supplier: mock.Supplier})
	if err := factory.Register(client, 1); err != nil {
		t.Fatal(err)
	}

	resp, err := factory.SendMessage(context.Background(), "mock-1", "18888888888", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.ConfigID != "mock-1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestFactorySendMessageChan(t *testing.T) {
	factory := NewFactory()
	client := mock.New(sms.BaseConfig{ConfigID: "mock-chan", Supplier: mock.Supplier})
	if err := factory.Register(client, 1); err != nil {
		t.Fatal(err)
	}

	result := <-factory.SendMessageChan(context.Background(), "mock-chan", "18888888888", "123456")
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Response == nil || !result.Response.Success || result.Response.ConfigID != "mock-chan" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestFactoryBlockAliases(t *testing.T) {
	factory := NewFactory(WithBlacklist(NewBlacklist(sms.NewMemoryDao(time.Hour), time.Hour)))
	client := mock.New(sms.BaseConfig{ConfigID: "mock-block", Supplier: mock.Supplier})
	if err := factory.Register(client, 1); err != nil {
		t.Fatal(err)
	}

	factory.Block("18888888888")
	if _, err := factory.SendMessage(context.Background(), "mock-block", "18888888888", "123456"); err != sms.ErrBlacklisted {
		t.Fatalf("expected blacklisted error, got %v", err)
	}

	factory.Unblock("18888888888")
	resp, err := factory.SendMessage(context.Background(), "mock-block", "18888888888", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
