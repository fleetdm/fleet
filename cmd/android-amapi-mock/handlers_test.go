package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

const (
	testESID         = "35B8F9A0-4C2E-4B1D-9F3A-7E6D5C4B3A21"
	testEnterpriseID = "LC01"
	testDeviceName   = "enterprises/LC01/devices/fakedevice"
)

func testPolicyName(esid string) string {
	return fmt.Sprintf("enterprises/%s/policies/%s", testEnterpriseID, esid)
}

// newTestMux builds the real route table with latency and simulated errors disabled.
func newTestMux(t *testing.T) (*http.ServeMux, *deviceStore) {
	t.Helper()
	store := newDeviceStore()
	return newMux(store, nil, 0, 0), store
}

// registerTestDevice registers a fake device through the coordination API the way
// osquery-perf's Android agents do, and asserts the expected status.
func registerTestDevice(t *testing.T, mux *http.ServeMux, req registerRequest, wantStatus int) {
	t.Helper()
	payload, err := json.Marshal(req)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/mock/devices/register", bytes.NewReader(payload)))
	require.Equal(t, wantStatus, rr.Code, "register response: %s", rr.Body.String())
}

func defaultRegisterRequest() registerRequest {
	return registerRequest{
		EnterpriseSpecificID: testESID,
		DeviceName:           testDeviceName,
		EnterpriseID:         testEnterpriseID,
	}
}

// certChangesBody builds a modifyPolicyApplications body from the same types Fleet uses to
// send one, so a change to either wire format fails to compile here.
func certChangesBody(t *testing.T, configs ...android.AgentManagedConfiguration) []byte {
	t.Helper()
	req := androidmanagement.ModifyPolicyApplicationsRequest{}
	for _, cfg := range configs {
		managedConfig, err := json.Marshal(cfg)
		require.NoError(t, err)
		req.Changes = append(req.Changes, &androidmanagement.ApplicationPolicyChange{
			Application: &androidmanagement.ApplicationPolicy{
				PackageName:          "com.fleetdm.agent",
				ManagedConfiguration: managedConfig,
			},
		})
	}
	body, err := json.Marshal(req)
	require.NoError(t, err)
	return body
}

func certConfig(templates ...android.AgentCertificateTemplate) android.AgentManagedConfiguration {
	return android.AgentManagedConfiguration{
		ServerURL:              "https://fleet.example.com",
		HostUUID:               testESID,
		EnrollSecret:           "secret",
		CertificateTemplateIDs: templates,
	}
}

func certTemplate(id uint, operation fleet.MDMOperationType) android.AgentCertificateTemplate {
	return android.AgentCertificateTemplate{ID: id, Status: "pending", Operation: string(operation)}
}

func postPolicyAction(t *testing.T, mux *http.ServeMux, policyPath string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", policyPath, bytes.NewReader(body)))
	return rr
}

// TestHandlePolicyActionStoresPendingCertificates covers the load-testing bug where the
// policy action suffix was left on the path value used to look up the device, so pending
// certificates were dropped and cert-backed profiles stayed Pending forever.
func TestHandlePolicyActionStoresPendingCertificates(t *testing.T) {
	// Any AMAPI custom method must resolve to the same policy, including one the mock has
	// never heard of.
	for _, action := range []string{"modifyPolicyApplications", "removePolicyApplications", "someFutureAction"} {
		t.Run(action, func(t *testing.T) {
			mux, store := newTestMux(t)
			registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

			body := certChangesBody(t, certConfig(certTemplate(7, fleet.MDMOperationTypeInstall)))
			rr := postPolicyAction(t, mux, fmt.Sprintf("/v1/enterprises/%s/policies/%s:%s", testEnterpriseID, testESID, action), body)
			require.Equal(t, http.StatusOK, rr.Code, "policy action response: %s", rr.Body.String())

			d := store.getByESID(testESID)
			require.NotNil(t, d, "device must be findable by its bare enterpriseSpecificID")
			assert.Equal(t, []uint{7}, d.PendingCertificates)

			// The policy version must be keyed to the suffix-stripped policy name, otherwise
			// the version the device reports never matches what Fleet recorded.
			assert.Positive(t, store.getPolicyVersion(testPolicyName(testESID)))
			assert.Zero(t, store.getPolicyVersion(testPolicyName(testESID+":"+action)))
		})
	}
}

