package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

// DownloadFile downloads a file from the given URL.
func (c *client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, ErrEmptyDownloadURL
	}

	url = strings.ReplaceAll(url, "\\u0026", "&")

	tempClient := resty.New().
		SetTimeout(ProcessingTimeout).
		SetRetryCount(3)

	resp, err := tempClient.R().
		SetContext(ctx).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("download file from %s failed: %w", url, err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("download file failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	data := resp.Body()
	if len(data) == 0 {
		return nil, fmt.Errorf("downloaded file is empty")
	}

	return data, nil
}
