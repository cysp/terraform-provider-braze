package provider

import (
	"fmt"
	"net/http"
)

type HTTPClientWithUserAgent struct {
	client httpDoer

	UserAgent string
}

func NewHTTPClientWithUserAgent(client httpDoer, userAgent string) *HTTPClientWithUserAgent {
	return &HTTPClientWithUserAgent{
		client:    client,
		UserAgent: userAgent,
	}
}

func (c *HTTPClientWithUserAgent) Do(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if req.Header == nil {
		req.Header = make(http.Header)
	}

	if req.Header.Get("User-Agent") == "" && c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client do: %w", err)
	}

	return resp, nil
}
