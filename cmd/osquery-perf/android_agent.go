package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/osquery_perf"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/google/uuid"
	"google.golang.org/api/androidmanagement/v1"
)

// androidAgent simulates a single Android device for load testing.
// It communicates with Fleet via PubSub messages (enrollment, status reports, command acks)
// and coordinates with a mock AMAPI proxy to get policy versions and pending commands.
type androidAgent struct {
	agentIndex    int
	serverAddress string
	enrollSecret  string
	pubSubToken   string
	proxyAddress  string
	stats         *osquery_perf.Stats

	// Device identity (stable across the agent lifetime)
	enterpriseSpecificID string
	serialNumber         string
	deviceName           string // AMAPI resource name: enterprises/{id}/devices/{id}
	enterpriseID         string

	// Hardware details
	brand    string
	model    string
	hardware string

	// Software
	androidVersion     string
	androidBuildNumber string

	// Memory
	totalRAM             int64
	totalInternalStorage int64

	// Installed apps reported in STATUS_REPORT
	installedApps []*androidmanagement.ApplicationReport

	// Timing
	statusReportInterval time.Duration

	// Non-compliance probability (fraction of STATUS_REPORTs that include non-compliance details)
	nonComplianceProb float64
}

// androidApp is a simplified app definition for generating realistic ApplicationReports.
var androidApps = []struct {
	displayName string
	packageName string
	baseVersion string
}{
	{"Google Chrome", "com.android.chrome", "126.0.6478.122"},
	{"Gmail", "com.google.android.gm", "2024.06.30.649015803"},
	{"Google Maps", "com.google.android.apps.maps", "11.125.0102"},
	{"YouTube", "com.google.android.youtube", "19.25.33"},
	{"Google Drive", "com.google.android.apps.docs", "2.24.277.0"},
	{"Google Photos", "com.google.android.apps.photos", "7.1.0.611579560"},
	{"Google Calendar", "com.google.android.calendar", "2024.25.0-647498253"},
	{"Google Meet", "com.google.android.apps.tachyon", "2024.06.30.643793517"},
	{"Slack", "com.Slack", "24.06.10.0"},
	{"Microsoft Teams", "com.microsoft.teams", "1416/1.0.0.2024063002"},
	{"Microsoft Outlook", "com.microsoft.office.outlook", "4.2425.1"},
	{"Zoom", "us.zoom.videomeetings", "6.1.1.21782"},
	{"Salesforce", "com.salesforce.chatter", "246.010.0"},
	{"1Password", "com.onepassword.android", "8.10.38"},
	{"Authenticator", "com.google.android.apps.authenticator2", "7.0"},
	{"Google Docs", "com.google.android.apps.docs.editors.docs", "1.24.272.01"},
	{"Google Sheets", "com.google.android.apps.docs.editors.sheets", "1.24.272.01"},
	{"Google Slides", "com.google.android.apps.docs.editors.slides", "1.24.272.01"},
	{"Google Keep", "com.google.android.keep", "5.24.272.00"},
	{"Google Messages", "com.google.android.apps.messaging", "20240625"},
	{"Files by Google", "com.google.android.apps.nbu.files", "1.4396.621459950"},
	{"Google Phone", "com.google.android.dialer", "130.0.631022283"},
	{"Google Contacts", "com.google.android.contacts", "4.32.33.621636488"},
	{"Google Clock", "com.google.android.deskclock", "7.8"},
	{"Google Calculator", "com.google.android.calculator", "8.8"},
	{"Google Camera", "com.google.android.GoogleCamera", "9.3.160.621982096"},
	{"Google Play Store", "com.android.vending", "41.6.26"},
	{"Google Play Services", "com.google.android.gms", "24.26.14"},
	{"Android System WebView", "com.google.android.webview", "126.0.6478.122"},
	{"Google Translate", "com.google.android.apps.translate", "8.7.29.626714160"},
	{"LinkedIn", "com.linkedin.android", "4.1.972"},
	{"Spotify", "com.spotify.music", "8.9.42.575"},
	{"WhatsApp", "com.whatsapp", "2.24.14.78"},
	{"Signal", "org.thoughtcrime.securesms", "7.11.3"},
	{"Firefox", "org.mozilla.firefox", "127.0.2"},
	{"Adobe Acrobat", "com.adobe.reader", "24.6.0.33768"},
	{"Dropbox", "com.dropbox.android", "372.2.2"},
	{"Evernote", "com.evernote", "10.95"},
	{"Trello", "com.trello", "2024.10"},
	{"Notion", "notion.id", "0.6.2413"},
	{"GitHub", "com.github.android", "1.148.0"},
	{"Jira Cloud", "com.atlassian.android.jira.core", "2024.06.30"},
	{"Okta Verify", "com.okta.android.auth", "9.6.1"},
	{"Duo Mobile", "com.duosecurity.duomobile", "4.62.0"},
	{"CrowdStrike Falcon", "com.crowdstrike.android.falcon", "7.19.17004"},
	{"Intune Company Portal", "com.microsoft.windowsintune.companyportal", "5.0.6233.0"},
	{"Fleet Agent", "com.fleetdm.agent", "1.3.0"},
	{"Samsung Knox", "com.samsung.android.knox.containercore", "2.7.1"},
	{"Google Admin", "com.google.android.apps.enterprise.cpanel", "2024.06.30.627"},
	{"LastPass", "com.lastpass.lpandroid", "5.21.0.13562"},
}

