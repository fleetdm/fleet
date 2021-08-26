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

func TestDecodeCreateInviteRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/invites", func(writer http.ResponseWriter, request *http.Request) {
		_, err := decodeCreateInviteRequest(context.Background(), request)
		assert.Nil(t, err)
	}).Methods("POST")

	t.Run("lowercase email", func(t *testing.T) {
		var body bytes.Buffer
		body.Write([]byte(`{
        "name": "foo",
        "email": "foo@fleet.co"
    }`))

		router.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/api/v1/fleet/invites", &body),
		)
	})

	t.Run("uppercase email", func(t *testing.T) {
		// email string should be lowerased after decode.
		var body bytes.Buffer
		body.Write([]byte(`{
        "name": "foo",
        "email": "Foo@fleet.co"
    }`))

		router.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/api/v1/fleet/invites", &body),
		)
	})

}

func TestDecodeVerifyInviteRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/invites/{token}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateInviteRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(verifyInviteRequest)
		assert.Equal(t, "test_token", params.Token)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/tokens/test_token", nil),
	)

}
