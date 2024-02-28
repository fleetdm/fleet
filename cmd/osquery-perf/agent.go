package main

import (
	"bytes"
	"compress/bzip2"
	cryptorand "crypto/rand"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
)

var (
	//go:embed *.tmpl
	templatesFS embed.FS

	//go:embed *.software
	macOSVulnerableSoftwareFS embed.FS

	//go:embed ubuntu_2204-software.json.bz2
	ubuntuSoftwareFS embed.FS
	//go:embed windows_11-software.json.bz2
	windowsSoftwareFS embed.FS

	macosVulnerableSoftware []fleet.Software
	windowsSoftware         []map[string]string
	ubuntuSoftware          []map[string]string
)

func loadMacOSVulnerableSoftware() {
	macOSVulnerableSoftwareData, err := macOSVulnerableSoftwareFS.ReadFile("macos_vulnerable.software")
	if err != nil {
		log.Fatal("reading vulnerable macOS software file: ", err)
	}
	lines := bytes.Split(macOSVulnerableSoftwareData, []byte("\n"))
	for _, line := range lines {
		parts := bytes.Split(line, []byte("##"))
		if len(parts) < 2 {
			log.Println("skipping", string(line))
			continue
		}
		macosVulnerableSoftware = append(macosVulnerableSoftware, fleet.Software{
			Name:    strings.TrimSpace(string(parts[0])),
			Version: strings.TrimSpace(string(parts[1])),
			Source:  "apps",
		})
	}
	log.Printf("Loaded %d vulnerable macOS software", len(macosVulnerableSoftware))
}

func loadSoftwareItems(fs embed.FS, path string) []map[string]string {
	bz2, err := fs.Open(path)
	if err != nil {
		panic(err)
	}

	type softwareJSON struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Release string `json:"release,omitempty"`
		Arch    string `json:"arch,omitempty"`
	}
	var softwareList []softwareJSON
	// ignoring "G110: Potential DoS vulnerability via decompression bomb", as this is test code.
	if err := json.NewDecoder(bzip2.NewReader(bz2)).Decode(&softwareList); err != nil { //nolint:gosec
		panic(err)
	}

	softwareRows := make([]map[string]string, 0, len(softwareList))
	for _, s := range softwareList {
		softwareRows = append(softwareRows, map[string]string{
			"name":    s.Name,
			"version": s.Version,
			"source":  "programs",
		})
	}
	return softwareRows
}

func init() {
	loadMacOSVulnerableSoftware()
	windowsSoftware = loadSoftwareItems(windowsSoftwareFS, "windows_11-software.json.bz2")
	ubuntuSoftware = loadSoftwareItems(ubuntuSoftwareFS, "ubuntu_2204-software.json.bz2")
}

type Stats struct {
	startTime              time.Time
	errors                 int
	osqueryEnrollments     int
	orbitEnrollments       int
	mdmEnrollments         int
	distributedWrites      int
	mdmCommandsReceived    int
	distributedReads       int
	configRequests         int
	configErrors           int
	resultLogRequests      int
	orbitErrors            int
	mdmErrors              int
	desktopErrors          int
	distributedReadErrors  int
	distributedWriteErrors int
	resultLogErrors        int
	bufferedLogs           int

	l sync.Mutex
}

func (s *Stats) IncrementErrors(errors int) {
	s.l.Lock()
	defer s.l.Unlock()
	s.errors += errors
}

func (s *Stats) IncrementEnrollments() {
	s.l.Lock()
	defer s.l.Unlock()
	s.osqueryEnrollments++
}

func (s *Stats) IncrementOrbitEnrollments() {
	s.l.Lock()
	defer s.l.Unlock()
	s.orbitEnrollments++
}

func (s *Stats) IncrementMDMEnrollments() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmEnrollments++
}

func (s *Stats) IncrementDistributedWrites() {
	s.l.Lock()
	defer s.l.Unlock()
	s.distributedWrites++
}

func (s *Stats) IncrementMDMCommandsReceived() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmCommandsReceived++
}

func (s *Stats) IncrementDistributedReads() {
	s.l.Lock()
	defer s.l.Unlock()
	s.distributedReads++
}

func (s *Stats) IncrementConfigRequests() {
	s.l.Lock()
	defer s.l.Unlock()
	s.configRequests++
}

func (s *Stats) IncrementConfigErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.configErrors++
}

func (s *Stats) IncrementResultLogRequests() {
	s.l.Lock()
	defer s.l.Unlock()
	s.resultLogRequests++
}

func (s *Stats) IncrementOrbitErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.orbitErrors++
}

func (s *Stats) IncrementMDMErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmErrors++
}

func (s *Stats) IncrementDesktopErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.desktopErrors++
}

func (s *Stats) IncrementDistributedReadErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.distributedReadErrors++
}

func (s *Stats) IncrementDistributedWriteErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.distributedWriteErrors++
}

func (s *Stats) IncrementResultLogErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.resultLogErrors++
}

func (s *Stats) UpdateBufferedLogs(v int) {
	s.l.Lock()
	defer s.l.Unlock()
	s.bufferedLogs += v
	if s.bufferedLogs < 0 {
		s.bufferedLogs = 0
	}
}

func (s *Stats) Log() {
	s.l.Lock()
	defer s.l.Unlock()

	log.Printf(
		"uptime: %s, error rate: %.2f, osquery enrolls: %d, orbit enrolls: %d, mdm enrolls: %d, distributed/reads: %d, distributed/writes: %d, config requests: %d, result log requests: %d, mdm commands received: %d, config errors: %d, distributed/read errors: %d, distributed/write errors: %d, log result errors: %d, orbit errors: %d, desktop errors: %d, mdm errors: %d, buffered logs: %d",
		time.Since(s.startTime).Round(time.Second),
		float64(s.errors)/float64(s.osqueryEnrollments),
		s.osqueryEnrollments,
		s.orbitEnrollments,
		s.mdmEnrollments,
		s.distributedReads,
		s.distributedWrites,
		s.configRequests,
		s.resultLogRequests,
		s.mdmCommandsReceived,
		s.configErrors,
		s.distributedReadErrors,
		s.distributedWriteErrors,
		s.resultLogErrors,
		s.orbitErrors,
		s.desktopErrors,
		s.mdmErrors,
		s.bufferedLogs,
	)
}

func (s *Stats) runLoop() {
	ticker := time.Tick(10 * time.Second)
	for range ticker {
		s.Log()
	}
}

type nodeKeyManager struct {
	filepath string

	l        sync.Mutex
	nodekeys []string
}

func (n *nodeKeyManager) LoadKeys() {
	if n.filepath == "" {
		return
	}

	n.l.Lock()
	defer n.l.Unlock()

	data, err := os.ReadFile(n.filepath)
	if err != nil {
		log.Println("WARNING (ignore if creating a new node key file): error loading nodekey file:", err)
		return
	}
	n.nodekeys = strings.Split(string(data), "\n")
	n.nodekeys = n.nodekeys[:len(n.nodekeys)-1] // remove last empty node key due to new line.
	log.Printf("loaded %d node keys", len(n.nodekeys))
}

func (n *nodeKeyManager) Get(i int) string {
	n.l.Lock()
	defer n.l.Unlock()

	if len(n.nodekeys) > i {
		return n.nodekeys[i]
	}
	return ""
}

