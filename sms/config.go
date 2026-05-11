package sms

import "time"

// ProxyConfig mirrors sms4j's per-provider proxy option.
type ProxyConfig struct {
	Enable bool   `json:"enable" yaml:"enable"`
	Host   string `json:"host" yaml:"host"`
	Port   int    `json:"port" yaml:"port"`
}

// BaseConfig contains common fields used by most providers.
type BaseConfig struct {
	ConfigID        string            `json:"configId" yaml:"configId"`
	Supplier        string            `json:"supplier" yaml:"supplier"`
	Factory         string            `json:"factory,omitempty" yaml:"factory,omitempty"`
	AccessKeyID     string            `json:"accessKeyId" yaml:"accessKeyId"`
	AccessKeySecret string            `json:"accessKeySecret" yaml:"accessKeySecret"`
	SDKAppID        string            `json:"sdkAppId,omitempty" yaml:"sdkAppId,omitempty"`
	Signature       string            `json:"signature" yaml:"signature"`
	TemplateID      string            `json:"templateId" yaml:"templateId"`
	TemplateName    string            `json:"templateName" yaml:"templateName"`
	RequestURL      string            `json:"requestUrl" yaml:"requestUrl"`
	URL             string            `json:"url,omitempty" yaml:"url,omitempty"`
	Action          string            `json:"action,omitempty" yaml:"action,omitempty"`
	Version         string            `json:"version,omitempty" yaml:"version,omitempty"`
	Territory       string            `json:"territory,omitempty" yaml:"territory,omitempty"`
	Service         string            `json:"service,omitempty" yaml:"service,omitempty"`
	Sender          string            `json:"sender,omitempty" yaml:"sender,omitempty"`
	StatusCallBack  string            `json:"statusCallBack,omitempty" yaml:"statusCallBack,omitempty"`
	CallbackURL     string            `json:"callbackUrl,omitempty" yaml:"callbackUrl,omitempty"`
	BaseURL         string            `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	Host            string            `json:"host,omitempty" yaml:"host,omitempty"`
	API             string            `json:"api,omitempty" yaml:"api,omitempty"`
	SingleMsgURL    string            `json:"singleMsgUrl,omitempty" yaml:"singleMsgUrl,omitempty"`
	MassMsgURL      string            `json:"massMsgUrl,omitempty" yaml:"massMsgUrl,omitempty"`
	SignatureID     string            `json:"signatureId,omitempty" yaml:"signatureId,omitempty"`
	TemplateParam   string            `json:"templateParam,omitempty" yaml:"templateParam,omitempty"`
	EnableMD5       bool              `json:"enableMd5,omitempty" yaml:"enableMd5,omitempty"`
	SignType        string            `json:"signType,omitempty" yaml:"signType,omitempty"`
	SignVersion     string            `json:"signVersion,omitempty" yaml:"signVersion,omitempty"`
	SignKey         string            `json:"signKey,omitempty" yaml:"signKey,omitempty"`
	IsSimple        bool              `json:"isSimple,omitempty" yaml:"isSimple,omitempty"`
	Custom          string            `json:"custom,omitempty" yaml:"custom,omitempty"`
	UserExtID       string            `json:"userExtId,omitempty" yaml:"userExtId,omitempty"`
	TemplateURL     string            `json:"templateUrl,omitempty" yaml:"templateUrl,omitempty"`
	CodeURL         string            `json:"codeUrl,omitempty" yaml:"codeUrl,omitempty"`
	VerifyURL       string            `json:"verifyUrl,omitempty" yaml:"verifyUrl,omitempty"`
	NeedUp          bool              `json:"needUp,omitempty" yaml:"needUp,omitempty"`
	SignID          string            `json:"signId,omitempty" yaml:"signId,omitempty"`
	Voice           string            `json:"voice,omitempty" yaml:"voice,omitempty"`
	TTL             int               `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	Tag             string            `json:"tag,omitempty" yaml:"tag,omitempty"`
	BaseAction      string            `json:"baseAction,omitempty" yaml:"baseAction,omitempty"`
	TemplateAction  string            `json:"templateAction,omitempty" yaml:"templateAction,omitempty"`
	SpCode          string            `json:"spCode,omitempty" yaml:"spCode,omitempty"`
	SignCode        string            `json:"signCode,omitempty" yaml:"signCode,omitempty"`
	F               string            `json:"f,omitempty" yaml:"f,omitempty"`
	MchID           string            `json:"mchId,omitempty" yaml:"mchId,omitempty"`
	AppKey          string            `json:"appKey,omitempty" yaml:"appKey,omitempty"`
	AppID           string            `json:"appId,omitempty" yaml:"appId,omitempty"`
	ECName          string            `json:"ecName,omitempty" yaml:"ecName,omitempty"`
	AddSerial       string            `json:"addSerial,omitempty" yaml:"addSerial,omitempty"`
	Region          string            `json:"region,omitempty" yaml:"region,omitempty"`
	Weight          int               `json:"weight" yaml:"weight"`
	Timeout         time.Duration     `json:"timeout" yaml:"timeout"`
	RetryInterval   time.Duration     `json:"retryInterval" yaml:"retryInterval"`
	MaxRetries      int               `json:"maxRetries" yaml:"maxRetries"`
	Maximum         int               `json:"maximum" yaml:"maximum"`
	Proxy           *ProxyConfig      `json:"proxy,omitempty" yaml:"proxy,omitempty"`
	Extra           map[string]string `json:"extra,omitempty" yaml:"extra,omitempty"`
}

func (c BaseConfig) ID() string {
	if c.ConfigID != "" {
		return c.ConfigID
	}
	return c.Supplier
}

func (c BaseConfig) LoadWeight() int {
	if c.Weight <= 0 {
		return 1
	}
	return c.Weight
}
