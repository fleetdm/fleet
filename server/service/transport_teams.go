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
	req := modifyTeamRequest{ID: id}
	err = json.NewDecoder(r.Body).Decode(&req.payload)
	if err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}

func decodeModifyTeamAgentOptionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := modifyTeamAgentOptionsRequest{ID: id}
	err = json.NewDecoder(r.Body).Decode(&req.options)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func decodeListTeamsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listTeamsRequest{ListOptions: opt}, nil
}

func decodeDeleteTeamRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteTeamRequest{ID: id}, nil
}

func decodeListTeamUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listTeamUsersRequest{TeamID: id, ListOptions: opt}, nil
}

func decodeModifyTeamUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := modifyTeamUsersRequest{TeamID: id}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func decodeTeamEnrollSecretsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := teamEnrollSecretsRequest{TeamID: id}
	return req, nil
}
