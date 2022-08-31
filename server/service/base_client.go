package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

// httpClient interface allows the HTTP methods to be mocked.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type baseClient struct {
	baseURL            *url.URL
	http               httpClient
	urlPrefix          string
	insecureSkipVerify bool
}

func (bc *baseClient) parseResponse(verb, path string, response *http.Response, responseDest interface{}) error {
	switch response.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusNotFound:
		return notFoundErr{}
	case http.StatusUnauthorized:
		return ErrUnauthenticated
	case http.StatusPaymentRequired:
		return ErrMissingLicense
	default:
		return fmt.Errorf(
			"%s %s received status %d %s",
			verb, path,
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	if err := json.NewDecoder(response.Body).Decode(&responseDest); err != nil {
		return fmt.Errorf("decode %s %s response: %w", verb, path, err)
	}

	if e, ok := responseDest.(errorer); ok {
		if e.error() != nil {
			return fmt.Errorf("%s %s error: %w", verb, path, e.error())
		}
	}

	return nil
}

func (bc *baseClient) url(path, rawQuery string) *url.URL {
	u := *bc.baseURL
	u.Path = bc.urlPrefix + path
	u.RawQuery = rawQuery
	return &u
}

func newBaseClient(addr string, insecureSkipVerify bool, rootCA, urlPrefix string) (*baseClient, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	if baseURL.Scheme != "https" && !strings.Contains(baseURL.Host, "localhost") && !strings.Contains(baseURL.Host, "127.0.0.1") {
		return nil, errors.New("address must start with https:// for remote connections")
	}

	rootCAPool := x509.NewCertPool()

	tlsConfig := &tls.Config{
		// Osquery itself requires >= TLS 1.2.
		// https://github.com/osquery/osquery/blob/9713ad9e28f1cfe6c16a823fb88bd531e39e192d/osquery/remote/transports/tls.cpp#L97-L98
		MinVersion: tls.VersionTLS12,
	}

	switch {
	case rootCA != "":
		// read in the root cert file specified in the context
		certs, err := ioutil.ReadFile(rootCA)
		if err != nil {
			return nil, fmt.Errorf("reading root CA: %w", err)
		}
		// add certs to pool
		if ok := rootCAPool.AppendCertsFromPEM(certs); !ok {
			return nil, errors.New("failed to add certificates to root CA pool")
		}
		tlsConfig.RootCAs = rootCAPool
	case insecureSkipVerify:
		// Ignoring "G402: TLS InsecureSkipVerify set true", needed for development/testing.
		tlsConfig.InsecureSkipVerify = true //nolint:gosec
	default:
		// Use only the system certs (doesn't work on Windows)
		rootCAPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("loading system cert pool: %w", err)
		}
		tlsConfig.RootCAs = rootCAPool
	}

	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(tlsConfig))
	client := &baseClient{
		baseURL:            baseURL,
		http:               httpClient,
		insecureSkipVerify: insecureSkipVerify,
		urlPrefix:          urlPrefix,
	}

	return client, nil
}
