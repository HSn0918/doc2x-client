package client

import "time"

const (
	ServiceName       = "doc2x"
	DefaultBaseURL    = "https://v2.doc2x.noedgeai.com"
	DefaultTimeout    = 5 * time.Minute
	ProcessingTimeout = 5 * time.Minute
	APIVersion        = "v2"
	TraceIDHeader     = "trace-id"
)

// Response codes and status constants
const (
	CodeSuccess = "success"
	CodeFailed  = "failed"

	StatusSuccess    = string(ParseStatusSuccess)
	StatusFailed     = string(ParseStatusFailed)
	StatusProcessing = string(ParseStatusProcessing)
)

// API endpoints
const (
	EndpointParsePDF               = "/api/" + APIVersion + "/parse/pdf"
	EndpointPreUpload              = "/api/" + APIVersion + "/parse/preupload"
	EndpointParseStatus            = "/api/" + APIVersion + "/parse/status"
	EndpointConvertParse           = "/api/" + APIVersion + "/convert/parse"
	EndpointConvertResult          = "/api/" + APIVersion + "/convert/parse/result"
	EndpointParseImageLayout       = "/api/" + APIVersion + "/parse/img/layout"
	EndpointAsyncParseImageLayout  = "/api/" + APIVersion + "/async/parse/img/layout"
	EndpointParseImageLayoutStatus = "/api/" + APIVersion + "/parse/img/layout/status"
)
