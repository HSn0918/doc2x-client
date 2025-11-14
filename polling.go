package client

import (
	"context"
	"fmt"
	"time"
)

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

// withProcessingTimeout wraps the context with ProcessingTimeout if it lacks a deadline.
func withProcessingTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, ProcessingTimeout)
	return ctxWithTimeout, cancel
}

// waitWithPolling repeatedly fetches task status until completion, failure, or timeout.
func waitWithPolling[T any](ctx context.Context, uid string, pollInterval time.Duration, operation string,
	fetch func(context.Context, string) (*T, error),
	evaluate func(*T) (bool, error),
) (*T, error) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ctx, cancel := withProcessingTimeout(ctx)
	defer cancel()

	result, err := fetch(ctx, uid)
	if err != nil {
		return nil, err
	}

	done, evalErr := evaluate(result)
	if evalErr != nil {
		return nil, evalErr
	}
	if done {
		return result, nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for %s cancelled: %w", operation, ctx.Err())
		case <-ticker.C:
			result, err := fetch(ctx, uid)
			if err != nil {
				return nil, err
			}

			done, evalErr = evaluate(result)
			if evalErr != nil {
				return nil, evalErr
			}

			if done {
				return result, nil
			}
		}
	}
}
