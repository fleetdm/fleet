package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func TestDecodeGetClientConfigRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetClientConfigRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(getClientConfigRequest)
		assert.Equal(t, "key", params.NodeKey)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "node_key": "key"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/", &body),
	)
}

func TestDecodeGetDistributedQueriesRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeGetDistributedQueriesRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(getDistributedQueriesRequest)
		assert.Equal(t, "key", params.NodeKey)
	}).Methods("POST")

	var body bytes.Buffer
	body.Write([]byte(`{
        "node_key": "key"
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/", &body),
	)
}

func TestDecodeSubmitDistributedQueryResultsRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeSubmitDistributedQueryResultsRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(SubmitDistributedQueryResultsRequest)
		assert.Equal(t, "key", params.NodeKey)
		assert.Equal(t, fleet.OsqueryDistributedQueryResults{
			"id1": {
				{"col1": "val1", "col2": "val2"},
				{"col1": "val3", "col2": "val4"},
			},
			"id2": {
				{"col3": "val5", "col4": "val6"},
			},
			"id3": {},
		}, params.Results)
		assert.Equal(t, map[string]fleet.OsqueryStatus{"id1": 0, "id3": 1}, params.Statuses)
	}).Methods("POST")

	// Note we explicitly test the case that requires using the shim
	// because of the inconsistent JSON schema
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
          ],
          "id3": ""
        },
        "statuses": {"id1": 0, "id3": "1"}
    }`))

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/", &body),
	)
}

func TestDecodeSubmitLogsRequest(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		r, err := decodeSubmitLogsRequest(context.Background(), request)
		require.Nil(t, err)

		params := r.(submitLogsRequest)
		assert.Equal(t, "xOCmmaTJJvGRi8prh4kdjkFMyh7K1bXb", params.NodeKey)
		assert.Equal(t, "status", params.LogType)
	}).Methods("POST")

	bodyJSON := []byte(`
            {
              "node_key":"xOCmmaTJJvGRi8prh4kdjkFMyh7K1bXb",
              "log_type":"status",
              "data":[
                {
                  "severity":"0",
                  "filename":"tls.cpp",
                  "line":"205",
                  "message":"TLS\/HTTPS POST request to URI: https:\/\/dockerhost:8080\/api\/v1\/osquery\/log",
                  "version":"2.3.2",
                  "decorations":{
                    "host_uuid":"EB714C9D-C1F8-A436-B6DA-3F853C5502EA",
                    "hostname":"9bed9dc098d9"
                  }
                }
              ]
            }
`)

	body := new(bytes.Buffer)
	_, err := body.Write(bodyJSON)
	require.Nil(t, err)

	router.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("POST", "/", body),
	)

	// Now try gzipped
	body.Reset()
	gzWriter := gzip.NewWriter(body)
	_, err = gzWriter.Write(bodyJSON)
	require.Nil(t, err)
	require.Nil(t, gzWriter.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Add("Content-Encoding", "gzip")

	router.ServeHTTP(
		httptest.NewRecorder(),
		req,
	)
}
