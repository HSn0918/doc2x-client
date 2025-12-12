package client

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type client struct {
	restyClient       *resty.Client
	processingTimeout time.Duration
}

var _ Client = (*client)(nil)

type Option func(*client)

// WithBaseURL sets the base URL that the client will use for API calls.
// This is useful for custom environments or testing against mock servers.
func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		c.restyClient.SetBaseURL(baseURL)
	}
}

// WithTimeout overrides the default HTTP timeout for the API client.
// A non-positive duration leaves the timeout unchanged.
func WithTimeout(timeout time.Duration) Option {
	return func(c *client) {
		if timeout > 0 {
			c.restyClient.SetTimeout(timeout)
		}
	}
}

// WithAPIKey assigns the API key that will be sent in the Authorization header.
func WithAPIKey(apiKey string) Option {
	return func(c *client) {
		setAuthHeader(c.restyClient, apiKey)
	}
}

// WithRestyClient allows callers to provide a preconfigured API client.
func WithRestyClient(restyClient *resty.Client) Option {
	return func(c *client) {
		if restyClient != nil {
			if auth := c.restyClient.Header.Get("Authorization"); auth != "" {
				restyClient.SetHeader("Authorization", auth)
			}
			c.restyClient = restyClient
		}
	}
}

// WithProcessingTimeout customizes the maximum wait time for long-running operations.
func WithProcessingTimeout(timeout time.Duration) Option {
	return func(c *client) {
		if timeout > 0 {
			c.processingTimeout = timeout
			if c.restyClient != nil {
				c.restyClient.SetTimeout(timeout)
			}
		}
	}
}

// NewClient creates a configured client that can speak to the doc2x service.
// Options allow overriding the base API client and processing timeout.
func NewClient(apiKey string, opts ...Option) Client {
	c := &client{
		restyClient:       newDefaultAPIClient(),
		processingTimeout: ProcessingTimeout,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.restyClient == nil {
		c.restyClient = newDefaultAPIClient()
	}

	setAuthHeader(c.restyClient, apiKey)

	return c
}

// Name returns the service name.
func (c *client) Name() string {
	return ServiceName
}

// Version returns the API version.
func (c *client) Version() string {
	return APIVersion
}

// newDefaultAPIClient returns a resty client preconfigured for doc2x API requests.
func newDefaultAPIClient() *resty.Client {
	return resty.New().
		SetBaseURL(DefaultBaseURL).
		SetTimeout(DefaultTimeout).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)
}

func setAuthHeader(restyClient *resty.Client, apiKey string) {
	if apiKey == "" || restyClient == nil {
		return
	}
	restyClient.SetHeader("Authorization", "Bearer "+apiKey)
}

func (c *client) transferClient() *resty.Client {
	timeout := c.processingTimeout
	if timeout <= 0 {
		timeout = ProcessingTimeout
	}

	if c.restyClient == nil {
		return newTransferClient(timeout, nil)
	}

	baseHTTP := *c.restyClient.GetClient()
	baseHTTP.Timeout = timeout

	return newTransferClient(timeout, &baseHTTP)
}

// newTransferClient builds an HTTP client tailored for transfer operations with the given timeout.
func newTransferClient(timeout time.Duration, httpClient *http.Client) *resty.Client {
	client := resty.New()
	if httpClient != nil {
		client = resty.NewWithClient(httpClient)
	}

	return client.
		SetTimeout(timeout).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)
}
