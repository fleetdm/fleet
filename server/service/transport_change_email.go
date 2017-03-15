package service

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

func decodeChangeEmailRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	token, ok := vars["token"]
	if !ok {
		return nil, errBadRoute
	}

	response := changeEmailRequest{
		Token: token,
	}

	return response, nil
}
