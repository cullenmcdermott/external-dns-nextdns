package nextdns

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRetryWithBackoff_SuccessNoRetry tests that successful operations do not retry
func TestRetryWithBackoff_SuccessNoRetry(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := retryWithBackoff(ctx, func() error {
		callCount++
		return nil // Success on first call
	}, "TestOperation")

	if err != nil {
		t.Errorf("retryWithBackoff() unexpected error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("retryWithBackoff() called operation %d times, expected 1", callCount)
	}
}

// TestRetryWithBackoff_RetryOn5xxEventualSuccess tests retry on 5xx errors with eventual success
func TestRetryWithBackoff_RetryOn5xxEventualSuccess(t *testing.T) {
	// Override retry delays for faster tests
	originalDelays := retryDelays
	retryDelays = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}
	defer func() { retryDelays = originalDelays }()

	ctx := context.Background()
	callCount := 0

	err := retryWithBackoff(ctx, func() error {
		callCount++
		if callCount < 3 {
			// Fail with 500 error on first two calls
			return errors.New("API error: 500 Internal Server Error")
		}
		return nil // Success on third call
	}, "TestOperation")

	if err != nil {
		t.Errorf("retryWithBackoff() unexpected error = %v", err)
	}
	if callCount != 3 {
		t.Errorf("retryWithBackoff() called operation %d times, expected 3", callCount)
	}
}

// TestRetryWithBackoff_ExhaustedRetries tests that all retries are exhausted on persistent errors
func TestRetryWithBackoff_ExhaustedRetries(t *testing.T) {
	// Override retry delays for faster tests
	originalDelays := retryDelays
	retryDelays = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}
	defer func() { retryDelays = originalDelays }()

	ctx := context.Background()
	callCount := 0
	serverError := errors.New("API error: 502 Bad Gateway")

	err := retryWithBackoff(ctx, func() error {
		callCount++
		return serverError
	}, "TestOperation")

	if err == nil {
		t.Error("retryWithBackoff() expected error, got nil")
	}
	// Should be called 4 times: initial + 3 retries
	if callCount != 4 {
		t.Errorf("retryWithBackoff() called operation %d times, expected 4", callCount)
	}
}

// TestRetryWithBackoff_NoRetryOn4xx tests that 4xx client errors are not retried
func TestRetryWithBackoff_NoRetryOn4xx(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	testCases := []struct {
		name  string
		error string
	}{
		{"400 Bad Request", "API error: 400 Bad Request"},
		{"401 Unauthorized", "API error: 401 Unauthorized"},
		{"403 Forbidden", "API error: 403 Forbidden"},
		{"404 Not Found", "API error: 404 Not Found"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callCount = 0

			err := retryWithBackoff(ctx, func() error {
				callCount++
				return errors.New(tc.error)
			}, "TestOperation")

			if err == nil {
				t.Error("retryWithBackoff() expected error, got nil")
			}
			// Should only be called once - no retry for 4xx errors
			if callCount != 1 {
				t.Errorf("retryWithBackoff() called operation %d times, expected 1 for %s", callCount, tc.name)
			}
		})
	}
}

// TestIsRetryableError tests the error classification function
func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name      string
		error     error
		retryable bool
	}{
		// Nil error
		{"nil error", nil, false},

		// Retryable server errors (5xx)
		{"500 Internal Server Error", errors.New("API error: 500 Internal Server Error"), true},
		{"502 Bad Gateway", errors.New("API error: 502 Bad Gateway"), true},
		{"503 Service Unavailable", errors.New("API error: 503 Service Unavailable"), true},
		{"504 Gateway Timeout", errors.New("API error: 504 Gateway Timeout"), true},

		// Retryable rate limit error
		{"429 Too Many Requests", errors.New("API error: 429 Too Many Requests"), true},

		// Retryable network errors
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"connection reset", errors.New("read tcp: connection reset by peer"), true},
		{"timeout", errors.New("context deadline exceeded (timeout)"), true},
		{"EOF", errors.New("unexpected EOF"), true},

		// Non-retryable client errors (4xx)
		{"400 Bad Request", errors.New("API error: 400 Bad Request"), false},
		{"401 Unauthorized", errors.New("API error: 401 Unauthorized"), false},
		{"403 Forbidden", errors.New("API error: 403 Forbidden"), false},
		{"404 Not Found", errors.New("API error: 404 Not Found"), false},

		// Unknown errors (not retryable by default)
		{"unknown error", errors.New("some random error"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetryableError(tc.error)
			if result != tc.retryable {
				t.Errorf("isRetryableError(%q) = %v, expected %v", tc.error, result, tc.retryable)
			}
		})
	}
}

// TestRetryWithBackoff_ContextCancellation tests that context cancellation stops retries
func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	// Override retry delays for faster tests
	originalDelays := retryDelays
	retryDelays = []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	defer func() { retryDelays = originalDelays }()

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	// Cancel context after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retryWithBackoff(ctx, func() error {
		callCount++
		return errors.New("API error: 500 Internal Server Error")
	}, "TestOperation")

	if err == nil {
		t.Error("retryWithBackoff() expected error after context cancellation, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		// The error might be the last API error depending on timing,
		// so we just verify it did not succeed
		t.Logf("retryWithBackoff() returned error: %v", err)
	}
	// Should be called at most 2 times due to context cancellation
	if callCount > 2 {
		t.Errorf("retryWithBackoff() called operation %d times, expected at most 2 due to cancellation", callCount)
	}
}
