package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper is a custom http.RoundTripper that redirects requests to a mock server
type mockRoundTripper struct {
	mockServer  string
	origBaseURL string
	next        http.RoundTripper
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// If the request URL contains the original base URL, replace it with the mock server URL
	if strings.Contains(req.URL.String(), rt.origBaseURL) {
		// Extract the path from the original URL
		path := strings.TrimPrefix(req.URL.Path, "/")

		// Create a new URL with the mock server
		newURL := fmt.Sprintf("%s/%s", rt.mockServer, path)
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}

		// Copy headers
		newReq.Header = req.Header

		// Use the next transport to perform the request
		return rt.next.RoundTrip(newReq)
	}

	// For other requests, use the next transport
	return rt.next.RoundTrip(req)
}

// Helper function to create HTTP responses for testing
func createTestResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// testRoundTripper2 is a mock implementation of http.RoundTripper for tracking URL calls
type testRoundTripper2 struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
	calls         []string // Track URLs that were called
}

func (m *testRoundTripper2) RoundTrip(req *http.Request) (*http.Response, error) {
	m.calls = append(m.calls, req.URL.String())
	return m.RoundTripFunc(req)
}

func TestExtractScriptNames(t *testing.T) {
	tests := []struct {
		name     string
		teams    []map[string]interface{}
		expected []string
	}{
		{
			name: "multiple teams with scripts",
			teams: []map[string]interface{}{
				{
					"name":    "Team1",
					"scripts": []interface{}{"script1.sh", "script2.sh"},
				},
				{
					"name":    "Team2",
					"scripts": []interface{}{"script2.sh", "script3.sh"}, // Note: script2.sh is duplicated
				},
				{
					"name": "Team3", // No scripts
				},
			},
			expected: []string{"script1.sh", "script2.sh", "script3.sh"},
		},
		{
			name:     "no teams",
			teams:    []map[string]interface{}{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create spec group from test data
			var teams []json.RawMessage
			for _, team := range tt.teams {
				teamRaw, err := json.Marshal(team)
				require.NoError(t, err)
				teams = append(teams, teamRaw)
			}
			specs := &spec.Group{Teams: teams}

			// Call the function
			scriptNames := ExtractScriptNames(specs)

			// Verify the results
			assert.Len(t, scriptNames, len(tt.expected))
			for _, name := range tt.expected {
				assert.Contains(t, scriptNames, name)
			}
		})
	}
}

func TestDownloadAndUpdateScripts(t *testing.T) {
	tests := []struct {
		name        string
		scriptNames []string
		scriptPaths []string
	}{
		{
			name:        "single script",
			scriptNames: []string{"test-script.sh"},
			scriptPaths: []string{"test-script.sh"},
		},
		{
			name:        "multiple scripts with nested path",
			scriptNames: []string{"test-script.sh", "subfolder/nested-script.sh"},
			scriptPaths: []string{"test-script.sh", "subfolder/nested-script.sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server to serve the scripts
			scriptContent := "#!/bin/bash\necho 'Hello, World!'"
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(scriptContent))
			}))
			defer mockServer.Close()

			// Save the original HTTP transport
			origTransport := http.DefaultTransport

			// Create a custom transport that redirects requests to our mock server
			mockTransport := &mockRoundTripper{
				mockServer:  mockServer.URL,
				origBaseURL: scriptsBaseURL,
				next:        origTransport,
			}

			// Replace the default transport with our mock transport
			http.DefaultTransport = mockTransport

			// Restore the original transport when the test is done
			defer func() {
				http.DefaultTransport = origTransport
			}()

			// Create a temporary directory
			tempDir, err := os.MkdirTemp("", "fleet-test-scripts-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create a test spec group
			teamData := map[string]interface{}{
				"name":    "Team1",
				"scripts": []interface{}{tt.scriptNames[0]},
			}
			teamRaw, err := json.Marshal(teamData)
			require.NoError(t, err)

			specs := &spec.Group{
				Teams: []json.RawMessage{teamRaw},
			}

			// Call the actual production function
			err = DownloadAndUpdateScripts(context.Background(), specs, tt.scriptNames, tempDir, kitlog.NewNopLogger())
			require.NoError(t, err)

			// Verify the scripts were downloaded
			for _, scriptName := range tt.scriptPaths {
				scriptPath := filepath.Join(tempDir, scriptName)
				_, err := os.Stat(scriptPath)
				assert.NoError(t, err, "Script should exist: %s", scriptPath)

				// Verify the content
				content, err := os.ReadFile(scriptPath)
				require.NoError(t, err)
				assert.Equal(t, scriptContent, string(content))
			}

			// Verify the specs were updated
			var updatedTeamData map[string]interface{}
			err = json.Unmarshal(specs.Teams[0], &updatedTeamData)
			require.NoError(t, err)

			updatedScripts, ok := updatedTeamData["scripts"].([]interface{})
			require.True(t, ok)

			// The scripts should now be local paths
			for i, script := range updatedScripts {
				scriptPath, ok := script.(string)
				require.True(t, ok)
				assert.Contains(t, scriptPath, tempDir)
				assert.Contains(t, scriptPath, tt.scriptNames[i])
			}
		})
	}
}

