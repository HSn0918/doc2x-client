package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ServiceName       = "doc2x"
	DefaultBaseURL    = "https://v2.doc2x.noedgeai.com"
	DefaultTimeout    = 30 * time.Second
	ProcessingTimeout = 5 * time.Minute
	APIVersion        = "v2"
)

// Response codes and status constants
const (
	CodeSuccess = "success"
	CodeFailed  = "failed"

	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusProcessing = "processing"
)

// API endpoints
const (
	EndpointParsePDF      = "/api/" + APIVersion + "/parse/pdf"
	EndpointPreUpload     = "/api/" + APIVersion + "/parse/preupload"
	EndpointParseStatus   = "/api/" + APIVersion + "/parse/status"
	EndpointConvertParse  = "/api/" + APIVersion + "/convert/parse"
	EndpointConvertResult = "/api/" + APIVersion + "/convert/parse/result"
)

// Info provides metadata about the client
type Info interface {
	Name() string
	Version() string
}

// Parser handles document parsing operations
type Parser interface {
	UploadPDF(ctx context.Context, pdfData []byte) (*UploadResponse, error)
	PreUpload(ctx context.Context) (*PreUploadResponse, error)
	UploadToPresignedURL(ctx context.Context, url string, fileData []byte) error
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
}

// Client combines all doc2x operations
type Client interface {
	Info
	Parser
	Converter
	Downloader
}

type client struct {
	restyClient *resty.Client
	apiKey      string
}

var _ Client = (*client)(nil)

type Option func(*client)

func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		c.restyClient.SetBaseURL(baseURL)
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *client) {
		c.restyClient.SetTimeout(timeout)
	}
}

func WithAPIKey(apiKey string) Option {
	return func(c *client) {
		c.apiKey = apiKey
		c.restyClient.SetHeader("Authorization", "Bearer "+apiKey)
	}
}

