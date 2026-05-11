package core

import (
	"context"
	"testing"

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
