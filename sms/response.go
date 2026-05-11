package sms

// Response is the normalized result returned by all SMS providers.
type Response struct {
	Success  bool   `json:"success" yaml:"success"`
	Data     any    `json:"data,omitempty" yaml:"data,omitempty"`
	ConfigID string `json:"configId,omitempty" yaml:"configId,omitempty"`
}

func Success(data any, configID string) *Response {
	return &Response{Success: true, Data: data, ConfigID: configID}
}

func Failure(data any, configID string) *Response {
	return &Response{Success: false, Data: data, ConfigID: configID}
}
