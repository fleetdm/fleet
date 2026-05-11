package client

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

var ErrInvalidScheme = errors.New("address must start with https:// for remote connections")

// HTTPClient interface allows the HTTP methods to be mocked.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type BaseClient struct {
	BaseURL            *url.URL
	HTTP               HTTPClient
	URLPrefix          string
	InsecureSkipVerify bool
	// ServerCapabilities is a map of capabilities that the server supports.
	// This map is updated on each response we receive from the server.
	ServerCapabilities fleet.CapabilityMap
	// ClientCapabilities is a map of capabilities that the client supports.
	// This list is given when the client is instantiated and shouldn't be
	// modified afterwards.
	ClientCapabilities fleet.CapabilityMap
}

// ParseResponse processes the status code and parses the response body.
// It does not close the response body (should be closed by the caller).
func (bc *BaseClient) ParseResponse(verb, path string, response *http.Response, responseDest any) error {
	switch response.StatusCode {
	case http.StatusNotFound:
		return &NotFoundErr{
			Msg: ExtractServerErrorText(response.Body),
		}
	case http.StatusUnauthorized:
		errText := ExtractServerErrorText(response.Body)
		if strings.Contains(errText, "password reset required") {
			return ErrPasswordResetRequired
		}
		if strings.Contains(errText, "END_USER_AUTH_REQUIRED") {
			return ErrEndUserAuthRequired
		}
		return ErrUnauthenticated
	case http.StatusPaymentRequired:
		return ErrMissingLicense
	default:
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			break
		}

		e := &StatusCodeErr{
			Code: response.StatusCode,
			Body: ExtractServerErrorText(response.Body),
		}
		return fmt.Errorf("%s %s received status %w", verb, path, e)
	}

	bc.SetServerCapabilities(response)

	if responseDest != nil {
		if e, ok := responseDest.(BodyHandler); ok {
			if err := e.Handle(response); err != nil {
				return fmt.Errorf("%s %s error with custom body handler contents: %w", verb, path, err)
			}
		} else if response.StatusCode != http.StatusNoContent {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("reading response body: %w", err)
			}
			if err := json.Unmarshal(b, &responseDest); err != nil {
				const maxBodyLen = 200
				truncatedBytes, isHTML := TruncateAndDetectHTML(b, maxBodyLen)

				if isHTML {
					return fmt.Errorf("decode %s %s response: %w, (server returned HTML instead of JSON), body: %s", verb, path, err, truncatedBytes)
				}
				return fmt.Errorf("decode %s %s response: %w, body: %s", verb, path, err, truncatedBytes)
			}
			if e, ok := responseDest.(fleet.Errorer); ok {
				if e.Error() != nil {
					return fmt.Errorf("%s %s error: %w", verb, path, e.Error())
				}
			}
		}
	}

	bc.SetServerCapabilities(response)

	return nil
}

func (bc *BaseClient) URL(path, rawQuery string) *url.URL {
	u := *bc.BaseURL
	u.Path = bc.URLPrefix + path
	u.RawQuery = rawQuery
	return &u
}

// SetServerCapabilities updates the server capabilities based on the response
// from the server.
func (bc *BaseClient) SetServerCapabilities(response *http.Response) {
	capabilities := response.Header.Get(fleet.CapabilitiesHeader)
	bc.ServerCapabilities.PopulateFromString(capabilities)
}

func (bc *BaseClient) GetServerCapabilities() fleet.CapabilityMap {
	return bc.ServerCapabilities
}

// SetClientCapabilitiesHeader is used to set a header with the client
// capabilities in the given request.
//
// This method is defined in BaseClient because other clients generally have
// custom implementations of a method to perform the requests to the server.
func (bc *BaseClient) SetClientCapabilitiesHeader(req *http.Request) {
	if len(bc.ClientCapabilities) == 0 {
		return
	}

	if req.Header == nil {
		req.Header = http.Header{}
	}

	req.Header.Set(fleet.CapabilitiesHeader, bc.ClientCapabilities.String())
}

func NewBaseClient(
	addr string,
	insecureSkipVerify bool,
	rootCA, urlPrefix string,
	fleetClientCert *tls.Certificate,
	capabilities fleet.CapabilityMap,
	signerWrapper func(*http.Client) *http.Client,
) (*BaseClient, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	allowHTTP := insecureSkipVerify || strings.Contains(baseURL.Host, "localhost") || strings.Contains(baseURL.Host, "127.0.0.1")
	if baseURL.Scheme != "https" && !allowHTTP {
		return nil, ErrInvalidScheme
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
	if signerWrapper != nil {
		httpClient = signerWrapper(httpClient)
	}
	client := &BaseClient{
		BaseURL:            baseURL,
		HTTP:               httpClient,
		InsecureSkipVerify: insecureSkipVerify,
		URLPrefix:          urlPrefix,
		ClientCapabilities: capabilities,
		ServerCapabilities: fleet.CapabilityMap{},
	}
	return client, nil
}

// BodyHandler is an interface for custom response body handling.
type BodyHandler interface {
	Handle(*http.Response) error
}

type FileResponse struct {
	DestPath      string
	DestFile      string
	DestFilePath  string
	SkipMediaType bool
	ProgressFunc  func(n int)
}

func (f *FileResponse) Handle(resp *http.Response) error {
	var filename string
	if !f.SkipMediaType {
		_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
		if err != nil {
			return fmt.Errorf("parsing media type from response header: %w", err)
		}
		filename = params["filename"]
	}

	if filename == "" {
		filename = f.DestFile
	}
	if filename == "" {
		filename = uuid.NewString()
	}

	f.DestFilePath = filepath.Join(f.DestPath, filename)
	destFile, err := os.Create(f.DestFilePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer destFile.Close()

	var respBodyReader io.Reader = resp.Body
	if f.ProgressFunc != nil {
		respBodyReader = &progressReader{
			Reader:       respBodyReader,
			progressFunc: f.ProgressFunc,
		}
	}

	_, err = io.Copy(destFile, respBodyReader)
	if err != nil {
		return fmt.Errorf("copying from http stream to file: %w", err)
	}

	if err := destFile.Close(); err != nil {
		return fmt.Errorf("closing file after copy: %w", err)
	}

	return nil
}

func (f *FileResponse) GetFilePath() string {
	return f.DestFilePath
}

type progressReader struct {
	io.Reader
	progressFunc func(n int)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.progressFunc(n)
	return n, err
}

// DoHTTPRequest performs an HTTP request using the underlying HTTP client.
func (bc *BaseClient) DoHTTPRequest(req *http.Request) (*http.Response, error) {
	return bc.HTTP.Do(req)
}

// GetRawHTTPClient returns the underlying HTTP client for type assertions (e.g., idle connection cleanup).
func (bc *BaseClient) GetRawHTTPClient() HTTPClient {
	return bc.HTTP
}

// SetHTTPClient sets the underlying HTTP client (used in tests).
func (bc *BaseClient) SetHTTPClient(c HTTPClient) {
	bc.HTTP = c
}