func (n *nodeKeyManager) Add(nodekey string) {
	if n.filepath == "" {
		return
	}

	// we lock just to make sure we write one at a time
	n.l.Lock()
	defer n.l.Unlock()

	f, err := os.OpenFile(n.filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("error opening nodekey file: %s", err.Error())
		return
	}
	defer f.Close()
	if _, err := f.WriteString(nodekey + "\n"); err != nil {
		log.Printf("error writing nodekey file: %s", err)
	}
}

type agent struct {
	agentIndex             int
	softwareCount          softwareEntityCount
	userCount              entityCount
	policyPassProb         float64
	munkiIssueProb         float64
	munkiIssueCount        int
	liveQueryFailProb      float64
	liveQueryNoResultsProb float64
	strings                map[string]string
	serverAddress          string
	stats                  *Stats
	nodeKeyManager         *nodeKeyManager
	nodeKey                string
	templates              *template.Template
	os                     string
	// deviceAuthToken holds Fleet Desktop device authentication token.
	//
	// Non-nil means the agent is identified as orbit osquery,
	// nil means the agent is identified as vanilla osquery.
	deviceAuthToken *string
	orbitNodeKey    *string

	// mdmClient simulates a device running the MDM protocol (client side).
	mdmClient *mdmtest.TestAppleMDMClient
	// isEnrolledToMDM is true when the mdmDevice has enrolled.
	isEnrolledToMDM bool
	// isEnrolledToMDMMu protects isEnrolledToMDM.
	isEnrolledToMDMMu sync.Mutex

	disableScriptExec   bool
	disableFleetDesktop bool
	loggerTLSMaxLines   int

	// atomic boolean is set to true when executing scripts, so that only a
	// single goroutine at a time can execute scripts.
	scriptExecRunning atomic.Bool

	//
	// The following are exported to be used by the templates.
	//

	EnrollSecret          string
	UUID                  string
	SerialNumber          string
	ConfigInterval        time.Duration
	LogInterval           time.Duration
	QueryInterval         time.Duration
	MDMCheckInInterval    time.Duration
	DiskEncryptionEnabled bool

	scheduledQueriesMu sync.Mutex // protects the below members
	scheduledQueries   []string
	scheduledQueryData []scheduledQuery
	bufferedResults    []resultLog
}

type entityCount struct {
	common int
	unique int
}

type softwareEntityCount struct {
	entityCount
	vulnerable                   int
	withLastOpened               int
	lastOpenedProb               float64
	commonSoftwareUninstallCount int
	commonSoftwareUninstallProb  float64
	uniqueSoftwareUninstallCount int
	uniqueSoftwareUninstallProb  float64
}

func newAgent(
	agentIndex int,
	serverAddress, enrollSecret string,
	templates *template.Template,
	configInterval, logInterval, queryInterval, mdmCheckInInterval time.Duration,
	softwareCount softwareEntityCount,
	userCount entityCount,
	policyPassProb float64,
	orbitProb float64,
	munkiIssueProb float64, munkiIssueCount int,
	emptySerialProb float64,
	mdmProb float64,
	mdmSCEPChallenge string,
	liveQueryFailProb float64,
	liveQueryNoResultsProb float64,
	disableScriptExec bool,
	disableFleetDesktop bool,
	loggerTLSMaxLines int,
) *agent {
	var deviceAuthToken *string
	if rand.Float64() <= orbitProb {
		deviceAuthToken = ptr.String(uuid.NewString())
	}
	serialNumber := mdmtest.RandSerialNumber()
	if rand.Float64() <= emptySerialProb {
		serialNumber = ""
	}
	uuid := strings.ToUpper(uuid.New().String())
	var mdmClient *mdmtest.TestAppleMDMClient
	if rand.Float64() <= mdmProb {
		mdmClient = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
			SCEPChallenge: mdmSCEPChallenge,
			SCEPURL:       serverAddress + apple_mdm.SCEPPath,
			MDMURL:        serverAddress + apple_mdm.MDMPath,
		})
		// Have the osquery agent match the MDM device serial number and UUID.
		serialNumber = mdmClient.SerialNumber
		uuid = mdmClient.UUID
	}
	return &agent{
		agentIndex:             agentIndex,
		serverAddress:          serverAddress,
		softwareCount:          softwareCount,
		userCount:              userCount,
		strings:                make(map[string]string),
		policyPassProb:         policyPassProb,
		munkiIssueProb:         munkiIssueProb,
		munkiIssueCount:        munkiIssueCount,
		liveQueryFailProb:      liveQueryFailProb,
		liveQueryNoResultsProb: liveQueryNoResultsProb,
		templates:              templates,
		deviceAuthToken:        deviceAuthToken,
		os:                     strings.TrimRight(templates.Name(), ".tmpl"),

		EnrollSecret:       enrollSecret,
		ConfigInterval:     configInterval,
		LogInterval:        logInterval,
		QueryInterval:      queryInterval,
		MDMCheckInInterval: mdmCheckInInterval,
		UUID:               uuid,
		SerialNumber:       serialNumber,

		mdmClient:           mdmClient,
		disableScriptExec:   disableScriptExec,
		disableFleetDesktop: disableFleetDesktop,
		loggerTLSMaxLines:   loggerTLSMaxLines,
	}
}

type enrollResponse struct {
	NodeKey string `json:"node_key"`
}

type distributedReadResponse struct {
	Queries map[string]string `json:"queries"`
}

type scheduledQuery struct {
	Query            string  `json:"query"`
	Name             string  `json:"name"`
	ScheduleInterval float64 `json:"interval"`
	Platform         string  `json:"platform"`
	Version          string  `json:"version"`
	Snapshot         bool    `json:"snapshot"`
	nextRun          float64
	numRows          uint
	packName         string
}

func (a *agent) isOrbit() bool {
	return a.deviceAuthToken != nil
}

func (a *agent) runLoop(i int, onlyAlreadyEnrolled bool) {
	if a.isOrbit() {
		if err := a.orbitEnroll(); err != nil {
			return
		}
	}

	if err := a.enroll(i, onlyAlreadyEnrolled); err != nil {
		return
	}

	_ = a.config()

	resp, err := a.DistributedRead()
	if err == nil {
		if len(resp.Queries) > 0 {
			_ = a.DistributedWrite(resp.Queries)
		}
	}

	if a.isOrbit() {
		go a.runOrbitLoop()
	}

	if a.mdmClient != nil {
		if err := a.mdmClient.Enroll(); err != nil {
			log.Printf("MDM enroll failed: %s", err)
			a.stats.IncrementMDMErrors()
			return
		}
		a.setMDMEnrolled()
		a.stats.IncrementMDMEnrollments()
		go a.runMDMLoop()
	}

	//
	// osquery runs three separate independent threads,
	//	- a thread for getting, running and submitting results for distributed queries (distributed).
	// 	- a thread for getting configuration from a remote server (config).
	//	- a thread for submitting log results (logger).
	//
	// Thus we try to simulate that as much as we can.

	// (1) distributed thread:
	go func() {
		liveQueryTicker := time.NewTicker(a.QueryInterval)
		defer liveQueryTicker.Stop()

		for range liveQueryTicker.C {
			if resp, err := a.DistributedRead(); err == nil && len(resp.Queries) > 0 {
				_ = a.DistributedWrite(resp.Queries)
			}
		}
	}()

	// (2) config thread:
	go func() {
		configTicker := time.NewTicker(a.ConfigInterval)
		defer configTicker.Stop()

		for range configTicker.C {
			_ = a.config()
		}
	}()

	// (3) logger thread:
	logTicker := time.NewTicker(a.LogInterval)
	defer logTicker.Stop()
	for range logTicker.C {
		// check if we have any scheduled queries that should be returning results
		var results []resultLog
		now := time.Now().Unix()
		a.scheduledQueriesMu.Lock()
		prevCount := len(a.bufferedResults)
		for i, query := range a.scheduledQueryData {
			if query.nextRun == 0 || now >= int64(query.nextRun) {
				results = append(results, resultLog{
					packName:  query.packName,
					queryName: query.Name,
					numRows:   int(query.numRows),
				})
				a.scheduledQueryData[i].nextRun = float64(now + int64(query.ScheduleInterval))
			}
		}
		a.bufferedResults = append(a.bufferedResults, results...)
		if len(a.bufferedResults) > 1_000_000 { // osquery buffered_log_max is 1M
			extra := len(a.bufferedResults) - 1_000_000
			a.bufferedResults = a.bufferedResults[extra:]
		}
		a.sendLogsBatch()
		newBufferedCount := len(a.bufferedResults) - prevCount
		a.stats.UpdateBufferedLogs(newBufferedCount)
		a.scheduledQueriesMu.Unlock()
	}
}

