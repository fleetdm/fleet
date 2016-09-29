package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestDecodeEnrollAgentRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/osquery/enroll", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeEnrollAgentRequest(context.Background(), request)
		assert.Nil(t, err)

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
		httptest.NewRequest("POST", "/api/v1/osquery/enroll", &body),
	)
}

func TestDecodeGetClientConfigRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/osquery/enroll", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetClientConfigRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getClientConfigRequest)
		assert.Equal(t, "key", params.NodeKey)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "node_key": "key"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/osquery/enroll", &body),
	)
}

func TestDecodeGetDistributedQueriesRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/osquery/enroll", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetDistributedQueriesRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(getDistributedQueriesRequest)
		assert.Equal(t, "key", params.NodeKey)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "node_key": "key"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/osquery/enroll", &body),
	)
}

func TestDecodeSubmitDistributedQueryResultsRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/osquery/enroll", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeSubmitDistributedQueryResultsRequest(context.Background(), request)
		assert.Nil(t, err)

		params := r.(submitDistributedQueryResultsRequest)
		assert.Equal(t, "key", params.NodeKey)
		assert.Equal(t, kolide.OsqueryDistributedQueryResults{
			"id1": {
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
			"id2": {
				{"col3": "val5", "col4": "val6"},
			},
		}, params.Results)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "node_key": "key",
        "queries": {
          "id1": [
            {"col1": "val1", "col2": "val2"},
            {"col1": "val3", "col2": "val4"}
          ],
          "id2": [
            {"col3": "val5", "col4": "val6"}
          ]
        }
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/api/v1/osquery/enroll", &body),
	)
}
