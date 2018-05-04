package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeDeleteLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteLabelRequest
	req.ID = id
	return req, nil
}

func decodeGetLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req getLabelRequest
	req.ID = id
	return req, nil
}

func decodeListLabelsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listLabelsRequest{ListOptions: opt}, nil
}

func decodeApplyLabelSpecsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyLabelSpecsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}
