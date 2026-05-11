package starter_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/CrazyLionCat/sms4go/provider/mock"
	"github.com/CrazyLionCat/sms4go/starter"
)

func TestNewFactoryFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sms.yml")

	content := []byte(`sms:
  config-type: yaml
  blends:
    mock:
      access-key-id: ak
      retry-interval: 5
      timeout: 10000
      weight: 2
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := starter.LoadYAML(path)
	if err != nil {
		t.Fatalf("load yaml: %v", err)
	}
	loaded := cfg.SMS.Blends["mock"]
	if loaded.Supplier != "mock" {
		t.Fatalf("expected default supplier mock, got %q", loaded.Supplier)
	}
	if loaded.AccessKeyID != "ak" {
		t.Fatalf("expected kebab-case access key to load, got %q", loaded.AccessKeyID)
	}
	if loaded.RetryInterval != 5*time.Second {
		t.Fatalf("expected retry interval 5s, got %s", loaded.RetryInterval)
	}
	if loaded.Timeout != 10*time.Second {
		t.Fatalf("expected timeout 10s, got %s", loaded.Timeout)
	}

	factory, err := starter.NewFactoryFromYAML(path)
	if err != nil {
		t.Fatalf("new factory: %v", err)
	}

	blend, err := starter.GetSmsBlend(factory, "mock")
	if err != nil {
		t.Fatalf("get sms blend: %v", err)
	}

	resp, err := blend.SendMessage(context.Background(), "18888888888", "123456")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if resp.ConfigID != "mock" {
		t.Fatalf("expected config id mock, got %q", resp.ConfigID)
	}
}