func TestDownloadAndUpdateScriptsWithInvalidPaths(t *testing.T) {
	tests := []struct {
		name        string
		scriptNames []string
		errorMsg    string
	}{
		{
			name:        "path traversal attempt",
			scriptNames: []string{"../test-script.sh"},
			errorMsg:    "invalid script name",
		},
		{
			name:        "absolute path attempt",
			scriptNames: []string{"/etc/passwd"},
			errorMsg:    "invalid script name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock server to serve the scripts
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("script content"))
			}))
			defer mockServer.Close()

			// Save the original HTTP transport
			origTransport := http.DefaultTransport

			// Create a custom transport that redirects requests to our mock server
			mockTransport := &mockRoundTripper{
				mockServer:  mockServer.URL,
				origBaseURL: scriptsBaseURL,
				next:        origTransport,
			}

			// Replace the default transport with our mock transport
			http.DefaultTransport = mockTransport

			// Restore the original transport when the test is done
			defer func() {
				http.DefaultTransport = origTransport
			}()

			// Create a temporary directory
			tempDir, err := os.MkdirTemp("", "fleet-test-scripts-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create a test spec group
			teamData := map[string]interface{}{
				"name":    "Team1",
				"scripts": []interface{}{tt.scriptNames[0]},
			}
			teamRaw, err := json.Marshal(teamData)
			require.NoError(t, err)

			specs := &spec.Group{
				Teams: []json.RawMessage{teamRaw},
			}

			// Call the actual production function
			err = DownloadAndUpdateScripts(context.Background(), specs, tt.scriptNames, tempDir, kitlog.NewNopLogger())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestFleetHTTPClientOverride(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer mockServer.Close()

	// Create a custom client that uses our mock server
	client := fleethttp.NewClient()
	client.Transport = &mockRoundTripper{
		mockServer:  mockServer.URL,
		origBaseURL: scriptsBaseURL,
		next:        http.DefaultTransport,
	}

	// Create a fleethttp client with our custom transport
	fleetClient := fleethttp.NewClient()
	// Replace the client's transport with our mock transport
	fleetClient.Transport = client.Transport

	// Create a request to the original URL
	req, err := http.NewRequest("GET", scriptsBaseURL+"/test.sh", nil)
	require.NoError(t, err)

	// Send the request
	resp, err := fleetClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the response
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "test response", string(body))
}

