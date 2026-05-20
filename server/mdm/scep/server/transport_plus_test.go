package scepserver_test

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestPKIOperationGET_LiteralPlus verifies that GET PKIOperation works
// with literal '+' in the base64 message query parameter (not percent-encoded).
// Regression test for https://github.com/fleetdm/fleet/issues/45291.
func TestPKIOperationGET_LiteralPlus(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()

	pkcsreq := loadTestFile(t, "../testdata/PKCSReq.der")
	message := base64.StdEncoding.EncodeToString(pkcsreq)

	if !strings.Contains(message, "+") {
		t.Fatal("test payload must contain '+' to be a valid regression test")
	}

	// Build request with literal '+' in the query string (NOT percent-encoded).
	// This is what jScep and other SCEP clients that don't percent-escape send.
	rawURL := fmt.Sprintf("%s/scep?operation=PKIOperation&message=%s", server.URL, message)
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Override RawQuery to ensure '+' stays literal and is not percent-encoded.
	req.URL.RawQuery = "operation=PKIOperation&message=" + message

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET PKIOperation with literal '+' returned %d (want 200): %s", resp.StatusCode, string(body))
	}
}

// TestPKIOperationGET_PercentEncodedPlus verifies that GET PKIOperation
// continues to work when '+' is percent-encoded as %%2B.
func TestPKIOperationGET_PercentEncodedPlus(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()

	pkcsreq := loadTestFile(t, "../testdata/PKCSReq.der")
	message := base64.StdEncoding.EncodeToString(pkcsreq)

	// Percent-encode '+' and '/' as a well-behaved client would.
	escaped := strings.ReplaceAll(message, "+", "%2B")
	escaped = strings.ReplaceAll(escaped, "/", "%2F")
	rawURL := fmt.Sprintf("%s/scep?operation=PKIOperation&message=%s", server.URL, escaped)

	resp, err := http.Get(rawURL) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET PKIOperation with percent-encoded '+' returned %d (want 200): %s", resp.StatusCode, string(body))
	}
}
