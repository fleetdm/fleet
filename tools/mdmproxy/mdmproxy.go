package main

import (
	"bufio"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"howett.net/plist"
)

type mdmProxy struct {
	migrateUDIDs      map[string]struct{}
	migratePercentage int
	existingServerURL string
	fleetServerURL    string
	// mutex is used to sync reads/updates to the migrateUDIDs and migratePercentage
	mutex sync.RWMutex
	// token is used to authenticate updates to the migrateUDIDs and migratePercentage
	token string
}

func (m *mdmProxy) handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Read the body of the request
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusUnprocessableEntity)
		return
	}

	// Get the UDID from request
	udid, err := udidFromRequestBody(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	_ = udid

	// Perform the redirect TODO
	// > MDM follows HTTP 3xx redirections without user interaction. However, it doesn’t save the URL
	// > given by HTTP 301 (Moved Permanently) redirections. Each transaction begins at the URL the MDM
	// > payload specifies.
	// https://developer.apple.com/documentation/devicemanagement/implementing_device_management/sending_mdm_commands_to_a_device
	http.Redirect(w, r, "/"+udid /* TODO */, http.StatusMovedPermanently)
}

func (m *mdmProxy) handleUpdatePercentage(w http.ResponseWriter, r *http.Request) {
	if m.token == "" {
		http.Error(w, "Set auth token to enable remote updates", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Authorization header must be provided", http.StatusUnauthorized)
		return

	}
	if token != m.token {
		http.Error(w, "Authorization header does not match", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusInternalServerError)
		return
	}
	percentage, err := strconv.Atoi(string(body))
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot read body as integer: %v", err), http.StatusUnprocessableEntity)
		return
	}
	if percentage < 0 || percentage > 100 {
		http.Error(w, "Percentage should be in range (0, 100)", http.StatusBadRequest)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.migratePercentage = percentage

	msg := fmt.Sprintf("Migrate percentage updated: %v\n", percentage)
	log.Printf(msg)
	fmt.Fprintf(w, msg)
}

func (m *mdmProxy) handleUpdateMigrateUDIDs(w http.ResponseWriter, r *http.Request) {
	if m.token == "" {
		http.Error(w, "Set auth token to enable remote updates", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Authorization header must be provided", http.StatusUnauthorized)
		return

	}
	if token != m.token {
		http.Error(w, "Authorization header does not match", http.StatusUnauthorized)
		return
	}

	scanner := bufio.NewScanner(r.Body)
	udids := make(map[string]struct{})
	defer r.Body.Close()
	for scanner.Scan() {
		udids[strings.TrimSpace(scanner.Text())] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan UDIDs: %v", err), http.StatusInternalServerError)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.migrateUDIDs = udids

	msg := fmt.Sprintf("Migrate UDIDs updated: %v\n", udids)
	log.Printf(msg)
	fmt.Fprintf(w, msg)
}

func (m *mdmProxy) isUDIDMigrated(udid string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	// If the UDID is manually included, it's always migrated
	if _, ok := m.migrateUDIDs[udid]; ok {
		return true
	}

	// Otherwise migrate by percentage
	return udidIncludedByPercentage(udid, m.migratePercentage)
}

func udidFromRequestBody(body []byte) (string, error) {
	if body == nil || len(body) == 0 {
		return "", fmt.Errorf("request has empty body")
	}

	type mdmRequest struct {
		UDID string `plist:""`
	}
	var req mdmRequest
	_, err := plist.Unmarshal(body, &req)
	if err != nil {
		return "", fmt.Errorf("unmarshal request: %w", err)
	}
	if req.UDID == "" {
		return "", fmt.Errorf("request body does not contain UDID")
	}

	return req.UDID, nil
}

func hashUDID(udid string) uint {
	hash := fnv.New32a()
	hash.Write([]byte(udid))
	return uint(hash.Sum32())
}

func udidIncludedByPercentage(udid string, percentage int) bool {
	index := hashUDID(udid) % 100
	return int(index) < percentage
}

func main() {
	authToken := flag.String("token", "", "Auth token for remote flag updates (remote updates disabled if not provided)")
	existingURL := flag.String("existing-url", "", "Existing MDM server URL (full path)")
	fleetURL := flag.String("fleet-url", "", "Fleet MDM server URL (full path)")
	migratePercentage := flag.Int("migrate-percentage", 0, "Percentage of clients to migrate from existing MDM to Fleet")
	flag.Parse()

	// TODO remove
	*authToken = "foo"
	proxy := mdmProxy{
		token:             *authToken,
		existingServerURL: *existingURL,
		fleetServerURL:    *fleetURL,
		migratePercentage: *migratePercentage,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/udids", proxy.handleUpdateMigrateUDIDs)
	mux.HandleFunc("/admin/percentage", proxy.handleUpdatePercentage)
	mux.HandleFunc("/", proxy.handleRedirect)

	fmt.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
