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

func WithBaseURL(baseURL string) Option {
	return func(c *client) {
		c.restyClient.SetBaseURL(baseURL)
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *client) {
		if timeout > 0 {
			c.restyClient.SetTimeout(timeout)
		}
	}
}

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

func NewClient(opts ...Option) Client {
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

func newDefaultAPIClient() *resty.Client {
	return resty.New().
		SetBaseURL(DefaultBaseURL).
		SetTimeout(DefaultTimeout).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)
}

func newTransferClient(timeout time.Duration) *resty.Client {
	client := resty.New().
		SetTimeout(timeout).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	return client
}
