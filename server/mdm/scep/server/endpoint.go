package scepserver

import (
	"bytes"
	"context"
	"errors"
	"net/url"
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

func (e *Endpoints) Supports(cap string) bool {
	e.mtx.RLock()
	defer e.mtx.RUnlock()

	if len(e.capabilities) == 0 {
		e.mtx.RUnlock()
		_, _ = e.GetCACaps(context.Background())
		e.mtx.RLock()
	}
	return bytes.Contains(e.capabilities, []byte(cap))
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

// MakeClientEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the remote instance, via a transport/http.Client.
// Useful in a SCEP client.
func MakeClientEndpoints(instance string, timeout *time.Duration) (*Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	var fleetOpts []fleethttp.ClientOpt
	if timeout != nil {
		fleetOpts = append(fleetOpts, fleethttp.WithTimeout(*timeout))
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
			resp.Data, resp.Err = svc.GetCACaps(ctx)
		case "GetCACert":
			resp.Data, resp.CACertNum, resp.Err = svc.GetCACert(ctx, string(req.Message))
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
