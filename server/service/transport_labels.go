package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeDeleteLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	name, err := nameFromRequest(r, "name")
	if err != nil {
		return nil, err
	}
	var req deleteLabelRequest
	req.Name = name
	return req, nil
}

func decodeDeleteLabelByIDRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteLabelByIDRequest
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

func decodeListHostsInLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}

	hopt, err := hostListOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}

	return listHostsInLabelRequest{ID: id, ListOptions: hopt}, nil
}

func decodeApplyLabelSpecsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyLabelSpecsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}

func decodeCreateLabelRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	return req, nil
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