type resultLog struct {
	packName  string
	queryName string
	numRows   int
}

func (r resultLog) emit() json.RawMessage {
	return scheduledQueryResults(r.packName, r.queryName, r.numRows)
}

// sendLogsBatch sends up to loggerTLSMaxLines logs and updates the buffer.
func (a *agent) sendLogsBatch() {
	if len(a.bufferedResults) == 0 {
		return
	}

	batchSize := a.loggerTLSMaxLines
	if len(a.bufferedResults) < batchSize {
		batchSize = len(a.bufferedResults)
	}
	batch := a.bufferedResults[:batchSize]
	batchLogs := make([]json.RawMessage, 0, len(batch))
	for _, result := range batch {
		batchLogs = append(batchLogs, result.emit())
	}
	if err := a.submitLogs(batchLogs); err != nil {
		return
	}
	a.bufferedResults = a.bufferedResults[batchSize:]
}

func (a *agent) runOrbitLoop() {
	orbitClient, err := service.NewOrbitClient(
		"",
		a.serverAddress,
		"",
		true,
		a.EnrollSecret,
		nil,
		fleet.OrbitHostInfo{
			HardwareUUID:   a.UUID,
			HardwareSerial: a.SerialNumber,
			Hostname:       a.CachedString("hostname"),
		},
		nil,
	)
	if err != nil {
		log.Println("creating orbit client: ", err)
	}

	orbitClient.TestNodeKey = *a.orbitNodeKey

	deviceClient, err := service.NewDeviceClient(a.serverAddress, true, "", nil, "")
	if err != nil {
		log.Fatal("creating device client: ", err)
	}

	// orbit does a config check when it starts
	if _, err := orbitClient.GetConfig(); err != nil {
		a.stats.IncrementOrbitErrors()
	}

	tokenRotationEnabled := false
	if !a.disableFleetDesktop {
		tokenRotationEnabled = orbitClient.GetServerCapabilities().Has(fleet.CapabilityOrbitEndpoints) &&
			orbitClient.GetServerCapabilities().Has(fleet.CapabilityTokenRotation)

		// it also writes and checks the device token
		if tokenRotationEnabled {
			if err := orbitClient.SetOrUpdateDeviceToken(*a.deviceAuthToken); err != nil {
				a.stats.IncrementOrbitErrors()
				log.Println("orbitClient.SetOrUpdateDeviceToken: ", err)
			}

			if err := deviceClient.CheckToken(*a.deviceAuthToken); err != nil {
				a.stats.IncrementOrbitErrors()
				log.Println("deviceClient.CheckToken: ", err)
			}
		}
	}

	// checkToken is used to simulate Fleet Desktop polling until a token is
	// valid, we make a random number of requests to properly emulate what
	// happens in the real world as there are delays that are not accounted by
	// the way this simulation is arranged.
	checkToken := func() {
		min := 1
		max := 5
		numberOfRequests := rand.Intn(max-min+1) + min
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			numberOfRequests--
			if err := deviceClient.CheckToken(*a.deviceAuthToken); err != nil {
				log.Println("deviceClient.CheckToken: ", err)
			}
			if numberOfRequests == 0 {
				break
			}
		}
	}

	// Fleet Desktop performs a burst of check token requests when it's initialized
	if !a.disableFleetDesktop {
		checkToken()
	}

	// orbit makes a call to check the config and update the CLI flags every 30
	// seconds
	orbitConfigTicker := time.Tick(30 * time.Second)
	// orbit makes a call every 5 minutes to check the validity of the device
	// token on the server
	orbitTokenRemoteCheckTicker := time.Tick(5 * time.Minute)
	// orbit pings the server every 1 hour to rotate the device token
	orbitTokenRotationTicker := time.Tick(1 * time.Hour)
	// orbit polls the /orbit/ping endpoint every 5 minutes to check if the
	// server capabilities have changed
	capabilitiesCheckerTicker := time.Tick(5 * time.Minute)
	// fleet desktop polls for policy compliance every 5 minutes
	fleetDesktopPolicyTicker := time.Tick(5 * time.Minute)

	for {
		select {
		case <-orbitConfigTicker:
			cfg, err := orbitClient.GetConfig()
			if err != nil {
				a.stats.IncrementOrbitErrors()
				continue
			}
			if len(cfg.Notifications.PendingScriptExecutionIDs) > 0 {
				// there are pending scripts to execute on this host, start a goroutine
				// that will simulate executing them.
				go a.execScripts(cfg.Notifications.PendingScriptExecutionIDs, orbitClient)
			}
		case <-orbitTokenRemoteCheckTicker:
			if !a.disableFleetDesktop && tokenRotationEnabled {
				if err := deviceClient.CheckToken(*a.deviceAuthToken); err != nil {
					a.stats.IncrementOrbitErrors()
					log.Println("deviceClient.CheckToken: ", err)
					continue
				}
			}
		case <-orbitTokenRotationTicker:
			if !a.disableFleetDesktop && tokenRotationEnabled {
				newToken := ptr.String(uuid.NewString())
				if err := orbitClient.SetOrUpdateDeviceToken(*newToken); err != nil {
					a.stats.IncrementOrbitErrors()
					log.Println("orbitClient.SetOrUpdateDeviceToken: ", err)
					continue
				}
				a.deviceAuthToken = newToken
				// fleet desktop performs a burst of check token requests after a token is rotated
				checkToken()
			}
		case <-capabilitiesCheckerTicker:
			if err := orbitClient.Ping(); err != nil {
				a.stats.IncrementOrbitErrors()
				continue
			}
		case <-fleetDesktopPolicyTicker:
			if !a.disableFleetDesktop {
				if _, err := deviceClient.DesktopSummary(*a.deviceAuthToken); err != nil {
					a.stats.IncrementDesktopErrors()
					log.Println("deviceClient.NumberOfFailingPolicies: ", err)
					continue
				}
			}
		}
	}
}

func (a *agent) runMDMLoop() {
	mdmCheckInTicker := time.Tick(a.MDMCheckInInterval)

	for range mdmCheckInTicker {
		mdmCommandPayload, err := a.mdmClient.Idle()
		if err != nil {
			log.Printf("MDM Idle request failed: %s", err)
			a.stats.IncrementMDMErrors()
			continue
		}
	INNER_FOR_LOOP:
		for mdmCommandPayload != nil {
			a.stats.IncrementMDMCommandsReceived()
			mdmCommandPayload, err = a.mdmClient.Acknowledge(mdmCommandPayload.CommandUUID)
			if err != nil {
				log.Printf("MDM Acknowledge request failed: %s", err)
				a.stats.IncrementMDMErrors()
				break INNER_FOR_LOOP
			}
		}
	}
}

