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

func TestDecodeGetScheduledQueriesInPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/packs/{id}/scheduled", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetScheduledQueriesInPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getScheduledQueriesInPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/packs/1/scheduled", nil),
	)
}

func TestDecodeScheduleQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/schedule", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeScheduleQueryRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(scheduleQueryRequest)
		assert.Equal(t, uint(5), params.PackID)
		assert.Equal(t, uint(1), params.QueryID)
		assert.Equal(t, uint(60), params.Interval)
		assert.Equal(t, true, *params.Snapshot)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
		"pack_id": 5,
		"query_id": 1,
		"interval": 60,
		"snapshot": true
	}`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/fleet/schedule", &body),
	)
}

func TestDecodeModifyScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
		assert.Equal(t, uint(5), *params.payload.PackID)
		assert.Equal(t, uint(6), *params.payload.QueryID)
		assert.Equal(t, true, *params.payload.Removed)
		assert.Equal(t, uint(60), *params.payload.Interval)
		assert.Equal(t, true, params.payload.Shard.Valid)
		assert.Equal(t, int64(1), params.payload.Shard.Int64)
	}).Methods("PATCH")

	var body bytes.Buffer
	body.Write([]byte(`{
	        "pack_id": 5,
		"query_id": 6,
		"removed": true,
		"interval": 60,
		"shard": 1
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("PATCH", "/api/v1/fleet/scheduled/1", &body),
	)
}

func TestDecodeDeleteScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/fleet/scheduled/1", nil),
	)
}

func TestDecodeGetScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/fleet/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/fleet/scheduled/1", nil),
	)
}
