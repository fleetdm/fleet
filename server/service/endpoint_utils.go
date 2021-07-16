package service

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type handlerFunc func(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error)

func makeDecoderForType(v interface{}) func(ctx context.Context, r *http.Request) (interface{}, error) {
	t := reflect.TypeOf(v)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		req := reflect.New(t).Interface()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		return req, nil
	}
}

func makeAuthenticatedServiceEndpoint(svc fleet.Service, f handlerFunc) endpoint.Endpoint {
	return authenticatedUser(svc, makeServiceEndpoint(svc, f))
}

func makeServiceEndpoint(svc fleet.Service, f handlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return f(ctx, request, svc)
	}
}