func (a *agent) execScripts(execIDs []string, orbitClient *service.OrbitClient) {
	if a.scriptExecRunning.Swap(true) {
		// if Swap returns true, the goroutine was already running, exit
		return
	}
	defer a.scriptExecRunning.Store(false)

	log.Printf("running scripts: %v", execIDs)
	for _, execID := range execIDs {
		if a.disableScriptExec {
			// send a no-op result without executing if script exec is disabled
			if err := orbitClient.SaveHostScriptResult(&fleet.HostScriptResultPayload{
				ExecutionID: execID,
				Output:      "Scripts are disabled",
				Runtime:     0,
				ExitCode:    -2,
			}); err != nil {
				log.Println("save disabled host script result:", err)
				return
			}
			log.Printf("did save disabled host script result: id=%s", execID)
			continue
		}

		script, err := orbitClient.GetHostScript(execID)
		if err != nil {
			log.Println("get host script:", err)
			return
		}

		// simulate script execution
		outputLen := rand.Intn(11000) // base64 encoding will make the actual output a bit bigger
		buf := make([]byte, outputLen)
		n, _ := io.ReadFull(cryptorand.Reader, buf)
		exitCode := rand.Intn(2)
		runtime := rand.Intn(5)
		time.Sleep(time.Duration(runtime) * time.Second)

		if err := orbitClient.SaveHostScriptResult(&fleet.HostScriptResultPayload{
			HostID:      script.HostID,
			ExecutionID: script.ExecutionID,
			Output:      base64.StdEncoding.EncodeToString(buf[:n]),
			Runtime:     runtime,
			ExitCode:    exitCode,
		}); err != nil {
			log.Println("save host script result:", err)
			return
		}
		log.Printf("did exec and save host script result: id=%s, output size=%d, runtime=%d, exit code=%d", execID, base64.StdEncoding.EncodedLen(n), runtime, exitCode)
	}
}

func (a *agent) waitingDo(request *http.Request) *http.Response {
	response, err := http.DefaultClient.Do(request)
	for err != nil || response.StatusCode != http.StatusOK {
		if err != nil {
			log.Printf("failed to run request: %s", err)
		} else { // res.StatusCode() != http.StatusOK
			log.Printf("request failed: %d", response.StatusCode)
		}
		a.stats.IncrementErrors(1)
		<-time.Tick(time.Duration(rand.Intn(120)+1) * time.Second)
		response, err = http.DefaultClient.Do(request)
	}
	return response
}

// TODO: add support to `alreadyEnrolled` akin to the `enroll` function.  for
// now, we assume that the agent is not already enrolled, if you kill the agent
// process then those Orbit node keys are gone.
func (a *agent) orbitEnroll() error {
	params := service.EnrollOrbitRequest{
		EnrollSecret:   a.EnrollSecret,
		HardwareUUID:   a.UUID,
		HardwareSerial: a.SerialNumber,
	}
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		log.Println("orbit json marshall:", err)
		return err
	}

	request, err := http.NewRequest("POST", a.serverAddress+"/api/fleet/orbit/enroll", bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}
	request.Header.Add("Content-type", "application/json")

	response := a.waitingDo(request)
	defer response.Body.Close()

	var parsedResp service.EnrollOrbitResponse
	if err := json.NewDecoder(response.Body).Decode(&parsedResp); err != nil {
		log.Println("orbit json parse:", err)
		return err
	}

	a.orbitNodeKey = &parsedResp.OrbitNodeKey
	a.stats.IncrementOrbitEnrollments()
	return nil
}

func (a *agent) enroll(i int, onlyAlreadyEnrolled bool) error {
	a.nodeKey = a.nodeKeyManager.Get(i)
	if a.nodeKey != "" {
		a.stats.IncrementEnrollments()
		return nil
	}

	if onlyAlreadyEnrolled {
		return errors.New("not enrolled")
	}

	var body bytes.Buffer
	if err := a.templates.ExecuteTemplate(&body, "enroll", a); err != nil {
		log.Println("execute template:", err)
		return err
	}

	request, err := http.NewRequest("POST", a.serverAddress+"/api/osquery/enroll", &body)
	if err != nil {
		return err
	}
	request.Header.Add("Content-type", "application/json")

	response := a.waitingDo(request)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Println("enroll status:", response.StatusCode)
		return fmt.Errorf("status code: %d", response.StatusCode)
	}

	var parsedResp enrollResponse
	if err := json.NewDecoder(response.Body).Decode(&parsedResp); err != nil {
		log.Println("json parse:", err)
		return err
	}

	a.nodeKey = parsedResp.NodeKey
	a.stats.IncrementEnrollments()

	a.nodeKeyManager.Add(a.nodeKey)

	return nil
}

func (a *agent) config() error {
	request, err := http.NewRequest("POST", a.serverAddress+"/api/osquery/config", bytes.NewReader([]byte(`{"node_key": "`+a.nodeKey+`"}`)))
	if err != nil {
		return err
	}
	request.Header.Add("Content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("config request failed to run: %w", err)
	}
	defer response.Body.Close()

	a.stats.IncrementConfigRequests()

	statusCode := response.StatusCode
	if statusCode != http.StatusOK {
		a.stats.IncrementConfigErrors()
		return fmt.Errorf("config request failed: %d", statusCode)
	}

	parsedResp := struct {
		Packs map[string]struct {
			Queries map[string]interface{} `json:"queries"`
		} `json:"packs"`
	}{}
	if err := json.NewDecoder(response.Body).Decode(&parsedResp); err != nil {
		return fmt.Errorf("json parse at config: %w", err)
	}

	var scheduledQueries []string
	var scheduledQueryData []scheduledQuery
	for packName, pack := range parsedResp.Packs {
		for queryName, query := range pack.Queries {
			scheduledQueries = append(scheduledQueries, packName+"_"+queryName)
			m, ok := query.(map[string]interface{})
			if !ok {
				return fmt.Errorf("processing scheduled query failed: %v", query)
			}

			q := scheduledQuery{}
			q.packName = packName
			q.Name = queryName
			q.numRows = 1
			parts := strings.Split(q.Name, "_")
			if len(parts) == 2 {
				num, err := strconv.ParseInt(parts[1], 10, 32)
				if err != nil {
					num = 1
				}
				q.numRows = uint(num)
			}
			q.ScheduleInterval = m["interval"].(float64)
			q.Query = m["query"].(string)
			scheduledQueryData = append(scheduledQueryData, q)
		}
	}

	a.scheduledQueriesMu.Lock()
	a.scheduledQueries = scheduledQueries
	a.scheduledQueryData = scheduledQueryData
	a.scheduledQueriesMu.Unlock()

	return nil
}

const stringVals = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_."

func randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(stringVals[rand.Int63()%int64(len(stringVals))])
	}
	return sb.String()
}

func (a *agent) CachedString(key string) string {
	if val, ok := a.strings[key]; ok {
		return val
	}
	val := randomString(12)
	a.strings[key] = val
	return val
}

