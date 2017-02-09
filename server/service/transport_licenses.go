package service

import (
	"encoding/json"
	"net/http"

	"golang.org/x/net/context"
)

func decodeLicenseRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req licenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}
