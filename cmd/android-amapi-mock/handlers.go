package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// ---- Coordination API handlers ----

func handleRegister(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var d fakeDevice
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if d.EnterpriseSpecificID == "" || d.DeviceName == "" {
			http.Error(w, "enterprise_specific_id and device_name required", http.StatusBadRequest)
			return
		}
		d.PolicyVersion = 0
		if d.EnterpriseID != "" {
			d.PolicyName = fmt.Sprintf("enterprises/%s/policies/default", d.EnterpriseID)
		}
		store.register(&d)
		log.Printf("Registered fake device: %s (name: %s)", d.EnterpriseSpecificID, d.DeviceName)
		w.WriteHeader(http.StatusOK)
	}
}

func handleGetState(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		esid := r.PathValue("esid")
		d := store.getByESID(esid)
		if d == nil {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}

		d.mu.Lock()
		policyVersion := d.PolicyVersion
		if d.PolicyName != "" {
			if v := store.getPolicyVersion(d.PolicyName); v > 0 {
				policyVersion = v
				d.PolicyVersion = v
			}
		}
		state := struct {
			PolicyVersion   int64    `json:"policy_version"`
			PolicyName      string   `json:"policy_name"`
			PendingCommands []string `json:"pending_commands"`
		}{
			PolicyVersion:   policyVersion,
			PolicyName:      d.PolicyName,
			PendingCommands: d.PendingCommands,
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
			"appliedPolicyVersion": d.PolicyVersion,
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
			if reqBody.PolicyName != "" {
				d.PolicyName = reqBody.PolicyName
			}
			if d.PolicyName != "" {
				appliedVersion = store.getPolicyVersion(d.PolicyName)
				d.PolicyVersion = appliedVersion
			}
			d.mu.Unlock()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":                 name,
			"appliedPolicyVersion": appliedVersion,
		})
	}
}

func handleDevicesDelete(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := deviceName(r)

		store.mu.Lock()
		if d, ok := store.byName[name]; ok {
			delete(store.byName, name)
			delete(store.byESID, d.EnterpriseSpecificID)
			log.Printf("Deleted fake device: %q (ESID: %q)", name, d.EnterpriseSpecificID)
		}
		store.mu.Unlock()

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
		enterpriseID := r.PathValue("eid")

		if !store.hasDevicesForEnterprise(enterpriseID) && google != nil {
			log.Printf("Forwarding policy patch to Google AMAPI: %q", name)
			google.ForwardPoliciesPatch(w, r)
			return
		}

		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}

		version := policyVersionCounter.Add(1)
		store.setPolicyVersion(name, version)

		if google != nil && hasSeenRealDevice.Load() {
			fwdReq := r.Clone(context.Background())
			fwdReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			go func() {
				google.ForwardPoliciesPatch(&discardResponseWriter{}, fwdReq)
			}()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    name,
			"version": version,
		})
	}
}

// handlePolicyAction handles POST on policies: modifyPolicyApplications and removePolicyApplications.
func handlePolicyAction(store *deviceStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := policyName(r)

		version := policyVersionCounter.Add(1)
		store.setPolicyVersion(name, version)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": version,
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

func handleCatchAll(google *googleForwarder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("ERROR: unhandled AMAPI endpoint: %q %q — add a handler or forwarding for this route", r.Method, r.URL.Path)
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
