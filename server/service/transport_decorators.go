package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeNewDecoratorRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	var dec newDecoratorRequest
	err := json.NewDecoder(req.Body).Decode(&dec)
	if err != nil {
		return nil, err
	}
	return dec, nil
}

func decodeDeleteDecoratorRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	id, err := idFromRequest(req, "id")
	if err != nil {
		return nil, err
	}
	return deleteDecoratorRequest{ID: id}, nil
}

func decodeModifyDecoratorRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	var request newDecoratorRequest
	id, err := idFromRequest(req, "id")
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		return nil, err
	}
	request.Payload.ID = id
	return request, nil
}
