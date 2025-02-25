package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"howett.net/plist"
)

type mdmProxy struct {
	migrateUDIDs      map[string]struct{}
	migratePercentage int
	existingServerURL string
	existingHostname  string
	fleetServerURL    string
	existingProxy     *httputil.ReverseProxy
	fleetProxy        *httputil.ReverseProxy
	// mutex is used to sync reads/updates to the migrateUDIDs and migratePercentage
	mutex sync.RWMutex
	// token is used to authenticate updates to the migrateUDIDs and migratePercentage
	token      string
	debug      bool
	logSkipped bool
}

func skipRequest(r *http.Request) bool {
	// Throw out a bunch of common junk requests
	return strings.Contains(r.URL.Path, ".php") ||
		strings.Contains(r.URL.Path, ".git") ||
		strings.Contains(r.URL.Path, ".yml") ||
		strings.Contains(r.URL.Path, ".txt") ||
		strings.Contains(r.URL.Path, ".py") ||
		strings.Contains(r.URL.Path, "wp-") ||
		strings.Contains(r.URL.Path, "private") ||
		(r.URL.Path == "/" && r.Method == http.MethodPost)
}

func (m *mdmProxy) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host != "" {
		log.Printf("%s %s Forbidden", r.Method, r.URL.String())
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if skipRequest(r) {
		if m.logSkipped {
			log.Printf("Forbidden skipped request: %s %s", r.Method, r.URL.String())
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Send all SCEP requests to the existing server
	if strings.Contains(r.URL.Path, "scep") {
		log.Printf("%s %s -> Existing (SCEP)", r.Method, r.URL.String())
		m.existingProxy.ServeHTTP(w, r)
		return
	}

	// Send all micromdm API requests to the existing server
	if strings.HasPrefix(r.URL.Path, "/v1") || strings.HasPrefix(r.URL.Path, "/push") {
		log.Printf("%s %s -> Existing (API)", r.Method, r.URL.String())
		m.existingProxy.ServeHTTP(w, r)
		return
	}

	if r.URL.Path == "/" && r.Method == http.MethodGet {
		log.Printf("%s %s -> Existing (Home)", r.Method, r.URL.String())
		m.existingProxy.ServeHTTP(w, r)
		return
	}

	// Send all micromdm repo requests to the existing server
	if strings.HasPrefix(r.URL.Path, "/repo") {
		log.Printf("%s %s -> Existing (Repo)", r.Method, r.URL.String())
		m.existingProxy.ServeHTTP(w, r)
		return

	}

	if !strings.HasPrefix(r.URL.Path, "/mdm") {
		if m.logSkipped {
			log.Printf("Forbidden non-mdm request: %s %s", r.Method, r.URL.String())
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Read the body of the request
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		log.Println("Failed to read request body: ", err.Error())
		http.Error(w, "Unable to read request body", http.StatusUnprocessableEntity)
		return
	}
	// Reset body so that the reverse proxy request includes it
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Get the UDID from request
	udid, err := udidFromRequestBody(body)
	if err != nil {
		log.Printf("%s %s Failed to get UDID: %v", r.Method, r.URL.String(), err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Migrated UDIDs go to the Fleet server, otherwise requests go to the existing server.
	if udid != "" && m.isUDIDMigrated(udid) {
		log.Printf("%s %s (%s) -> Fleet", r.Method, r.URL.String(), udid)
		if m.debug {
			log.Printf("Fleet request: %s", string(body))
		}
		m.fleetProxy.ServeHTTP(w, r)
	} else {
		log.Printf("%s %s (%s) -> Existing", r.Method, r.URL.String(), udid)
		m.existingProxy.ServeHTTP(w, r)
	}
}

func (m *mdmProxy) handleUpdatePercentage(w http.ResponseWriter, r *http.Request) {
	if m.token == "" {
		http.Error(w, "Set auth token to enable remote updates", http.StatusUnauthorized)
		return
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header must be provided", http.StatusUnauthorized)
		return

	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Authorization header must start with \"Bearer \"", http.StatusUnauthorized)
		return
	}
	if authHeader != "Bearer "+m.token {
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
	log.Print(msg)
	fmt.Fprint(w, msg)
}

func (m *mdmProxy) handleUpdateMigrateUDIDs(w http.ResponseWriter, r *http.Request) {
	if m.token == "" {
		http.Error(w, "Set auth token to enable remote updates", http.StatusUnauthorized)
		return
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header must be provided", http.StatusUnauthorized)
		return

	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Authorization header must start with \"Bearer \"", http.StatusUnauthorized)
		return
	}
	if authHeader != "Bearer "+m.token {
		http.Error(w, "Authorization header does not match", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()
	udids, err := processUDIDs(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.migrateUDIDs = udids

	msg := fmt.Sprintf("Migrate UDIDs updated: %v\n", udids)
	log.Print(msg)
	fmt.Fprint(w, msg)
}

func processUDIDs(in io.Reader) (map[string]struct{}, error) {
	scanner := bufio.NewScanner(in)
	scanner.Split(bufio.ScanWords)
	udids := make(map[string]struct{})
	for scanner.Scan() {
		udids[strings.TrimSpace(scanner.Text())] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Failed to scan UDIDs: %w", err)
	}
	return udids, nil
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
	// Not all requests (eg. SCEP) contain a UDID. Return empty without an error in this case.
	if len(body) == 0 {
		return "", nil
	}

	type mdmRequest struct {
		UDID string `plist:""`
	}
	var req mdmRequest
	_, err := plist.Unmarshal(body, &req)
	if err != nil {
		return "", fmt.Errorf("unmarshal request: %w body: %s", err, string(body))
	}
	if req.UDID == "" {
		return "", errors.New("request body does not contain UDID")
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
	return int(index) < percentage //nolint:gosec // G115 false positive
}

func makeExistingProxy(existingURL, existingDNSName string) *httputil.ReverseProxy {
	targetURL, err := url.Parse(existingURL)
	if err != nil {
		panic("failed to parse fleet-url: " + err.Error())
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Allow TLS validation to use the "old" server name
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig.ServerName = existingDNSName
	proxy.Transport = transport

	return proxy
}

func makeFleetProxy(fleetURL string, debug bool) *httputil.ReverseProxy {
	targetURL, err := url.Parse(fleetURL)
	if err != nil {
		panic("failed to parse fleet-url: " + err.Error())
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	if debug {
		proxy.ModifyResponse = func(r *http.Response) error {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			err = r.Body.Close()
			if err != nil {
				return err
			}
			r.Body = io.NopCloser(bytes.NewReader(b))

			log.Println("Fleet response: ", string(b))

			return nil
		}
	}

	return proxy
}

func main() {
	authToken := flag.String("auth-token", "", "Auth token for remote flag updates (remote updates disabled if not provided)")
	existingURL := flag.String("existing-url", "", "Existing MDM server URL (full path) (required)")
	existingHostname := flag.String("existing-hostname", "", "Hostname for existing MDM server (eg. 'mdm.example.com') (required)")
	fleetURL := flag.String("fleet-url", "", "Fleet MDM server URL (full path) (required)")
	migratePercentage := flag.Int("migrate-percentage", 0, "Percentage of clients to migrate from existing MDM to Fleet")
	migrateUDIDs := flag.String("migrate-udids", "", "Space/newline-delimited list of UDIDs to migrate always")
	serverAddr := flag.String("server-address", ":8080", "Address for server to listen on")
	debug := flag.Bool("debug", false, "Enable debug logging")
	logSkipped := flag.Bool("log-skipped", false, "Log skipped requests (usually from web scanners)")
	check := flag.String("check", "", "Print whether the specified UDID is migrated with the current configuration, then exit")
	flag.Parse()

	udids, err := processUDIDs(bytes.NewBufferString(*migrateUDIDs))
	if err != nil {
		panic(err)
	}

	proxy := mdmProxy{
		token:             *authToken,
		existingServerURL: *existingURL,
		fleetServerURL:    *fleetURL,
		existingHostname:  *existingHostname,
		migratePercentage: *migratePercentage,
		migrateUDIDs:      udids,
		existingProxy:     makeExistingProxy(*existingURL, *existingHostname),
		fleetProxy:        makeFleetProxy(*fleetURL, *debug),
		debug:             *debug,
		logSkipped:        *logSkipped,
	}

	if len(*check) > 0 {
		if proxy.isUDIDMigrated(*check) {
			fmt.Printf("%s IS migrated\n", *check)
		} else {
			fmt.Printf("%s IS NOT migrated\n", *check)
		}
		os.Exit(0)
	}

	// Check required flags
	if *existingURL == "" {
		log.Fatal("--existing-url must be set")
	}
	if *existingHostname == "" {
		log.Fatal("--existing-hostname must be set")
	}
	if *fleetURL == "" {
		log.Fatal("--fleet-url must be set")
	}

	log.Printf("--migrate-udids set: %v", udids)
	log.Printf("--migrate-percentage set: %d", *migratePercentage)
	log.Printf("--existing-url set: %s", *existingURL)
	log.Printf("--existing-hostname set: %s", *existingHostname)
	log.Printf("--fleet-url set: %s", *fleetURL)
	log.Printf("--debug set: %v", *debug)
	log.Printf("--log-skipped set: %v", *logSkipped)
	if *authToken != "" {
		log.Printf("--auth-token set. Remote configuration enabled.")
	} else {
		log.Printf("--auth-token is empty. Remote configuration disabled.")
	}

	mux := http.NewServeMux()
	// Health check endpoint used for load balancers
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Printf("/healthz error: %v", err)
		}
	})
	// Remote management of migration (enabled if auth token set)
	mux.HandleFunc("/admin/udids", proxy.handleUpdateMigrateUDIDs)
	mux.HandleFunc("/admin/percentage", proxy.handleUpdatePercentage)
	// Handler for the actual proxying
	mux.HandleFunc("/", proxy.handleProxy)

	log.Printf("Starting server on %s", *serverAddr)
	server := &http.Server{
		Addr:              *serverAddr,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           mux,
	}
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}
