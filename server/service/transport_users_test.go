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
