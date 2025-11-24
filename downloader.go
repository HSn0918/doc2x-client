package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

// DownloadFile downloads a file from the given URL.
func (c *client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, ErrEmptyDownloadURL
	}

	var buf bytes.Buffer
	if err := c.DownloadFileTo(ctx, url, &buf); err != nil {
		return nil, err
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("downloaded file is empty")
	}

	return buf.Bytes(), nil
}

// DownloadFileTo streams the file into the provided writer, avoiding buffering large payloads.
func (c *client) DownloadFileTo(ctx context.Context, url string, dst io.Writer) error {
	if url == "" {
		return ErrEmptyDownloadURL
	}

	if dst == nil {
		return ErrNilWriter
	}

	url = strings.ReplaceAll(url, "\\u0026", "&")

	resp, err := c.transferClient.R().
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Get(url)

	if err != nil {
		return fmt.Errorf("download file from %s failed: %w", url, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("download file failed with status %d: %s", resp.StatusCode(), resp.Status())
	}

	body := resp.RawBody()
	if body == nil {
		return fmt.Errorf("downloaded file is empty")
	}
	defer body.Close()

	written, copyErr := io.Copy(dst, body)
	if copyErr != nil {
		return fmt.Errorf("writing downloaded file failed: %w", copyErr)
	}

	if written == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	return nil
}