// newAndroidAgent creates a new Android device simulator.
func newAndroidAgent(
	agentIndex int,
	serverAddress string,
	enrollSecret string,
	pubSubToken string,
	proxyAddress string,
	enterpriseID string,
	statusReportInterval time.Duration,
	appCount int,
	nonComplianceProb float64,
	stats *osquery_perf.Stats,
) *androidAgent {
	enterpriseSpecificID := strings.ToUpper(uuid.New().String())
	deviceID := "fake" + strings.ReplaceAll(uuid.New().String()[:28], "-", "")
	serialNumber := fmt.Sprintf("AND%s", randomString(10))

	brands := []string{"Google", "Samsung", "OnePlus", "Motorola", "Nokia"}
	models := []string{"Pixel 8 Pro", "Pixel 7a", "Galaxy S24", "Galaxy A54", "Nord CE 3", "Edge 40", "X30"}
	hardwareTypes := []string{"qcom", "exynos", "tensor", "dimensity"}

	brand := brands[rand.IntN(len(brands))]                  // #nosec G404 -- load testing only
	model := models[rand.IntN(len(models))]                  // #nosec G404 -- load testing only
	hardware := hardwareTypes[rand.IntN(len(hardwareTypes))] // #nosec G404 -- load testing only

	// Android versions 13-15
	androidVersions := []string{"13", "14", "15"}
	androidVersion := androidVersions[rand.IntN(len(androidVersions))]                                     // #nosec G404 -- load testing only
	buildNumber := fmt.Sprintf("TP1A.%d%02d%02d.003", 2024+rand.IntN(2), 1+rand.IntN(12), 1+rand.IntN(28)) // #nosec G404 -- load testing only

	// Generate installed apps list
	if appCount > len(androidApps) {
		appCount = len(androidApps)
	}
	// Shuffle and pick appCount apps
	perm := rand.Perm(len(androidApps))
	apps := make([]*androidmanagement.ApplicationReport, 0, appCount)
	for i := 0; i < appCount; i++ {
		app := androidApps[perm[i]]
		apps = append(apps, &androidmanagement.ApplicationReport{
			DisplayName: app.displayName,
			PackageName: app.packageName,
			VersionName: app.baseVersion,
			State:       "INSTALLED",
		})
	}

	// Memory: 4-12 GB RAM, 64-256 GB storage
	ramOptions := []int64{4, 6, 8, 12}
	storageOptions := []int64{64, 128, 256}
	totalRAM := ramOptions[rand.IntN(len(ramOptions))] * 1024 * 1024 * 1024             // #nosec G404 -- load testing only
	totalStorage := storageOptions[rand.IntN(len(storageOptions))] * 1024 * 1024 * 1024 // #nosec G404 -- load testing only

	return &androidAgent{
		agentIndex:           agentIndex,
		serverAddress:        serverAddress,
		enrollSecret:         enrollSecret,
		pubSubToken:          pubSubToken,
		proxyAddress:         proxyAddress,
		enterpriseID:         enterpriseID,
		stats:                stats,
		enterpriseSpecificID: enterpriseSpecificID,
		serialNumber:         serialNumber,
		deviceName:           fmt.Sprintf("enterprises/%s/devices/%s", enterpriseID, deviceID),
		brand:                brand,
		model:                model,
		hardware:             hardware,
		androidVersion:       androidVersion,
		androidBuildNumber:   buildNumber,
		totalRAM:             totalRAM,
		totalInternalStorage: totalStorage,
		installedApps:        apps,
		statusReportInterval: statusReportInterval,
		nonComplianceProb:    nonComplianceProb,
	}
}

