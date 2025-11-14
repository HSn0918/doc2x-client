package client

import (
	"context"
	"fmt"
	"time"
)

// ParseImageLayout uploads an image and parses it synchronously.
func (c *client) ParseImageLayout(ctx context.Context, imageData []byte) (*ImageLayoutSyncResponse, error) {
	if len(imageData) == 0 {
		return nil, ErrEmptyImageData
	}

	var result ImageLayoutSyncResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/octet-stream").
		SetBody(imageData).
		SetResult(&result).
		Post(EndpointParseImageLayout)

	if err != nil {
		return nil, fmt.Errorf("parse image layout failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("parse image layout failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, fmt.Errorf("parse image layout succeeded but response data is empty")
	}

	return &result, nil
}

// AsyncParseImageLayout submits an image parsing task and returns the UID for polling.
func (c *client) AsyncParseImageLayout(ctx context.Context, imageData []byte) (*ImageLayoutAsyncResponse, error) {
	if len(imageData) == 0 {
		return nil, ErrEmptyImageData
	}

	var result ImageLayoutAsyncResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/octet-stream").
		SetBody(imageData).
		SetResult(&result).
		Post(EndpointAsyncParseImageLayout)

	if err != nil {
		return nil, fmt.Errorf("async parse image layout failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("async parse image layout failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, fmt.Errorf("async parse image layout succeeded but no UID returned")
	}

	return &result, nil
}

// GetImageLayoutStatus checks the processing status for an async image parsing task.
func (c *client) GetImageLayoutStatus(ctx context.Context, uid string) (*ImageLayoutStatusResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	var result ImageLayoutStatusResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetQueryParam("uid", uid).
		SetResult(&result).
		Get(EndpointParseImageLayoutStatus)

	if err != nil {
		return nil, fmt.Errorf("get image layout status for UID %s failed: %w", uid, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get image layout status failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, err
	}

	return &result, nil
}

// WaitForImageLayout polls the image layout status until completion or failure.
func (c *client) WaitForImageLayout(ctx context.Context, uid string, pollInterval time.Duration) (*ImageLayoutStatusResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	return waitWithPolling(ctx, uid, pollInterval, "image layout", c.GetImageLayoutStatus, func(status *ImageLayoutStatusResponse) (bool, error) {
		if status.Data == nil {
			return false, nil
		}

		switch status.Data.Status {
		case StatusSuccess:
			return true, nil
		case StatusFailed:
			detail := status.Data.Detail
			if detail == "" {
				detail = "unknown error"
			}
			return false, fmt.Errorf("image layout failed: %s", detail)
		default:
			return false, nil
		}
	})
}
