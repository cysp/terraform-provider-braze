package provider

import (
	"fmt"
	"net/http"
)

type HTTPClientWithUserAgent struct {
	client *http.Client

	UserAgent string
}

func NewHTTPClientWithUserAgent(client *http.Client, userAgent string) *HTTPClientWithUserAgent {
	return &HTTPClientWithUserAgent{
		client:    client,
		UserAgent: userAgent,
	}
}

func (c *HTTPClientWithUserAgent) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" && c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client do: %w", err)
	}

	return resp, nil
}
