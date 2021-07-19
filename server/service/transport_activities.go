package service

import (
	"context"
	"net/http"
)

func decodeListActivitiesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listActivitiesRequest{ListOptions: opt}, nil
}
