package provider

import (
	"net/http"
)

type BrazeProviderOption func(*brazeProvider)

func WithBaseURL(url string) BrazeProviderOption {
	return func(p *brazeProvider) {
		p.baseURL = url
	}
}

func WithHTTPClient(httpClient *http.Client) BrazeProviderOption {
	return func(p *brazeProvider) {
		p.httpClient = httpClient
	}
}

func WithAPIKey(apiKey string) BrazeProviderOption {
	return func(p *brazeProvider) {
		p.apiKey = apiKey
	}
}