func TestDownloadAndUpdateScriptsTimeout(t *testing.T) {
	tests := []struct {
		name        string
		sleepTime   time.Duration
		contextTime time.Duration
		expectError bool
	}{
		{
			name:        "timeout occurs",
			sleepTime:   6 * time.Second, // Server sleeps longer than timeout
			contextTime: 2 * time.Second, // Short context timeout
			expectError: true,
		},
		{
			name:        "no timeout",
			sleepTime:   1 * time.Second, // Server responds quickly
			contextTime: 5 * time.Second, // Longer context timeout
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip the long-running test in short mode
			if tt.sleepTime > 2*time.Second && testing.Short() {
				t.Skip("Skipping long-running test in short mode")
			}

			// Create a mock server that delays its response
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.sleepTime)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test response"))
			}))
			defer mockServer.Close()

			// Save the original HTTP transport
			origTransport := http.DefaultTransport

			// Create a custom transport that redirects requests to our mock server
			mockTransport := &mockRoundTripper{
				mockServer:  mockServer.URL,
				origBaseURL: scriptsBaseURL,
				next:        origTransport,
			}

			// Replace the default transport with our mock transport
			http.DefaultTransport = mockTransport

			// Restore the original transport when the test is done
			defer func() {
				http.DefaultTransport = origTransport
			}()

			// Create a temporary directory
			tempDir, err := os.MkdirTemp("", "fleet-test-scripts-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create a test spec group
			teamData := map[string]interface{}{
				"name":    "Team1",
				"scripts": []interface{}{"test-script.sh"},
			}
			teamRaw, err := json.Marshal(teamData)
			require.NoError(t, err)

			specs := &spec.Group{
				Teams: []json.RawMessage{teamRaw},
			}

			// Define script names
			scriptNames := []string{"test-script.sh"}

			// Set context timeout
			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTime)
			defer cancel()

			// Call the actual production function
			err = DownloadAndUpdateScripts(ctx, specs, scriptNames, tempDir, kitlog.NewNopLogger())

			if tt.expectError {
				require.Error(t, err)
				// The error message might vary depending on the HTTP client implementation,
				// but it should contain either "timeout", "deadline exceeded", or "context canceled"
				errorMsg := strings.ToLower(err.Error())
				timeoutRelated := strings.Contains(errorMsg, "timeout") ||
					strings.Contains(errorMsg, "deadline exceeded") ||
					strings.Contains(errorMsg, "context canceled")
				assert.True(t, timeoutRelated, "Expected a timeout-related error, got: %s", err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestApplyStarterLibraryWithMockClient(t *testing.T) {
	// Read the real production starter library YAML file
	starterLibraryPath := "../../docs/01-Using-Fleet/starter-library/starter-library.yml"
	starterLibraryContent, err := os.ReadFile(starterLibraryPath)
	require.NoError(t, err, "Should be able to read starter library YAML file")

	// Create mock HTTP client for downloading the starter library and scripts
	mockRT := &testRoundTripper2{
		calls: []string{},
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.String() == starterLibraryURL:
				// Return the real starter library content
				return createTestResponse(200, string(starterLibraryContent)), nil
			case strings.Contains(req.URL.String(), "uninstall-fleetd"):
				// Return a simple script for any script URL
				return createTestResponse(200, "#!/bin/bash\necho ok"), nil
			default:
				// For any other URL, return a 404
				return createTestResponse(404, "Not found"), nil
			}
		},
	}

	httpClientFactory := func(opts ...fleethttp.ClientOpt) *http.Client {
		client := fleethttp.NewClient(opts...)
		client.Transport = mockRT
		return client
	}

	// Create a client factory that returns a real client
	// We're not testing the token setting functionality
	clientFactory := NewClient

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
	testErr := ApplyStarterLibrary(
		context.Background(),
		"https://example.com",
		"test-token",
		kitlog.NewNopLogger(),
		httpClientFactory,
		clientFactory,
		mockApplyGroup,
		false, // Don't skip teams on free license for this test
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

	// Verify that the starter library URL was requested
	assert.Contains(t, mockRT.calls, starterLibraryURL, "The starter library URL should have been requested")
}

func TestApplyStarterLibraryWithMalformedYAML(t *testing.T) {
	// Create mock HTTP client that returns malformed YAML
	malformedYAML := `
	teams:
	- name: "Malformed Team
	  # Missing closing quote and improper indentation
	scripts:
	- "script1.sh
	`

	mockRT := &testRoundTripper2{
		calls: []string{},
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.String() == starterLibraryURL:
				// Return malformed YAML content
				return createTestResponse(200, malformedYAML), nil
			default:
				// For any other URL, return a 404
				return createTestResponse(404, "Not found"), nil
			}
		},
	}

	httpClientFactory := func(opts ...fleethttp.ClientOpt) *http.Client {
		client := fleethttp.NewClient(opts...)
		client.Transport = mockRT
		return client
	}

	// Create a client factory that returns a real client
	clientFactory := NewClient

	// Create a mock ApplyGroup function that should not be called
	mockApplyGroup := func(ctx context.Context, specs *spec.Group) error {
		t.Error("ApplyGroup should not be called with malformed YAML")
		return nil
	}

	// Use a defer/recover to explicitly catch any panics
	var panicValue interface{}

	defer func() {
		if r := recover(); r != nil {
			panicValue = r
			t.Fatalf("Panic occurred when processing malformed YAML: %v", panicValue)
		}
	}()

	// Call the function under test
	testErr := ApplyStarterLibrary(
		context.Background(),
		"https://example.com",
		"test-token",
		kitlog.NewNopLogger(),
		httpClientFactory,
		clientFactory,
		mockApplyGroup,
		false, // Don't skip teams on free license for this test
	)

	// Verify results
	require.Error(t, testErr, "Should return an error with malformed YAML")
	assert.Contains(t, testErr.Error(), "failed to parse starter library",
		"Error should indicate YAML parsing failure")

	// Verify that the starter library URL was requested
	assert.Contains(t, mockRT.calls, starterLibraryURL, "The starter library URL should have been requested")

	// If we reach here, no panic occurred and the setup flow was not interrupted
}

