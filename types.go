package client

// ConvertFormat enumerates supported conversion targets.
type ConvertFormat string

const (
	FormatMarkdown ConvertFormat = "md"
	FormatTex      ConvertFormat = "tex"
	FormatDocx     ConvertFormat = "docx"
	FormatMDDollar ConvertFormat = "md_dollar"
)

// FormulaMode enumerates available formula rendering modes.
type FormulaMode string

const (
	FormulaModeNormal FormulaMode = "normal"
	FormulaModeDollar FormulaMode = "dollar"
	FormulaModeLatex  FormulaMode = "latex"
)

// ParseStatus enumerates parse task states.
type ParseStatus string

const (
	ParseStatusProcessing ParseStatus = "processing"
	ParseStatusSuccess    ParseStatus = "success"
	ParseStatusFailed     ParseStatus = "failed"
)

// ConvertStatus enumerates conversion task states.
type ConvertStatus string

const (
	ConvertStatusProcessing ConvertStatus = "processing"
	ConvertStatusSuccess    ConvertStatus = "success"
	ConvertStatusFailed     ConvertStatus = "failed"
)

// Operation enumerates named long-running tasks for polling.
type Operation string

const (
	OperationParsing               Operation = "parsing"
	OperationConversion            Operation = "conversion"
	OperationImageLayout           Operation = "image layout"
	OperationUploadPDF             Operation = "upload PDF"
	OperationPreUpload             Operation = "preupload"
	OperationGetStatus             Operation = "get status"
	OperationConvertParse          Operation = "convert parse"
	OperationGetConvertResult      Operation = "get convert result"
	OperationParseImageLayout      Operation = "parse image layout"
	OperationAsyncParseImageLayout Operation = "async parse image layout"
	OperationGetImageLayoutStatus  Operation = "get image layout status"
)

// UploadResponse represents the response from direct PDF upload
type UploadResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`          // Response status code, "success" on successful upload
	Msg     string `json:"msg,omitempty"` // Optional server message
	Data    struct {
		UID string `json:"uid"` // Unique document identifier for tracking and subsequent operations
	} `json:"data"`
}

// PreUploadResponse represents the response from preupload request
// Used to obtain presigned URLs for large file uploads
type PreUploadResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`          // Response status code, "success" on successful request
	Msg     string `json:"msg,omitempty"` // Optional server message
	Data    struct {
		UID string `json:"uid"` // Unique document identifier
		URL string `json:"url"` // Presigned upload URL for direct file upload
	} `json:"data"`
}

// StatusResponse represents the document parsing status response
type StatusResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`          // Response status code, "success" on successful request
	Msg     string `json:"msg,omitempty"` // Error message, only present on failure
	Data    *struct {
		Progress int         `json:"progress"` // Parsing progress percentage (0-100)
		Status   ParseStatus `json:"status"`   // Parsing status: "processing", "success", or "failed"
		Detail   string      `json:"detail"`   // Detailed status description or error message
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
	UID                 string        `json:"uid"`                              // Document unique identifier from upload response
	To                  ConvertFormat `json:"to"`                               // Target format: "md", "tex", "docx", etc.
	FormulaMode         FormulaMode   `json:"formula_mode"`                     // Formula rendering mode: "normal", "dollar"
	Filename            string        `json:"filename,omitempty"`               // Output filename (optional)
	MergeCrossPageForms bool          `json:"merge_cross_page_forms,omitempty"` // Whether to merge tables across pages (optional)
}

// ConvertResponse represents the document conversion response
type ConvertResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"` // Response status code, "success" on successful request
	Msg     string `json:"msg,omitempty"`
	Data    struct {
		Status ConvertStatus `json:"status"` // Conversion status: "processing", "success", or "failed"
		URL    string        `json:"url"`    // Download URL for converted file (available when conversion is complete)
	} `json:"data"`
}

// ConvertResultResponse represents the conversion result query response
type ConvertResultResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"` // Response status code, "success" on successful request
	Msg     string `json:"msg,omitempty"`
	Data    struct {
		Status ConvertStatus `json:"status"` // Conversion status: "processing", "success", or "failed"
		URL    string        `json:"url"`    // Download URL for the converted file
	} `json:"data"`
}

// ImageLayoutPage represents a single parsed image page result.
type ImageLayoutPage struct {
	URL        string `json:"url,omitempty"`
	PageIdx    int    `json:"page_idx,omitempty"`
	PageWidth  int    `json:"page_width"`
	PageHeight int    `json:"page_height"`
	Md         string `json:"md"`
}

// ImageLayoutResult groups pages returned by the image layout API.
type ImageLayoutResult struct {
	Pages []ImageLayoutPage `json:"pages"`
}

// ImageLayoutSyncResponse represents the synchronous image layout parse response.
type ImageLayoutSyncResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`
	Msg     string `json:"msg,omitempty"`
	Data    *struct {
		ConvertZIP string             `json:"convert_zip"`
		Result     *ImageLayoutResult `json:"result"`
		UID        string             `json:"uid"`
	} `json:"data"`
}

// ImageLayoutAsyncResponse represents the async image layout submission response.
type ImageLayoutAsyncResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`
	Msg     string `json:"msg,omitempty"`
	Data    *struct {
		UID string `json:"uid"`
	} `json:"data"`
}

// ImageLayoutStatusResponse represents the async image layout processing status.
type ImageLayoutStatusResponse struct {
	TraceID string `json:"-"`
	Code    string `json:"code"`
	Msg     string `json:"msg,omitempty"`
	Data    *struct {
		Status     string             `json:"status"`
		Detail     string             `json:"detail,omitempty"`
		Progress   int                `json:"progress,omitempty"`
		Result     *ImageLayoutResult `json:"result"`
		ConvertZIP string             `json:"convert_zip"`
	} `json:"data"`
}
