package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeCreateTeamRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeModifyTeamRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	var resp modifyTeamRequest
	err = json.NewDecoder(r.Body).Decode(&resp.payload)
	if err != nil {
		return nil, err
	}
	resp.ID = id
	return resp, nil
}
