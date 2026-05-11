package base

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/CrazyLionCat/sms4go/sms"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(config sms.BaseConfig) (*HTTPClient, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.Proxy != nil && config.Proxy.Enable {
		proxyURL := &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(config.Proxy.Host, strconv.Itoa(config.Proxy.Port)),
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &HTTPClient{client: &http.Client{Timeout: timeout, Transport: transport}}, nil
}

func (h *HTTPClient) Do(req *http.Request) ([]byte, int, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (h *HTTPClient) Get(ctx context.Context, endpoint string, header http.Header) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header = header.Clone()
	return h.Do(req)
}

func (h *HTTPClient) PostJSON(ctx context.Context, endpoint string, payload any, header http.Header) ([]byte, int, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	return h.PostRaw(ctx, endpoint, data, "application/json", header)
}

func (h *HTTPClient) PostRaw(ctx context.Context, endpoint string, data []byte, contentType string, header http.Header) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header = header.Clone()
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return h.Do(req)
}

func (h *HTTPClient) PostForm(ctx context.Context, endpoint string, values url.Values, header http.Header) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header = header.Clone()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return h.Do(req)
}
