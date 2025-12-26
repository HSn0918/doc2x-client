package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// FetchConvertZIP resolves an image layout convert_zip payload into raw zip bytes.
// The payload is expected to be base64 (optionally with a data URI prefix).
func (c *client) FetchConvertZIP(ctx context.Context, convertZIP string) ([]byte, error) {
	_ = ctx
	payload, err := normalizeConvertZIP(convertZIP)
	if err != nil {
		return nil, err
	}

	data, err := decodeBase64Payload(payload)
	if err != nil {
		return nil, fmt.Errorf("decode convert_zip failed: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("convert_zip payload is empty")
	}

	return data, nil
}

// FetchConvertZIPTo resolves an image layout convert_zip payload into the provided writer.
// The payload is expected to be base64 (optionally with a data URI prefix).
func (c *client) FetchConvertZIPTo(ctx context.Context, convertZIP string, dst io.Writer) error {
	_ = ctx
	if dst == nil {
		return ErrNilWriter
	}

	payload, err := normalizeConvertZIP(convertZIP)
	if err != nil {
		return err
	}

	data, err := decodeBase64Payload(payload)
	if err != nil {
		return fmt.Errorf("decode convert_zip failed: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("convert_zip payload is empty")
	}

	if _, err := dst.Write(data); err != nil {
		return fmt.Errorf("write convert_zip payload failed: %w", err)
	}

	return nil
}

func normalizeConvertZIP(convertZIP string) (payload string, err error) {
	payload = strings.TrimSpace(convertZIP)
	if payload == "" {
		return "", ErrEmptyConvertZIP
	}

	payload = stripBase64DataURL(payload)
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", ErrEmptyConvertZIP
	}

	return payload, nil
}

func stripBase64DataURL(value string) string {
	if !strings.HasPrefix(value, "data:") {
		return value
	}

	idx := strings.Index(value, "base64,")
	if idx == -1 {
		return value
	}

	return value[idx+len("base64,"):]
}

func decodeBase64Payload(payload string) ([]byte, error) {
	if payload == "" {
		return nil, ErrEmptyConvertZIP
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err == nil {
		return data, nil
	}

	data, rawErr := base64.RawStdEncoding.DecodeString(payload)
	if rawErr == nil {
		return data, nil
	}

	return nil, err
}