func TestApplyStarterLibraryWithFreeLicense(t *testing.T) {
	// Read the real production starter library YAML file
	starterLibraryPath := "../../docs/01-Using-Fleet/starter-library/starter-library.yml"
	starterLibraryContent, err := os.ReadFile(starterLibraryPath)
	require.NoError(t, err, "Should be able to read starter library YAML file")

	// Create mock HTTP client for downloading the starter library and scripts
	mockRT := &testRoundTripper2{
		calls: []string{},
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.String() == starterLibraryURL:
				// Return the real starter library content
				return createTestResponse(200, string(starterLibraryContent)), nil
			case strings.Contains(req.URL.String(), "uninstall-fleetd"):
				// Return a simple script for any script URL
				return createTestResponse(200, "#!/bin/bash\necho ok"), nil
			default:
				// For any other URL, return a 404
				return createTestResponse(404, "Not found"), nil
			}
		},
	}

	httpClientFactory := func(opts ...fleethttp.ClientOpt) *http.Client {
		client := fleethttp.NewClient(opts...)
		client.Transport = mockRT
		return client
	}

	// Create a mock client that returns a free license
	// Create a properly structured EnrichedAppConfig
	mockEnrichedAppConfig := &fleet.EnrichedAppConfig{}
	// Set the License field using json marshaling/unmarshaling to bypass unexported field access
	configJSON := []byte(`{"license":{"tier":"free"}}`)
	if err := json.Unmarshal(configJSON, mockEnrichedAppConfig); err != nil {
		t.Fatal("Failed to unmarshal mock config:", err)
	}

	// Create a mock client factory
	clientFactory := func(serverURL string, insecureSkipVerify bool, rootCA, urlPrefix string, options ...ClientOption) (*Client, error) {
		mockClient := &Client{}

		// Override the baseClient with a mock implementation
		// Create a mock HTTP client
		mockHTTPClient := &mockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				// Mock the GetAppConfig response
				if req.URL.Path == "/api/v1/fleet/config" && req.Method == http.MethodGet {
					respBody, _ := json.Marshal(mockEnrichedAppConfig)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBuffer(respBody)),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("{}")),
					Header:     make(http.Header),
				}, nil
			},
		}

		// Set up the baseClient with the mock HTTP client and a valid baseURL
		baseURL, _ := url.Parse(serverURL)
		mockClient.baseClient = &baseClient{
			http:    mockHTTPClient,
			baseURL: baseURL,
		}

		return mockClient, nil
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

	// Call the function under test with skipTeamsOnFreeLicense=true
	testErr := ApplyStarterLibrary(
		context.Background(),
		"https://example.com",
		"test-token",
		kitlog.NewNopLogger(),
		httpClientFactory,
		clientFactory,
		mockApplyGroup,
		true, // Skip teams on free license
	)

	// Verify results
	require.NoError(t, testErr)
	assert.True(t, applyGroupCalled, "ApplyGroup should have been called")

	// Verify that the specs were correctly parsed
	require.NotNil(t, capturedSpecs, "Specs should not be nil")

	// Verify that teams were removed
	require.Empty(t, capturedSpecs.Teams, "Teams should be empty for free license")

	// Verify that policies referencing teams were filtered out
	if capturedSpecs.Policies != nil {
		for _, policy := range capturedSpecs.Policies {
			assert.Empty(t, policy.Team, "Policies should not reference teams for free license")
		}
	}

	// Verify that scripts were removed from AppConfig
	if capturedSpecs.AppConfig != nil {
		appConfigMap, ok := capturedSpecs.AppConfig.(map[string]interface{})
		if ok {
			_, hasScripts := appConfigMap["scripts"]
			assert.False(t, hasScripts, "AppConfig should not contain scripts for free license")
		}
	}

	// Verify that the starter library URL was requested
	assert.Contains(t, mockRT.calls, starterLibraryURL, "The starter library URL should have been requested")
}

// mockHTTPClient is a mock implementation of the http.Client
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}
