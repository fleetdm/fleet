package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestDecodeCreatePackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreatePackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createPackRequest)
		assert.Equal(t, "foo", *params.payload.Name)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/kolide/packs", &body),
	)
}

func TestDecodeModifyPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyPackRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("PATCH")

	var body bytes.Buffer
	body.Write([]byte(`{
        "name": "foo"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("PATCH", "/api/v1/kolide/packs/1", &body),
	)
}

func TestDecodeDeletePackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeletePackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deletePackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/kolide/packs/1", nil),
	)
}

func TestDecodeGetPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1", nil),
	)
}

func TestDecodeAddQueryToPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{pid}/queries/{qid}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeAddQueryToPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(addQueryToPackRequest)
		assert.Equal(t, uint(1), params.PackID)
		assert.Equal(t, uint(2), params.QueryID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1/queries/2", nil),
	)
}

func TestDecodeGetQueriesInPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}/queries", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetQueriesInPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getQueriesInPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1/queries", nil),
	)
}

func TestDecodeDeleteQueryFromPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{pid}/queries/{qid}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteQueryFromPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteQueryFromPackRequest)
		assert.Equal(t, uint(1), params.PackID)
		assert.Equal(t, uint(2), params.QueryID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/kolide/packs/1/queries/2", nil),
	)
}
