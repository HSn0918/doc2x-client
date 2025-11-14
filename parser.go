package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// UploadPDF uploads PDF data for parsing.
func (c *client) UploadPDF(ctx context.Context, pdfData []byte) (*UploadResponse, error) {
	if len(pdfData) == 0 {
		return nil, ErrEmptyPDFData
	}

	var result UploadResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/pdf").
		SetBody(pdfData).
		SetResult(&result).
		Post(EndpointParsePDF)

	if err != nil {
		return nil, fmt.Errorf("upload PDF failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("upload PDF failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, errCode("upload PDF", result.Code, result.Msg)
	}

	return &result, nil
}

// PreUpload initiates the presigned upload flow.
func (c *client) PreUpload(ctx context.Context) (*PreUploadResponse, error) {
	var result PreUploadResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetResult(&result).
		Post(EndpointPreUpload)

	if err != nil {
		return nil, fmt.Errorf("preupload failed: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("preupload failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, ""); err != nil {
		return nil, errCode("preupload", result.Code, "")
	}

	return &result, nil
}

// UploadToPresignedURL uploads file data to the provided OSS URL.
func (c *client) UploadToPresignedURL(ctx context.Context, url string, fileData []byte) error {
	if url == "" {
		return ErrEmptyPresignedURL
	}

	if len(fileData) == 0 {
		return ErrEmptyFileData
	}

	tempClient := resty.New().
		SetTimeout(ProcessingTimeout).
		SetRetryCount(3)

	resp, err := tempClient.R().
		SetContext(ctx).
		SetBody(fileData).
		Put(url)

	if err != nil {
		return fmt.Errorf("upload to presigned URL failed: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("upload to presigned URL failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	return nil
}

// GetStatus checks the parsing status for a given UID.
func (c *client) GetStatus(ctx context.Context, uid string) (*StatusResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	var result StatusResponse
	resp, err := c.restyClient.R().
		SetContext(ctx).
		SetQueryParam("uid", uid).
		SetResult(&result).
		Get(EndpointParseStatus)

	if err != nil {
		return nil, fmt.Errorf("get status for UID %s failed: %w", uid, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("get status failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	if err := ensureAPISuccess(result.Code, result.Msg); err != nil {
		return nil, errCode("get status", result.Code, result.Msg)
	}

	return &result, nil
}

// WaitForParsing polls the parsing status until completion, failure, or context cancellation.
func (c *client) WaitForParsing(ctx context.Context, uid string, pollInterval time.Duration) (*StatusResponse, error) {
	if uid == "" {
		return nil, ErrEmptyUID
	}

	return waitWithPolling(ctx, uid, pollInterval, "parsing", c.GetStatus, func(status *StatusResponse) (bool, error) {
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
			return false, fmt.Errorf("parse failed: %s", detail)
		default:
			return false, nil
		}
	})
}
