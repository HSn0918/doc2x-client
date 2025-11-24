package client

import (
	"context"
	"io"
	"time"
)

// Info provides metadata about the client
type Info interface {
	Name() string
	Version() string
}

// Parser handles document parsing operations
type Parser interface {
	UploadPDF(ctx context.Context, pdfData []byte) (*UploadResponse, error)
	UploadPDFReader(ctx context.Context, r io.Reader) (*UploadResponse, error)
	PreUpload(ctx context.Context) (*PreUploadResponse, error)
	UploadToPresignedURL(ctx context.Context, url string, fileData []byte) error
	UploadToPresignedURLFrom(ctx context.Context, url string, r io.Reader) error
	GetStatus(ctx context.Context, uid string) (*StatusResponse, error)
	WaitForParsing(ctx context.Context, uid string, pollInterval time.Duration) (*StatusResponse, error)
}

// Converter handles document conversion operations
type Converter interface {
	ConvertParse(ctx context.Context, req ConvertRequest) (*ConvertResponse, error)
	GetConvertResult(ctx context.Context, uid string) (*ConvertResultResponse, error)
	WaitForConversion(ctx context.Context, uid string, pollInterval time.Duration) (*ConvertResultResponse, error)
}

// Downloader handles file download operations
type Downloader interface {
	DownloadFile(ctx context.Context, url string) ([]byte, error)
	DownloadFileTo(ctx context.Context, url string, dst io.Writer) error
}

// ImageParser handles image parsing operations
type ImageParser interface {
	ParseImageLayout(ctx context.Context, imageData []byte) (*ImageLayoutSyncResponse, error)
	AsyncParseImageLayout(ctx context.Context, imageData []byte) (*ImageLayoutAsyncResponse, error)
	GetImageLayoutStatus(ctx context.Context, uid string) (*ImageLayoutStatusResponse, error)
	WaitForImageLayout(ctx context.Context, uid string, pollInterval time.Duration) (*ImageLayoutStatusResponse, error)
}

// Client combines all doc2x operations
type Client interface {
	Info
	Parser
	Converter
	Downloader
	ImageParser
}
