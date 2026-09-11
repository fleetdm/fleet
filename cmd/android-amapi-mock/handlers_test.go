package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

	// A policy patched before the device registers again leaves this process holding a low
	// version for that policy. Reporting it would tell Fleet the device's applied version
	// went backwards, and Fleet only verifies profiles whose included_in_policy_version is
	// <= the applied version — so every profile delivered before the restart would be stuck
	// Pending until some later patch happened to exceed the pre-restart version.
	t.Run("a version issued before the restore does not regress the reported version", func(t *testing.T) {
		mux, _ := newTestMux(t)
		require.Equal(t, http.StatusOK, patchPolicy(t, mux, testESID).Code)

		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 57
		registerTestDevice(t, mux, req, http.StatusOK)

		state := getTestState(t, mux, testESID, http.StatusOK)
		assert.GreaterOrEqual(t, state.PolicyVersion, int64(57),
			"the reported version must never go backwards")
	})

	t.Run("a first registration gets the default policy", func(t *testing.T) {
		mux, store := newTestMux(t)
		registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, "enterprises/LC01/policies/default", d.PolicyName)
		assert.Zero(t, d.PolicyVersion)
	})

	// An out-of-range version must not reach Fleet at all. Fleet verifies profiles whose
	// included_in_policy_version is <= the applied version, so an absurdly high version would
	// flip every pending profile to Verified; a negative one would verify nothing ever.
	for _, version := range []int64{1<<62 + 1, maxRestorablePolicyVersion + 1, -1} {
		t.Run(fmt.Sprintf("an out-of-range version %d is ignored", version), func(t *testing.T) {
			mux, store := newTestMux(t)
			req := defaultRegisterRequest()
			req.PolicyName = testPolicyName(testESID)
			req.PolicyVersion = version
			registerTestDevice(t, mux, req, http.StatusOK)

			d := store.getByESID(testESID)
			require.NotNil(t, d)
			assert.Zero(t, d.PolicyVersion, "the bogus version must not be kept on the device")
			assert.Equal(t, "enterprises/LC01/policies/default", d.PolicyName,
				"the device falls back to a fresh registration")

			// And it must not be reported to Fleet either.
			state := getTestState(t, mux, testESID, http.StatusOK)
			assert.Zero(t, state.PolicyVersion)

			// The version counter must still issue sane, positive versions.
			require.Equal(t, http.StatusOK, patchPolicy(t, mux, testESID).Code)
			issued := store.getPolicyVersion(testPolicyName(testESID))
			assert.Positive(t, issued, "issued versions must stay positive")
			assert.LessOrEqual(t, issued, int64(maxRestorablePolicyVersion))
		})
	}

	// A registration that reports no policy must not clobber the policy this process already
	// has for the device: reporting the default policy at version 0 is exactly the regression
	// that strands every already-delivered profile.
	t.Run("a registration without a policy keeps the known policy", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 57
		registerTestDevice(t, mux, req, http.StatusOK)

		// Same device registers again, reporting nothing (e.g. a freshly started agent).
		registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, testPolicyName(testESID), d.PolicyName)
		assert.Equal(t, int64(57), d.PolicyVersion)
		assert.Equal(t, int64(57), getTestState(t, mux, testESID, http.StatusOK).PolicyVersion)
	})

	// Same protection when the reported version is rejected as out of range.
	t.Run("an out-of-range version keeps the known policy", func(t *testing.T) {
		mux, store := newTestMux(t)
		req := defaultRegisterRequest()
		req.PolicyName = testPolicyName(testESID)
		req.PolicyVersion = 57
		registerTestDevice(t, mux, req, http.StatusOK)

		bogus := defaultRegisterRequest()
		bogus.PolicyName = testPolicyName(testESID)
		bogus.PolicyVersion = maxRestorablePolicyVersion + 1
		registerTestDevice(t, mux, bogus, http.StatusOK)

		d := store.getByESID(testESID)
		require.NotNil(t, d)
		assert.Equal(t, testPolicyName(testESID), d.PolicyName)
		assert.Equal(t, int64(57), d.PolicyVersion)
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

// TestDeleteOfUnknownDeviceStaysDeleted covers the delete that lands while this process has
// forgotten the device — i.e. exactly the restart window this change is about. The ESID isn't
// known then, so the deletion is recorded by resource name and the agent's next registration
// must still be refused. Otherwise the device resurrects, and if Fleet also deleted the host
// the next STATUS_REPORT re-creates it as a ghost.
func TestDeleteOfUnknownDeviceStaysDeleted(t *testing.T) {
	mux, store := newTestMux(t)

	// No device is registered: this process restarted and the agent has not come back yet.
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("DELETE", "/v1/"+testDeviceName, nil))
	require.Equal(t, http.StatusNotFound, rr.Code, "an unknown device is reported as absent")

	// The agent polls (404, since the ESID was never seen), then tries to register.
	getTestState(t, mux, testESID, http.StatusNotFound)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusGone)

	assert.Nil(t, store.getByESID(testESID))
	assert.Empty(t, store.allDeviceNames(), "a deleted device must stay out of the AMAPI device list")
}

