//nolint:testpackage
package provider

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestRateLimitDelayUsesRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "12")
	header.Set(rateLimitResetHeader, "1234567890")

	delay, ok := rateLimitDelay(header, fixedTime)
	if !ok {
		t.Fatal("expected rate limit delay")
	}

	if delay != 12*time.Second {
		t.Fatalf("expected 12s delay, got %s", delay)
	}
}

func TestRateLimitDelayUsesRetryAfterHTTPDate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 29, 10, 0, 0, 0, time.UTC)
	header := http.Header{}
	header.Set("Retry-After", now.Add(2*time.Minute).Format(http.TimeFormat))

	delay, ok := rateLimitDelay(header, func() time.Time { return now })
	if !ok {
		t.Fatal("expected rate limit delay")
	}

	if delay != 2*time.Minute {
		t.Fatalf("expected 2m delay, got %s", delay)
	}
}

func TestRateLimitDelayUsesRateLimitReset(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 29, 10, 0, 0, 0, time.UTC)
	header := http.Header{}
	header.Set(rateLimitResetHeader, strconvFormatInt(now.Add(45*time.Second).Unix()))

	delay, ok := rateLimitDelay(header, func() time.Time { return now })
	if !ok {
		t.Fatal("expected rate limit delay")
	}

	if delay != 45*time.Second {
		t.Fatalf("expected 45s delay, got %s", delay)
	}
}

func TestRateLimitDelayReturnsZeroForPastReset(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.May, 29, 10, 0, 0, 0, time.UTC)
	header := http.Header{}
	header.Set(rateLimitResetHeader, strconvFormatInt(now.Add(-time.Second).Unix()))

	delay, ok := rateLimitDelay(header, func() time.Time { return now })
	if !ok {
		t.Fatal("expected rate limit delay")
	}

	if delay != 0 {
		t.Fatalf("expected no delay for past reset, got %s", delay)
	}
}

func TestRateLimitDelayRejectsInvalidHeaders(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "invalid")
	header.Set(rateLimitResetHeader, "invalid")

	delay, ok := rateLimitDelay(header, fixedTime)
	if ok {
		t.Fatalf("expected invalid headers to be ignored, got %s", delay)
	}
}

func fixedTime() time.Time {
	return time.Date(2026, time.May, 29, 10, 0, 0, 0, time.UTC)
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
