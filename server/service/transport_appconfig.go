package service

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func decodeModifyAppConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return appConfigRequest{Payload: payload}, nil
}

func decodeApplyEnrollSecretSpecRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req applyEnrollSecretSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil

}