func (a *agent) hostUsers() []map[string]string {
	groupNames := []string{"staff", "nobody", "wheel", "tty", "daemon"}
	shells := []string{"/bin/zsh", "/bin/sh", "/usr/bin/false", "/bin/bash"}
	commonUsers := make([]map[string]string, a.userCount.common)
	for i := 0; i < len(commonUsers); i++ {
		commonUsers[i] = map[string]string{
			"uid":       fmt.Sprint(i),
			"username":  fmt.Sprintf("Common_%d", i),
			"type":      "", // Empty for macOS.
			"groupname": groupNames[i%len(groupNames)],
			"shell":     shells[i%len(shells)],
		}
	}
	uniqueUsers := make([]map[string]string, a.userCount.unique)
	for i := 0; i < len(uniqueUsers); i++ {
		uniqueUsers[i] = map[string]string{
			"uid":       fmt.Sprint(i),
			"username":  fmt.Sprintf("Unique_%d_%d", a.agentIndex, i),
			"type":      "", // Empty for macOS.
			"groupname": groupNames[i%len(groupNames)],
			"shell":     shells[i%len(shells)],
		}
	}
	users := append(commonUsers, uniqueUsers...)
	rand.Shuffle(len(users), func(i, j int) {
		users[i], users[j] = users[j], users[i]
	})
	return users
}

func (a *agent) softwareMacOS() []map[string]string {
	var lastOpenedCount int
	commonSoftware := make([]map[string]string, a.softwareCount.common)
	for i := 0; i < len(commonSoftware); i++ {
		var lastOpenedAt string
		if l := a.genLastOpenedAt(&lastOpenedCount); l != nil {
			lastOpenedAt = l.Format(time.UnixDate)
		}
		commonSoftware[i] = map[string]string{
			"name":              fmt.Sprintf("Common_%d", i),
			"version":           "0.0.1",
			"bundle_identifier": "com.fleetdm.osquery-perf",
			"source":            "osquery-perf",
			"last_opened_at":    lastOpenedAt,
			"installed_path":    fmt.Sprintf("/some/path/Common_%d", i),
		}
	}
	if a.softwareCount.commonSoftwareUninstallProb > 0.0 && rand.Float64() <= a.softwareCount.commonSoftwareUninstallProb {
		rand.Shuffle(len(commonSoftware), func(i, j int) {
			commonSoftware[i], commonSoftware[j] = commonSoftware[j], commonSoftware[i]
		})
		commonSoftware = commonSoftware[:a.softwareCount.common-a.softwareCount.commonSoftwareUninstallCount]
	}
	uniqueSoftware := make([]map[string]string, a.softwareCount.unique)
	for i := 0; i < len(uniqueSoftware); i++ {
		var lastOpenedAt string
		if l := a.genLastOpenedAt(&lastOpenedCount); l != nil {
			lastOpenedAt = l.Format(time.UnixDate)
		}
		uniqueSoftware[i] = map[string]string{
			"name":              fmt.Sprintf("Unique_%s_%d", a.CachedString("hostname"), i),
			"version":           "1.1.1",
			"bundle_identifier": "com.fleetdm.osquery-perf",
			"source":            "osquery-perf",
			"last_opened_at":    lastOpenedAt,
			"installed_path":    fmt.Sprintf("/some/path/Unique_%s_%d", a.CachedString("hostname"), i),
		}
	}
	if a.softwareCount.uniqueSoftwareUninstallProb > 0.0 && rand.Float64() <= a.softwareCount.uniqueSoftwareUninstallProb {
		rand.Shuffle(len(uniqueSoftware), func(i, j int) {
			uniqueSoftware[i], uniqueSoftware[j] = uniqueSoftware[j], uniqueSoftware[i]
		})
		uniqueSoftware = uniqueSoftware[:a.softwareCount.unique-a.softwareCount.uniqueSoftwareUninstallCount]
	}
	randomVulnerableSoftware := make([]map[string]string, a.softwareCount.vulnerable)
	for i := 0; i < len(randomVulnerableSoftware); i++ {
		sw := macosVulnerableSoftware[rand.Intn(len(macosVulnerableSoftware))]
		var lastOpenedAt string
		if l := a.genLastOpenedAt(&lastOpenedCount); l != nil {
			lastOpenedAt = l.Format(time.UnixDate)
		}
		randomVulnerableSoftware[i] = map[string]string{
			"name":              sw.Name,
			"version":           sw.Version,
			"bundle_identifier": sw.BundleIdentifier,
			"source":            sw.Source,
			"last_opened_at":    lastOpenedAt,
			"installed_path":    fmt.Sprintf("/some/path/%s", sw.Name),
		}
	}
	software := append(commonSoftware, uniqueSoftware...)
	software = append(software, randomVulnerableSoftware...)
	rand.Shuffle(len(software), func(i, j int) {
		software[i], software[j] = software[j], software[i]
	})
	return software
}

func (a *agent) DistributedRead() (*distributedReadResponse, error) {
	request, err := http.NewRequest("POST", a.serverAddress+"/api/osquery/distributed/read", bytes.NewReader([]byte(`{"node_key": "`+a.nodeKey+`"}`)))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("distributed/read request failed to run: %w", err)
	}
	defer response.Body.Close()

	a.stats.IncrementDistributedReads()

	statusCode := response.StatusCode
	if statusCode != http.StatusOK {
		a.stats.IncrementDistributedReadErrors()
		return nil, fmt.Errorf("distributed/read request failed: %d", statusCode)
	}

	var parsedResp distributedReadResponse
	if err := json.NewDecoder(response.Body).Decode(&parsedResp); err != nil {
		log.Printf("json parse: %s", err)
		return nil, err
	}

	return &parsedResp, nil
}

var defaultQueryResult = []map[string]string{
	{"foo": "bar"},
}

func (a *agent) genLastOpenedAt(count *int) *time.Time {
	if *count >= a.softwareCount.withLastOpened {
		return nil
	}
	*count++
	if rand.Float64() <= a.softwareCount.lastOpenedProb {
		now := time.Now()
		return &now
	}
	return nil
}

func (a *agent) runPolicy(query string) []map[string]string {
	if rand.Float64() <= a.policyPassProb {
		return []map[string]string{
			{"1": "1"},
		}
	}
	return []map[string]string{}
}

func (a *agent) randomQueryStats() []map[string]string {
	a.scheduledQueriesMu.Lock()
	defer a.scheduledQueriesMu.Unlock()

	var stats []map[string]string
	for _, scheduledQuery := range a.scheduledQueries {
		stats = append(stats, map[string]string{
			"name":           scheduledQuery,
			"delimiter":      "_",
			"average_memory": fmt.Sprint(rand.Intn(200) + 10),
			"denylisted":     "false",
			"executions":     fmt.Sprint(rand.Intn(100) + 1),
			"interval":       fmt.Sprint(rand.Intn(100) + 1),
			"last_executed":  fmt.Sprint(time.Now().Unix()),
			"output_size":    fmt.Sprint(rand.Intn(100) + 1),
			"system_time":    fmt.Sprint(rand.Intn(4000) + 10),
			"user_time":      fmt.Sprint(rand.Intn(4000) + 10),
			"wall_time":      fmt.Sprint(rand.Intn(4) + 1),
			"wall_time_ms":   fmt.Sprint(rand.Intn(4000) + 10),
		})
	}
	return stats
}

var possibleMDMServerURLs = []string{
	"https://kandji.com/1",
	"https://jamf.com/1",
	"https://airwatch.com/1",
	"https://microsoft.com/1",
	"https://simplemdm.com/1",
	"https://example.com/1",
	"https://kandji.com/2",
	"https://jamf.com/2",
	"https://airwatch.com/2",
	"https://microsoft.com/2",
	"https://simplemdm.com/2",
	"https://example.com/2",
}

