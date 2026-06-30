package logging

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplunkIntegration tests the Splunk HEC writer against a real Splunk instance.
//
// Prerequisites:
//
//	docker run -d --name splunk-test --platform linux/amd64 \
//	  -p 8000:8000 -p 8088:8088 -p 8089:8089 \
//	  -e SPLUNK_GENERAL_TERMS=--accept-sgt-current-at-splunk-com \
//	  -e SPLUNK_START_ARGS=--accept-license \
//	  -e SPLUNK_PASSWORD=changeme123 \
//	  -e SPLUNK_HEC_TOKEN=test-hec-token-1234 \
//	  splunk/splunk:latest
//
// Run with: SPLUNK_INTEGRATION_TEST=1 go test ./server/logging/ -run TestSplunkIntegration -v
func TestSplunkIntegration(t *testing.T) {
	if os.Getenv("SPLUNK_INTEGRATION_TEST") == "" {
		t.Skip("set SPLUNK_INTEGRATION_TEST=1 to run this test (requires a running Splunk instance)")
	}

	splunkURL := "https://localhost:8088"
	splunkToken := "test-hec-token-1234"

	ctx := t.Context()

	// 1. Create writer with insecureSkipVerify for the self-signed cert
	writer, err := NewSplunkLogWriter(splunkURL, splunkToken, "main", "fleet-integration-test", "fleet:json", true, slog.Default())
	require.NoError(t, err, "NewSplunkLogWriter should connect to the running Splunk instance")

	// 2. Send test events
	marker := fmt.Sprintf("integration-test-%d", time.Now().UnixNano())
	testLogs := []json.RawMessage{
		json.RawMessage(fmt.Sprintf(`{"marker":"%s","seq":1,"host":"test-host-1","status":"ok"}`, marker)),
		json.RawMessage(fmt.Sprintf(`{"marker":"%s","seq":2,"host":"test-host-2","status":"warning"}`, marker)),
		json.RawMessage(fmt.Sprintf(`{"marker":"%s","seq":3,"host":"test-host-3","status":"error"}`, marker)),
	}

	err = writer.Write(ctx, testLogs)
	require.NoError(t, err, "Write should send events to Splunk HEC without error")

	// 3. Query Splunk REST API to verify events landed.
	// Give Splunk a moment to index the events.
	time.Sleep(5 * time.Second)

	events := searchSplunk(t, marker)
	require.Len(t, events, 3, "should find all 3 test events in Splunk")

	// Verify event content (Splunk returns newest first)
	for _, evt := range events {
		assert.Contains(t, evt, marker, "event should contain our unique marker")
	}

	t.Logf("Successfully sent and retrieved %d events from Splunk (marker: %s)", len(events), marker)
}

// TestSplunkIntegrationBatch tests batch splitting against a real Splunk instance.
func TestSplunkIntegrationBatch(t *testing.T) {
	if os.Getenv("SPLUNK_INTEGRATION_TEST") == "" {
		t.Skip("set SPLUNK_INTEGRATION_TEST=1 to run this test (requires a running Splunk instance)")
	}

	splunkURL := "https://localhost:8088"
	splunkToken := "test-hec-token-1234"

	ctx := t.Context()

	writer, err := NewSplunkLogWriter(splunkURL, splunkToken, "main", "fleet-batch-test", "fleet:json", true, slog.Default())
	require.NoError(t, err)

	// Send 100 events to verify batching works
	marker := fmt.Sprintf("batch-test-%d", time.Now().UnixNano())
	testLogs := make([]json.RawMessage, 100)
	for i := range testLogs {
		testLogs[i] = json.RawMessage(fmt.Sprintf(`{"marker":"%s","seq":%d,"data":"%s"}`, marker, i, "payload-data-for-batch-test"))
	}

	err = writer.Write(ctx, testLogs)
	require.NoError(t, err, "Write should handle batch of 100 events")

	time.Sleep(5 * time.Second)

	events := searchSplunk(t, marker)
	require.Len(t, events, 100, "all 100 events should be indexed in Splunk")

	t.Logf("Successfully sent and retrieved %d batched events from Splunk", len(events))
}

// TestSplunkIntegrationBadToken tests that sending with a bad token is rejected by HEC.
// Note: the HEC /health endpoint returns 200 regardless of token validity (it reports
// overall HEC health), so token validation only happens on the event endpoint.
func TestSplunkIntegrationBadToken(t *testing.T) {
	if os.Getenv("SPLUNK_INTEGRATION_TEST") == "" {
		t.Skip("set SPLUNK_INTEGRATION_TEST=1 to run this test (requires a running Splunk instance)")
	}

	splunkURL := "https://localhost:8088"
	ctx := t.Context()

	// Health check passes (it doesn't validate tokens), but Write should fail.
	writer, err := NewSplunkLogWriter(splunkURL, "bad-token-12345", "main", "fleet", "fleet:json", true, slog.Default())
	require.NoError(t, err, "health check passes regardless of token")

	err = writer.Write(ctx, []json.RawMessage{json.RawMessage(`{"test":"bad-token"}`)})
	require.Error(t, err, "Write should fail with an invalid token")
	require.Contains(t, err.Error(), "403")
}

// searchSplunk queries the Splunk REST API for events containing the given marker string.
func searchSplunk(t *testing.T, marker string) []string {
	t.Helper()

	searchQuery := fmt.Sprintf(`search index=main "%s" | fields _raw`, marker)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // test-only, local Docker Splunk
			},
		},
	}

	body := fmt.Sprintf("search=%s&output_mode=json&earliest_time=-5m", searchQuery)
	req, err := http.NewRequest(http.MethodPost, "https://localhost:8089/services/search/jobs/export", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.SetBasicAuth("admin", "changeme123")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Splunk search API returned: %s", string(respBody))

	// Parse the NDJSON response (one JSON object per line).
	var events []string
	dec := json.NewDecoder(bytes.NewReader(respBody))
	for dec.More() {
		var result map[string]interface{}
		if err := dec.Decode(&result); err != nil {
			break
		}
		if raw, ok := result["result"].(map[string]interface{}); ok {
			if rawStr, ok := raw["_raw"].(string); ok {
				events = append(events, rawStr)
			}
		}
	}

	return events
}
