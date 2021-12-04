package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

	writer io.Writer
}

type ClientOption func(*Client) error

func NewClient(addr string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error) {
	// TODO #265 refactor all optional parameters to functional options
	// API breaking change, needs a major version release
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	if baseURL.Scheme != "https" && !strings.Contains(baseURL.Host, "localhost") && !strings.Contains(baseURL.Host, "127.0.0.1") {
		return nil, errors.New("address must start with https:// for remote connections")
	}

	rootCAPool := x509.NewCertPool()
	if rootCA != "" {
		// read in the root cert file specified in the context
		certs, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return nil, fmt.Errorf("reading root CA: %w", err)
		}

		// add certs to pool
		if ok := rootCAPool.AppendCertsFromPEM(certs); !ok {
			return nil, errors.New("failed to add certificates to root CA pool")
		}
	} else if !insecureSkipVerify {
		// Use only the system certs (doesn't work on Windows)
		rootCAPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("loading system cert pool: %w", err)
		}
	}

	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		RootCAs:            rootCAPool,
	}))

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

func SetClientWriter(w io.Writer) ClientOption {
	return func(c *Client) error {
		c.writer = w
		return nil
	}
}

func (c *Client) doContextWithHeaders(ctx context.Context, verb, path, rawQuery string, params interface{}, headers map[string]string) (*http.Response, error) {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "marshaling json")
		}
	}

	request, err := http.NewRequestWithContext(
		ctx,
		verb,
		c.url(path, rawQuery).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating request object")
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	resp, err := c.http.Do(request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "do request")
	}

	if resp.Header.Get(fleet.HeaderLicenseKey) == fleet.HeaderLicenseValueExpired {
		fleet.WriteExpiredLicenseBanner(c.writer)
	}

	return resp, nil
}

func (c *Client) Do(verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	return c.DoContext(context.Background(), verb, path, rawQuery, params)
}

func (c *Client) DoContext(ctx context.Context, verb, path, rawQuery string, params interface{}) (*http.Response, error) {
	headers := map[string]string{
		"Content-type": "application/json",
		"Accept":       "application/json",
	}

	return c.doContextWithHeaders(ctx, verb, path, rawQuery, params, headers)
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

	return c.doContextWithHeaders(context.Background(), verb, path, rawQuery, params, headers)
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
	took := time.Since(start).Truncate(time.Millisecond)
	fmt.Fprintf(os.Stderr, "%s %s %s (%s)\n", res.Request.Method, res.Request.URL, res.Status, took)

	resBody := &bytes.Buffer{}
	resBodyReader := io.TeeReader(res.Body, resBody)
	if _, err := io.Copy(os.Stderr, resBodyReader); err != nil {
		fmt.Fprintf(os.Stderr, "Read body error: %v", err)
		return nil, err
	}
	res.Body = io.NopCloser(resBody)

	return res, nil
}

func (c *Client) authenticatedRequestWithQuery(params interface{}, verb string, path string, responseDest interface{}, query string) error {
	response, err := c.AuthenticatedDo(verb, path, query, params)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%s %s received status %d %s",
			verb, path,
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	err = json.NewDecoder(response.Body).Decode(&responseDest)
	if err != nil {
		return fmt.Errorf("decode %s %s response: %w", verb, path, err)
	}

	if e, ok := responseDest.(errorer); ok {
		if e.error() != nil {
			return fmt.Errorf("%s %s error: %s", verb, path, e.error())
		}
	}

	return nil
}

func (c *Client) authenticatedRequest(params interface{}, verb string, path string, responseDest interface{}) error {
	return c.authenticatedRequestWithQuery(params, verb, path, responseDest, "")
}
