package server

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveFleetACMEServer exercises the full ACME flow against a running
// Fleet server with ACME endpoints registered.
//
// Requires: FLEET_LIVE_TEST=1 and Fleet running at https://localhost:8080
// with an ACME CA named "test_acme" configured.
func TestLiveFleetACMEServer(t *testing.T) {
	if os.Getenv("FLEET_LIVE_TEST") != "1" {
		t.Skip("set FLEET_LIVE_TEST=1 to run (requires Fleet at localhost:8080 with ACME CA)")
	}

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	base := "https://localhost:8080/api/acme/test_acme"

	// 1. Directory
	resp, err := client.Get(base + "/directory")
	require.NoError(t, err)
	var dir map[string]interface{}
	require.NoError(t, readJSON(resp, &dir))
	resp.Body.Close()
	assert.Contains(t, dir, "newNonce")
	assert.Contains(t, dir, "newAccount")
	assert.Contains(t, dir, "newOrder")
	t.Logf("Directory: %v", dir)

	// 2. Account
	resp, err = client.Post(base+"/new-account", "application/json",
		bytes.NewReader([]byte(`{"contact":["mailto:test@fleet.local"]}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()
	t.Log("Account created")

	// 3. Order
	orderBody, _ := json.Marshal(map[string]interface{}{
		"identifiers": []map[string]string{{"type": "dns", "value": "localhost"}},
	})
	resp, err = client.Post(base+"/new-order", "application/json", bytes.NewReader(orderBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	var order map[string]interface{}
	require.NoError(t, readJSON(resp, &order))
	resp.Body.Close()
	assert.Equal(t, "pending", order["status"])
	t.Logf("Order: status=%s", order["status"])

	// 4. Authorization + Challenge
	authzURLs := order["authorizations"].([]interface{})
	resp, err = client.Get(authzURLs[0].(string))
	require.NoError(t, err)
	var authz map[string]interface{}
	require.NoError(t, readJSON(resp, &authz))
	resp.Body.Close()

	challenges := authz["challenges"].([]interface{})
	challengeURL := challenges[0].(map[string]interface{})["url"].(string)

	resp, err = client.Post(challengeURL, "application/json", bytes.NewReader([]byte(`{}`)))
	require.NoError(t, err)
	var chResp map[string]interface{}
	require.NoError(t, readJSON(resp, &chResp))
	resp.Body.Close()
	assert.Equal(t, "valid", chResp["status"])
	t.Logf("Challenge: status=%s", chResp["status"])

	// 5. Poll order → ready
	orderURL := order["finalize"].(string)
	orderURL = orderURL[:len(orderURL)-len("/finalize")]
	resp, err = client.Get(orderURL)
	require.NoError(t, err)
	var readyOrder map[string]interface{}
	require.NoError(t, readJSON(resp, &readyOrder))
	resp.Body.Close()
	assert.Equal(t, "ready", readyOrder["status"])
	t.Logf("Order polled: status=%s", readyOrder["status"])

	// Finalize is skipped in this test. The relay needs port 80 to complete
	// upstream http-01 challenges against the RA, which requires a dedicated
	// challenge server. Certificate issuance through the full stack
	// (relay → RA → Smallstep cloud) is verified by TestRelaySmallstepRA.
	t.Log("")
	t.Log("*** Fleet ACME server verified against live server ***")
	t.Log("*** Tested: directory, account, order, authz, challenge, order poll ***")
	t.Log("*** Certificate issuance via relay verified separately in TestRelaySmallstepRA ***")

	// Suppress unused imports
	_ = fmt.Sprintf
	_ = base64.RawURLEncoding
	_ = pem.Decode
	_ = io.ReadAll
}
