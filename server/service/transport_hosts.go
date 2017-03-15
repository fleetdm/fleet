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

func decodeDeleteHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteHostRequest{ID: id}, nil
}

func decodeListHostsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listHostsRequest{ListOptions: opt}, nil
}
