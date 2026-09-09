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
	"fmt"
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

// maxRestorablePolicyVersion bounds the policy version a device may report when it
// registers again, so a bogus value can't push the version counter near overflow and turn
// every later version negative. It is far above any version a load test will reach.
const maxRestorablePolicyVersion = 1 << 40

// deviceStore is the in-memory registry of fake devices and policy versions.
type deviceStore struct {
	mu sync.RWMutex
	// byESID maps EnterpriseSpecificID -> device
	byESID map[string]*fakeDevice
	// byName maps AMAPI device resource name -> device
	byName map[string]*fakeDevice
	// deletedESIDs and deletedNames record devices Fleet deleted through AMAPI. Deletion is a
	// real unenrollment, not lost state, so such a device must never come back: a device that
	// registered again after being deleted would re-enroll itself in Fleet and produce an
	// unenroll/enroll flip-flop.
	//
	// Both keys are needed. A delete that arrives while this process has forgotten the device
	// (restarted, and the agent has not registered again yet) can only be recorded by resource
	// name, since the ESID is not known then; the agent's next registration is matched on
	// either key. Devices are never removed from these sets, which is bounded by how many
	// devices a load test deletes.
	deletedESIDs map[string]struct{}
	deletedNames map[string]struct{}

	// policyVersions tracks the latest version for each policy name.
	// Fleet uses per-device policies named enterprises/{id}/policies/{hostUUID}.
	// policyVersion is the counter versions are issued from; it is guarded by policyMu so
	// issuing a version and recording it against a policy is one atomic step.
	policyMu       sync.RWMutex
	policyVersions map[string]int64
	policyVersion  int64
}

func newDeviceStore() *deviceStore {
	return &deviceStore{
		byESID:         make(map[string]*fakeDevice),
		byName:         make(map[string]*fakeDevice),
		deletedESIDs:   make(map[string]struct{}),
		deletedNames:   make(map[string]struct{}),
		policyVersions: make(map[string]int64),
		policyVersion:  1,
	}
}

// nextPolicyVersion issues the next version and records it as the current version of
// policyName.
func (ds *deviceStore) nextPolicyVersion(policyName string) int64 {
	ds.policyMu.Lock()
	defer ds.policyMu.Unlock()
	ds.policyVersion++
	ds.policyVersions[policyName] = ds.policyVersion
	return ds.policyVersion
}

func (ds *deviceStore) getPolicyVersion(policyName string) int64 {
	ds.policyMu.RLock()
	defer ds.policyMu.RUnlock()
	return ds.policyVersions[policyName]
}

// raisePolicyVersionCounter pulls the counter up to a version a device reports when it
// registers again after this process lost its state, so versions issued from here on stay
// above the version the device already told Fleet about.
//
// It deliberately does NOT record the version against the policy: doing so would let a
// device claim a high applied version that this process never issued, and Fleet verifies
// profiles whose included_in_policy_version is <= the applied version — so pending profiles
// would flip to Verified without the policy ever having been applied, hiding the very
// "stuck in Pending" bugs this harness exists to find.
func (ds *deviceStore) raisePolicyVersionCounter(version int64) {
	if version <= 0 || version > maxRestorablePolicyVersion {
		return
	}
	ds.policyMu.Lock()
	defer ds.policyMu.Unlock()
	if ds.policyVersion < version {
		ds.policyVersion = version
	}
}

