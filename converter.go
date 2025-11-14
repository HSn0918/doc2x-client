package client

import (
	"context"
	"fmt"
	"time"
)

// ConvertParse initiates document conversion with specified parameters.
func (c *client) ConvertParse(ctx context.Context, req ConvertRequest) (*ConvertResponse, error) {
	if req.UID == "" {
		return nil, ErrEmptyUID
	}

	if req.To == "" {
		return nil, ErrEmptyTargetFormat
	}

	if req.FormulaMode == "" {
		req.FormulaMode = "latex"
	}

	var result ConvertResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		Post(EndpointConvertParse)

	if err != nil {
		return nil, fmt.Errorf("convert parse for UID %s failed: %w", req.UID, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("convert parse failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, errCode("convert parse", result.Code, result.Msg)
	}

	return &result, nil
}

// GetConvertResult retrieves conversion results for a given UID.
func (c *client) GetConvertResult(ctx context.Context, uid string) (*ConvertResultResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	var result ConvertResultResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetQueryParam("uid", uid).
		SetResult(&result).
		Get(EndpointConvertResult)

	if err != nil {
		return nil, fmt.Errorf("get convert result for UID %s failed: %w", uid, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get convert result failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, errCode("get convert result", result.Code, result.Msg)
	}

	return &result, nil
}

// WaitForConversion polls the conversion status until completion, failure, or context cancellation.
func (c *client) WaitForConversion(ctx context.Context, uid string, pollInterval time.Duration) (*ConvertResultResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	return waitWithPolling(ctx, uid, pollInterval, "conversion", c.GetConvertResult, func(result *ConvertResultResponse) (bool, error) {
		switch result.Data.Status {
		case StatusSuccess:
			if result.Data.URL == "" {
				return false, fmt.Errorf("conversion succeeded but no download URL provided")
			}
			return true, nil
		case StatusFailed:
			return false, fmt.Errorf("conversion failed for UID: %s", uid)
		default:
			return false, nil
		}
	})
}