// runLoop is the main loop for the Android agent.
// It registers with the mock proxy, sends enrollment to Fleet, then periodically sends status reports.
func (a *androidAgent) runLoop() {
	// Step 1: Register with mock AMAPI proxy
	if err := a.registerWithProxy(); err != nil {
		log.Printf("Android agent %d: failed to register with proxy: %v", a.agentIndex, err)
		return
	}

	// Step 2: Send ENROLLMENT PubSub to Fleet
	if err := a.sendEnrollment(); err != nil {
		log.Printf("Android agent %d: enrollment failed: %v", a.agentIndex, err)
		return
	}
	a.stats.IncrementAndroidEnrollments()

	// Step 3: Periodic status reports + command ack loop
	statusTicker := time.NewTicker(a.statusReportInterval)
	defer statusTicker.Stop()

	for range statusTicker.C {
		// Poll proxy for current state (policy version, pending commands)
		state, err := a.pollProxyState()
		if err != nil {
			log.Printf("Android agent %d: failed to poll proxy: %v", a.agentIndex, err)
			a.stats.IncrementAndroidErrors()
			continue
		}

		// Send STATUS_REPORT
		if err := a.sendStatusReport(state); err != nil {
			log.Printf("Android agent %d: status report failed: %v", a.agentIndex, err)
			a.stats.IncrementAndroidErrors()
			continue
		}
		a.stats.IncrementAndroidStatusReports()

		// Ack any pending commands
		for _, opName := range state.PendingCommands {
			if err := a.sendCommandAck(opName); err != nil {
				log.Printf("Android agent %d: command ack failed for %s: %v", a.agentIndex, opName, err)
				a.stats.IncrementAndroidErrors()
				continue
			}
			a.stats.IncrementAndroidCommandAcks()
		}
	}
}

// proxyDeviceState is the response from the mock proxy's coordination API.
type proxyDeviceState struct {
	PolicyVersion   int64    `json:"policy_version"`
	PolicyName      string   `json:"policy_name"`
	PendingCommands []string `json:"pending_commands"`
}

