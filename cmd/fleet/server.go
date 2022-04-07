package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/openapi"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

/*
type ErrorContextKey struct{}

type Handler func(w http.ResponseWriter, r *http.Request) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		// set the error in the request context
		errPtr, ok := r.Context().Value(ErrorContextKey{}).(*error)
		if !ok {
			panic("request context does not contain error pointer")
		}
		*errPtr = err
	}
}
*/

// ErrorMiddleware accepts a callback that will be called if there is an error returned from the handler.
func ErrorMiddleware(h func(w http.ResponseWriter, r *http.Request, err error)) openapi.MiddlewareFunc {
	return func(next openapi.HandlerFunc) openapi.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			var err error
			ctx := context.WithValue(r.Context(), openapi.ErrorContextKey{}, &err)
			next.ServeHTTP(w, r.WithContext(ctx))
			if err != nil {
				h(w, r, err)
			}
			return nil
		}
	}
}

type Server struct {
	svc fleet.Service
}

// Ensure that we implement the server interface
var _ openapi.ServerInterface = (*Server)(nil)

// TODO: add status code?
func respondJSON(w http.ResponseWriter, v interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func UserToAPI(user *fleet.User) *openapi.User {
	return &openapi.User{
		Id: user.ID,
	}
}

func (s *Server) GetUserByID(w http.ResponseWriter, r *http.Request, userID uint) error {
	ctx := r.Context()

	user, err := s.svc.User(ctx, userID)
	if err != nil {
		return err
	}

	// TODO: list teams

	userResponse := openapi.UserResponse{
		User: UserToAPI(user),
	}

	return respondJSON(w, &userResponse)
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) error {
	var req openapi.Login
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	req.Email = strings.ToLower(req.Email)
	_, session, err := s.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		return err
	}

	// TODO: add viewer to context
	token := openapi.Token{
		Token: session.Key,
	}

	return respondJSON(w, &token)
}

func (s *Server) GetHosts(w http.ResponseWriter, r *http.Request, params openapi.GetHostsParams) error {
	panic("not implemented")
}
