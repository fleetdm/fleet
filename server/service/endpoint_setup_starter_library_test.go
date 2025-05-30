package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRoundTripper is a mock implementation of http.RoundTripper for testing
type testRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
	calls         []string // Track URLs that were called
}

func (m *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, req.URL.String())
	return m.RoundTripFunc(req)
}

// Helper function to create HTTP responses for testing
func respond(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestApplyStarterLibrary(t *testing.T) {
	// Read the real production starter library YAML file
	starterLibraryPath := "../../docs/01-Using-Fleet/starter-library/starter-library.yml"
	starterLibraryContent, err := os.ReadFile(starterLibraryPath)
	require.NoError(t, err, "Should be able to read starter library YAML file")

	// Create mock HTTP client for downloading the starter library and scripts
	mockRT := &testRoundTripper{
		calls: []string{},
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.String() == starterLibraryURL:
				// Return the real starter library content
				return respond(200, string(starterLibraryContent)), nil
			case strings.Contains(req.URL.String(), "uninstall-fleetd"):
				// Return a simple script for any script URL
				return respond(200, "#!/bin/bash\necho ok"), nil
			default:
				// For any other URL, return a 404
				return respond(404, "Not found"), nil
			}
		},
	}

	// For debugging: uncomment to print the expected script URL
	// fmt.Printf("Expected script URL: %s\n", scriptsBaseURL+"it-and-security/lib/macos/scripts/uninstall-fleetd-macos.sh")

	httpClientFactory := func(opts ...fleethttp.ClientOpt) *http.Client {
		return &http.Client{Transport: mockRT}
	}

	// Create a client factory that returns a real client
	clientFactory := func(serverURL string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error) {
		// Return a real client - we're not testing the token setting functionality
		return NewClient(serverURL, insecureSkipVerify, rootCA, urlPrefix, options...)
	}

	// Track if ApplyGroup was called and capture the specs
	applyGroupCalled := false
	var capturedSpecs *spec.Group
	// Create a mock ApplyGroup function
	mockApplyGroup := func(ctx context.Context, specs *spec.Group) error {
		applyGroupCalled = true
		capturedSpecs = specs
		return nil
	}

	// Call the function under test
	testErr := applyStarterLibrary(
		context.Background(),
		"https://example.com",
		"test-token",
		kitlog.NewNopLogger(),
		httpClientFactory,
		clientFactory,
		mockApplyGroup,
	)

	// Verify results
	require.NoError(t, testErr)
	assert.True(t, applyGroupCalled, "ApplyGroup should have been called")

	// Verify that the specs were correctly parsed
	require.NotNil(t, capturedSpecs, "Specs should not be nil")
	require.NotEmpty(t, capturedSpecs.Teams, "Specs should contain teams")

	// Verify that the first team has the expected structure
	var team1 map[string]interface{}
	unmarshalErr := json.Unmarshal(capturedSpecs.Teams[0], &team1)
	require.NoError(t, unmarshalErr, "Should be able to unmarshal team JSON")

	// Verify that the team has a name
	require.Contains(t, team1, "name", "Team should have a name")
	teamName := team1["name"].(string)
	require.NotEmpty(t, teamName, "Team name should not be empty")

	// Verify the scripts
	scripts, ok := team1["scripts"].([]interface{})
	require.True(t, ok, "Team should have scripts")
	require.Len(t, scripts, 3, "Team should have 3 scripts")

	// Verify that script references were rewritten to point to the temporary directory
	for _, v := range scripts {
		path := v.(string)
		assert.Contains(t, path, os.TempDir(), "Script path should contain the temporary directory")
		assert.True(t, filepath.IsAbs(path), "Script path should be absolute")
	}

	// Note: We're not testing the token setting functionality

	// Use the fleet package to avoid the unused import error
	_ = fleet.ApplyClientSpecOptions{}

	// Verify that the starter library URL was requested
	assert.Contains(t, mockRT.calls, starterLibraryURL, "The starter library URL should have been requested")
}
