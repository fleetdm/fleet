package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDecodeGetInfoAboutSessionRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/sessions/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetInfoAboutSessionRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getInfoAboutSessionRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/sessions/1", nil),
	)
}

func TestDecodeGetInfoAboutSessionsForUserRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/user/{id}/sessions", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetInfoAboutSessionsForUserRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getInfoAboutSessionsForUserRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/users/1/sessions", nil),
	)
}

func TestDecodeDeleteSessionRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/sessions/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteSessionRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteSessionRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/sessions/1", nil),
	)
}

func TestDecodeDeleteSessionsForUserRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/user/{id}/sessions", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteSessionsForUserRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteSessionsForUserRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/users/1/sessions", nil),
	)
}

func TestDecodeLoginRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/login", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeLoginRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(loginRequest)
		assert.Equal(t, "foo", params.Email)
		assert.Equal(t, "bar", params.Password)
	}).Methods("POST")
	t.Run("lowercase email", func(t *testing.T) {
		var body bytes.Buffer
		body.Write([]byte(`{
        "email": "foo",
        "password": "bar"
    }`))

		router.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/api/v1/fleet/login", &body),
		)
	})
	t.Run("uppercase email", func(t *testing.T) {
		var body bytes.Buffer
		body.Write([]byte(`{
        "email": "Foo",
        "password": "bar"
    }`))

		router.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/api/v1/fleet/login", &body),
		)
	})

}
