package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kolide/kolide-ose/errors"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationEnrollHostBadSecret(t *testing.T) {
	var req IntegrationRequests
	req.New(t)

	// Check that a bad enroll secret causes the appropriate error code and
	// error JSON to be returned

	resp := req.EnrollHost("bad secret", "fake_uuid")

	if resp.Code != http.StatusUnauthorized {
		t.Error("Should error with invalid enroll secret")
	}

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("JSON decode error: %s JSON contents:\n %s", err.Error(), resp.Body.Bytes())
	}

	if _, ok := body["node_key"]; ok {
		t.Errorf("Should not return node key when secret is invalid")
	}
}

func TestIntegrationEnrollHostMissingIdentifier(t *testing.T) {
	var req IntegrationRequests
	req.New(t)

	// Check that an empty host identifier causes the appropriate error code and
	// error JSON to be returned

	resp := req.EnrollHost("super secret", "")

	if resp.Code != errors.StatusUnprocessableEntity {
		t.Error("Should error with missing host identifier")
	}

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("JSON decode error: %s JSON contents:\n %s", err.Error(), resp.Body.Bytes())
	}

	assert.Equal(t, "Validation error", body["message"])
}

func TestIntegrationEnrollHostGood(t *testing.T) {
	var req IntegrationRequests
	req.New(t)

	// Make a good request and verify that a node key is returned. Also
	// check that the DB recorded the information.

	resp := req.EnrollHost("super secret", "fake_host_1")

	if resp.Code != http.StatusOK {
		t.Error("Status should be StatusOK")
	}

	t.Logf("Response body:\n%s", string(resp.Body.Bytes()))

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("JSON decode error: %s JSON contents:\n %s", err.Error(), resp.Body.Bytes())
	}

	if _, ok := body["error"]; ok {
		t.Errorf("Unexpected error message: %s", body["error"])
	}

	if invalid, ok := body["node_invalid"]; ok && invalid == true {
		t.Errorf("Expected node_invalid = false")
	}

	nodeKey, ok := body["node_key"]
	if !ok || nodeKey == "" {
		t.Errorf("Expected node_key")
	}

	var host Host
	err = req.db.Where("uuid = ?", "fake_host_1").First(&host).Error
	if err != nil {
		t.Fatalf("Host not saved to DB: %s", err.Error())
	}

	if host.NodeKey != nodeKey {
		t.Errorf("Saved node key different than response key, %s != %s",
			host.NodeKey, nodeKey)
	}

	// Enroll again and check that node key changes

	resp = req.EnrollHost("super secret", "fake_host_1")

	if resp.Code != http.StatusOK {
		t.Error("Status should be StatusOK")
	}

	t.Logf("Response body:\n%s", string(resp.Body.Bytes()))

	body = map[string]interface{}{}
	err = json.Unmarshal(resp.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("JSON decode error: %s JSON contents:\n %s", err.Error(), resp.Body.Bytes())
	}

	if _, ok := body["error"]; ok {
		t.Errorf("Unexpected error message: %s", body["error"])
	}

	if invalid, ok := body["node_invalid"]; ok && invalid == true {
		t.Errorf("Expected node_invalid = false")
	}

	newNodeKey, ok := body["node_key"]
	if !ok || nodeKey == "" {
		t.Errorf("Expected node_key")
	}

	if newNodeKey == nodeKey {
		t.Errorf("Node key should have changed, %s == %s", newNodeKey, nodeKey)
	}

}
