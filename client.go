package client

import (
	"time"

	"github.com/go-resty/resty/v2"
)

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

	c.restyClient.
		SetBaseURL(DefaultBaseURL).
		SetTimeout(DefaultTimeout).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)

	for _, opt := range opts {
		opt(c)
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
