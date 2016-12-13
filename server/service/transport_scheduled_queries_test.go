package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"
)

func TestDecodeScheduleQueriesRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/schedule", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeScheduleQueriesRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(scheduleQueriesRequest)
		require.Len(t, params.Options, 2)

		accessed := struct {
			Five bool
			Six  bool
		}{}

		for _, q := range params.Options {
			switch q.PackID {
			case uint(5):
				accessed.Five = true
				assert.Equal(t, uint(60), q.Interval)
				assert.Equal(t, true, *q.Snapshot)
				assert.Nil(t, q.Removed)
				assert.Len(t, q.QueryIDs, 1)
				assert.Equal(t, q.QueryIDs[0], uint(1))
			case uint(6):
				accessed.Six = true
				assert.Equal(t, uint(120), q.Interval)
				assert.Nil(t, q.Removed)
				assert.Nil(t, q.Snapshot)
				assert.Len(t, q.QueryIDs, 3)
			default:
				t.Errorf("Found an unexpected pack_id: %d", q.PackID)
			}
		}

		if !accessed.Five {
			t.Error("Create scheduled query for pack 5 not read")
		}

		if !accessed.Six {
			t.Error("Create scheduled query for pack 6 not read")
		}

	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
		"options": [{
			"pack_id": 5,
			"interval": 60,
			"snapshot": true,
			"query_ids": [1]
		},{
			"pack_id": 6,
			"interval": 120,
			"query_ids": [1, 2, 3]
		}]
	}`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/kolide/schedule", &body),
	)
}

func TestDecodeModifyScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeModifyScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(modifyScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
		assert.Equal(t, uint(5), params.payload.PackID)
		assert.Equal(t, uint(6), params.payload.QueryID)
		assert.Equal(t, true, *params.payload.Removed)
		assert.Equal(t, uint(60), params.payload.Interval)
		assert.Equal(t, uint(1), *params.payload.Shard)
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
		httptest.NewRequest("PATCH", "/api/v1/kolide/scheduled/1", &body),
	)
}

func TestDecodeDeleteScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeDeleteScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(deleteScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("DELETE")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("DELETE", "/api/v1/kolide/scheduled/1", nil),
	)
}

func TestDecodeGetScheduledQueryRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/scheduled/{id}", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetScheduledQueryRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getScheduledQueryRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/scheduled/1", nil),
	)
}

func TestDecodeGetScheduledQueriesInPackRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/kolide/packs/{id}/scheduled", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetScheduledQueriesInPackRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getScheduledQueriesInPackRequest)
		assert.Equal(t, uint(1), params.ID)
	}).Methods("GET")

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/api/v1/kolide/packs/1/scheduled", nil),
	)
}
