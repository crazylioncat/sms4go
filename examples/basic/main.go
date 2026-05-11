package main

import (
	"context"
	"fmt"
	"time"

	"github.com/CrazyLionCat/sms4go/core"
	"github.com/CrazyLionCat/sms4go/provider/mock"
	"github.com/CrazyLionCat/sms4go/sms"
)

func main() {
	factory := core.NewFactory(
		core.WithBlacklist(core.NewBlacklist(sms.NewMemoryDao(24*time.Hour), 24*time.Hour)),
	)

	client := mock.New(sms.BaseConfig{ConfigID: "mock-1", Supplier: mock.Supplier, Weight: 1})
	if err := factory.Register(client, client.Config.LoadWeight()); err != nil {
		panic(err)
	}

	resp, err := factory.SendMessage(context.Background(), "mock-1", "18888888888", "123456")
	fmt.Printf("resp=%+v err=%v\n", resp, err)
}