// mdmMac returns the results for the `mdm` table query.
//
// If the host is enrolled via MDM it will return installed_from_dep as false
// (which means the host will be identified as manually enrolled).
//
// NOTE: To support proper DEP simulation in a loadtest environment
// we may need to implement a mocked Apple DEP endpoint.
func (a *agent) mdmMac() []map[string]string {
	if !a.mdmEnrolled() {
		return []map[string]string{
			{"enrolled": "false", "server_url": "", "installed_from_dep": "false"},
		}
	}
	return []map[string]string{
		{"enrolled": "true", "server_url": a.mdmClient.EnrollInfo.MDMURL, "installed_from_dep": "false"},
	}
}

func (a *agent) mdmEnrolled() bool {
	a.isEnrolledToMDMMu.Lock()
	defer a.isEnrolledToMDMMu.Unlock()

	return a.isEnrolledToMDM
}

func (a *agent) setMDMEnrolled() {
	a.isEnrolledToMDMMu.Lock()
	defer a.isEnrolledToMDMMu.Unlock()

	a.isEnrolledToMDM = true
}

func (a *agent) mdmWindows() []map[string]string {
	autopilot := rand.Intn(2) == 1
	ix := rand.Intn(len(possibleMDMServerURLs))
	serverURL := possibleMDMServerURLs[ix]
	providerID := fleet.MDMNameFromServerURL(serverURL)
	installType := "Microsoft Workstation"
	if rand.Intn(4) == 1 {
		installType = "Microsoft Server"
	}

	rows := []map[string]string{
		{"key": "discovery_service_url", "value": serverURL},
		{"key": "installation_type", "value": installType},
	}
	if providerID != "" {
		rows = append(rows, map[string]string{"key": "provider_id", "value": providerID})
	}
	if autopilot {
		rows = append(rows, map[string]string{"key": "autopilot", "value": "true"})
	}
	return rows
}

var munkiIssues = func() []string {
	// generate a list of random munki issues (messages)
	issues := make([]string, 1000)
	for i := range issues {
		// message size: between 60 and 200, with spaces between each 10-char word so
		// that it can still make a bit of sense for UI tests.
		numParts := rand.Intn(15) + 6 // number between 0-14, add 6 to get between 6-20
		var sb strings.Builder
		for j := 0; j < numParts; j++ {
			if j > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(randomString(10))
		}
		issues[i] = sb.String()
	}
	return issues
}()

func (a *agent) munkiInfo() []map[string]string {
	var errors, warnings []string

	if rand.Float64() <= a.munkiIssueProb {
		for i := 0; i < a.munkiIssueCount; i++ {
			if rand.Intn(2) == 1 {
				errors = append(errors, munkiIssues[rand.Intn(len(munkiIssues))])
			} else {
				warnings = append(warnings, munkiIssues[rand.Intn(len(munkiIssues))])
			}
		}
	}

	errList := strings.Join(errors, ";")
	warnList := strings.Join(warnings, ";")
	return []map[string]string{
		{"version": "1.2.3", "errors": errList, "warnings": warnList},
	}
}

func (a *agent) googleChromeProfiles() []map[string]string {
	count := rand.Intn(5) // return between 0 and 4 emails
	result := make([]map[string]string, count)
	for i := range result {
		email := fmt.Sprintf("user%d@example.com", i)
		if i == len(result)-1 {
			// if the maximum number of emails is returned, set a random domain name
			// so that we have email addresses that match a lot of hosts, and some
			// that match few hosts.
			domainRand := rand.Intn(10)
			email = fmt.Sprintf("user%d@example%d.com", i, domainRand)
		}
		result[i] = map[string]string{"email": email}
	}
	return result
}

func (a *agent) batteries() []map[string]string {
	count := rand.Intn(3) // return between 0 and 2 batteries
	result := make([]map[string]string, count)
	for i := range result {
		health := "Good"
		cycleCount := rand.Intn(2000)
		switch {
		case cycleCount > 1500:
			health = "Poor"
		case cycleCount > 1000:
			health = "Fair"
		}
		result[i] = map[string]string{
			"serial_number": fmt.Sprintf("%04d", i),
			"cycle_count":   strconv.Itoa(cycleCount),
			"health":        health,
		}
	}
	return result
}

func (a *agent) diskSpace() []map[string]string {
	// between 1-100 gigs, between 0-99 percentage available
	gigs := rand.Intn(100)
	gigs++
	pct := rand.Intn(100)
	available := gigs * pct / 100
	return []map[string]string{
		{
			"percent_disk_space_available": strconv.Itoa(pct),
			"gigs_disk_space_available":    strconv.Itoa(available),
			"gigs_total_disk_space":        strconv.Itoa(gigs),
		},
	}
}

func (a *agent) diskEncryption() []map[string]string {
	// 50% of results have encryption enabled
	a.DiskEncryptionEnabled = rand.Intn(2) == 1
	if a.DiskEncryptionEnabled {
		return []map[string]string{{"1": "1"}}
	}
	return []map[string]string{}
}

func (a *agent) diskEncryptionLinux() []map[string]string {
	// 50% of results have encryption enabled
	a.DiskEncryptionEnabled = rand.Intn(2) == 1
	if a.DiskEncryptionEnabled {
		return []map[string]string{
			{"path": "/etc", "encrypted": "0"},
			{"path": "/tmp", "encrypted": "0"},
			{"path": "/", "encrypted": "1"},
		}
	}
	return []map[string]string{
		{"path": "/etc", "encrypted": "0"},
		{"path": "/tmp", "encrypted": "0"},
	}
}

func (a *agent) runLiveQuery(query string) (results []map[string]string, status *fleet.OsqueryStatus, message *string, stats *fleet.Stats) {
	if a.liveQueryFailProb > 0.0 && rand.Float64() <= a.liveQueryFailProb {
		ss := fleet.OsqueryStatus(1)
		return []map[string]string{}, &ss, ptr.String("live query failed with error foobar"), nil
	}
	ss := fleet.OsqueryStatus(0)
	if a.liveQueryNoResultsProb > 0.0 && rand.Float64() <= a.liveQueryNoResultsProb {
		return []map[string]string{}, &ss, nil, nil
	}
	return []map[string]string{
			{
				"admindir":   "/var/lib/dpkg",
				"arch":       "amd64",
				"maintainer": "foobar",
				"name":       "netconf",
				"priority":   "optional",
				"revision":   "",
				"section":    "default",
				"size":       "112594",
				"source":     "",
				"status":     "install ok installed",
				"version":    "20230224000000",
			},
		}, &ss, nil, &fleet.Stats{
			WallTimeMs: uint64(rand.Intn(1000) * 1000),
			UserTime:   uint64(rand.Intn(1000)),
			SystemTime: uint64(rand.Intn(1000)),
			Memory:     uint64(rand.Intn(1000)),
		}
}

