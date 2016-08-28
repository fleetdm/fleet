package kitserver

import (
	"net/http"

	"golang.org/x/net/context"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/kolide/kolide-ose/kolide"
)

// MakeHandler creates an http handler for the Kolide API
func MakeHandler(ctx context.Context, svc kolide.Service, logger kitlog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(encodeError),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}

	createUserHandler := kithttp.NewServer(
		ctx,
		makeCreateUserEndpoint(svc),
		decodeCreateUserRequest,
		encodeResponse,
		opts...,
	)

	var ds kolide.Datastore
	api := mux.NewRouter()
	api.Handle("/api/v1/kolide/users", createUserHandler).Methods("POST")
	r := mux.NewRouter()

	r.PathPrefix("/api/v1/kolide").Handler(authMiddleware(api))
	r.Handle("/login", login(ds, logger)).Methods("POST")
	r.Handle("/logout", logout(ds, logger)).Methods("GET")
	return r
}
