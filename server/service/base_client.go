package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

var errInvalidScheme = errors.New("address must start with https:// for remote connections")

// httpClient interface allows the HTTP methods to be mocked.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type baseClient struct {
	baseURL            *url.URL
	http               httpClient
	urlPrefix          string
	insecureSkipVerify bool
	// serverCapabilities is a map of capabilities that the server supports.
	// This map is updated on each response we receive from the server.
	serverCapabilities fleet.CapabilityMap
	// clientCapabilities is a map of capabilities that the client supports.
	// This list is given when the client is instantiated and shouldn't be
	// modified afterwards.
	clientCapabilities fleet.CapabilityMap
}

// parseResponse processes the status code and parses the response body.
// It does not close the response body (should be closed by the caller).
func (bc *baseClient) parseResponse(verb, path string, response *http.Response, responseDest interface{}) error {
	switch response.StatusCode {
	case http.StatusNotFound:
		return notFoundErr{
			msg: extractServerErrorText(response.Body),
		}
	case http.StatusUnauthorized:
		return ErrUnauthenticated
	case http.StatusPaymentRequired:
		return ErrMissingLicense
	default:
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			break
		}

		e := &statusCodeErr{
			code: response.StatusCode,
			body: extractServerErrorText(response.Body),
		}
		return fmt.Errorf("%s %s received status %w", verb, path, e)
	}

	bc.setServerCapabilities(response)

	if responseDest != nil {
		if e, ok := responseDest.(bodyHandler); ok {
			if err := e.Handle(response); err != nil {
				return fmt.Errorf("%s %s error with custom body handler contents: %w", verb, path, err)
			}
		} else if response.StatusCode != http.StatusNoContent {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("reading response body: %w", err)
			}
			if err := json.Unmarshal(b, &responseDest); err != nil {
				return fmt.Errorf("decode %s %s response: %w, body: %s", verb, path, err, b)
			}
			if e, ok := responseDest.(errorer); ok {
				if e.error() != nil {
					return fmt.Errorf("%s %s error: %w", verb, path, e.error())
				}
			}
		}
	}

	bc.setServerCapabilities(response)

	return nil
}

func (bc *baseClient) url(path, rawQuery string) *url.URL {
	u := *bc.baseURL
	u.Path = bc.urlPrefix + path
	u.RawQuery = rawQuery
	return &u
}

// setServerCapabilities updates the server capabilities based on the response
// from the server.
func (bc *baseClient) setServerCapabilities(response *http.Response) {
	capabilities := response.Header.Get(fleet.CapabilitiesHeader)
	bc.serverCapabilities.PopulateFromString(capabilities)
}

func (bc *baseClient) GetServerCapabilities() fleet.CapabilityMap {
	return bc.serverCapabilities
}

// setClientCapabilities header is used to set a header with the client
// capabilities in the given request.
//
// This method is defined in baseClient because other clients generally have
// custom implementations of a method to perform the requests to the server.
func (bc *baseClient) setClientCapabilitiesHeader(req *http.Request) {
	if len(bc.clientCapabilities) == 0 {
		return
	}

	if req.Header == nil {
		req.Header = http.Header{}
	}

	req.Header.Set(fleet.CapabilitiesHeader, bc.clientCapabilities.String())
}

func newBaseClient(
	addr string,
	insecureSkipVerify bool,
	rootCA, urlPrefix string,
	fleetClientCert *tls.Certificate,
	capabilities fleet.CapabilityMap,
) (*baseClient, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	allowHTTP := insecureSkipVerify || strings.Contains(baseURL.Host, "localhost") || strings.Contains(baseURL.Host, "127.0.0.1")
	if baseURL.Scheme != "https" && !allowHTTP {
		return nil, errInvalidScheme
	}

	rootCAPool := x509.NewCertPool()

	tlsConfig := &tls.Config{
		// Osquery itself requires >= TLS 1.2.
		// https://github.com/osquery/osquery/blob/9713ad9e28f1cfe6c16a823fb88bd531e39e192d/osquery/remote/transports/tls.cpp#L97-L98
		MinVersion: tls.VersionTLS12,
	}

	if fleetClientCert != nil {
		tlsConfig.Certificates = []tls.Certificate{*fleetClientCert}
	}

	switch {
	case rootCA != "":
		// read in the root cert file specified in the context
		certs, err := os.ReadFile(rootCA)
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
		clientCapabilities: capabilities,
		serverCapabilities: fleet.CapabilityMap{},
	}
	return client, nil
}

type bodyHandler interface {
	Handle(*http.Response) error
}

type FileResponse struct {
	DestPath     string
	DestFile     string
	destFilePath string
}

func (f *FileResponse) Handle(resp *http.Response) error {
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return fmt.Errorf("parsing media type from response header: %w", err)
	}

	filename := params["filename"]
	if filename == "" {
		filename = uuid.NewString()
	}

	f.destFilePath = filepath.Join(f.DestPath, filename)
	destFile, err := os.Create(f.destFilePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("copying from http stream to file: %w", err)
	}

	if err := destFile.Close(); err != nil {
		return fmt.Errorf("closing file after copy: %w", err)
	}

	return nil
}

func (f *FileResponse) GetFilePath() string {
	return f.destFilePath
}
