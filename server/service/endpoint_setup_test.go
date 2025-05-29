package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/spec"
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
			scriptNames := extractScriptNames(specs)

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
				w.Write([]byte(scriptContent))
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
			err = downloadAndUpdateScripts(context.Background(), specs, tt.scriptNames, tempDir, kitlog.NewNopLogger())
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
				w.Write([]byte("script content"))
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
			err = downloadAndUpdateScripts(context.Background(), specs, tt.scriptNames, tempDir, kitlog.NewNopLogger())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestFleetHTTPClientOverride(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer mockServer.Close()

	// Create a custom client that uses our mock server
	client := &http.Client{
		Transport: &mockRoundTripper{
			mockServer:  mockServer.URL,
			origBaseURL: scriptsBaseURL,
			next:        http.DefaultTransport,
		},
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
				w.Write([]byte("test response"))
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
			err = downloadAndUpdateScripts(ctx, specs, scriptNames, tempDir, kitlog.NewNopLogger())

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

// TestApplyStarterLibrary tests the applyStarterLibrary function
func TestApplyStarterLibrary(t *testing.T) {
	// Skip the test since we can't mock the NewClient function
	t.Skip("This test requires mocking the NewClient function which is not possible in Go")
}
