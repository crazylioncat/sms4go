package oa

import "errors"

var (
	ErrSenderNotFound = errors.New("sms4go/oa: sender not found")
	ErrEmptyContent   = errors.New("sms4go/oa: content is empty")
	ErrMessageType    = errors.New("sms4go/oa: unsupported message type")
)