func (a *agent) processQuery(name, query string) (
	handled bool, results []map[string]string, status *fleet.OsqueryStatus, message *string, stats *fleet.Stats,
) {
	const (
		hostPolicyQueryPrefix = "fleet_policy_query_"
		hostDetailQueryPrefix = "fleet_detail_query_"
		liveQueryPrefix       = "fleet_distributed_query_"
	)
	statusOK := fleet.StatusOK
	statusNotOK := fleet.OsqueryStatus(1)

	switch {
	case strings.HasPrefix(name, liveQueryPrefix):
		results, status, message, stats = a.runLiveQuery(query)
		return true, results, status, message, stats
	case strings.HasPrefix(name, hostPolicyQueryPrefix):
		return true, a.runPolicy(query), &statusOK, nil, nil
	case name == hostDetailQueryPrefix+"scheduled_query_stats":
		return true, a.randomQueryStats(), &statusOK, nil, nil
	case name == hostDetailQueryPrefix+"mdm":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.mdmMac()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"mdm_windows":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.mdmWindows()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"munki_info":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.munkiInfo()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"google_chrome_profiles":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.googleChromeProfiles()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"battery":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.batteries()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"users":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.hostUsers()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"software_macos":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.softwareMacOS()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"software_windows":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = windowsSoftware
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"software_linux":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			switch a.os {
			case "ubuntu_22.04":
				results = ubuntuSoftware
			}
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"disk_space_unix" || name == hostDetailQueryPrefix+"disk_space_windows":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.diskSpace()
		}
		return true, results, &ss, nil, nil

	case strings.HasPrefix(name, hostDetailQueryPrefix+"disk_encryption_linux"):
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.diskEncryptionLinux()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"disk_encryption_darwin" ||
		name == hostDetailQueryPrefix+"disk_encryption_windows":
		ss := fleet.OsqueryStatus(rand.Intn(2))
		if ss == fleet.StatusOK {
			results = a.diskEncryption()
		}
		return true, results, &ss, nil, nil
	case name == hostDetailQueryPrefix+"kubequery_info" && a.os != "kubequery":
		// Real osquery running on hosts would return no results if it was not
		// running kubequery (due to discovery query). Returning true here so that
		// the caller knows it is handled, will not try to return lorem-ipsum-style
		// results.
		return true, nil, &statusNotOK, nil, nil
	default:
		// Look for results in the template file.
		if t := a.templates.Lookup(name); t == nil {
			return false, nil, nil, nil, nil
		}
		var ni bytes.Buffer
		err := a.templates.ExecuteTemplate(&ni, name, a)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(ni.Bytes(), &results)
		if err != nil {
			panic(err)
		}

		return true, results, &statusOK, nil, nil
	}
}

func (a *agent) DistributedWrite(queries map[string]string) error {
	r := service.SubmitDistributedQueryResultsRequest{
		Results:  make(fleet.OsqueryDistributedQueryResults),
		Statuses: make(map[string]fleet.OsqueryStatus),
		Messages: make(map[string]string),
		Stats:    make(map[string]*fleet.Stats),
	}
	r.NodeKey = a.nodeKey
	for name, query := range queries {
		handled, results, status, message, stats := a.processQuery(name, query)
		if !handled {
			// If osquery-perf does not handle the incoming query,
			// always return status OK and the default query result.
			r.Results[name] = defaultQueryResult
			r.Statuses[name] = fleet.StatusOK
		} else {
			if results != nil {
				r.Results[name] = results
			}
			if status != nil {
				r.Statuses[name] = *status
			}
			if message != nil {
				r.Messages[name] = *message
			}
			if stats != nil {
				r.Stats[name] = stats
			}
		}
	}
	body, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", a.serverAddress+"/api/osquery/distributed/write", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("distributed/write request failed to run: %w", err)
	}
	defer response.Body.Close()

	a.stats.IncrementDistributedWrites()

	statusCode := response.StatusCode
	if statusCode != http.StatusOK {
		a.stats.IncrementDistributedWriteErrors()
		return fmt.Errorf("distributed/write request failed: %d", statusCode)
	}

	// No need to read the distributed write body
	return nil
}

func scheduledQueryResults(packName, queryName string, numResults int) json.RawMessage {
	return json.RawMessage(`{
  "snapshot": [` + rows(numResults) + `
  ],
  "action": "snapshot",
  "name": "pack/` + packName + `/` + queryName + `",
  "hostIdentifier": "EF9595F0-CE81-493A-9B06-D8A9D2CCB952",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "osquery-perf"
  }
}`)
}

func (a *agent) submitLogs(results []json.RawMessage) error {
	// Connection check to prevent unnecessary JSON marshaling when the server is down.
	conn, err := net.Dial("tcp", strings.TrimPrefix(a.serverAddress, "https://"))
	if err != nil {
		return err
	}
	conn.Close()

	jsonResults, err := json.Marshal(results)
	if err != nil {
		panic(err)
	}
	type submitLogsRequest struct {
		NodeKey string          `json:"node_key"`
		LogType string          `json:"log_type"`
		Data    json.RawMessage `json:"data"`
	}
	slr := submitLogsRequest{
		NodeKey: a.nodeKey,
		LogType: "result",
		Data:    jsonResults,
	}
	body, err := json.Marshal(slr)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", a.serverAddress+"/api/osquery/log", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Add("Content-type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("log request failed to run: %w", err)
	}
	defer response.Body.Close()

	a.stats.IncrementResultLogRequests()

	statusCode := response.StatusCode
	if statusCode != http.StatusOK {
		a.stats.IncrementResultLogErrors()
		return fmt.Errorf("log request failed: %d", statusCode)
	}

	return nil
}

// rows returns a set of rows for use in tests for query results.
func rows(num int) string {
	b := strings.Builder{}
	for i := 0; i < num; i++ {
		b.WriteString(`    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "EF9595F0-CE81-493A-9B06-D8A9D2CCB95",
      "version": "5.9.2",
      "watcher": "3570"
    }`)
		if i != num-1 {
			b.WriteString(",")
		}
	}

	return b.String()
}

