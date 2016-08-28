package kitserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"golang.org/x/net/context"
)

func decodeCreateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeGetUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, errBadRoute
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return getUserRequest{ID: uint(uid)}, nil
}

func decodeModifyUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req modifyUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req.payload); err != nil {
		return nil, err
	}

	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, errBadRoute
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	req.ID = uint(uid)

	return req, nil
}
