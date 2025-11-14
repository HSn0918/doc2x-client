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
)

func errCode(operation, code, msg string) error {
	if msg == "" {
		return fmt.Errorf("%s failed with code %s", operation, code)
	}
	return fmt.Errorf("%s failed with code %s: %s", operation, code, msg)
}
