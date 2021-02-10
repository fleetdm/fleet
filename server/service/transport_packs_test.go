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

func TestDecodeCreatePackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/packs", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeCreatePackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(createPackRequest)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "bar", *params.payload.Description)
		require.NotNil(t, params.payload.HostIDs)
		assert.Len(t, *params.payload.HostIDs, 3)
		require.NotNil(t, params.payload.LabelIDs)
		assert.Len(t, *params.payload.LabelIDs, 2)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
		"name": "foo",
		"description": "bar",
		"host_ids": [1, 2, 3],
		"label_ids": [1, 5]
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/packs", &body),
	)
}

func TestDecodeModifyPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyPackRequest)
		assert.Equal(t, uint(1), params.ID)
		assert.Equal(t, "foo", *params.payload.Name)
		assert.Equal(t, "bar", *params.payload.Description)
		require.NotNil(t, params.payload.HostIDs)
		assert.Len(t, *params.payload.HostIDs, 3)
		require.NotNil(t, params.payload.LabelIDs)
		assert.Len(t, *params.payload.LabelIDs, 2)
	}).Methods("PATCH")

	var body bytes.Buffer
	body.Write([]byte(`{
		"name": "foo",
		"description": "bar",
		"host_ids": [1, 2, 3],
		"label_ids": [1, 5]
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("PATCH", "/api/v1/fleet/packs/1", &body),
	)
}

func TestDecodeDeletePackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/packs/{name}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeletePackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deletePackRequest)
		assert.Equal(t, "packaday", params.Name)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/packs/packaday", nil),
	)
}

func TestDecodeGetPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/packs/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/packs/1", nil),
	)
}