func TestExtractAndStoreCertTemplateIDs(t *testing.T) {
	testCases := []struct {
		name     string
		configs  []android.AgentManagedConfiguration
		expected []uint
	}{
		{
			name: "install ops in a later change are not dropped",
			configs: []android.AgentManagedConfiguration{
				certConfig(),
				certConfig(certTemplate(11, fleet.MDMOperationTypeInstall)),
			},
			expected: []uint{11},
		},
		{
			name: "a change with no install op does not stop later changes",
			configs: []android.AgentManagedConfiguration{
				certConfig(certTemplate(3, fleet.MDMOperationTypeRemove)),
				certConfig(certTemplate(4, fleet.MDMOperationTypeInstall)),
			},
			expected: []uint{4},
		},
		{
			name: "several install ops in one change are all collected",
			configs: []android.AgentManagedConfiguration{
				certConfig(
					certTemplate(1, fleet.MDMOperationTypeInstall),
					certTemplate(2, fleet.MDMOperationTypeInstall),
				),
			},
			expected: []uint{1, 2},
		},
		{
			name: "non-install ops are ignored",
			configs: []android.AgentManagedConfiguration{
				certConfig(certTemplate(6, fleet.MDMOperationTypeRemove)),
			},
			expected: nil,
		},
		{
			name:     "no managed configuration at all",
			configs:  nil,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mux, store := newTestMux(t)
			registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

			rr := postPolicyAction(t, mux,
				fmt.Sprintf("/v1/enterprises/%s/policies/%s:modifyPolicyApplications", testEnterpriseID, testESID),
				certChangesBody(t, tc.configs...))
			require.Equal(t, http.StatusOK, rr.Code)

			d := store.getByESID(testESID)
			require.NotNil(t, d)
			assert.Equal(t, tc.expected, d.PendingCertificates)
		})
	}
}

// TestGetStateReportsPendingCertificates checks the full path a fake agent takes: a policy
// action delivers cert templates, and the next state poll hands them to the device.
func TestGetStateReportsPendingCertificates(t *testing.T) {
	mux, _ := newTestMux(t)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

	body := certChangesBody(t, certConfig(certTemplate(42, fleet.MDMOperationTypeInstall)))
	rr := postPolicyAction(t, mux,
		fmt.Sprintf("/v1/enterprises/%s/policies/%s:modifyPolicyApplications", testEnterpriseID, testESID), body)
	require.Equal(t, http.StatusOK, rr.Code)

	state := getTestState(t, mux, testESID, http.StatusOK)
	assert.Equal(t, []uint{42}, state.PendingCertificates)
}

type testDeviceState struct {
	PolicyVersion       int64    `json:"policy_version"`
	PolicyName          string   `json:"policy_name"`
	PendingCommands     []string `json:"pending_commands"`
	PendingCertificates []uint   `json:"pending_certificates"`
}

func getTestState(t *testing.T, mux *http.ServeMux, esid string, wantStatus int) testDeviceState {
	t.Helper()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET", "/mock/devices/"+esid+"/state", nil))
	require.Equal(t, wantStatus, rr.Code, "state response: %s", rr.Body.String())

	var state testDeviceState
	if wantStatus == http.StatusOK {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &state))
	}
	return state
}

// TestPoliciesPatchIdentifiesFakeDeviceByPolicyID guards the fake-vs-real routing decision,
// which also reads the policy ID out of the path.
func TestPoliciesPatchIdentifiesFakeDeviceByPolicyID(t *testing.T) {
	mux, store := newTestMux(t)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

	rr := patchPolicy(t, mux, testESID)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, testPolicyName(testESID), resp.Name)
	assert.Positive(t, store.getPolicyVersion(testPolicyName(testESID)))
}

func patchPolicy(t *testing.T, mux *http.ServeMux, policyID string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("PATCH",
		fmt.Sprintf("/v1/enterprises/%s/policies/%s", testEnterpriseID, policyID),
		bytes.NewReader([]byte(`{}`))))
	return rr
}