func NewClient(opts ...Option) Client {
	c := &client{
		restyClient: resty.New(),
	}

	// Set default configurations
	c.restyClient.
		SetBaseURL(DefaultBaseURL).
		SetTimeout(DefaultTimeout).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	// Apply custom options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// UploadResponse represents the response from direct PDF upload
type UploadResponse struct {
	Code string `json:"code"` // Response status code, "success" on successful upload
	Data struct {
		UID string `json:"uid"` // Unique document identifier for tracking and subsequent operations
	} `json:"data"`
}

// PreUploadResponse represents the response from preupload request
// Used to obtain presigned URLs for large file uploads
type PreUploadResponse struct {
	Code string `json:"code"` // Response status code, "success" on successful request
	Data struct {
		UID string `json:"uid"` // Unique document identifier
		URL string `json:"url"` // Presigned upload URL for direct file upload
	} `json:"data"`
}

// StatusResponse represents the document parsing status response
type StatusResponse struct {
	Code string `json:"code"`          // Response status code, "success" on successful request
	Msg  string `json:"msg,omitempty"` // Error message, only present on failure
	Data *struct {
		Progress int    `json:"progress"` // Parsing progress percentage (0-100)
		Status   string `json:"status"`   // Parsing status: "processing", "success", or "failed"
		Detail   string `json:"detail"`   // Detailed status description or error message
		Result   *struct {
			Version string `json:"version"` // Parser engine version
			Pages   []struct {
				URL        string `json:"url"`         // Page preview image URL
				PageIdx    int    `json:"page_idx"`    // Page index, starting from 0
				PageWidth  int    `json:"page_width"`  // Page width in pixels
				PageHeight int    `json:"page_height"` // Page height in pixels
				Md         string `json:"md"`          // Parsed Markdown content for this page
			} `json:"pages"`
		} `json:"result"` // Parsing result, only present on successful parsing
	} `json:"data"`
}

// ConvertRequest represents a document conversion request
type ConvertRequest struct {
	UID                 string `json:"uid"`                              // Document unique identifier from upload response
	To                  string `json:"to"`                               // Target format: "markdown", "html", "docx", etc.
	FormulaMode         string `json:"formula_mode"`                     // Formula rendering mode: "latex", "mathml", "image"
	Filename            string `json:"filename,omitempty"`               // Output filename (optional)
	MergeCrossPageForms bool   `json:"merge_cross_page_forms,omitempty"` // Whether to merge tables across pages (optional)
}

// ConvertResponse represents the document conversion response
type ConvertResponse struct {
	Code string `json:"code"` // Response status code, "success" on successful request
	Data struct {
		Status string `json:"status"` // Conversion status: "processing", "success", or "failed"
		URL    string `json:"url"`    // Download URL for converted file (available when conversion is complete)
	} `json:"data"`
}

// ConvertResultResponse represents the conversion result query response
type ConvertResultResponse struct {
	Code string `json:"code"` // Response status code, "success" on successful request
	Data struct {
		Status string `json:"status"` // Conversion status: "processing", "success", or "failed"
		URL    string `json:"url"`    // Download URL for the converted file
	} `json:"data"`
}

// UploadPDF uploads PDF data for parsing.
// It returns the upload response containing the UID for tracking.
func (c *client) UploadPDF(ctx context.Context, pdfData []byte) (*UploadResponse, error) {
	if len(pdfData) == 0 {
		return nil, fmt.Errorf("PDF data cannot be empty")
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

	if result.Code != CodeSuccess {
		return nil, fmt.Errorf("upload PDF failed with code: %s", result.Code)
	}

	return &result, nil
}

// PreUpload initiates a presigned upload flow.
// It returns presigned URL for direct file upload.
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

	if result.Code != CodeSuccess {
		return nil, fmt.Errorf("preupload failed with code: %s", result.Code)
	}

	return &result, nil
}

// UploadToPresignedURL uploads file data to a presigned URL.
// This is used in conjunction with PreUpload for large file uploads.
func (c *client) UploadToPresignedURL(ctx context.Context, url string, fileData []byte) error {
	if url == "" {
		return fmt.Errorf("presigned URL cannot be empty")
	}

	if len(fileData) == 0 {
		return fmt.Errorf("file data cannot be empty")
	}

	// Create a temporary client without base URL for presigned URL upload
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
// It returns detailed status information including progress and results.
func (c *client) GetStatus(ctx context.Context, uid string) (*StatusResponse, error) {
	if uid == "" {
		return nil, fmt.Errorf("UID cannot be empty")
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

	return &result, nil
}

// ConvertParse initiates document conversion with specified parameters.
// It returns conversion tracking information.
func (c *client) ConvertParse(ctx context.Context, req ConvertRequest) (*ConvertResponse, error) {
	if req.UID == "" {
		return nil, fmt.Errorf("UID cannot be empty in conversion request")
	}

	if req.To == "" {
		return nil, fmt.Errorf("target format cannot be empty in conversion request")
	}

	// Set default formula mode if not specified
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

	if result.Code != CodeSuccess {
		return nil, fmt.Errorf("convert parse failed with code: %s", result.Code)
	}

	return &result, nil
}

// GetConvertResult retrieves conversion results for a given UID.
// It returns the final converted document information.
func (c *client) GetConvertResult(ctx context.Context, uid string) (*ConvertResultResponse, error) {
	if uid == "" {
		return nil, fmt.Errorf("UID cannot be empty")
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

	return &result, nil
}

// DownloadFile downloads a file from the given URL.
// It handles URL unescaping and returns the raw file content.
func (c *client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("download URL cannot be empty")
	}

	// Handle URL unescaping
	url = strings.ReplaceAll(url, "\\u0026", "&")

	// Create a temporary client for external URL download
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

// WaitForParsing polls the parsing status until completion, failure, or context cancellation.
// It uses time.Ticker for precise timing and automatically applies ProcessingTimeout if context has no deadline.
// Returns the final status or an error if parsing fails or context is cancelled.
func (c *client) WaitForParsing(ctx context.Context, uid string, pollInterval time.Duration) (*StatusResponse, error) {
	if uid == "" {
		return nil, fmt.Errorf("UID cannot be empty")
	}

	if pollInterval <= 0 {
		pollInterval = 2 * time.Second // Default poll interval
	}

	// Apply timeout if context doesn't have one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ProcessingTimeout)
		defer cancel()
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Check status immediately before waiting
	status, err := c.GetStatus(ctx, uid)
	if err != nil {
		return nil, err
	}

	if status.Code != CodeSuccess {
		return nil, fmt.Errorf("parse API returned error: %s - %s", status.Code, status.Msg)
	}

	if status.Data != nil {
		switch status.Data.Status {
		case StatusSuccess:
			return status, nil
		case StatusFailed:
			return nil, fmt.Errorf("parse failed with detail: %s", status.Data.Detail)
		}
	}

	// Start polling
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for parsing cancelled: %w", ctx.Err())
		case <-ticker.C:
			status, err := c.GetStatus(ctx, uid)
			if err != nil {
				return nil, err
			}

			if status.Code != CodeSuccess {
				return nil, fmt.Errorf("parse API returned error: %s - %s", status.Code, status.Msg)
			}

			if status.Data != nil {
				switch status.Data.Status {
				case StatusSuccess:
					return status, nil
				case StatusFailed:
					detail := "unknown error"
					if status.Data.Detail != "" {
						detail = status.Data.Detail
					}
					return nil, fmt.Errorf("parse failed: %s", detail)
				case StatusProcessing:
					// Continue polling
				default:
					// Unknown status, continue polling
				}
			}
		}
	}
}

// WaitForConversion polls the conversion status until completion, failure, or context cancellation.
// It uses time.Ticker for precise timing and automatically applies ProcessingTimeout if context has no deadline.
// Returns the final result or an error if conversion fails or context is cancelled.
func (c *client) WaitForConversion(ctx context.Context, uid string, pollInterval time.Duration) (*ConvertResultResponse, error) {
	if uid == "" {
		return nil, fmt.Errorf("UID cannot be empty")
	}

	if pollInterval <= 0 {
		pollInterval = 2 * time.Second // Default poll interval
	}

	// Apply timeout if context doesn't have one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ProcessingTimeout)
		defer cancel()
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Check status immediately before waiting
	result, err := c.GetConvertResult(ctx, uid)
	if err != nil {
		return nil, err
	}

	if result.Code != CodeSuccess {
		return nil, fmt.Errorf("convert API returned error: %s", result.Code)
	}

	switch result.Data.Status {
	case StatusSuccess:
		return result, nil
	case StatusFailed:
		return nil, fmt.Errorf("conversion failed for UID: %s", uid)
	}

	// Start polling
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for conversion cancelled: %w", ctx.Err())
		case <-ticker.C:
			result, err := c.GetConvertResult(ctx, uid)
			if err != nil {
				return nil, err
			}

			if result.Code != CodeSuccess {
				return nil, fmt.Errorf("convert API returned error: %s", result.Code)
			}

			switch result.Data.Status {
			case StatusSuccess:
				if result.Data.URL == "" {
					return nil, fmt.Errorf("conversion succeeded but no download URL provided")
				}
				return result, nil
			case StatusFailed:
				return nil, fmt.Errorf("conversion failed for UID: %s", uid)
			case StatusProcessing:
				// Continue polling
			default:
				// Unknown status, continue polling
			}
		}
	}
}

// Name returns the service name
func (c *client) Name() string {
	return ServiceName
}

// Version returns the API version
func (c *client) Version() string {
	return APIVersion
}