// TestDevicesPatchDoesNotLowerReportedVersion covers the device patch after a restart: the
// policy has no version in this process yet, and zeroing the device's restored version would
// make it report applied version 0, verifying nothing.
func TestDevicesPatchDoesNotLowerReportedVersion(t *testing.T) {
	mux, store := newTestMux(t)
	req := defaultRegisterRequest()
	req.PolicyName = testPolicyName(testESID)
	req.PolicyVersion = 57
	registerTestDevice(t, mux, req, http.StatusOK)

	body := fmt.Sprintf(`{"policyName":%q,"state":"ACTIVE"}`, testPolicyName(testESID))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("PATCH", "/v1/"+testDeviceName, bytes.NewReader([]byte(body))))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		AppliedPolicyVersion string `json:"appliedPolicyVersion"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "57", resp.AppliedPolicyVersion)

	d := store.getByESID(testESID)
	require.NotNil(t, d)
	assert.Equal(t, int64(57), d.PolicyVersion)
	assert.Equal(t, int64(57), getTestState(t, mux, testESID, http.StatusOK).PolicyVersion)
}

// TestDevicesPatchToNewPolicyUsesNewPolicyVersion pins the other direction: moving a device to
// a different policy must not carry the old policy's version over, which would over-verify.
func TestDevicesPatchToNewPolicyUsesNewPolicyVersion(t *testing.T) {
	mux, store := newTestMux(t)
	req := defaultRegisterRequest()
	req.PolicyName = testPolicyName("old-policy")
	req.PolicyVersion = 57
	registerTestDevice(t, mux, req, http.StatusOK)

	// Fleet patches the new policy first, then points the device at it.
	require.Equal(t, http.StatusOK, patchPolicy(t, mux, testESID).Code)
	issued := store.getPolicyVersion(testPolicyName(testESID))
	require.Positive(t, issued)

	body := fmt.Sprintf(`{"policyName":%q,"state":"ACTIVE"}`, testPolicyName(testESID))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("PATCH", "/v1/"+testDeviceName, bytes.NewReader([]byte(body))))
	require.Equal(t, http.StatusOK, rr.Code)

	d := store.getByESID(testESID)
	require.NotNil(t, d)
	assert.Equal(t, testPolicyName(testESID), d.PolicyName)
	assert.Equal(t, issued, d.PolicyVersion, "the old policy's version must not carry over")
}

// TestStoreConcurrentAccess exercises the store from several goroutines so the race detector
// has something to look at: every other test drives it from a single goroutine.
func TestStoreConcurrentAccess(t *testing.T) {
	mux, store := newTestMux(t)
	registerTestDevice(t, mux, defaultRegisterRequest(), http.StatusOK)

	// Everything the goroutines need is built up front: assertions may only run on the test's
	// own goroutine, so the workers just drive requests.
	registerPayload, err := json.Marshal(defaultRegisterRequest())
	require.NoError(t, err)
	certBody := certChangesBody(t, certConfig(certTemplate(1, fleet.MDMOperationTypeInstall)))
	policyActionPath := fmt.Sprintf("/v1/enterprises/%s/policies/%s:modifyPolicyApplications", testEnterpriseID, testESID)
	policyPatchPath := fmt.Sprintf("/v1/enterprises/%s/policies/%s", testEnterpriseID, testESID)

	newRequest := []func() *http.Request{
		func() *http.Request {
			return httptest.NewRequest("POST", "/mock/devices/register", bytes.NewReader(registerPayload))
		},
		func() *http.Request {
			return httptest.NewRequest("GET", "/mock/devices/"+testESID+"/state", nil)
		},
		func() *http.Request {
			return httptest.NewRequest("PATCH", policyPatchPath, bytes.NewReader([]byte(`{}`)))
		},
		func() *http.Request {
			return httptest.NewRequest("POST", policyActionPath, bytes.NewReader(certBody))
		},
	}

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mux.ServeHTTP(httptest.NewRecorder(), newRequest[i%len(newRequest)]())
		}(i)
	}
	wg.Wait()

	// The store is still coherent: the device is present exactly once.
	assert.Len(t, store.allDeviceNames(), 1)
	assert.NotNil(t, store.getByESID(testESID))
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
