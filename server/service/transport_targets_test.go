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

func TestDecodeSearchTargetsRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/targets", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeSearchTargetsRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(searchTargetsRequest)
		assert.Equal(t, "bar", params.Query)
		assert.Len(t, params.Selected.Hosts, 3)
		assert.Len(t, params.Selected.Labels, 2)
	}).Methods("POST")
	var body bytes.Buffer

	body.Write([]byte(`{
        "query": "bar",
		"selected": {
			"hosts": [
				1,
				2,
				3
			],
			"labels": [
				1,
				2
			]
		}
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/targets", &body),
	)
}
