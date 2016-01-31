package restclient

import "net/http"

type DefaultClient struct {
	baseURL string
	debug   bool
	client  *http.Client
}

func NewDefaultClient(baseURL string, debug bool, client *http.Client) Client {
	return &DefaultClient{
		baseURL,
		debug,
		client,
	}
}

func (c *DefaultClient) BaseURL() string {
	return c.baseURL
}

func (c *DefaultClient) Debug() bool {
	return c.debug
}

func (c *DefaultClient) HttpClient() *http.Client {
	return c.client
}
