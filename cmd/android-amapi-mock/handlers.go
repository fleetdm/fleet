package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/google/uuid"
)

// ---- Coordination API handlers ----

// registerRequest is the coordination API's registration body. It is deliberately narrower
// than fakeDevice: only these fields come from the agent, so pending commands and pending
// certificates can't be injected through it.
type registerRequest struct {
	EnterpriseSpecificID string `json:"enterprise_specific_id"`
	DeviceName           string `json:"device_name"`
	EnterpriseID         string `json:"enterprise_id"`
	// PolicyName and PolicyVersion are sent only by a device registering again after this
	// process lost its state; they carry the policy the device last observed.
	PolicyName    string `json:"policy_name"`
	PolicyVersion int64  `json:"policy_version"`
}

func handleRegister(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.EnterpriseSpecificID == "" || req.DeviceName == "" {
			http.Error(w, "enterprise_specific_id and device_name required", http.StatusBadRequest)
			return
		}
		d := fakeDevice{
			EnterpriseSpecificID: req.EnterpriseSpecificID,
			DeviceName:           req.DeviceName,
			EnterpriseID:         req.EnterpriseID,
			PolicyName:           req.PolicyName,
			PolicyVersion:        req.PolicyVersion,
		}
		// A device that registers again after this process lost its state (restart) reports the
		// policy it last observed, so it doesn't tell Fleet its applied policy regressed.
		//
		// A version outside the restorable range is a bug in the caller rather than recovered
		// state, and must not be reported to Fleet: Fleet verifies profiles whose
		// included_in_policy_version is <= the applied version, so an absurdly high version
		// would flip every pending profile to Verified. Drop the reported policy in that case;
		// store.register then keeps whatever policy it already has for the device, or starts a
		// new device on the default policy.
		if d.PolicyVersion < 0 || d.PolicyVersion > maxRestorablePolicyVersion {
			log.Printf("Ignoring out-of-range policy version %d reported by device %s", d.PolicyVersion, d.EnterpriseSpecificID) // #nosec G706 -- load testing tool
			d.PolicyName = ""
			d.PolicyVersion = 0
		} else if d.PolicyName != "" {
			store.raisePolicyVersionCounter(d.PolicyVersion)
		}

		// A device Fleet deleted through AMAPI was genuinely unenrolled; letting it register
		// again would resurrect it and re-enroll the host.
		if !store.register(&d) {
			http.Error(w, "device was deleted", http.StatusGone)
			return
		}
		log.Printf("Registered fake device: %s (name: %s)", d.EnterpriseSpecificID, d.DeviceName)
		w.WriteHeader(http.StatusOK)
	}
}

func handleGetState(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		esid := r.PathValue("esid")
		d := store.getByESID(esid)
		if d == nil {
			// Distinguish "this process forgot the device" (the agent should register again)
			// from "Fleet deleted the device" (the agent must stop).
			if store.wasDeleted(esid) {
				http.Error(w, "device was deleted", http.StatusGone)
				return
			}
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		d.mu.Lock()
		policyVersion := d.PolicyVersion
		if d.PolicyName != "" {
			// Report the higher of the version this process issued for the policy and the
			// version the device already observed (which is higher after a restart, since the
			// version counter starts over). Reporting a lower version would tell Fleet the
			// device's applied policy went backwards, and Fleet only verifies profiles whose
			// included_in_policy_version is <= the applied version — so profiles delivered
			// before the restart would be stuck Pending.
			if v := store.getPolicyVersion(d.PolicyName); v > policyVersion {
				policyVersion = v
			}
			d.PolicyVersion = policyVersion
		}
		state := struct {
			PolicyVersion       int64    `json:"policy_version"`
			PolicyName          string   `json:"policy_name"`
			PendingCommands     []string `json:"pending_commands"`
			PendingCertificates []uint   `json:"pending_certificates"`
		}{
			PolicyVersion:       policyVersion,
			PolicyName:          d.PolicyName,
			PendingCommands:     d.PendingCommands,
			PendingCertificates: d.PendingCertificates,
		}
		d.PendingCommands = nil
		d.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	}
}

// ---- Device handlers ----

