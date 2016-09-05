package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/datastore"
	"golang.org/x/net/context"
)

var (
	// errBadRoute is used for mux errors
	errBadRoute = errors.New("bad route")
)

type invalidArgumentError struct {
	field    string
	required bool
}

// invalidArgumentError is returned when one or more arguments are invalid.
func (e invalidArgumentError) Error() string {
	req := "optional"
	if e.required {
		req = "required"
	}
	return fmt.Sprintf("%s argument invalid or missing: %s", req, e.field)
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	return json.NewEncoder(w).Encode(response)
}

// erroer interface is implemented by response structs to encode business logic errors
type errorer interface {
	error() error
}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err {
	case datastore.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case datastore.ErrExists:
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(typeErrsStatus(err))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

const unprocessableEntity int = 422

func typeErrsStatus(err error) int {
	switch err.(type) {
	case invalidArgumentError:
		return unprocessableEntity
	case authError:
		return http.StatusUnauthorized
	case forbiddenError:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
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
