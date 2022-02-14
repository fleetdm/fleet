package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

func decodeDeleteInviteRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteInviteRequest{ID: uint(id)}, nil
}

func decodeVerifyInviteRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	token, ok := vars["token"]
	if !ok {
		return 0, errBadRoute
	}
	return verifyInviteRequest{Token: token}, nil
}

func decodeListInvitesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return nil, err
	}
	return listInvitesRequest{ListOptions: opt}, nil
}