// TestRegisterRestoresPolicyState covers recovery after this process is restarted: the agent
// registers again reporting the policy state it last observed, and the mock must keep it
// instead of resetting the device to the default policy at version 0. Reporting a regressed
// policy would make Fleet see already-applied profiles as un-applied.
func TestRegisterRestoresPolicyState(t *testing.T) {
	t.Run("reported policy state is kept", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 57
		registerTestDevice(t, mux, req, http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, testPolicyName(testESID), d.PolicyName)
		assert.Equal(t, int64(57), d.PolicyVersion)

		// The device reports the version it last observed, so profiles Fleet already
		// delivered are not seen as un-applied.
		state := getTestState(t, mux, testESID, http.StatusOK)
		assert.Equal(t, int64(57), state.PolicyVersion)
		assert.Equal(t, testPolicyName(testESID), state.PolicyName)
	})

	// A restored version must NOT be recorded as the policy's current version: Fleet verifies
	// profiles whose included_in_policy_version is <= the applied version, so a device
	// claiming a version this process never issued would flip pending profiles to Verified
	// without the policy ever being applied — hiding stuck-in-Pending bugs.
	t.Run("a restored version is not recorded as the policy version", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 1000
		registerTestDevice(t, mux, req, http.StatusOK)

		assert.Zero(t, store.getPolicyVersion(testPolicyName(testESID)),
			"no version was issued for this policy by this process")
	})

	// Versions issued after a restore must still be above what the device already reported,
	// so the version the device reports never goes backwards.
	t.Run("later versions stay above the restored version", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 57
		registerTestDevice(t, mux, req, http.StatusOK)

		require.Equal(t, http.StatusOK, patchPolicy(t, mux, testESID).Code)
		assert.Greater(t, store.getPolicyVersion(testPolicyName(testESID)), int64(57))
	})

	t.Run("a first registration gets the default policy", func(t *testing.T) {
		mux, store := newTestMux(t)
		registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, "enterprises/LC01/policies/default", d.PolicyName)
		assert.Zero(t, d.PolicyVersion)
	})

	// A bogus version must not push the counter near overflow: the next issued version would
	// wrap negative and every version after it would be negative and decreasing.
	t.Run("an absurd version cannot overflow the counter", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 1<<62 + 1
		registerTestDevice(t, mux, req, http.StatusOK)

		require.Equal(t, http.StatusOK, patchPolicy(t, mux, testESID).Code)
		assert.Positive(t, store.getPolicyVersion(testPolicyName(testESID)),
			"issued versions must stay positive")
		assert.Less(t, store.getPolicyVersion(testPolicyName(testESID)), int64(maxRestorablePolicyVersion)+1)
	})

	// Re-registration must not discard state other handlers already attached to the device.
	t.Run("re-registration keeps pending state", func(t *testing.T) {
		mux, store := newTestMux(t)
		registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

		body := certChangesBody(t, certConfig(certTemplate(9, fleet.MDMOperationTypeInstall)))
		require.Equal(t, http.StatusOK, postPolicyAction(t, mux,
			fmt.Sprintf("/v1/enterprises/%s/policies/%s:modifyPolicyApplications", testEnterpriseID, testESID), body).Code)

		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 5
		registerTestDevice(t, mux, req, http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, []uint{9}, d.PendingCertificates, "pending certificates must survive")
		assert.Equal(t, testPolicyName(testESID), d.PolicyName)
		assert.Len(t, store.allDeviceNames(), 1, "the device must not be duplicated")
	})
}

// TestRegisterRejectsInjectedPendingState pins that the registration body can only carry the
// fields the agent sends: pending commands would otherwise be acked to Fleet as if issued.
func TestRegisterRejectsInjectedPendingState(t *testing.T) {
	mux, store := newTestMux(t)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/mock/devices/register", bytes.NewReader([]byte(`{
		"enterprise_specific_id": "`+testESID+`",
		"device_name": "`+testDeviceName+`",
		"enterprise_id": "`+testEnterpriseID+`",
		"pending_commands": ["enterprises/LC01/devices/fakedevice/operations/injected"],
		"pending_certificates": [99]
	}`))))
	require.Equal(t, http.StatusOK, rr.Code)

	d := store.getByESID(testESID)
	require.NotNil(t, d)
	assert.Empty(t, d.PendingCommands)
	assert.Empty(t, d.PendingCertificates)
}

// TestDeletedDeviceStaysDeleted covers the resurrection bug: Fleet unenrolls a non-BYO
// Android host by deleting the device through AMAPI, so a device that registers again after
// being deleted would re-appear in the device list the reconciler reads and re-enroll the
// host, flip-flopping between mdm_unenrolled and mdm_enrolled.
func TestDeletedDeviceStaysDeleted(t *testing.T) {
	mux, store := newTestMux(t)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("DELETE", "/v1/"+testDeviceName, nil))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Empty(t, store.allDeviceNames())

	// The agent is told the device is gone for good, not merely forgotten.
	getTestState(t, mux, testESID, http.StatusGone)

	// Registering again must not bring it back.
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusGone)
	assert.Nil(t, store.getByESID(testESID))
	assert.Empty(t, store.allDeviceNames(), "a deleted device must stay out of the AMAPI device list")
}

// TestGetStateUnknownDeviceReturns404 pins the status code the agent relies on to tell that
// this process lost its registration and that it should register again.
func TestGetStateUnknownDeviceReturns404(t *testing.T) {
	mux, _ := newTestMux(t)
	getTestState(t, mux, testESID, http.StatusNotFound)
}

// TestDevicesListIncludesFakeDevices pins what the reconciler reads: a registered device must
// be listed, since Fleet marks any enrolled device missing from this list as unenrolled.
func TestDevicesListIncludesFakeDevices(t *testing.T) {
	mux, _ := newTestMux(t)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("GET",
		"/v1/enterprises/"+testEnterpriseID+"/devices?pageSize=100&fields=nextPageToken,devices/name", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp androidmanagement.ListDevicesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Devices, 1)
	assert.Equal(t, testDeviceName, resp.Devices[0].Name)
	assert.Empty(t, resp.NextPageToken)
}
