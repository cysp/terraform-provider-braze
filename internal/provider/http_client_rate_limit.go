package provider

import (
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const rateLimitResetHeader = "X-Ratelimit-Reset"

func brazeRateLimitBackoff(minDelay, maxDelay time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if delay, ok := rateLimitDelay(resp.Header, time.Now); ok {
			return delay
		}
	}

	return retryablehttp.LinearJitterBackoff(minDelay, maxDelay, attemptNum, resp)
}

func rateLimitDelay(header http.Header, now func() time.Time) (time.Duration, bool) {
	if delay, ok := retryAfterDelay(header.Get("Retry-After"), now); ok {
		return delay, true
	}

	resetAt, err := strconv.ParseInt(header.Get(rateLimitResetHeader), 10, 64)
	if err != nil {
		return 0, false
	}

	delay := time.Unix(resetAt, 0).Sub(now())
	if delay < 0 {
		return 0, true
	}

	return delay, true
}

func retryAfterDelay(value string, now func() time.Time) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if seconds < 0 {
			return 0, false
		}

		return time.Duration(seconds) * time.Second, true
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}

	delay := retryAt.Sub(now())
	if delay < 0 {
		return 0, true
	}

	return delay, true
}
