package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type brazeHTTPClient struct {
	retryable *http.Client
	mutations *http.Client
}

func newBrazeHTTPClient(client *http.Client) *brazeHTTPClient {
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}

	return &brazeHTTPClient{
		retryable: newBrazeRetryClient(client, brazeRetryPolicy),
		mutations: newBrazeRetryClient(client, brazeMutationRetryPolicy),
	}
}

func newBrazeRetryClient(client *http.Client, policy retryablehttp.CheckRetry) *http.Client {
	retry := retryablehttp.NewClient()
	retry.HTTPClient = client
	retry.Logger = nil
	retry.RetryWaitMin = time.Second
	retry.RetryWaitMax = 3 * time.Second //nolint:mnd
	retry.Backoff = brazeRateLimitBackoff
	retry.CheckRetry = policy
	retry.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return retry.StandardClient()
}

//nolint:wrapcheck // Transport delegates preserve the HTTP client error contract.
func (c *brazeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return c.retryable.Do(req)
	default:
		// A failed POST may have created an object before the response was lost.
		return c.mutations.Do(req)
	}
}

//nolint:wrapcheck // Retry callbacks preserve cancellation and retryablehttp errors.
func brazeRetryPolicy(ctx context.Context, response *http.Response, err error) (bool, error) {
	if response != nil && response.StatusCode == http.StatusTooManyRequests {
		const maxRateLimitWait = 5 * time.Minute
		if delay, ok := rateLimitDelay(response.Header, time.Now); ok && delay > maxRateLimitWait {
			return false, ctx.Err()
		}
	}

	return retryablehttp.DefaultRetryPolicy(ctx, response, err)
}

//nolint:wrapcheck // Retry callbacks preserve cancellation errors.
func brazeMutationRetryPolicy(ctx context.Context, response *http.Response, err error) (bool, error) {
	if response == nil || response.StatusCode != http.StatusTooManyRequests {
		return false, ctx.Err()
	}

	return brazeRetryPolicy(ctx, response, err)
}
