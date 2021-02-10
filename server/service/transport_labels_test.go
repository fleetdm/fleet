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

func TestDecodeDeleteLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/labels/{name}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteLabelRequest)
		assert.Equal(t, "foo", params.Name)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/labels/foo", nil),
	)
}

func TestDecodeGetLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/labels/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getLabelRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/labels/1", nil),
	)
}

func TestDecodeCreateLabelRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/labels", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateLabelRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createLabelRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "select * from foo;", *params.payload.Query)
		assert.Equal(t, "darwin", *params.payload.Platform)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "query": "select * from foo;",
		"platform": "darwin"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/labels", &body),
	)
}
