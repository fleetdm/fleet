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
