package client

import (
	"encoding/json"
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

// API response codes
const (
	CodeSuccess = "success"
	CodeFailed  = "failed"
)

// Processing status constants
const (
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusProcessing = "processing"
)

// API endpoints
const (
	EndpointParsePDF      = "/api/v2/parse/pdf"
	EndpointPreUpload     = "/api/v2/parse/preupload"
	EndpointParseStatus   = "/api/v2/parse/status"
	EndpointConvertParse  = "/api/v2/convert/parse"
	EndpointConvertResult = "/api/v2/convert/parse/result"
)

// Info provides metadata about the client
type Info interface {
	Name() string
	Version() string
}

// Parser handles document parsing operations
type Parser interface {
	UploadPDF(pdfData []byte) (*UploadResponse, error)
	PreUpload() (*PreUploadResponse, error)
	UploadToPresignedURL(url string, fileData []byte) error
	GetStatus(uid string) (*StatusResponse, error)
	WaitForParsing(uid string, pollInterval time.Duration) (*StatusResponse, error)
}

// Converter handles document conversion operations
type Converter interface {
	ConvertParse(req ConvertRequest) (*ConvertResponse, error)
	GetConvertResult(uid string) (*ConvertResultResponse, error)
	WaitForConversion(uid string, pollInterval time.Duration) (*ConvertResultResponse, error)
}

// Downloader handles file download operations
type Downloader interface {
	DownloadFile(url string) ([]byte, error)
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
	baseURL     string
}

var _ Client = (*client)(nil)

type Option func(*client)

func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		c.baseURL = baseURL
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
		c.restyClient.SetHeader("Authorization", "Bearer "+apiKey)
	}
}

func NewClient(opts ...Option) Client {
	c := &client{
		baseURL:     DefaultBaseURL,
		restyClient: resty.New(),
	}

	c.restyClient.
		SetBaseURL(DefaultBaseURL).
		SetTimeout(DefaultTimeout).
		SetHeader("Content-Type", "application/json")

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
func (c *client) UploadPDF(pdfData []byte) (*UploadResponse, error) {
	resp, err := c.restyClient.R().
		SetHeader("Content-Type", "application/pdf").
		SetBody(pdfData).
		Post(EndpointParsePDF)

	if err != nil {
		return nil, fmt.Errorf("upload PDF failed: %w", err)
	}

	var result UploadResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// PreUpload initiates a presigned upload flow.
// It returns presigned URL for direct file upload.
func (c *client) PreUpload() (*PreUploadResponse, error) {
	resp, err := c.restyClient.R().
		Post(EndpointPreUpload)

	if err != nil {
		return nil, fmt.Errorf("preupload failed: %w", err)
	}

	var result PreUploadResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// UploadToPresignedURL uploads file data to a presigned URL.
// This is used in conjunction with PreUpload for large file uploads.
func (c *client) UploadToPresignedURL(url string, fileData []byte) error {
	_, err := resty.New().R().
		SetBody(fileData).
		Put(url)

	if err != nil {
		return fmt.Errorf("upload to presigned URL failed: %w", err)
	}

	return nil
}

// GetStatus checks the parsing status for a given UID.
// It returns detailed status information including progress and results.
func (c *client) GetStatus(uid string) (*StatusResponse, error) {
	resp, err := c.restyClient.R().
		SetQueryParam("uid", uid).
		Get(EndpointParseStatus)

	if err != nil {
		return nil, fmt.Errorf("get status failed: %w", err)
	}

	var result StatusResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// ConvertParse initiates document conversion with specified parameters.
// It returns conversion tracking information.
func (c *client) ConvertParse(req ConvertRequest) (*ConvertResponse, error) {
	resp, err := c.restyClient.R().
		SetBody(req).
		Post(EndpointConvertParse)

	if err != nil {
		return nil, fmt.Errorf("convert parse failed: %w", err)
	}

	var result ConvertResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// GetConvertResult retrieves conversion results for a given UID.
// It returns the final converted document information.
func (c *client) GetConvertResult(uid string) (*ConvertResultResponse, error) {
	resp, err := c.restyClient.R().
		SetQueryParam("uid", uid).
		Get(EndpointConvertResult)

	if err != nil {
		return nil, fmt.Errorf("get convert result failed: %w", err)
	}

	var result ConvertResultResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return &result, nil
}

// DownloadFile downloads a file from the given URL.
// It handles URL unescaping and returns the raw file content.
func (c *client) DownloadFile(url string) ([]byte, error) {
	// Fix URL encoding issues
	url = strings.ReplaceAll(url, "\\u0026", "&")

	resp, err := resty.New().R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("download file failed: %w", err)
	}

	return resp.Body(), nil
}

// WaitForParsing polls the parsing status until completion or failure.
// It uses the provided poll interval to check status periodically.
// Returns the final status or an error if parsing fails.
func (c *client) WaitForParsing(uid string, pollInterval time.Duration) (*StatusResponse, error) {
	for {
		status, err := c.GetStatus(uid)
		if err != nil {
			return nil, err
		}

		if status.Code != CodeSuccess {
			return nil, fmt.Errorf("parse failed: %s - %s", status.Code, status.Msg)
		}

		switch status.Data.Status {
		case StatusSuccess:
			return status, nil
		case StatusFailed:
			return nil, fmt.Errorf("parse failed: %s", status.Data.Detail)
		case StatusProcessing:
			time.Sleep(pollInterval)
		default:
			// Unknown status, continue polling
			time.Sleep(pollInterval)
		}
	}
}

// WaitForConversion polls the conversion status until completion or failure.
// It uses the provided poll interval to check status periodically.
// Returns the final result or an error if conversion fails.
func (c *client) WaitForConversion(uid string, pollInterval time.Duration) (*ConvertResultResponse, error) {
	for {
		result, err := c.GetConvertResult(uid)
		if err != nil {
			return nil, err
		}

		if result.Code != CodeSuccess {
			return nil, fmt.Errorf("convert failed: %s", result.Code)
		}

		switch result.Data.Status {
		case StatusSuccess:
			return result, nil
		case StatusFailed:
			return nil, fmt.Errorf("convert failed")
		case StatusProcessing:
			time.Sleep(pollInterval)
		default:
			// Unknown status, continue polling
			time.Sleep(pollInterval)
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