func main() {
	// Start HTTP server for pprof. See https://pkg.go.dev/net/http/pprof.
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// #nosec (osquery-perf is only used for testing)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = tlsConfig
	http.DefaultClient.Transport = tr

	validTemplateNames := map[string]bool{
		"macos_13.6.2.tmpl":         true,
		"macos_14.1.2.tmpl":         true,
		"windows_11.tmpl":           true,
		"windows_11_22H2_2861.tmpl": true,
		"windows_11_22H2_3007.tmpl": true,
		"ubuntu_22.04.tmpl":         true,
	}
	allowedTemplateNames := make([]string, 0, len(validTemplateNames))
	for k := range validTemplateNames {
		allowedTemplateNames = append(allowedTemplateNames, k)
	}

	var (
		serverURL      = flag.String("server_url", "https://localhost:8080", "URL (with protocol and port of osquery server)")
		enrollSecret   = flag.String("enroll_secret", "", "Enroll secret to authenticate enrollment")
		hostCount      = flag.Int("host_count", 10, "Number of hosts to start (default 10)")
		randSeed       = flag.Int64("seed", time.Now().UnixNano(), "Seed for random generator (default current time)")
		startPeriod    = flag.Duration("start_period", 10*time.Second, "Duration to spread start of hosts over")
		configInterval = flag.Duration("config_interval", 1*time.Minute, "Interval for config requests")
		// Flag logger_tls_period defines how often to check for sending scheduled query results.
		// osquery-perf will send log requests with results only if there are scheduled queries configured AND it's their time to run.
		logInterval         = flag.Duration("logger_tls_period", 10*time.Second, "Interval for scheduled queries log requests")
		queryInterval       = flag.Duration("query_interval", 10*time.Second, "Interval for live query requests")
		mdmCheckInInterval  = flag.Duration("mdm_check_in_interval", 10*time.Second, "Interval for performing MDM check ins")
		onlyAlreadyEnrolled = flag.Bool("only_already_enrolled", false, "Only start agents that are already enrolled")
		nodeKeyFile         = flag.String("node_key_file", "", "File with node keys to use")

		commonSoftwareCount          = flag.Int("common_software_count", 10, "Number of common installed applications reported to fleet")
		commonSoftwareUninstallCount = flag.Int("common_software_uninstall_count", 1, "Number of common software to uninstall")
		commonSoftwareUninstallProb  = flag.Float64("common_software_uninstall_prob", 0.1, "Probability of uninstalling common_software_uninstall_count unique software/s")

		uniqueSoftwareCount          = flag.Int("unique_software_count", 1, "Number of unique software installed on each host")
		uniqueSoftwareUninstallCount = flag.Int("unique_software_uninstall_count", 1, "Number of unique software to uninstall")
		uniqueSoftwareUninstallProb  = flag.Float64("unique_software_uninstall_prob", 0.1, "Probability of uninstalling unique_software_uninstall_count common software/s")

		vulnerableSoftwareCount     = flag.Int("vulnerable_software_count", 10, "Number of vulnerable installed applications reported to fleet")
		withLastOpenedSoftwareCount = flag.Int("with_last_opened_software_count", 10, "Number of applications that may report a last opened timestamp to fleet")
		lastOpenedChangeProb        = flag.Float64("last_opened_change_prob", 0.1, "Probability of last opened timestamp to be reported as changed [0, 1]")
		commonUserCount             = flag.Int("common_user_count", 10, "Number of common host users reported to fleet")
		uniqueUserCount             = flag.Int("unique_user_count", 10, "Number of unique host users reported to fleet")
		policyPassProb              = flag.Float64("policy_pass_prob", 1.0, "Probability of policies to pass [0, 1]")
		orbitProb                   = flag.Float64("orbit_prob", 0.5, "Probability of a host being identified as orbit install [0, 1]")
		munkiIssueProb              = flag.Float64("munki_issue_prob", 0.5, "Probability of a host having munki issues (note that ~50% of hosts have munki installed) [0, 1]")
		munkiIssueCount             = flag.Int("munki_issue_count", 10, "Number of munki issues reported by hosts identified to have munki issues")
		// E.g. when running with `-host_count=10`, you can set host count for each template the following way:
		// `-os_templates=windows_11.tmpl:3,macos_14.1.2.tmpl:4,ubuntu_22.04.tmpl:3`
		osTemplates     = flag.String("os_templates", "macos_14.1.2", fmt.Sprintf("Comma separated list of host OS templates to use and optionally their host count separated by ':' (any of %v, with or without the .tmpl extension)", allowedTemplateNames))
		emptySerialProb = flag.Float64("empty_serial_prob", 0.1, "Probability of a host having no serial number [0, 1]")

		mdmProb          = flag.Float64("mdm_prob", 0.0, "Probability of a host enrolling via MDM (for macOS) [0, 1]")
		mdmSCEPChallenge = flag.String("mdm_scep_challenge", "", "SCEP challenge to use when running MDM enroll")

		liveQueryFailProb      = flag.Float64("live_query_fail_prob", 0.0, "Probability of a live query failing execution in the host")
		liveQueryNoResultsProb = flag.Float64("live_query_no_results_prob", 0.2, "Probability of a live query returning no results")

		disableScriptExec = flag.Bool("disable_script_exec", false, "Disable script execution support")

		disableFleetDesktop = flag.Bool("disable_fleet_desktop", false, "Disable Fleet Desktop")
		// logger_tls_max_lines is simulating the osquery setting with the same name.
		loggerTLSMaxLines = flag.Int("", 1024, "Maximum number of buffered result log lines to send on every log request")
	)

	flag.Parse()
	rand.Seed(*randSeed)

	if *onlyAlreadyEnrolled {
		// Orbit enrollment does not support the "already enrolled" mode at the
		// moment (see TODO in this file).
		*orbitProb = 0
	}

	if *commonSoftwareUninstallCount > *commonSoftwareCount {
		log.Fatalf("Argument common_software_uninstall_count cannot be bigger than common_software_count")
	}
	if *uniqueSoftwareUninstallCount > *uniqueSoftwareCount {
		log.Fatalf("Argument unique_software_uninstall_count cannot be bigger than unique_software_count")
	}

	tmplsm := make(map[*template.Template]int)
	requestedTemplates := strings.Split(*osTemplates, ",")
	tmplsTotalHostCount := 0
	for _, nm := range requestedTemplates {
		numberOfHosts := 0
		if strings.Contains(nm, ":") {
			parts := strings.Split(nm, ":")
			nm = parts[0]
			hc, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				log.Fatalf("Invalid template host count: %s", parts[1])
			}
			numberOfHosts = int(hc)
		}
		if !strings.HasSuffix(nm, ".tmpl") {
			nm += ".tmpl"
		}
		if !validTemplateNames[nm] {
			log.Fatalf("Invalid template name: %s (accepted values: %v)", nm, allowedTemplateNames)
		}

		tmpl, err := template.ParseFS(templatesFS, nm)
		if err != nil {
			log.Fatal("parse templates: ", err)
		}
		tmplsm[tmpl] = numberOfHosts
		tmplsTotalHostCount += numberOfHosts
	}
	if tmplsTotalHostCount != 0 && tmplsTotalHostCount != *hostCount {
		log.Fatalf("Invalid host count in templates: total=%d vs host_count=%d", tmplsTotalHostCount, *hostCount)
	}

	// Spread starts over the interval to prevent thundering herd
	sleepTime := *startPeriod / time.Duration(*hostCount)

	stats := &Stats{
		startTime: time.Now(),
	}
	go stats.runLoop()

	nodeKeyManager := &nodeKeyManager{}
	if nodeKeyFile != nil {
		nodeKeyManager.filepath = *nodeKeyFile
		nodeKeyManager.LoadKeys()
	}

	var tmplss []*template.Template
	for tmpl := range tmplsm {
		tmplss = append(tmplss, tmpl)
	}

	for i := 0; i < *hostCount; i++ {
		var tmpl *template.Template
		if tmplsTotalHostCount > 0 {
			for tmpl_, hostCount := range tmplsm {
				if hostCount > 0 {
					tmpl = tmpl_
					tmplsm[tmpl_] = tmplsm[tmpl_] - 1
					break
				}
			}
			if tmpl == nil {
				log.Fatalf("Failed to determine template for host: %d", i)
			}
		} else {
			tmpl = tmplss[i%len(tmplss)]
		}

		a := newAgent(i+1,
			*serverURL,
			*enrollSecret,
			tmpl,
			*configInterval,
			*logInterval,
			*queryInterval,
			*mdmCheckInInterval,
			softwareEntityCount{
				entityCount: entityCount{
					common: *commonSoftwareCount,
					unique: *uniqueSoftwareCount,
				},
				vulnerable:                   *vulnerableSoftwareCount,
				withLastOpened:               *withLastOpenedSoftwareCount,
				lastOpenedProb:               *lastOpenedChangeProb,
				commonSoftwareUninstallCount: *commonSoftwareUninstallCount,
				commonSoftwareUninstallProb:  *commonSoftwareUninstallProb,
				uniqueSoftwareUninstallCount: *uniqueSoftwareUninstallCount,
				uniqueSoftwareUninstallProb:  *uniqueSoftwareUninstallProb,
			}, entityCount{
				common: *commonUserCount,
				unique: *uniqueUserCount,
			},
			*policyPassProb,
			*orbitProb,
			*munkiIssueProb,
			*munkiIssueCount,
			*emptySerialProb,
			*mdmProb,
			*mdmSCEPChallenge,
			*liveQueryFailProb,
			*liveQueryNoResultsProb,
			*disableScriptExec,
			*disableFleetDesktop,
			*loggerTLSMaxLines,
		)
		a.stats = stats
		a.nodeKeyManager = nodeKeyManager
		go a.runLoop(i, *onlyAlreadyEnrolled)
		time.Sleep(sleepTime)
	}

	log.Println("Agents running. Kill with C-c.")
	<-make(chan struct{})
}
