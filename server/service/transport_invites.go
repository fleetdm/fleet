package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/net/context"
)

func decodeCreateInviteRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	if req.payload.Email != nil {
		*req.payload.Email = strings.ToLower(*req.payload.Email)
	}

	return req, nil
}

func decodeDeleteInviteRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var req deleteInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}
