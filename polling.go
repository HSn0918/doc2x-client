package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

const transientFetchRetryBudget = 3

// ensureAPISuccess normalizes API error handling by checking response codes.
func ensureAPISuccess(code, msg string) error {
	if code == "" || code == CodeSuccess {
		return nil
	}

	if msg == "" {
		msg = "no message provided"
	}

	return fmt.Errorf("api returned code %s: %s", code, msg)
}

// withProcessingTimeout wraps the context with the provided timeout if it lacks a deadline.
func withProcessingTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	if timeout <= 0 {
		timeout = ProcessingTimeout
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	return ctxWithTimeout, cancel
}

// waitWithPolling repeatedly fetches task status until completion, failure, or timeout.
func waitWithPolling[T any](ctx context.Context, uid string, pollInterval time.Duration, operation string,
	timeout time.Duration,
	fetch func(context.Context, string) (*T, error),
	evaluate func(*T) (bool, error),
) (*T, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ctx, cancel := withProcessingTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	retriesLeft := transientFetchRetryBudget

	for {
		result, err := fetch(ctx, uid)
		if err != nil {
			if retriesLeft > 0 && isTransientError(err) {
				retriesLeft--
				if err := waitForNextPoll(ctx, ticker, operation); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		retriesLeft = transientFetchRetryBudget

		done, evalErr := evaluate(result)
		if evalErr != nil {
			return nil, evalErr
		}
		if done {
			return result, nil
		}

		if err := waitForNextPoll(ctx, ticker, operation); err != nil {
			return nil, err
		}
	}
}

// waitForNextPoll blocks until the next ticker pulse or context cancellation.
func waitForNextPoll(ctx context.Context, ticker *time.Ticker, operation string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting for %s cancelled: %w", operation, ctx.Err())
	case <-ticker.C:
		return nil
	}
}

// isTransientError reports whether an error is temporary and merits a retry.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	type temporary interface {
		Temporary() bool
	}

	var tempErr temporary
	return errors.As(err, &tempErr) && tempErr.Temporary()
}
