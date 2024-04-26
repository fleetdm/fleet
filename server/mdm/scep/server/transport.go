package scepserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/groob/finalizer/logutil"
)

func MakeHTTPHandler(e *Endpoints, svc Service, logger kitlog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerFinalizer(logutil.NewHTTPLogger(logger).LoggingFinalizer),
	}

	r := mux.NewRouter()
	r.Methods("GET").Handler(kithttp.NewServer(
		e.GetEndpoint,
		decodeSCEPRequest,
		encodeSCEPResponse,
		opts...,
	))
	r.Methods("POST").Handler(kithttp.NewServer(
		e.PostEndpoint,
		decodeSCEPRequest,
		encodeSCEPResponse,
		opts...,
	))

	return r
}

// EncodeSCEPRequest encodes a SCEP HTTP Request. Used by the client.
func EncodeSCEPRequest(ctx context.Context, r *http.Request, request interface{}) error {
	req := request.(SCEPRequest)
	params := r.URL.Query()
	params.Set("operation", req.Operation)
	switch r.Method {
	case "GET":
		if len(req.Message) > 0 {
			var msg string
			if req.Operation == "PKIOperation" {
				msg = base64.URLEncoding.EncodeToString(req.Message)
			} else {
				msg = string(req.Message)
			}
			params.Set("message", msg)
		}
		r.URL.RawQuery = params.Encode()
		return nil
	case "POST":
		body := bytes.NewReader(req.Message)
		// recreate the request here because IIS does not support chunked encoding by default
		// and Go doesn't appear to set Content-Length if we use an io.ReadCloser
		u := r.URL
		u.RawQuery = params.Encode()
		rr, err := http.NewRequest("POST", u.String(), body)
		rr.Header.Set("Content-Type", "application/octet-stream")
		if err != nil {
			return errors.Join(err, fmt.Errorf("creating new POST request for %s", req.Operation))
		}
		*r = *rr
		return nil
	default:
		return fmt.Errorf("scep: %s method not supported", r.Method)
	}
}

const maxPayloadSize = 2 << 20

func decodeSCEPRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	msg, err := message(r)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	operation := r.URL.Query().Get("operation")
	if len(operation) == 0 {
		return nil, &BadRequestError{Message: "missing operation"}
	}

	request := SCEPRequest{
		Message:   msg,
		Operation: r.URL.Query().Get("operation"),
	}

	return request, nil
}

// extract message from request
func message(r *http.Request) ([]byte, error) {
	switch r.Method {
	case "GET":
		var msg string
		q := r.URL.Query()
		if _, ok := q["message"]; ok {
			msg = q.Get("message")
		}
		op := q.Get("operation")
		if op == "PKIOperation" {
			if len(msg) == 0 {
				return nil, &BadRequestError{Message: "missing PKIOperation message"}
			}

			msg2, err := url.PathUnescape(msg)
			if err != nil {
				return nil, &BadRequestError{Message: fmt.Sprintf("invalid PKIOperation message: %s", msg)}
			}

			decoded, err := base64.StdEncoding.DecodeString(msg2)
			if err != nil {
				return nil, &BadRequestError{Message: fmt.Sprintf("failed to base64 decode message: %s: %s", err.Error(), msg2)}
			}

			return decoded, nil
		}
		return []byte(msg), nil
	case "POST":
		return ioutil.ReadAll(io.LimitReader(r.Body, maxPayloadSize))
	default:
		return nil, errors.New("method not supported")
	}
}

// BadRequestError is an error type that generates a 400 status code.
type BadRequestError struct {
	Message string
}

// Error returns the error message.
func (e *BadRequestError) Error() string {
	return e.Message
}

// StatusCode implements the kithttp StatusCoder interface
func (e *BadRequestError) StatusCode() int { return http.StatusBadRequest }

// EncodeSCEPResponse writes a SCEP response back to the SCEP client.
func encodeSCEPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(SCEPResponse)
	if resp.Err != nil {
		status := http.StatusInternalServerError
		var esc kithttp.StatusCoder
		if errors.As(resp.Err, &esc) {
			status = esc.StatusCode()
		}

		http.Error(w, resp.Err.Error(), status)
		return nil
	}
	w.Header().Set("Content-Type", contentHeader(resp.operation, resp.CACertNum))
	_, _ = w.Write(resp.Data)
	return nil
}

// DecodeSCEPResponse decodes a SCEP response
func DecodeSCEPResponse(ctx context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK && r.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 4096))
		return nil, fmt.Errorf("http request failed with status %s, msg: %s",
			r.Status,
			string(body),
		)
	}
	data, err := ioutil.ReadAll(io.LimitReader(r.Body, maxPayloadSize))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	resp := SCEPResponse{
		Data: data,
	}
	header := r.Header.Get("Content-Type")
	if header == certChainHeader {
		// we only set it to two to indicate a cert chain.
		// the actual number of certs will be in the payload.
		resp.CACertNum = 2
	}
	return resp, nil
}

const (
	certChainHeader = "application/x-x509-ca-ra-cert"
	leafHeader      = "application/x-x509-ca-cert"
	pkiOpHeader     = "application/x-pki-message"
)

func contentHeader(op string, certNum int) string {
	switch op {
	case "GetCACert":
		if certNum > 1 {
			return certChainHeader
		}
		return leafHeader
	case "PKIOperation":
		return pkiOpHeader
	default:
		return "text/plain"
	}
}
