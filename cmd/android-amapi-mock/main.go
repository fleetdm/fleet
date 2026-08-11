// Command android-amapi-mock is a lightweight mock of Google's Android Management API
// for load testing Fleet with fake Android devices.
//
// It serves two roles:
//  1. AMAPI surface — Fleet calls these endpoints (policy patches, device patches, commands, etc.).
//     For registered fake devices, it returns canned responses. For real devices, it forwards
//     requests to the real Google AMAPI using service account credentials.
//  2. Coordination API — osquery-perf's Android agents call these to register devices and poll for
//     state (policy versions, pending commands) so they can send realistic PubSub messages to Fleet.
//
// Usage:
//
//	android-amapi-mock --listen :9999
//	android-amapi-mock --listen :9999 --google-credentials "$(cat service-account.json)"
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// fakeDevice holds the in-memory state for a single fake Android device.
type fakeDevice struct {
	mu                   sync.Mutex
	EnterpriseSpecificID string   `json:"enterprise_specific_id"`
	DeviceName           string   `json:"device_name"`
	EnterpriseID         string   `json:"enterprise_id"`
	PolicyVersion        int64    `json:"policy_version"`
	PolicyName           string   `json:"policy_name"`
	PendingCommands      []string `json:"pending_commands"`
	PendingCertificates  []uint   `json:"pending_certificates"`
}

// deviceStore is the in-memory registry of fake devices and policy versions.
type deviceStore struct {
	mu sync.RWMutex
	// byESID maps EnterpriseSpecificID -> device
	byESID map[string]*fakeDevice
	// byName maps AMAPI device resource name -> device
	byName map[string]*fakeDevice

	// policyVersions tracks the latest version for each policy name.
	// Fleet uses per-device policies named enterprises/{id}/policies/{hostUUID}.
	policyMu       sync.RWMutex
	policyVersions map[string]int64
}

func newDeviceStore() *deviceStore {
	return &deviceStore{
		byESID:         make(map[string]*fakeDevice),
		byName:         make(map[string]*fakeDevice),
		policyVersions: make(map[string]int64),
	}
}

func (ds *deviceStore) setPolicyVersion(policyName string, version int64) {
	ds.policyMu.Lock()
	defer ds.policyMu.Unlock()
	ds.policyVersions[policyName] = version
}

func (ds *deviceStore) getPolicyVersion(policyName string) int64 {
	ds.policyMu.RLock()
	defer ds.policyMu.RUnlock()
	return ds.policyVersions[policyName]
}

func (ds *deviceStore) register(d *fakeDevice) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.byESID[d.EnterpriseSpecificID] = d
	ds.byName[d.DeviceName] = d
}

func (ds *deviceStore) getByESID(esid string) *fakeDevice {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.byESID[esid]
}

func (ds *deviceStore) getByName(name string) *fakeDevice {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.byName[name]
}

func (ds *deviceStore) allDeviceNames() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	names := make([]string, 0, len(ds.byName))
	for name := range ds.byName {
		names = append(names, name)
	}
	return names
}

// policyVersionCounter is a global atomic counter for policy versions.
var policyVersionCounter atomic.Int64

// hasSeenRealDevice indicates that a real device has been seen.
var hasSeenRealDevice atomic.Bool

func main() {
	listen := flag.String("listen", ":9999", "Address to listen on")
	googleCredentials := flag.String("google-credentials", "", "Google service account JSON credentials (enables forwarding for real devices). Pass via: --google-credentials \"$(cat credentials.json)\" or set GOOGLE_CREDENTIALS env var")
	latencyMean := flag.Duration("latency", 200*time.Millisecond, "Mean latency added to AMAPI responses (simulates Google API latency)")
	errorRate := flag.Float64("error-rate", 0.01, "Fraction of AMAPI requests that return 429/5xx errors [0, 1]")
	flag.Parse()

	// Fall back to env var if flag not provided (for ECS Secrets Manager injection)
	credJSON := *googleCredentials
	if credJSON == "" {
		credJSON = os.Getenv("GOOGLE_CREDENTIALS")
	}

	policyVersionCounter.Store(1)

	store := newDeviceStore()

	// Set up authenticated Google API client for real device forwarding
	var google *googleForwarder
	if credJSON != "" {
		var err error
		google, err = newGoogleForwarder(credJSON)
		if err != nil {
			log.Fatalf("Failed to create Google forwarder: %v", err)
		}
		log.Printf("Google credentials loaded — forwarding real device requests to Google AMAPI")
	}

	mux := http.NewServeMux()

	// ---- Health check ----
	mux.HandleFunc("GET /mock/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ---- Coordination API (osquery-perf calls these) ----
	mux.HandleFunc("POST /mock/devices/register", handleRegister(store))
	mux.HandleFunc("GET /mock/devices/{esid}/state", handleGetState(store))

	// sim wraps AMAPI handlers with simulated latency and occasional errors
	sim := func(h http.HandlerFunc) http.HandlerFunc {
		return simulateLatencyAndErrors(*latencyMean, *errorRate, h)
	}

	// ---- AMAPI: Devices ----
	fwd := forwardForRealDevice(store, google)
	mux.HandleFunc("GET /v1/enterprises/{eid}/devices/{did}", fwd(sim(handleDevicesGet(store))))
	mux.HandleFunc("PATCH /v1/enterprises/{eid}/devices/{did}", fwd(sim(handleDevicesPatch(store))))
	mux.HandleFunc("DELETE /v1/enterprises/{eid}/devices/{did}", fwd(sim(handleDevicesDelete(store))))
	mux.HandleFunc("POST /v1/enterprises/{eid}/devices/{did}", fwd(sim(handleIssueCommand(store))))
	mux.HandleFunc("GET /v1/enterprises/{eid}/devices", sim(handleDevicesList(store, google)))

	// ---- AMAPI: Policies ----
	mux.HandleFunc("PATCH /v1/enterprises/{eid}/policies/{pid}", sim(handlePoliciesPatch(store, google)))
	mux.HandleFunc("POST /v1/enterprises/{eid}/policies/{pid}", sim(handlePolicyAction(store)))

	// ---- AMAPI: Other ----
	mux.HandleFunc("POST /v1/enterprises/{eid}/enrollmentTokens", sim(forwardOrMock(google, handleEnrollmentTokenCreate())))
	mux.HandleFunc("GET /v1/enterprises/{eid}/applications/{pkg}", sim(forwardOrMock(google, handleApplicationsGet())))
	mux.HandleFunc("POST /v1/enterprises/{eid}/webApps", sim(forwardOrMock(google, handleWebAppsCreate())))
	mux.HandleFunc("GET /v1/enterprises", sim(forwardOrMock(google, handleEnterprisesList(store))))

	// Catch-all for unmatched /v1/ requests
	mux.HandleFunc("/v1/", handleCatchAll(google))

	srv := &http.Server{
		Addr:         *listen,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("Mock AMAPI proxy listening on %s", *listen)
	log.Fatal(srv.ListenAndServe())
}

// ---- Route helpers ----

// deviceName builds the AMAPI resource name from path values.
func deviceName(r *http.Request) string {
	did := r.PathValue("did")
	did = strings.TrimSuffix(did, ":issueCommand")
	return "enterprises/" + r.PathValue("eid") + "/devices/" + did
}

// policyName builds the AMAPI policy resource name from path values.
func policyName(r *http.Request) string {
	pid := r.PathValue("pid")
	pid = strings.TrimSuffix(pid, ":modifyPolicyApplications")
	pid = strings.TrimSuffix(pid, ":removePolicyApplications")
	return "enterprises/" + r.PathValue("eid") + "/policies/" + pid
}
