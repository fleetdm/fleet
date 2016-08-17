package app

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kolide/kolide-ose/errors"
	"github.com/stretchr/testify/assert"
)

func TestOsqueryIntegrationEnrollHostBadSecret(t *testing.T) {
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

func TestOsqueryIntegrationEnrollHostMissingIdentifier(t *testing.T) {
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

	assert.Equal(t, "Validation error", body["error"])
}

func TestOsqueryIntegrationEnrollHostGood(t *testing.T) {
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

func TestOsqueryIntegrationOsqueryLogErrors(t *testing.T) {
	var req IntegrationRequests
	req.New(t)

	// Missing log type should give error
	data := json.RawMessage("{}")
	resp := req.OsqueryLog("node_key", "", &data)

	assert.Equal(t, errors.StatusUnprocessableEntity, resp.Code)

	body := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Contains(t, body, "error")

	// Bad node key should give error
	resp = req.OsqueryLog("bad_node_key", "status", &data)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	body = map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Contains(t, body, "error")
	assert.Contains(t, body, "node_invalid")
	assert.Equal(t, "true", body["node_invalid"])
}

func TestOsqueryIntegrationOsqueryLogSuccess(t *testing.T) {
	var req IntegrationRequests
	req.New(t)

	// First enroll
	resp := req.EnrollHost("super secret", "fake_host_1")

	body := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, body, "node_key")
	nodeKey := body["node_key"].(string)

	// Now logging should be successful

	expectStatus := []OsqueryStatusLog{
		OsqueryStatusLog{
			Severity: "bad",
			Filename: "nope.cpp",
			Line:     "42",
			Message:  "bad stuff happened",
			Version:  "1.8.0",
			Decorations: map[string]string{
				"foo": "bar",
			},
		},
		OsqueryStatusLog{
			Severity: "worse",
			Filename: "uhoh.cpp",
			Line:     "42",
			Message:  "bad stuff happened",
			Version:  "1.8.0",
			Decorations: map[string]string{
				"foo": "bar",
				"baz": "bang",
			},
		},
	}

	expectResult := []OsqueryResultLog{
		OsqueryResultLog{
			Name:           "query",
			HostIdentifier: "somehost",
			UnixTime:       "the time",
			CalendarTime:   "other time",
			Columns: map[string]string{
				"foo": "bar",
				"baz": "bang",
			},
			Action: "none",
		},
	}

	// Send status log
	jsonVal, err := json.Marshal(&expectStatus)
	assert.NoError(t, err)
	data := json.RawMessage(jsonVal)

	resp = req.OsqueryLog(nodeKey, "status", &data)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Send result log
	jsonVal, err = json.Marshal(&expectResult)
	assert.NoError(t, err)
	data = json.RawMessage(jsonVal)

	resp = req.OsqueryLog(nodeKey, "result", &data)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Check that correct logs were logged
	assert.Equal(t, expectStatus, req.statusHandler.Logs)
	assert.Equal(t, expectResult, req.resultHandler.Logs)

}
