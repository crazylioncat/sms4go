package email

import "errors"

var (
	ErrNoRecipients       = errors.New("sms4go/email: no recipients")
	ErrMonitorUnsupported = errors.New("sms4go/email: IMAP monitoring requires an IMAP client dependency")
)