// registerWithProxy registers this fake device with the mock AMAPI proxy.
func (a *androidAgent) registerWithProxy() error {
	body := struct {
		EnterpriseSpecificID string `json:"enterprise_specific_id"`
		DeviceName           string `json:"device_name"`
		EnterpriseID         string `json:"enterprise_id"`
	}{
		EnterpriseSpecificID: a.enterpriseSpecificID,
		DeviceName:           a.deviceName,
		EnterpriseID:         a.enterpriseID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal register body: %w", err)
	}

	resp, err := http.Post(a.proxyAddress+"/mock/devices/register", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// pollProxyState asks the mock proxy for the current state this device should report.
func (a *androidAgent) pollProxyState() (*proxyDeviceState, error) {
	resp, err := http.Get(a.proxyAddress + "/mock/devices/" + a.enterpriseSpecificID + "/state")
	if err != nil {
		return nil, fmt.Errorf("poll state request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("poll state returned %d: %s", resp.StatusCode, string(respBody))
	}

	var state proxyDeviceState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	return &state, nil
}

// sendEnrollment sends an ENROLLMENT PubSub message to Fleet.
func (a *androidAgent) sendEnrollment() error {
	device := androidmanagement.Device{
		Name:                a.deviceName,
		Ownership:           "COMPANY_OWNED",
		EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, a.enrollSecret),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: a.enterpriseSpecificID,
			SerialNumber:         a.serialNumber,
			Brand:                a.brand,
			Model:                a.model,
			Hardware:             a.hardware,
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidVersion:     a.androidVersion,
			AndroidBuildNumber: a.androidBuildNumber,
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             a.totalRAM,
			TotalInternalStorage: a.totalInternalStorage,
		},
		MemoryEvents: a.generateMemoryEvents(),
	}

	return a.sendPubSubMessage(android.PubSubEnrollment, device)
}

// sendStatusReport sends a STATUS_REPORT PubSub message to Fleet.
func (a *androidAgent) sendStatusReport(state *proxyDeviceState) error {
	now := time.Now().UTC()

	device := androidmanagement.Device{
		Name:      a.deviceName,
		Ownership: "COMPANY_OWNED",
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: a.enterpriseSpecificID,
			SerialNumber:         a.serialNumber,
			Brand:                a.brand,
			Model:                a.model,
			Hardware:             a.hardware,
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidVersion:     a.androidVersion,
			AndroidBuildNumber: a.androidBuildNumber,
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             a.totalRAM,
			TotalInternalStorage: a.totalInternalStorage,
		},
		MemoryEvents:         a.generateMemoryEvents(),
		ApplicationReports:   a.installedApps,
		AppliedPolicyVersion: state.PolicyVersion,
		AppliedPolicyName:    state.PolicyName,
		LastPolicySyncTime:   now.Format(time.RFC3339),
		LastStatusReportTime: now.Format(time.RFC3339),
		EnrollmentTokenData:  fmt.Sprintf(`{"EnrollSecret": "%s"}`, a.enrollSecret),
	}

	// Optionally add non-compliance details
	nonCompliant := rand.Float64() < a.nonComplianceProb // #nosec G404 -- load testing only
	if nonCompliant {
		device.NonComplianceDetails = []*androidmanagement.NonComplianceDetail{
			{
				SettingName:               "passwordPolicies",
				NonComplianceReason:       "USER_ACTION",
				InstallationFailureReason: "",
			},
		}
	}

	return a.sendPubSubMessage(android.PubSubStatusReport, device)
}

// sendCommandAck sends a COMMAND PubSub message to Fleet acknowledging a completed command.
func (a *androidAgent) sendCommandAck(operationName string) error {
	op := androidmanagement.Operation{
		Name: operationName,
		Done: true,
	}
	return a.sendPubSubMessage(android.PubSubCommand, op)
}

// sendPubSubMessage constructs and sends a PubSub push message to Fleet's endpoint.
func (a *androidAgent) sendPubSubMessage(notificationType android.NotificationType, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	encodedData := base64.StdEncoding.EncodeToString(data)

	msg := struct {
		Message android.PubSubMessage `json:"message"`
	}{
		Message: android.PubSubMessage{
			Attributes: map[string]string{
				"notificationType": string(notificationType),
			},
			Data: encodedData,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal pubsub message: %w", err)
	}

	// POST to Fleet's PubSub endpoint with the token as a query parameter
	url := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/pubsub?token=%s", a.serverAddress, a.pubSubToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) // #nosec G107 -- URL is constructed from trusted config
	if err != nil {
		return fmt.Errorf("pubsub POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pubsub returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// generateMemoryEvents creates realistic memory events for the device.
func (a *androidAgent) generateMemoryEvents() []*androidmanagement.MemoryEvent {
	now := time.Now().UTC()
	// External storage = half of internal for simplicity
	externalTotal := a.totalInternalStorage / 2
	// Available = 30-80% of total
	internalAvail := int64(float64(a.totalInternalStorage) * (0.3 + rand.Float64()*0.5)) // #nosec G404 -- load testing only
	externalAvail := int64(float64(externalTotal) * (0.3 + rand.Float64()*0.5))          // #nosec G404 -- load testing only

	return []*androidmanagement.MemoryEvent{
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  externalTotal,
			CreateTime: now.Add(-24 * time.Hour).Format(time.RFC3339),
		},
		{
			EventType:  "INTERNAL_STORAGE_MEASURED",
			ByteCount:  internalAvail,
			CreateTime: now.Format(time.RFC3339),
		},
		{
			EventType:  "EXTERNAL_STORAGE_MEASURED",
			ByteCount:  externalAvail,
			CreateTime: now.Format(time.RFC3339),
		},
	}
}
