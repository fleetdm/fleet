package integrationtest

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test/httptest"
	"github.com/go-json-experiment/json"
	"github.com/stretchr/testify/require"
)

func (s *BaseSuite) DoJSON(t *testing.T, verb, path string, params interface{}, expectedStatusCode int, v interface{}, queryParams ...string) {
	resp := s.Do(t, verb, path, params, expectedStatusCode, queryParams...)
	err := json.UnmarshalRead(resp.Body, v)
	require.NoError(t, err)
	if e, ok := v.(fleet.Errorer); ok {
		require.NoError(t, e.Error())
	}
}

func (s *BaseSuite) Do(t *testing.T, verb, path string, params interface{}, expectedStatusCode int, queryParams ...string) *http.Response {
	j, err := json.Marshal(params)
	require.NoError(t, err)

	resp := s.DoRaw(t, verb, path, j, expectedStatusCode, queryParams...)

	t.Cleanup(func() {
		resp.Body.Close()
	})
	return resp
}

func (s *BaseSuite) DoRaw(t *testing.T, verb string, path string, rawBytes []byte, expectedStatusCode int, queryParams ...string) *http.Response {
	return s.DoRawWithHeaders(t, verb, path, rawBytes, expectedStatusCode, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.Token),
	}, queryParams...)
}

func (s *BaseSuite) DoRawWithHeaders(
	t *testing.T, verb string, path string, rawBytes []byte, expectedStatusCode int, headers map[string]string, queryParams ...string,
) *http.Response {
	opts := []fleethttp.ClientOpt{}
	if expectedStatusCode >= 300 && expectedStatusCode <= 399 {
		opts = append(opts, fleethttp.WithFollowRedir(false))
	}
	client := fleethttp.NewClient(opts...)
	return httptest.DoHTTPReq(t, client, decodeJSON, verb, rawBytes, s.Server.URL+path, headers, expectedStatusCode, queryParams...)
}

func decodeJSON(r io.Reader, v interface{}) error {
	return json.UnmarshalRead(r, v)
}
