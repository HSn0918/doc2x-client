package client

import (
	"time"

	"github.com/go-resty/resty/v2"
)

type client struct {
	restyClient       *resty.Client
	transferClient    *resty.Client
	apiKey            string
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
// The key is shared with both the main and transfer clients.
func WithAPIKey(apiKey string) Option {
	return func(c *client) {
		c.apiKey = apiKey
		c.restyClient.SetHeader("Authorization", "Bearer "+apiKey)
	}
}

// WithRestyClient allows callers to provide a preconfigured API client.
func WithRestyClient(restyClient *resty.Client) Option {
	return func(c *client) {
		if restyClient != nil {
			c.restyClient = restyClient
		}
	}
}

// WithTransferClient overrides the client used for uploads/downloads to OSS/pre-signed URLs.
func WithTransferClient(transfer *resty.Client) Option {
	return func(c *client) {
		if transfer != nil {
			c.transferClient = transfer
		}
	}
}

// WithProcessingTimeout customizes the maximum wait time for long-running operations.
func WithProcessingTimeout(timeout time.Duration) Option {
	return func(c *client) {
		if timeout > 0 {
			c.processingTimeout = timeout
			if c.transferClient != nil {
				c.transferClient.SetTimeout(timeout)
			}
		}
	}
}

// NewClient creates a configured client that can speak to the doc2x service.
// Options allow overriding the base API client, transfer client, and processing timeout.
func NewClient(apiKey string, opts ...Option) Client {
	c := &client{
		restyClient:       newDefaultAPIClient(),
		processingTimeout: ProcessingTimeout,
		apiKey:            apiKey,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.restyClient == nil {
		c.restyClient = newDefaultAPIClient()
	}

	if c.transferClient == nil {
		c.transferClient = newTransferClient(c.processingTimeout)
	}

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

// newTransferClient builds an HTTP client tailored for transfer operations with the given timeout.
func newTransferClient(timeout time.Duration) *resty.Client {
	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	return client
}
