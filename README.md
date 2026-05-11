# sms4go

sms4go 是 sms4j 的 Go 版实现，保留原项目中与 Java 框架无关的核心能力，不包含 Spring Boot、Solon 等 Java 生态适配。

实现原则：以 sms4j 已经完成真实联调的 Java 实现为行为基准，Go 版尽量保持相同的参数构造、签名逻辑、请求路径、响应成功判断和重试策略。

当前模块路径：

```text
github.com/CrazyLionCat/sms4go
```

## 安装

```bash
go get github.com/CrazyLionCat/sms4go
```

本仓库当前不依赖第三方 Go 包，核心功能基于 Go 标准库实现。

## 快速开始

```go
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

	client := mock.New(sms.BaseConfig{
		ConfigID: "mock-1",
		Supplier: mock.Supplier,
		Weight:   1,
	})

	if err := factory.Register(client, client.Config.LoadWeight()); err != nil {
		panic(err)
	}

	resp, err := factory.SendMessage(context.Background(), "mock-1", "18888888888", "123456")
	fmt.Printf("resp=%+v err=%v\n", resp, err)
}
```

## Provider 注册

provider 包会通过 `init` 自动注册。使用配置创建 provider 时，需要对要启用的 provider 做空导入：

```go
import (
	_ "github.com/CrazyLionCat/sms4go/provider/aliyun"
	_ "github.com/CrazyLionCat/sms4go/provider/tencent"
)
```

手动创建 provider 时，直接调用对应 provider 的 `New` 方法即可：

```go
client, err := aliyun.New(sms.BaseConfig{
	ConfigID:        "aliyun-main",
	AccessKeyID:     "your-access-key-id",
	AccessKeySecret: "your-access-key-secret",
	Signature:       "短信签名",
	TemplateID:      "SMS_123456",
	TemplateName:    "code",
})
```

## JSON 配置加载

配置文件示例：

```json
{
  "sms": {
    "blends": {
      "aliyun-main": {
        "supplier": "alibaba",
        "accessKeyId": "your-access-key-id",
        "accessKeySecret": "your-access-key-secret",
        "signature": "短信签名",
        "templateId": "SMS_123456",
        "templateName": "code",
        "weight": 1
      },
      "tencent-main": {
        "supplier": "tencent",
        "accessKeyId": "your-secret-id",
        "accessKeySecret": "your-secret-key",
        "sdkAppId": "your-sdk-app-id",
        "signature": "短信签名",
        "templateId": "123456",
        "weight": 1
      }
    }
  }
}
```

加载并注册：

```go
package main

import (
	"github.com/CrazyLionCat/sms4go/config"
	"github.com/CrazyLionCat/sms4go/core"

	_ "github.com/CrazyLionCat/sms4go/provider/aliyun"
	_ "github.com/CrazyLionCat/sms4go/provider/tencent"
)

func main() {
	cfg, err := config.LoadJSON("sms.json")
	if err != nil {
		panic(err)
	}

	factory := core.NewFactory()
	if err := config.RegisterBlends(factory, cfg); err != nil {
		panic(err)
	}
}
```

说明：当前内置配置加载只支持 JSON。`LoadYAML` 保留了函数入口，但会返回 unsupported 错误。

## 短信发送

按配置 ID 发送：

```go
resp, err := factory.SendMessage(ctx, "aliyun-main", "18888888888", "123456")
```

模板短信：

```go
resp, err := factory.SendTemplate(ctx, "aliyun-main", "18888888888", "SMS_123456", map[string]string{
	"code": "123456",
})
```

群发：

```go
resp, err := factory.MassTexting(ctx, "aliyun-main", []string{
	"18888888888",
	"16666666666",
}, "123456")
```

模板群发：

```go
resp, err := factory.MassTextingTemplate(ctx, "aliyun-main", []string{
	"18888888888",
	"16666666666",
}, "SMS_123456", map[string]string{
	"code": "123456",
})
```

如果 `configID` 传空字符串，会使用平滑加权负载均衡选择一个已注册客户端：

```go
resp, err := factory.SendMessage(ctx, "", "18888888888", "123456")
```

## 异步和延迟发送

异步发送：

```go
factory.SendMessageAsync(ctx, "aliyun-main", "18888888888", "123456", func(resp *sms.Response, err error) {
	// 处理发送结果
})
```

延迟发送：

```go
timer := factory.DelayedMessage(ctx, "aliyun-main", "18888888888", "123456", 10*time.Second, nil)

// 如需取消：
timer.Stop()
```

## 黑名单

```go
dao := sms.NewMemoryDao(24 * time.Hour)
blacklist := core.NewBlacklist(dao, 24*time.Hour)

factory := core.NewFactory(core.WithBlacklist(blacklist))

factory.JoinInBlacklist("18888888888")
factory.RemoveFromBlacklist("18888888888")
factory.BatchJoinBlacklist([]string{"18888888888", "16666666666"})
factory.BatchRemovalFromBlacklist([]string{"18888888888", "16666666666"})
```

黑名单会通过 middleware 在发送前拦截。

## 参数校验

默认会校验手机号非空、消息非空、模板 ID 非空等基础参数。

如需自定义手机号校验：

```go
factory := core.NewFactory(core.WithPhoneVerifier(sms.PhoneVerifierFunc(func(phone string) bool {
	return len(phone) == 11
})))
```

## 自定义 Middleware

