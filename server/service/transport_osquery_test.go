package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEnrollAgentRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeEnrollAgentRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(enrollAgentRequest)
		assert.Equal(t, "secret", params.EnrollSecret)
		assert.Equal(t, "uuid", params.HostIdentifier)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "enroll_secret": "secret",
        "host_identifier": "uuid"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/", &body),
	)
}
