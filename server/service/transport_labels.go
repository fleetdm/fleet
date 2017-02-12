package service

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

func decodeCreateLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	return req, nil
}

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

func decodeModifyLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var resp modifyLabelRequest
	err = json.NewDecoder(r.Body).Decode(&resp.payload)
	if err != nil {
		return nil, err
	}
	resp.ID = id
	return resp, nil
}
