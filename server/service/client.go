package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type Client struct {
	addr               string
	baseURL            *url.URL
	token              string
	http               *http.Client
	insecureSkipVerify bool
}

func NewClient(addr string, insecureSkipVerify bool) (*Client, error) {
	if !strings.HasPrefix(addr, "https://") {
		return nil, errors.New("Addrress must start with https://")
	}

	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err, "parsing URL")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	return &Client{
		addr:               addr,
		baseURL:            baseURL,
		http:               httpClient,
		insecureSkipVerify: insecureSkipVerify,
	}, nil
}

func (c *Client) doWithHeaders(verb, path string, params interface{}, headers map[string]string) (*http.Response, error) {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return nil, errors.Wrap(err, "marshaling json")
		}
	}

	request, err := http.NewRequest(
		verb,
		c.url(path).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, errors.Wrap(err, "creating request object")
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	return c.http.Do(request)
}

func (c *Client) Do(verb, path string, params interface{}) (*http.Response, error) {
	headers := map[string]string{
		"Content-type": "application/json",
		"Accept":       "application/json",
	}

	return c.doWithHeaders(verb, path, params, headers)
}

func (c *Client) AuthenticatedDo(verb, path string, params interface{}) (*http.Response, error) {
	if c.token == "" {
		return nil, errors.New("authentication token is empty")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.token),
	}

	return c.doWithHeaders(verb, path, params, headers)
}

func (c *Client) SetToken(t string) {
	c.token = t
}

func (c *Client) url(path string) *url.URL {
	u := *c.baseURL
	u.Path = path
	return &u
}
