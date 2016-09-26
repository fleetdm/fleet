package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

var (
	// errBadRoute is used for mux errors
	errBadRoute = errors.New("bad route")
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.status())
		if e.status() == http.StatusNoContent {
			return nil
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

// statuser allows response types to implement a custom
// http success status - default is 200 OK
type statuser interface {
	status() int
}

func idFromRequest(r *http.Request, name string) (uint, error) {
	vars := mux.Vars(r)
	id, ok := vars[name]
	if !ok {
		return 0, errBadRoute
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}
	return uint(uid), nil
}

func decodeNoParamsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}
