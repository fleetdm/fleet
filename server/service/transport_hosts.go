package service

import (
	"context"
	"net/http"
)

func decodeGetHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getHostRequest{ID: id}, nil
}

func decodeHostByIdentifierRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	identifier, err := nameFromRequest(r, "identifier")
	if err != nil {
		return nil, err
	}
	return hostByIdentifierRequest{Identifier: identifier}, nil
}

func decodeDeleteHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteHostRequest{ID: id}, nil
}

func decodeListHostsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	hopt, err := hostListOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}

	return listHostsRequest{ListOptions: hopt}, nil
}
