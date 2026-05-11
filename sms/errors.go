package sms

import "errors"

var (
	ErrNilClient      = errors.New("sms4j: client is nil")
	ErrClientNotFound = errors.New("sms4j: client not found")
	ErrNoClients      = errors.New("sms4j: no clients registered")
	ErrInvalidPhone   = errors.New("sms4j: invalid phone")
	ErrEmptyMessage   = errors.New("sms4j: message is empty")
	ErrEmptyTemplate  = errors.New("sms4j: template id is empty")
	ErrBlacklisted    = errors.New("sms4j: phone is blacklisted")
)
