package client

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyPDFData      = errors.New("pdf data cannot be empty")
	ErrEmptyImageData    = errors.New("image data cannot be empty")
	ErrEmptyUID          = errors.New("uid cannot be empty")
	ErrEmptyFileData     = errors.New("file data cannot be empty")
	ErrEmptyPresignedURL = errors.New("presigned url cannot be empty")
	ErrEmptyTargetFormat = errors.New("target format cannot be empty")
	ErrEmptyDownloadURL  = errors.New("download url cannot be empty")
	ErrEmptyConvertZIP   = errors.New("convert_zip cannot be empty")
	ErrNilReader         = errors.New("reader cannot be nil")
	ErrNilWriter         = errors.New("writer cannot be nil")
)

// errCode formats a failure message when the API reports a non-success code.
func errCode(operation Operation, code, msg, traceID string) error {
	traceID = normalizeTraceID(traceID)
	if msg == "" {
		return fmt.Errorf("%s failed with code %s (trace-id: %s)", operation, code, traceID)
	}
	return fmt.Errorf("%s failed with code %s: %s (trace-id: %s)", operation, code, msg, traceID)
}

// errStatus formats an error with HTTP status and trace id.
func errStatus(operation Operation, statusCode int, status, traceID string) error {
	traceID = normalizeTraceID(traceID)
	return fmt.Errorf("%s failed with status %d: %s (trace-id: %s)", operation, statusCode, status, traceID)
}

func normalizeTraceID(traceID string) string {
	if traceID == "" {
		return "unknown"
	}
	return traceID
}