func handleDevicesGet(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := deviceName(r)
		d := store.getByName(name)
		if d == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":{"code":404,"message":"Device not found","status":"NOT_FOUND"}}`)
			return
		}

		d.mu.Lock()
		resp := map[string]any{
			"name":                 name,
			"appliedPolicyVersion": fmt.Sprintf("%d", d.PolicyVersion),
			"appliedPolicyName":    d.PolicyName,
		}
		d.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleDevicesPatch(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := deviceName(r)
		d := store.getByName(name)

		var reqBody struct {
			PolicyName string `json:"policyName"`
		}
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body: "+err.Error(), http.StatusBadRequest)
				return
			}
			if len(body) > 0 {
				if err := json.Unmarshal(body, &reqBody); err != nil {
					http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
					return
				}
			}
		}

		var appliedVersion int64
		if d != nil {
			d.mu.Lock()
			if reqBody.PolicyName != "" && reqBody.PolicyName != d.PolicyName {
				// The version the device holds belongs to the policy it is moving off of.
				d.PolicyName = reqBody.PolicyName
				d.PolicyVersion = 0
			}
			if d.PolicyName != "" {
				// As in handleGetState, never lower the version the device already observed.
				appliedVersion = max(d.PolicyVersion, store.getPolicyVersion(d.PolicyName))
				d.PolicyVersion = appliedVersion
			}
			d.mu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":                 name,
			"appliedPolicyVersion": fmt.Sprintf("%d", appliedVersion),
		})
	}
}

func handleDevicesDelete(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := deviceName(r)

		if d, ok := store.markDeleted(name); ok {
			log.Printf("Deleted fake device: %q (ESID: %q)", name, d.EnterpriseSpecificID) // #nosec G706 -- load testing tool
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}
}

func handleIssueCommand(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := deviceName(r)
		operationID := uuid.New().String()
		operationName := fmt.Sprintf("%s/operations/%s", name, operationID)

		d := store.getByName(name)
		if d != nil {
			d.mu.Lock()
			d.PendingCommands = append(d.PendingCommands, operationName)
			d.mu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": operationName,
			"done": false,
		})
	}
}

func handleDevicesList(store *deviceStore, google *googleForwarder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fakeNames := store.allDeviceNames()
		sort.Strings(fakeNames)

		var realDevices []map[string]string
		if google != nil {
			enterpriseName := "enterprises/" + r.PathValue("eid")
			var err error
			realDevices, err = google.ForwardDevicesList(enterpriseName, r.Context())
			if err != nil {
				log.Printf("Failed to list real devices from Google: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    502,
						"message": "failed to list real devices: " + err.Error(),
						"status":  "BAD_GATEWAY",
					},
				})
				return
			}
			if len(realDevices) > 0 {
				hasSeenRealDevice.Store(true)
			}
		}

		allDevices := make([]map[string]string, 0, len(realDevices)+len(fakeNames))
		allDevices = append(allDevices, realDevices...)
		for _, name := range fakeNames {
			allDevices = append(allDevices, map[string]string{"name": name})
		}

		pageSize := 100
		offset := 0
		if pt := r.URL.Query().Get("pageToken"); pt != "" {
			if v, err := strconv.Atoi(pt); err == nil {
				offset = v
			}
		}
		if offset < 0 {
			offset = 0
		}
		if offset > len(allDevices) {
			offset = len(allDevices)
		}

		end := min(offset+pageSize, len(allDevices))

		resp := map[string]any{
			"devices": allDevices[offset:end],
		}
		if end < len(allDevices) {
			resp["nextPageToken"] = fmt.Sprintf("%d", end)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// ---- Policy handlers ----

func handlePoliciesPatch(store *deviceStore, google *googleForwarder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := policyName(r)
		hostUUID := policyID(r)

		// Check if this policy is for a fake device (hostUUID == enterpriseSpecificID).
		// If it's not a fake device and we have Google credentials, forward to real AMAPI.
		isFakeDevice := store.getByESID(hostUUID) != nil
		if !isFakeDevice && google != nil {
			log.Printf("Forwarding policy patch to Google AMAPI: %q", name) // #nosec G706 -- load testing tool
			google.ForwardPoliciesPatch(w, r)
			return
		}

		version := store.nextPolicyVersion(name)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    name,
			"version": fmt.Sprintf("%d", version),
		})
	}
}

