package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestServeFrontend(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}
	logger := log.NewLogfmtLogger(os.Stdout)
	h := ServeFrontend("", false, logger)
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.Close()
	})

	// Simulate a misconfigured osquery sending log requests to the root endpoint.
	requestBody := []byte(`
	{"data":[{"snapshot":[{"build_distro":"10.14","build_platform":"darwin","config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5",
	"config_valid":"1","extensions":"active","instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21",
	"start_time":"1707768989","uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_50","hostIdentifier":"589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:40 2024 UTC",
	"unixTime":1707769000,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"589966AE-074A-503B-B17B-54B05684A120",
	"hostname":"foobar.local"}},{"snapshot":[{"build_distro":"10.14","build_platform":"darwin",
	"config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5","config_valid":"1","extensions":"active",
	"instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21","start_time":"1707768989",
	"uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_28","hostIdentifier": "589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:41 2024 UTC",
	"unixTime":1707769001,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"408F3B27-434F-4776-8538-DA394A3D545F",
	"hostname":"foobar.local"}}],"log_type":"result","node_key":"J9pA1CmjydHGi0bqS1XkkR9pOJQJzoPA"}`)
	response, err := http.DefaultClient.Post(ts.URL, "", bytes.NewReader(requestBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

func TestServeEndUserEnrollOTA(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}

	ds := new(mock.DataStore)
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return []*fleet.User{{}}, nil
	}
	svc, _ := newTestService(t, ds, nil, nil)

	logger := log.NewLogfmtLogger(os.Stdout)
	h := ServeEndUserEnrollOTA(svc, "", logger)
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.Close()
	})

	// assert html is returned
	response, err := http.DefaultClient.Get(ts.URL + "?enroll_secret=foo")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, response.Header.Get("Content-Type"), "text/html; charset=utf-8")

	// assert it contains the content we expect
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	bodyString := string(bodyBytes)
	require.Contains(t, bodyString, "api/v1/fleet/enrollment_profiles/ota?enroll_secret=foo")
}