// register adds a device, or updates the identity and policy fields of one that is already
// known. An existing device is updated in place so that state other handlers hold a pointer
// to (pending commands, pending certificates) survives a device registering again.
//
// A device's policy is only changed by a registration that actually reports one (d.PolicyName
// is set). A registration without a policy — a freshly started agent, or one whose reported
// version was rejected — leaves the known policy alone: overwriting it with the default policy
// at version 0 would tell Fleet the applied policy regressed and strand every profile already
// delivered. A new device with no reported policy starts on the default policy.
//
// It reports false, and registers nothing, for a device that was deleted. The check shares
// the write lock with the insert so a delete can't land between the two.
func (ds *deviceStore) register(d *fakeDevice) bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, ok := ds.deletedESIDs[d.EnterpriseSpecificID]; ok {
		return false
	}
	if _, ok := ds.deletedNames[d.DeviceName]; ok {
		return false
	}

	if existing, ok := ds.byESID[d.EnterpriseSpecificID]; ok {
		existing.mu.Lock()
		delete(ds.byName, existing.DeviceName)
		existing.DeviceName = d.DeviceName
		existing.EnterpriseID = d.EnterpriseID
		switch {
		case d.PolicyName == "":
			// Nothing reported: keep what we already know.
		case d.PolicyName == existing.PolicyName:
			// Same policy: never lower the version the device already observed.
			existing.PolicyVersion = max(existing.PolicyVersion, d.PolicyVersion)
		default:
			existing.PolicyName = d.PolicyName
			existing.PolicyVersion = d.PolicyVersion
		}
		existing.mu.Unlock()
		ds.byName[existing.DeviceName] = existing
		return true
	}

	if d.PolicyName == "" && d.EnterpriseID != "" {
		d.PolicyName = fmt.Sprintf("enterprises/%s/policies/default", d.EnterpriseID)
		d.PolicyVersion = 0
	}
	ds.byESID[d.EnterpriseSpecificID] = d
	ds.byName[d.DeviceName] = d
	return true
}

// markDeleted records that the device with this resource name was deleted, not merely
// forgotten, and removes it if it is currently known. The name is recorded either way: a
// delete that arrives after this process lost its state has no other identifier to key on,
// and the device must still not be able to register again.
func (ds *deviceStore) markDeleted(name string) (*fakeDevice, bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.deletedNames[name] = struct{}{}
	d, ok := ds.byName[name]
	if !ok {
		return nil, false
	}
	delete(ds.byName, name)
	delete(ds.byESID, d.EnterpriseSpecificID)
	ds.deletedESIDs[d.EnterpriseSpecificID] = struct{}{}
	return d, true
}

// wasDeleted reports whether this process deleted the device with the given ESID.
func (ds *deviceStore) wasDeleted(esid string) bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	_, ok := ds.deletedESIDs[esid]
	return ok
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

	mux := newMux(store, google, *latencyMean, *errorRate)

	srv := &http.Server{
		Addr:         *listen,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Printf("Mock AMAPI proxy listening on %s", *listen)
	log.Fatal(srv.ListenAndServe())
}

// newMux registers every mock route. Route patterns matter to the handlers (they read path
// values), so this is shared between main and the tests rather than duplicated.
func newMux(store *deviceStore, google *googleForwarder, latencyMean time.Duration, errorRate float64) *http.ServeMux {
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
		return simulateLatencyAndErrors(latencyMean, errorRate, h)
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

	return mux
}

// ---- Route helpers ----

// trimAction removes an AMAPI custom method suffix (":issueCommand",
// ":modifyPolicyApplications", ...) from a resource ID. Every action is routed to a single
// handler per method, so the suffix is stripped generically: naming the known actions here
// would silently reintroduce the resource-lookup bug the moment AMAPI grows another one.
// Resource IDs themselves are UUIDs or numeric IDs and never contain a colon.
func trimAction(resourceID string) string {
	id, _, _ := strings.Cut(resourceID, ":")
	return id
}

// deviceName builds the AMAPI resource name from path values.
func deviceName(r *http.Request) string {
	return "enterprises/" + r.PathValue("eid") + "/devices/" + trimAction(r.PathValue("did"))
}

// policyID returns the policy ID from path values with any action suffix removed.
// Fleet names per-device policies after the host UUID (the enterpriseSpecificID for
// Android), so callers can use this to look up the fake device the policy belongs to.
func policyID(r *http.Request) string {
	return trimAction(r.PathValue("pid"))
}

// policyName builds the AMAPI policy resource name from path values.
func policyName(r *http.Request) string {
	return "enterprises/" + r.PathValue("eid") + "/policies/" + policyID(r)
}
