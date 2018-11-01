package service

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func NewClient(addr string, insecureSkipVerify bool, rootCA string) (*Client, error) {
	if !strings.HasPrefix(addr, "https://") {
		return nil, errors.New("Address must start with https://")
	}

	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err, "parsing URL")
	}

	rootCAPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "loading system cert pool")
	}

	if rootCA != "" {
		// set up empty cert pool
		rootCAPool = x509.NewCertPool()

		// read in the root cert file specified in the context
		certs, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return nil, errors.Wrap(err, "reading root CA")
		}

		// add certs to new cert pool
		if ok := rootCAPool.AppendCertsFromPEM(certs); !ok {
			return nil, errors.Wrap(err, "adding root CA")
		}
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
				RootCAs:            rootCAPool,
			},
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