// handlePolicyAction handles POST on policies: modifyPolicyApplications and removePolicyApplications.
func handlePolicyAction(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := policyName(r)
		// policyID strips the action suffix, without which the device lookup in
		// extractAndStoreCertTemplateIDs never matches.
		hostUUID := policyID(r)

		// Try to extract cert template IDs from the request body
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}
		if len(bodyBytes) > 0 {
			extractAndStoreCertTemplateIDs(store, hostUUID, bodyBytes)
		}

		version := store.nextPolicyVersion(name)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": fmt.Sprintf("%d", version),
		})
	}
}

// ---- Other AMAPI handlers ----

func handleEnrollmentTokenCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		token := uuid.New().String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":   "enterprises/mock/enrollmentTokens/" + token,
			"value":  token,
			"qrCode": fmt.Sprintf(`{"android.app.extra.PROVISIONING_DEVICE_ADMIN_COMPONENT_NAME":"com.google.android.apps.work.clouddpc/.receivers.CloudDeviceAdminReceiver","android.app.extra.PROVISIONING_DEVICE_ADMIN_SIGNATURE_CHECKSUM":"I5YvS0O5hXY46mb01BlRjq4oJJGs2kuUcHvVkAPEXlg","android.app.extra.PROVISIONING_DEVICE_ADMIN_PACKAGE_DOWNLOAD_LOCATION":"https://play.google.com/managed/downloadManagingApp?identifier=setup","android.app.extra.PROVISIONING_ADMIN_EXTRAS_BUNDLE":{"com.google.android.apps.work.clouddpc.EXTRA_ENROLLMENT_TOKEN":"%s"}}`, token),
		})
	}
}

func handleApplicationsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":  "mock-app",
			"title": "Mock Application",
		})
	}
}

func handleWebAppsCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":  "enterprises/mock/webApps/" + uuid.New().String(),
			"title": "Mock Web App",
		})
	}
}

func handleEnterprisesList(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		store.mu.RLock()
		seen := make(map[string]bool)
		for _, d := range store.byESID {
			if d.EnterpriseID != "" {
				seen[d.EnterpriseID] = true
			}
		}
		store.mu.RUnlock()

		enterprises := make([]map[string]string, 0, len(seen))
		for id := range seen {
			enterprises = append(enterprises, map[string]string{"name": "enterprises/" + id})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"enterprises": enterprises,
		})
	}
}

func extractAndStoreCertTemplateIDs(store *deviceStore, hostUUID string, body []byte) {
	var req struct {
		Changes []struct {
			Application struct {
				ManagedConfiguration json.RawMessage `json:"managedConfiguration"`
			} `json:"application"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}

	// Keep looking after a change that carries no install operation: returning early here
	// dropped certificates that were listed in a later change.
	var certIDs []uint
	for _, change := range req.Changes {
		if change.Application.ManagedConfiguration == nil {
			continue
		}
		var config android.AgentManagedConfiguration
		if err := json.Unmarshal(change.Application.ManagedConfiguration, &config); err != nil {
			continue
		}

		for _, ct := range config.CertificateTemplateIDs {
			if ct.Operation == string(fleet.MDMOperationTypeInstall) {
				certIDs = append(certIDs, ct.ID)
			}
		}
	}
	if len(certIDs) == 0 {
		return
	}

	// The hostUUID from the policy path is the enterpriseSpecificID for android devices.
	// A real device's policy action reaches this handler too (it is never forwarded), so
	// this is expected in a mixed real + fake run.
	d := store.getByESID(hostUUID)
	if d == nil {
		log.Printf("Policy %q is not a registered fake device; dropping %d pending certificate(s)", hostUUID, len(certIDs)) // #nosec G706 -- load testing tool
		return
	}
	d.mu.Lock()
	d.PendingCertificates = certIDs
	d.mu.Unlock()
}

func handleCatchAll(_ *googleForwarder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ERROR: unhandled AMAPI endpoint: %q %q — add a handler or forwarding for this route", r.Method, r.URL.Path) // #nosec G706 -- load testing tool
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    501,
				"message": "mock does not handle " + r.Method + " " + r.URL.Path,
				"status":  "NOT_IMPLEMENTED",
			},
		})
	}
}
