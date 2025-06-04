package httptest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/stretchr/testify/require"
)

func DoHTTPReq(t *testing.T, jsonDecoder func(r io.Reader, v interface{}) error, verb string, rawBytes []byte, urlPath string,
	headers map[string]string, expectedStatusCode int, queryParams ...string) *http.Response {
	requestBody := io.NopCloser(bytes.NewBuffer(rawBytes))
	req, err := http.NewRequest(verb, urlPath, requestBody)
	require.NoError(t, err)
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	opts := []fleethttp.ClientOpt{}
	if expectedStatusCode >= 300 && expectedStatusCode <= 399 {
		opts = append(opts, fleethttp.WithFollowRedir(false))
	}
	client := fleethttp.NewClient(opts...)

	if len(queryParams)%2 != 0 {
		require.Fail(t, "need even number of params: key value")
	}
	if len(queryParams) > 0 {
		q := req.URL.Query()
		for i := 0; i < len(queryParams); i += 2 {
			q.Add(queryParams[i], queryParams[i+1])
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	require.NoError(t, err)

	if resp.StatusCode != expectedStatusCode {
		defer resp.Body.Close()
		var je endpoint_utils.JsonError
		err := jsonDecoder(resp.Body, &je)
		if err != nil {
			t.Logf("Error trying to decode response body as Fleet jsonError: %s", err)
			require.Equal(t, expectedStatusCode, resp.StatusCode, fmt.Sprintf("response: %+v", resp))
		}
		require.Equal(t, expectedStatusCode, resp.StatusCode, fmt.Sprintf("Fleet jsonError: %+v", je))
	}
	return resp
}
