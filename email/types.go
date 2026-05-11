package email

import "time"

type SMTPConfig struct {
	Port          string
	FromAddress   string
	NickName      string
	SMTPServer    string
	Username      string
	Password      string
	SSL           bool
	Auth          bool
	RetryInterval time.Duration
	MaxRetries    int
}

type IMAPConfig struct {
	IMAPServer  string
	Username    string
	AccessToken string
	Cycle       time.Duration
}

type Message struct {
	To          []string
	Title       string
	Body        string
	HTMLContent string
	HTMLValues  map[string]string
	CC          []string
	BCC         []string
	Files       map[string]string
	ZipName     string
}

type MonitorMessage struct {
	Title        string
	Text         string
	HTMLText     string
	SendDate     time.Time
	FromAddress  string
	MessageIndex int
	AcceptTime   time.Time
}

type Blacklist interface {
	GetBlacklist() []string
}

type Monitor func(MonitorMessage) bool
