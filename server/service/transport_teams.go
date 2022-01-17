package service

import (
	"context"
	"encoding/json"
	"net/http"
)

func decodeModifyTeamAgentOptionsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := modifyTeamAgentOptionsRequest{ID: uint(id)}
	err = json.NewDecoder(r.Body).Decode(&req.options)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func decodeDeleteTeamRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteTeamRequest{ID: uint(id)}, nil
}

func decodeListTeamUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listTeamUsersRequest{TeamID: uint(id), ListOptions: opt}, nil
}

func decodeModifyTeamUsersRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := modifyTeamUsersRequest{TeamID: uint(id)}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func decodeTeamEnrollSecretsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	req := teamEnrollSecretsRequest{TeamID: uint(id)}
	return req, nil
}
