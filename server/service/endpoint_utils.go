package service

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
	"github.com/pkg/errors"
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

func makeDecoderForIDs(v interface{}, idKeys ...string) func(ctx context.Context, r *http.Request) (interface{}, error) {
	t := reflect.TypeOf(v)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		value := reflect.New(t)
		for _, idKey := range idKeys {
			err := setIDFromKey(r, t, value, idKey)
			if err != nil {
				return nil, err
			}
		}

		return value.Interface(), nil
	}
}

func makeDecoderForTypeAndIDs(v interface{}, idKeys ...string) func(ctx context.Context, r *http.Request) (interface{}, error) {
	t := reflect.TypeOf(v)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		req, err := makeDecoderForType(v)(ctx, r)
		if err != nil {
			return nil, err
		}

		value := reflect.ValueOf(req)
		for _, idKey := range idKeys {
			err := setIDFromKey(r, t, value, idKey)
			if err != nil {
				return nil, err
			}
		}

		return req, nil
	}
}

func setIDFromKey(r *http.Request, t reflect.Type, v reflect.Value, idKey string) error {
	id, err := idFromRequest(r, idKey)
	if err != nil {
		return err
	}
	name := ""
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get("url") == idKey {
			name = f.Name
		}
	}
	if name == "" {
		return errors.Errorf("%s not found in URL", idKey)
	}

	field := v.Elem().FieldByName(name)
	field.SetUint(uint64(id))

	return nil
}

func makeDecoderForOptionsAndIDs(v interface{}, idKeys ...string) func(ctx context.Context, r *http.Request) (interface{}, error) {
	t := reflect.TypeOf(v)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		req, err := makeDecoderForIDs(v, idKeys...)(ctx, r)
		if err != nil {
			return nil, err
		}

		value := reflect.ValueOf(req)
		err = setListOptions(r, t, value)
		if err != nil {
			return nil, err
		}

		return req, nil
	}
}

func makeDecoderForTypeOptionsAndIDs(v interface{}, idKeys ...string) func(ctx context.Context, r *http.Request) (interface{}, error) {
	t := reflect.TypeOf(v)
	return func(ctx context.Context, r *http.Request) (interface{}, error) {
		req, err := makeDecoderForTypeAndIDs(v, idKeys...)(ctx, r)
		if err != nil {
			return nil, err
		}

		value := reflect.ValueOf(req)
		err = setListOptions(r, t, value)
		if err != nil {
			return nil, err
		}

		return req, nil
	}
}

func setListOptions(r *http.Request, t reflect.Type, v reflect.Value) error {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return err
	}
	name := ""
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get("url") == "list_options" {
			name = f.Name
		}
	}
	// ListOptions are optional
	if name == "" {
		return nil
	}

	field := v.Elem().FieldByName(name)
	field.Set(reflect.ValueOf(opt))

	return nil
}

func makeAuthenticatedServiceEndpoint(svc fleet.Service, f handlerFunc) endpoint.Endpoint {
	return authenticatedUser(svc, makeServiceEndpoint(svc, f))
}

func makeServiceEndpoint(svc fleet.Service, f handlerFunc) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return f(ctx, request, svc)
	}
}
