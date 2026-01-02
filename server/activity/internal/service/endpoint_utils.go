package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	eu "github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

// default number of items to include per page
const defaultPerPage = 20

// AuthMiddleware is a type alias for endpoint middleware functions.
type AuthMiddleware = func(endpoint.Endpoint) endpoint.Endpoint

// encodeResponse encodes the response as JSON.
func encodeResponse(ctx context.Context, w http.ResponseWriter, response any) error {
	return eu.EncodeCommonResponse(ctx, w, response,
		func(w http.ResponseWriter, response any) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			return enc.Encode(response)
		},
		nil, // no domain-specific error encoder
	)
}

// makeDecoder creates a decoder for the given request type.
func makeDecoder(iface any) kithttp.DecodeRequestFunc {
	return eu.MakeDecoder(iface, func(body io.Reader, req any) error {
		return json.NewDecoder(body).Decode(req)
	}, parseCustomTags, nil, nil, nil)
}

// parseCustomTags handles custom URL tag values for activity requests.
func parseCustomTags(urlTagValue string, r *http.Request, field reflect.Value) (bool, error) {
	if urlTagValue == "list_options" {
		opts, err := listOptionsFromRequest(r)
		if err != nil {
			return false, err
		}
		field.Set(reflect.ValueOf(opts))
		return true, nil
	}
	return false, nil
}

// listOptionsFromRequest parses list options from query parameters.
func listOptionsFromRequest(r *http.Request) (api.ListOptions, error) {
	var err error

	pageString := r.URL.Query().Get("page")
	perPageString := r.URL.Query().Get("per_page")
	orderKey := r.URL.Query().Get("order_key")
	orderDirectionString := r.URL.Query().Get("order_direction")

	var page int
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "non-int page value"})
		}
		if page < 0 {
			return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "negative page value"})
		}
	}

	var perPage int
	if perPageString != "" {
		perPage, err = strconv.Atoi(perPageString)
		if err != nil {
			return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "non-int per_page value"})
		}
		if perPage <= 0 {
			return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "invalid per_page value"})
		}
	}

	if perPage == 0 && pageString != "" {
		// We explicitly set a non-zero default if a page is specified
		perPage = defaultPerPage
	}

	if orderKey == "" && orderDirectionString != "" {
		return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "order_key must be specified with order_direction"})
	}

	var orderDirection string
	switch orderDirectionString {
	case "desc":
		orderDirection = "desc"
	case "asc", "":
		orderDirection = "asc"
	default:
		return api.ListOptions{}, ctxerr.Wrap(r.Context(), &platform_http.BadRequestError{Message: "unknown order_direction: " + orderDirectionString})
	}

	return api.ListOptions{
		Page:           uint(page),    //nolint:gosec // dismiss G115
		PerPage:        uint(perPage), //nolint:gosec // dismiss G115
		OrderKey:       orderKey,
		OrderDirection: orderDirection,
	}, nil
}

// handlerFunc is the handler function type for Activity service endpoints.
type handlerFunc func(ctx context.Context, request any, svc api.Service) platform_http.Errorer

// Compile-time check to ensure endpointer implements Endpointer.
var _ eu.Endpointer[handlerFunc] = &endpointer{}

type endpointer struct {
	svc api.Service
}

func (e *endpointer) CallHandlerFunc(f handlerFunc, ctx context.Context, request any,
	svc any) (platform_http.Errorer, error) {
	return f(ctx, request, svc.(api.Service)), nil
}

func (e *endpointer) Service() any {
	return e.svc
}

func newUserAuthenticatedEndpointer(svc api.Service, authMiddleware AuthMiddleware, opts []kithttp.ServerOption, r *mux.Router,
	versions ...string) *eu.CommonEndpointer[handlerFunc] {
	return &eu.CommonEndpointer[handlerFunc]{
		EP: &endpointer{
			svc: svc,
		},
		MakeDecoderFn:  makeDecoder,
		EncodeFn:       encodeResponse,
		Opts:           opts,
		AuthMiddleware: authMiddleware,
		Router:         r,
		Versions:       versions,
	}
}

// fillListOptions sets default values for list options.
// Note: IncludeMetadata is set internally by the service layer.
func fillListOptions(opt *api.ListOptions) {
	// Default ordering by created_at descending (newest first) if not specified
	if opt.OrderKey == "" {
		opt.OrderKey = "created_at"
		opt.OrderDirection = "desc"
	}
}