```go
factory.Use(func(next core.Handler) core.Handler {
	return func(ctx context.Context, client sms.Client, req core.Request) (*sms.Response, error) {
		// 前置处理
		resp, err := next(ctx, client, req)
		// 后置处理
		return resp, err
	}
})
```

## 已创建的短信 Provider

- `mock`
- `alibaba`
- `tencent`
- `huawei`
- `yunpian`
- `qiniu`
- `huyi`
- `montnets`
- `luosimao`
- `zhutong`
- `chuanglan`
- `mysubmail`
- `unisms`
- `baidu`
- `buding_v2`
- `ctyun`
- `netease`
- `jiguang`
- `danmi`
- `dingzhong`
- `yixintong`
- `cloopen`
- `emay`
- `lianlu`
- `mas`
- `jdcloud`

注意：这些 provider 按 sms4j Java 版逻辑迁移，包括参数构造、签名逻辑、请求路径和响应成功判断。`jdcloud` 当前是包骨架，因为 Java 版依赖京东云 Java SDK，Go 版仍需要接入京东云 Go SDK 或补齐 OpenAPI 签名后才能真实发送。

## OA 通知

已支持：

- 钉钉：`text`、`markdown`、`link`
- 企业微信：`text`、`markdown`、`news`
- 飞书：`text`

### 钉钉示例

```go
package main

import (
	"context"

	"github.com/CrazyLionCat/sms4go/oa"
	"github.com/CrazyLionCat/sms4go/oa/provider/dingtalk"
)

func main() {
	sender := dingtalk.New(oa.Config{
		ConfigID: "ding-main",
		TokenID:  "your-access-token",
		Sign:     "your-secret",
	})

	resp, err := sender.Send(context.Background(), oa.Request{
		Title:   "系统通知",
		Content: "服务启动成功",
	}, oa.DingTalkText)

	_, _ = resp, err
}
```

### 使用 OA Factory

```go
factory := oa.NewFactory()
factory.Register(dingtalk.New(oa.Config{
	ConfigID: "ding-main",
	TokenID:  "your-access-token",
	Sign:     "your-secret",
}))

resp, err := factory.Send(context.Background(), "ding-main", oa.Request{
	Content: "服务启动成功",
}, oa.DingTalkText)
```

异步发送：

```go
factory.SendAsync(context.Background(), "ding-main", oa.Request{
	Content: "异步消息",
}, oa.DingTalkText, func(resp *oa.Response, err error) {
	// 处理结果
})
```

优先级发送：

```go
factory.SendByPriority("ding-main", oa.Request{
	Content:  "高优先级消息",
	Priority: 100,
}, oa.DingTalkText)
```

## 邮件发送

### SMTP 文本邮件

```go
package main

import "github.com/CrazyLionCat/sms4go/email"

func main() {
	client := email.NewClient(email.SMTPConfig{
		SMTPServer:  "smtp.example.com",
		Port:        "465",
		FromAddress: "sender@example.com",
		NickName:    "sms4go",
		Username:    "sender@example.com",
		Password:    "password-or-token",
		SSL:         true,
		Auth:        true,
		MaxRetries:  2,
	}, nil)

	err := client.Send(email.Message{
		To:    []string{"receiver@example.com"},
		Title: "测试邮件",
		Body:  "这是一封测试邮件",
	})

	_ = err
}
```

### HTML 邮件

```go
err := client.Send(email.Message{
	To:          []string{"receiver@example.com"},
	Title:       "验证码",
	HTMLContent: "<h1>验证码：{{code}}</h1>",
	HTMLValues: map[string]string{
		"code": "123456",
	},
})
```

### 附件和 zip

```go
err := client.Send(email.Message{
	To:    []string{"receiver@example.com"},
	Title: "附件邮件",
	Body:  "请查收附件",
	Files: map[string]string{
		"report.txt": "D:/tmp/report.txt",
		"image.png":  "https://example.com/image.png",
	},
})
```

打包为 zip：

```go
err := client.Send(email.Message{
	To:      []string{"receiver@example.com"},
	Title:   "附件压缩包",
	Body:    "请查收附件",
	ZipName: "files.zip",
	Files: map[string]string{
		"report.txt": "D:/tmp/report.txt",
	},
})
```

### 邮件黑名单

```go
type MailBlacklist struct{}

func (MailBlacklist) GetBlacklist() []string {
	return []string{"blocked@example.com"}
}

client := email.NewClient(config, MailBlacklist{})
```

## IMAP 监听

IMAP 监听接口已保留：

```go
monitor := email.NewMonitorService(email.IMAPConfig{
	IMAPServer:  "imap.example.com",
	Username:    "receiver@example.com",
	AccessToken: "password-or-token",
}, func(message email.MonitorMessage) bool {
	return true
})

err := monitor.Start()
```

当前 `Start` 会返回 unsupported 错误。Go 标准库没有 IMAP 客户端，后续需要选择第三方 IMAP 依赖后再实现。

## 本地验证

```powershell
$env:GOCACHE='D:\git-sources\sms4j\sms4go\.gocache'; go test ./...
```

## 当前限制

- 短信 provider 应继续补充与 Java sms4j 行为一致的迁移测试，例如签名字符串、请求体、Header、URL 和成功响应判断。
- `jdcloud` 仍需要接入京东云 Go SDK 或补齐 OpenAPI 签名。
- 邮件 IMAP 监听尚未实现。
- 目前配置加载只内置 JSON，不内置 YAML。
- 需要继续增加签名算法单元测试和 HTTP mock 测试，确保 Go 版输出与 Java 版逻辑一致。
