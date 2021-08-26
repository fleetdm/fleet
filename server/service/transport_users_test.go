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

func TestDecodeCreateUserRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/users", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateUserRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createUserRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "foo@fleet.co", *params.payload.Email)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "email": "foo@fleet.co"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/users", &body),
	)
}

func TestDecodeGetUserRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/users/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetUserRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getUserRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/users/1", nil),
	)
}

func TestDecodeChangePasswordRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/change_password", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeChangePasswordRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(changePasswordRequest)
		assert.Equal(t, "foo", params.OldPassword)
		assert.Equal(t, "bar", params.NewPassword)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "old_password": "foo",
        "new_password": "bar"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/change_password", &body),
	)
}

func TestDecodeResetPasswordRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/users/{id}/password", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeResetPasswordRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(resetPasswordRequest)
		assert.Equal(t, "bar", params.NewPassword)
		assert.Equal(t, "baz", params.PasswordResetToken)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "new_password": "bar",
        "password_reset_token": "baz"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/users/1/password", &body),
	)
}

func TestDecodeModifyUserRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/users/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyUserRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyUserRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "foo@fleet.co", *params.payload.Email)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("PATCH")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "email": "foo@fleet.co"
    }`))

	request := httptest.NewRequest("PATCH", "/api/v1/fleet/users/1", &body)
	router.ServeHTTP(
		httptest.NewRecorder(),
		request,
	)
}
