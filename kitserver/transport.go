package kitserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kolide/kolide-ose/datastore"
	"golang.org/x/net/context"
)

var (
	// errInvalidArgument is returned when one or more arguments are invalid.
	errInvalidArgument = errors.New("invalid argument")

	// errBadRoute is used for mux errors
	errBadRoute = errors.New("bad route")
)

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
	case errInvalidArgument:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(typeErrsStatus(err))
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func typeErrsStatus(err error) int {
	switch err.(type) {
	case authError:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
