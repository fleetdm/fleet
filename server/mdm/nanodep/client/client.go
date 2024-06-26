// Package client implements HTTP primitives for talking with and authenticating with the Apple DEP APIs.
package client

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

const DefaultBaseURL = "https://mdmenrollment.apple.com/"

// Doer executes an HTTP request.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Config represents the configuration of a DEP name.
type Config struct {
	BaseURL string `json:"base_url,omitempty"`
}

type ConfigRetriever interface {
	RetrieveConfig(context.Context, string) (*Config, error)
}

// DefaultConfigRetreiver wraps a ConfigRetriever to return a default configuration.
type DefaultConfigRetreiver struct {
	next ConfigRetriever
}

func NewDefaultConfigRetreiver(next ConfigRetriever) *DefaultConfigRetreiver {
	return &DefaultConfigRetreiver{next: next}
}

// RetrieveConfig retrieves the Config from the wrapped retreiver and returns
// it. If the config is empty a default config is returned.
func (c *DefaultConfigRetreiver) RetrieveConfig(ctx context.Context, name string) (*Config, error) {
	config, err := c.next.RetrieveConfig(ctx, name)
	if config == nil || config.BaseURL == "" {
		config = &Config{BaseURL: DefaultBaseURL}
	}
	return config, err
}

// RetrieveAndResolveURL retrieves the base URL for a DEP name using store
// and resolves the full DEP request URL using path.
func RetrieveAndResolveURL(ctx context.Context, name string, store ConfigRetriever, path string) (*url.URL, error) {
	store = NewDefaultConfigRetreiver(store)
	config, err := store.RetrieveConfig(ctx, name)
	if err != nil {
		return nil, err
	}
	urlBase, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}
	urlPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	return urlBase.ResolveReference(urlPath), nil
}

// NewDEPRequestWithContext creates a new request for a DEP name. Note that
// path is the relative path of the DEP endpoint name like "account".
func NewRequestWithContext(ctx context.Context, name string, store ConfigRetriever, method, path string, body io.Reader) (*http.Request, error) {
	url, err := RetrieveAndResolveURL(ctx, name, store, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return req, err
	}
	return req.WithContext(WithName(req.Context(), name)), nil
}

// NewClient is a helper that returns a copy of client with transport set.
func NewClient(client *http.Client, transport http.RoundTripper) *http.Client {
	depClient := *client
	depClient.Transport = transport
	return &depClient
}
