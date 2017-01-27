package service

import (
	"net/http"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
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
