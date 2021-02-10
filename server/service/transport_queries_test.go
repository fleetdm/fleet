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

func TestDecodeCreateQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/queries", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreateQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createQueryRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "select * from time;", *params.payload.Query)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo",
        "query": "select * from time;"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/queries", &body),
	)
}

func TestDecodeModifyQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/queries/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyQueryRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("PATCH")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("PATCH", "/api/v1/fleet/queries/1", &body),
	)
}

func TestDecodeDeleteQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/queries/{name}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteQueryRequest)
		assert.Equal(t, "qwerty", params.Name)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/queries/qwerty", nil),
	)
}

func TestDecodeGetQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/queries/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getQueryRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/queries/1", nil),
	)
}
