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
	router.HandleFunc("/api/v1/kolide/invites", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateInviteRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createInviteRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "foo@kolide.co", *params.payload.Email)
		assert.Equal(t, uint(1), *params.payload.InvitedBy)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "email": "foo@kolide.co",
        "invited_by": 1
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/kolide/invites", &body),
	)
}
