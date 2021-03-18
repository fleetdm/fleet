package service

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// httpClient interface allows the HTTP methods to be mocked.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	addr               string
	baseURL            *url.URL
	urlPrefix          string
	token              string
	http               httpClient
	insecureSkipVerify bool
}

type ClientOption func(*Client) error

func NewClient(addr string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error) {
	// TODO #265 refactor all optional parameters to functional options
	// API breaking change, needs a major version release
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err, "parsing URL")
	}

	if baseURL.Scheme != "https" && !strings.Contains(baseURL.Host, "localhost") {
		return nil, errors.New("address must start with https:// for remote connections")
	}

	rootCAPool := x509.NewCertPool()
	if rootCA != "" {
		// read in the root cert file specified in the context
		certs, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return nil, errors.Wrap(err, "reading root CA")
		}

		// add certs to pool
		if ok := rootCAPool.AppendCertsFromPEM(certs); !ok {
			return nil, errors.Wrap(err, "adding root CA")
		}
	} else if !insecureSkipVerify {
		// Use only the system certs (doesn't work on Windows)
		rootCAPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "loading system cert pool")
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

	client := &Client{
		addr:               addr,
		baseURL:            baseURL,
		http:               httpClient,
		insecureSkipVerify: insecureSkipVerify,
		urlPrefix:          urlPrefix,
	}

	for _, option := range options {
		err := option(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func EnableClientDebug() ClientOption {
	return func(c *Client) error {
		httpClient, ok := c.http.(*http.Client)
		if !ok {
			return errors.New("client is not *http.Client")
		}
		httpClient.Transport = &logRoundTripper{roundtripper: httpClient.Transport}

		return nil
	}
}

func (c *Client) doWithHeaders(verb, path, rawQuery string, params interface{}, headers map[string]string) (*http.Response, error) {
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
		c.url(path, rawQuery).String(),
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

func (c *Client) Do(verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	headers := map[string]string{
		"Content-type": "application/json",
		"Accept":       "application/json",
	}

	return c.doWithHeaders(verb, path, rawQuery, params, headers)
}

func (c *Client) AuthenticatedDo(verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	if c.token == "" {
		return nil, errors.New("authentication token is empty")
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.token),
	}

	return c.doWithHeaders(verb, path, rawQuery, params, headers)
}

func (c *Client) SetToken(t string) {
	c.token = t
}

func (c *Client) url(path, rawQuery string) *url.URL {
	u := *c.baseURL
	u.Path = c.urlPrefix + path
	u.RawQuery = rawQuery
	return &u
}

// http.RoundTripper that will log debug information about the request and
// response, including paths, timing, and body.
//
// Inspired by https://stackoverflow.com/a/39528716/491710 and
// github.com/motemen/go-loghttp
type logRoundTripper struct {
	roundtripper http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (l *logRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log request
	fmt.Fprintf(os.Stderr, "%s %s\n", req.Method, req.URL)
	reqBody, err := req.GetBody()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetBody error: %v\n", err)
	} else {
		defer reqBody.Close()
		if _, err := io.Copy(os.Stderr, reqBody); err != nil {
			fmt.Fprintf(os.Stderr, "Copy body error: %v\n", err)
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Perform request using underlying roundtripper
	start := time.Now()
	res, err := l.roundtripper.RoundTrip(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RoundTrip error: %v", err)
		return nil, err
	}

	// Log response
	took := time.Now().Sub(start).Truncate(time.Millisecond)
	fmt.Fprintf(os.Stderr, "%s %s %s (%s)\n", res.Request.Method, res.Request.URL, res.Status, took)

	resBody := &bytes.Buffer{}
	resBodyReader := io.TeeReader(res.Body, resBody)
	if _, err := io.Copy(os.Stderr, resBodyReader); err != nil {
		fmt.Fprintf(os.Stderr, "Read body error: %v", err)
		return nil, err
	}
	res.Body = ioutil.NopCloser(resBody)

	return res, nil
}
