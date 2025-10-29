package tests

import (
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test/httptest"
	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/require"
)

func (ts *WithServer) DoJSON(verb, path string, params interface{}, expectedStatusCode int, v interface{}, queryParams ...string) {
	resp := ts.Do(verb, path, params, expectedStatusCode, queryParams...)
	err := json.UnmarshalRead(resp.Body, v)
	require.NoError(ts.T(), err)
	if e, ok := v.(fleet.Errorer); ok {
		require.NoError(ts.T(), e.Error())
	}
}

func (ts *WithServer) Do(verb, path string, params interface{}, expectedStatusCode int, queryParams ...string) *http.Response {
	j, err := json.Marshal(params)
	require.NoError(ts.T(), err)

	resp := ts.DoRaw(verb, path, j, expectedStatusCode, queryParams...)

	ts.T().Cleanup(func() {
		resp.Body.Close()
	})
	return resp
}

func (ts *WithServer) DoRaw(verb string, path string, rawBytes []byte, expectedStatusCode int, queryParams ...string) *http.Response {
	return ts.DoRawWithHeaders(verb, path, rawBytes, expectedStatusCode, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ts.Token),
	}, queryParams...)
}

func (ts *WithServer) DoRawWithHeaders(
	verb string, path string, rawBytes []byte, expectedStatusCode int, headers map[string]string, queryParams ...string,
) *http.Response {
	opts := []fleethttp.ClientOpt{}
	if expectedStatusCode >= 300 && expectedStatusCode <= 399 {
		opts = append(opts, fleethttp.WithFollowRedir(false))
	}
	client := fleethttp.NewClient(opts...)
	return httptest.DoHTTPReq(ts.T(), client, decodeJSON, verb, rawBytes, ts.Server.URL+path, headers, expectedStatusCode, queryParams...)
}

func decodeJSON(r io.Reader, v interface{}) error {
	return json.UnmarshalRead(r, v)
}
