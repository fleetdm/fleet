package scepserver

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

// possible SCEP operations
const (
	getCACaps     = "GetCACaps"
	getCACert     = "GetCACert"
	pkiOperation  = "PKIOperation"
	getNextCACert = "GetNextCACert"
)

type Endpoints struct {
	GetEndpoint  endpoint.Endpoint
	PostEndpoint endpoint.Endpoint

	mtx          sync.RWMutex
	capabilities []byte
}

func (e *Endpoints) GetCACaps(ctx context.Context) ([]byte, error) {
	request := SCEPRequest{Operation: getCACaps}
	response, err := e.GetEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	resp := response.(SCEPResponse)

	e.mtx.Lock()
	e.capabilities = resp.Data
	e.mtx.Unlock()

	return resp.Data, resp.Err
}

func (e *Endpoints) Supports(capacity string) bool {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	if len(e.capabilities) == 0 {
		e.mtx.RUnlock()
		_, _ = e.GetCACaps(context.Background())
		e.mtx.RLock()
	}
	return bytes.Contains(e.capabilities, []byte(capacity))
}

func (e *Endpoints) GetCACert(ctx context.Context, message string) ([]byte, int, error) {
	request := SCEPRequest{Operation: getCACert, Message: []byte(message)}
	response, err := e.GetEndpoint(ctx, request)
	if err != nil {
		return nil, 0, err
	}
	resp := response.(SCEPResponse)
	return resp.Data, resp.CACertNum, resp.Err
}

func (e *Endpoints) PKIOperation(ctx context.Context, msg []byte) ([]byte, error) {
	var ee endpoint.Endpoint
	if e.Supports("POSTPKIOperation") || e.Supports("SCEPStandard") {
		ee = e.PostEndpoint
	} else {
		ee = e.GetEndpoint
	}

	request := SCEPRequest{Operation: pkiOperation, Message: msg}
	response, err := ee(ctx, request)
	if err != nil {
		return nil, err
	}
	resp := response.(SCEPResponse)
	return resp.Data, resp.Err
}

func (e *Endpoints) GetNextCACert(ctx context.Context) ([]byte, error) {
	var request SCEPRequest
	response, err := e.GetEndpoint(ctx, request)
	if err != nil {
		return nil, err
	}
	resp := response.(SCEPResponse)
	return resp.Data, resp.Err
}

func MakeServerEndpoints(svc Service) *Endpoints {
	e := MakeSCEPEndpoint(svc)
	return &Endpoints{
		GetEndpoint:  e,
		PostEndpoint: e,
	}
}

func MakeServerEndpointsWithIdentifier(svc ServiceWithIdentifier) *Endpoints {
	e := MakeSCEPEndpointWithIdentifier(svc)
	return &Endpoints{
		GetEndpoint:  e,
		PostEndpoint: e,
	}
}

type clientOpts struct {
	timeout  *time.Duration
	rootCA   string
	insecure bool
}

// ClientOption is a functional option for configuring a SCEP Client
type ClientOption func(*clientOpts)

// WithClientRootCA sets the root CA file to use when connecting to the SCEP server.
func WithClientRootCA(rootCA string) ClientOption {
	return func(c *clientOpts) {
		c.rootCA = rootCA
	}
}

// ClientInsecure configures the client to not verify server certificates.
// Only used for tests.
func ClientInsecure() ClientOption {
	return func(c *clientOpts) {
		c.insecure = true
	}
}

// WithClientTimeout configures the timeout for SCEP client requests.
func WithClientTimeout(timeout *time.Duration) ClientOption {
	return func(c *clientOpts) {
		c.timeout = timeout
	}
}

// MakeClientEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the remote instance, via a transport/http.Client.
// Useful in a SCEP client.
func MakeClientEndpoints(instance string, opts ...ClientOption) (*Endpoints, error) {
	var co clientOpts
	for _, fn := range opts {
		fn(&co)
	}

	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	var fleetOpts []fleethttp.ClientOpt
	if co.timeout != nil {
		fleetOpts = append(fleetOpts, fleethttp.WithTimeout(*co.timeout))
	}

	if co.rootCA != "" || co.insecure {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		switch {
		case co.rootCA != "":
			certs, err := os.ReadFile(co.rootCA)
			if err != nil {
				return nil, fmt.Errorf("reading root CA: %w", err)
			}
			rootCAPool := x509.NewCertPool()
			if ok := rootCAPool.AppendCertsFromPEM(certs); !ok {
				return nil, errors.New("failed to add certificates to root CA pool")
			}
			tlsConfig.RootCAs = rootCAPool
		case co.insecure:
			// Ignoring "G402: TLS InsecureSkipVerify set true", needed for development/testing.
			tlsConfig.InsecureSkipVerify = true //nolint:gosec
		}
		fleetOpts = append(fleetOpts, fleethttp.WithTLSClientConfig(tlsConfig))
	}

	options := []httptransport.ClientOption{httptransport.SetClient(fleethttp.NewClient(fleetOpts...))}

	return &Endpoints{
		GetEndpoint: httptransport.NewClient(
			"GET",
			tgt,
			EncodeSCEPRequest,
			DecodeSCEPResponse,
			options...).Endpoint(),
		PostEndpoint: httptransport.NewClient(
			"POST",
			tgt,
			EncodeSCEPRequest,
			DecodeSCEPResponse,
			options...).Endpoint(),
	}, nil
}

func MakeSCEPEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SCEPRequest)
		resp := SCEPResponse{operation: req.Operation}
		switch req.Operation {
		case "GetCACaps":
			resp.Data, resp.Err = svc.GetCACaps(ctx)
		case "GetCACert":
			resp.Data, resp.CACertNum, resp.Err = svc.GetCACert(ctx, string(req.Message))
		case "PKIOperation":
			resp.Data, resp.Err = svc.PKIOperation(ctx, req.Message)
		default:
			return nil, &BadRequestError{Message: "operation not implemented"}
		}
		return resp, nil
	}
}

// SCEPRequest is a SCEP server request.
type SCEPRequest struct {
	Operation string
	Message   []byte
}

func (r SCEPRequest) scepOperation() string { return r.Operation }

func MakeSCEPEndpointWithIdentifier(svc ServiceWithIdentifier) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SCEPRequestWithIdentifier)
		resp := SCEPResponse{operation: req.Operation}
		switch req.Operation {
		case "GetCACaps":
			resp.Data, resp.Err = svc.GetCACaps(ctx, req.Identifier)
		case "GetCACert":
			resp.Data, resp.CACertNum, resp.Err = svc.GetCACert(ctx, string(req.Message), req.Identifier)
		case "PKIOperation":
			resp.Data, resp.Err = svc.PKIOperation(ctx, req.Message, req.Identifier)
		default:
			return nil, &BadRequestError{Message: "operation not implemented"}
		}
		if errors.Is(resp.Err, context.DeadlineExceeded) {
			return nil, &TimeoutError{Message: resp.Err.Error()}
		}
		return resp, resp.Err
	}
}

// SCEPRequestWithIdentifier is a SCEP server request.
type SCEPRequestWithIdentifier struct {
	SCEPRequest
	Identifier string `url:"identifier"`
}

// SCEPResponse is a SCEP server response.
// Business errors will be encoded as a CertRep message
// with pkiStatus FAILURE and a failInfo attribute.
type SCEPResponse struct {
	operation string
	CACertNum int
	Data      []byte
	Err       error
}

func (r SCEPResponse) scepOperation() string { return r.operation }

// EndpointLoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func EndpointLoggingMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			var keyvals []interface{}
			// check if this is a scep endpoint, if it is, append the method to the log.
			if oper, ok := request.(interface {
				scepOperation() string
			}); ok {
				keyvals = append(keyvals, "op", oper.scepOperation())
			}
			defer func(begin time.Time) {
				logger.Log(append(keyvals, "error", err, "took", time.Since(begin))...)
			}(time.Now())
			return next(ctx, request)
		}
	}
}
