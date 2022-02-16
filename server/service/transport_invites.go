package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

func decodeVerifyInviteRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	token, ok := vars["token"]
	if !ok {
		return 0, errBadRoute
	}
	return verifyInviteRequest{Token: token}, nil
}
