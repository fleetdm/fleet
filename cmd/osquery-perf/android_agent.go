package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"math/big"
	"math/rand/v2"
	"net/http"
	neturl "net/url"
	"slices"
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
//
// The proxy keeps devices in memory only, so a proxy restart leaves it listing no devices
// until each agent registers again. Registering again only happens on the next status-report
// tick, so a restart exposes a window of up to one --android_status_interval in which Fleet's
// device reconciler can see the fleet as absent and unenroll it (one mdm_unenrolled activity
// per host) before the next status report re-enrolls it. Keep that interval short when
// restarting the proxy mid-test; closing the window entirely means persisting the proxy's
// device store across restarts.
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
	orbitNodeKey         string // obtained from orbit enrollment, used for certificate API auth

	// Hardware details
	brand        string
	model        string
	hardware     string
	manufacturer string

	// Software
	androidVersion      string
	androidBuildNumber  string
	apiLevel            int64
	securityPatchLevel  string
	deviceKernelVersion string
	bootloaderVersion   string
	systemUpdateStatus  string

	// Device settings, security posture and eSIM activation. Fleet stores these
	// as host vitals. Unlike the fields above they describe the device's current
	// state rather than its identity, so a status report re-rolls them (see
	// vitalsChurnProb).
	adbEnabled        bool
	deviceSecure      bool
	verifyAppsEnabled bool
	encryptionStatus  string
	securityPosture   string
	postureDetails    []*androidmanagement.PostureDetail

	// The SIM cards themselves are hardware and stay put; only their eSIM
	// activation state and config mode move.
	telephonyInfos []*androidmanagement.TelephonyInfo

	// The device's radio identifier. A device has one radio, so exactly one of
	// these is set: IMEI on a GSM device, MEID on a CDMA one.
	imei string
	meid string

	// Whether the applied policy collects these sections at all. AMAPI omits a
	// section it is not asked for, and Fleet nulls the matching columns when a
	// device stops reporting one.
	reportsDeviceSettings  bool
	reportsSecurityPosture bool

	// vitalsChurnProb is the chance a status report re-rolls the volatile vitals,
	// and policyEditProb the chance such a re-roll also changes which sections
	// the applied policy collects.
	vitalsChurnProb float64
	policyEditProb  float64

	// Memory
	totalRAM             int64
	totalInternalStorage int64

	// Installed apps reported in STATUS_REPORT
	installedApps []*androidmanagement.ApplicationReport

	// Timing
	statusReportInterval time.Duration

	// lastState is the most recent state successfully polled from the mock proxy. It is
	// reused when a later poll fails so the agent keeps reporting instead of going silent.
	// Fleet's device reconciler unenrolls hosts that the proxy stops listing, and a status
	// report is what re-enrolls one, so an agent that goes quiet cannot recover.
	// staleStateReports counts how many consecutive reports have used it.
	lastState         *proxyDeviceState
	staleStateReports int

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

// AMAPI's "no data" members for the eSIM-only telephony enums, which every
// physical SIM reports and Fleet stores as no value at all.
const (
	activationStateUnspecified = "ACTIVATION_STATE_UNSPECIFIED"
	configModeUnspecified      = "CONFIG_MODE_UNSPECIFIED"
)

// defaultPolicyEditProb is the chance a re-roll of a device's volatile vitals
// also changes which sections its applied policy collects, standing in for an
// admin editing the policy's status reporting settings.
const defaultPolicyEditProb = 0.05

// defaultVitalsChurnProb is the default for --android_vitals_churn_prob: the
// share of status reports that re-roll a device's volatile vitals instead of
// repeating the previous ones.
//
// A real device's posture and settings move far more rarely than this, but a
// load test needs the vitals table to actually see row writes. Fleet's UPDATE
// overwrites every column on every status report, yet MySQL changes no row,
// writes no redo or binlog entry and does not advance updated_at when the values
// are identical — so a fleet whose vitals never move measures none of the write
// cost a real one generates.
const defaultVitalsChurnProb = 0.2

// androidManufacturers maps a brand to the Build.MANUFACTURER value real devices
// of that brand report, which is not always the brand itself.
var androidManufacturers = map[string]string{
	"Google":   "Google",
	"Samsung":  "samsung",
	"OnePlus":  "OnePlus",
	"Motorola": "motorola",
	"Nokia":    "HMD Global",
}

// androidAPILevels maps the Android release to its SDK level, which AMAPI reports
// alongside the version.
var androidAPILevels = map[string]int64{
	"13": 33,
	"14": 34,
	"15": 35,
}

// androidKernelBases maps the Android release to the Linux kernel line it ships on.
var androidKernelBases = map[string]string{
	"13": "5.15",
	"14": "6.1",
	"15": "6.6",
}

// androidSecurityPatchLevels is the set of patch levels devices report, resolved
// once at startup. It is a handful of recent months rather than a freely
// generated date because Fleet folds the security patch level into
// hosts.os_version: a distinct value per device would inflate the os_versions
// aggregation to one row per host, which no real fleet looks like. Deriving them
// from the clock rather than hardcoding keeps a long-lived load-test harness from
// reporting patch levels that are years stale, at the cost of the exact
// os_versions rows differing between runs in different months.
var androidSecurityPatchLevels = recentSecurityPatchLevels(time.Now().UTC())

// recentSecurityPatchLevels returns the monthly Android security bulletin dates
// for the four months ending at now, newest last. AMAPI reports these as
// YYYY-MM-DD, and the bulletins land on the first of the month.
func recentSecurityPatchLevels(now time.Time) []string {
	const months = 4
	// Anchored to the first of the month before subtracting: AddDate normalizes a
	// day the target month doesn't have, so subtracting a month from the 31st
	// lands in the month after the one intended and yields a duplicate.
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	levels := make([]string, 0, months)
	for i := months - 1; i >= 0; i-- {
		levels = append(levels, firstOfMonth.AddDate(0, -i, 0).Format("2006-01-02"))
	}
	return levels
}

// androidCarrier is a mobile carrier a simulated SIM can report.
type androidCarrier struct {
	name        string
	dialCode    string
	nationalLen int
}

// androidCarriers pairs a carrier with the country calling code and national
// number length its subscriber numbers use, so a generated phone number and the
// carrier reported beside it don't contradict each other.
var androidCarriers = []androidCarrier{
	{"Verizon", "+1", 10},
	{"AT&T", "+1", 10},
	{"T-Mobile", "+1", 10},
	{"Rogers", "+1", 10},
	{"Vodafone", "+44", 10},
	{"Orange", "+33", 9},
	{"Telefonica", "+34", 9},
}

// generateStableVitals produces the vitals that identify the device and never
// change over its lifetime: who built it, what it runs, and which SIM cards are
// in it. The distributions are weighted so a simulated fleet has a realistic
// spread — the aggregate shape matters more than any one device.
//
// securityPatchLevel is pinned here even though a real device's does move when
// it installs an update, which is why a simulated device can report
// SECURITY_UPDATE_AVAILABLE and then UP_TO_DATE without its patch level ever
// changing. Fleet folds the patch level into hosts.os_version, so churning it
// would also churn operating_systems and host_operating_system — a much wider
// blast radius than the vitals table this change is about.
func (a *androidAgent) generateStableVitals() {
	a.manufacturer = androidManufacturers[a.brand]
	a.apiLevel = androidAPILevels[a.androidVersion]
	a.securityPatchLevel = androidSecurityPatchLevels[rand.IntN(len(androidSecurityPatchLevels))] // #nosec G404 -- load testing only
	// e.g. "6.6.42-android15-11-gb1c0ffee1234", the format Android kernels report.
	a.deviceKernelVersion = fmt.Sprintf("%s.%d-android%s-%d-g%s",
		androidKernelBases[a.androidVersion], rand.IntN(100), a.androidVersion, 1+rand.IntN(30), randomHexString(12)) // #nosec G404 -- load testing only
	a.bootloaderVersion = fmt.Sprintf("%s-%s.0-%d", a.hardware, a.androidVersion, 10000000+rand.IntN(9000000)) // #nosec G404 -- load testing only

	// Almost every managed device is encrypted; ACTIVE_PER_USER is what modern
	// file-based encryption reports and ACTIVE what older full-disk encryption
	// does. The unencrypted states are rare but not absent: they are the only
	// ones that make Fleet store an encryption_type other than "on".
	//
	// Stable rather than volatile because none of these transitions exist on real
	// hardware: UNSUPPORTED is a property of the device, and file-based vs
	// full-disk encryption is fixed at first boot and changes only on a factory
	// reset.
	a.encryptionStatus = weightedChoice([]weightedValue{
		{"ACTIVE_PER_USER", 0.84},
		{"ACTIVE", 0.1},
		{"INACTIVE", 0.04},
		{"UNSUPPORTED", 0.02},
	})

	a.telephonyInfos = generateTelephonyInfos()
	a.imei, a.meid = generateRadioIdentifier(a.telephonyInfos)
	a.rollCollectedSections()
}

// generateRadioIdentifier builds the device's hardware radio identifier and
// returns it as (imei, meid). AMAPI reports whichever the radio uses and never
// both, so exactly one of the two is non-empty.
//
// CDMA is the rare case, and only for a device on a North American carrier:
// the networks that ran it elsewhere are gone, and the ones that ran it in the
// US shut it down years ago, leaving only stragglers.
func generateRadioIdentifier(sims []*androidmanagement.TelephonyInfo) (imei, meid string) {
	northAmerican := len(sims) > 0 && strings.HasPrefix(sims[0].PhoneNumber, "+1")
	if northAmerican && rand.Float64() < 0.1 { // #nosec G404 -- load testing only
		// An MEID is 14 hexadecimal digits, conventionally upper case.
		return "", strings.ToUpper(randomHexString(14))
	}
	// An IMEI is 15 digits: 14 identifying the device plus a Luhn check digit,
	// which is left random here because nothing in Fleet validates it.
	return randomDigits(15), ""
}

// generateVolatileVitals rolls the vitals that describe what the device is doing
// now rather than what it is: settings a user or admin can change, the posture
// Play Integrity reports, whether an update is pending, eSIM activation, and
// which sections the applied policy collects at all.
func (a *androidAgent) generateVolatileVitals() {
	a.adbEnabled = rand.Float64() < 0.05       // #nosec G404 -- load testing only
	a.deviceSecure = rand.Float64() < 0.95     // #nosec G404 -- load testing only
	a.verifyAppsEnabled = rand.Float64() < 0.9 // #nosec G404 -- load testing only
	a.systemUpdateStatus = weightedChoice([]weightedValue{
		{"UP_TO_DATE", 0.7},
		{"SECURITY_UPDATE_AVAILABLE", 0.15},
		{"OS_UPDATE_AVAILABLE", 0.1},
		{"UNKNOWN_UPDATE_AVAILABLE", 0.05},
	})

	a.securityPosture = weightedChoice([]weightedValue{
		{"SECURE", 0.85},
		{"AT_RISK", 0.1},
		{"POTENTIALLY_COMPROMISED", 0.05},
	})
	// AMAPI only attaches posture details to a device that is not secure, so a
	// device becoming secure again has to drop the details it used to report.
	a.postureDetails = nil
	switch a.securityPosture {
	case "AT_RISK":
		risk := "UNKNOWN_OS"
		advice := "This device is running an unrecognized version of Android. Update it to a certified build."
		if rand.Float64() < 0.5 { // #nosec G404 -- load testing only
			risk = "HARDWARE_BACKED_EVALUATION_FAILED"
			advice = "This device does not support hardware-backed integrity checks. Replace it with a device that does."
		}
		a.postureDetails = []*androidmanagement.PostureDetail{{
			SecurityRisk: risk,
			Advice:       []*androidmanagement.UserFacingMessage{{DefaultMessage: advice}},
		}}
	case "POTENTIALLY_COMPROMISED":
		a.postureDetails = []*androidmanagement.PostureDetail{{
			SecurityRisk: "COMPROMISED_OS",
			Advice: []*androidmanagement.UserFacingMessage{
				{DefaultMessage: "This device is running a modified version of Android. Factory reset it."},
			},
		}}
	}

	// Which sections are collected belongs to the applied policy, not to the
	// moment, so it is decided per device in generateStableVitals. Re-rolling it
	// rarely stands in for an admin editing the policy's status reporting
	// settings, which is what makes Fleet null the columns of a section the
	// device used to report. Re-rolling it every time would instead leave every
	// host flickering between populated and empty vitals, which no real fleet
	// does and which reads as a Fleet bug in the host UI.
	if rand.Float64() < a.policyEditProb { // #nosec G404 -- load testing only
		a.rollCollectedSections()
	}

	a.rollSIMActivation()
}

// rollCollectedSections decides which optional sections the device's applied
// policy collects. AMAPI omits a section it is not asked for entirely.
func (a *androidAgent) rollCollectedSections() {
	a.reportsDeviceSettings = rand.Float64() < 0.9  // #nosec G404 -- load testing only
	a.reportsSecurityPosture = rand.Float64() < 0.9 // #nosec G404 -- load testing only
}

// rollSIMActivation re-rolls the activation state and config mode of the device's
// eSIMs. A physical SIM reports neither, so it is left alone; its *_UNSPECIFIED
// activation state is what marks it as physical.
func (a *androidAgent) rollSIMActivation() {
	for _, sim := range a.telephonyInfos {
		if sim.ActivationState == activationStateUnspecified {
			continue
		}
		sim.ActivationState = "ACTIVATED"
		if rand.Float64() < 0.05 { // #nosec G404 -- load testing only
			sim.ActivationState = "NOT_ACTIVATED"
		}
		sim.ConfigMode = "ADMIN_CONFIGURED"
		if rand.Float64() < 0.3 { // #nosec G404 -- load testing only
			sim.ConfigMode = "USER_CONFIGURED"
		}
	}
}

// generateTelephonyInfos builds the SIM cards a device reports. Most devices have
// one, which is usually physical; a second one is always an eSIM, so a dual-SIM
// device is either physical + eSIM or two eSIMs. Only eSIMs report an activation
// state and config mode; a physical SIM reports the *_UNSPECIFIED sentinels,
// which Fleet stores as no value at all.
func generateTelephonyInfos() []*androidmanagement.TelephonyInfo {
	simCount := 1
	if rand.Float64() < 0.3 { // #nosec G404 -- load testing only
		simCount = 2
	}

	infos := make([]*androidmanagement.TelephonyInfo, 0, simCount)
	for i := 0; i < simCount; i++ {
		// The first SIM is usually physical; a second one is always an eSIM.
		eSIM := i > 0 || rand.Float64() < 0.4                       // #nosec G404 -- load testing only
		carrier := androidCarriers[rand.IntN(len(androidCarriers))] // #nosec G404 -- load testing only
		info := &androidmanagement.TelephonyInfo{
			PhoneNumber: carrier.dialCode + randomDigits(carrier.nationalLen),
			CarrierName: carrier.name,
			// 19 digits in total: the 89 telecom major industry identifier plus
			// 17 more. The last of those is random rather than a real Luhn check
			// digit, since nothing in Fleet validates it.
			IccId:           "89" + randomDigits(17),
			ActivationState: activationStateUnspecified,
			ConfigMode:      configModeUnspecified,
		}
		if eSIM {
			// Marks the SIM as an eSIM; rollSIMActivation picks the real values
			// here and on every later re-roll.
			info.ActivationState = "ACTIVATED"
			info.ConfigMode = "ADMIN_CONFIGURED"
		}
		infos = append(infos, info)
	}
	return infos
}

// weightedValue is one option of weightedChoice, with the fraction of devices
// that should get it.
type weightedValue struct {
	value  string
	weight float64
}

// weightedChoice picks one value according to its weight. The weights are
// expected to sum to 1; any rounding shortfall is absorbed by the last value
// that can actually be picked, so a zero-weight option is never returned.
func weightedChoice(values []weightedValue) string {
	roll := rand.Float64() // #nosec G404 -- load testing only
	var cumulative float64
	fallback := values[len(values)-1].value
	for _, v := range values {
		if v.weight > 0 {
			fallback = v.value
		}
		cumulative += v.weight
		if roll < cumulative {
			return v.value
		}
	}
	return fallback
}

func randomDigits(n int) string {
	const digitVals = "0123456789"
	sb := strings.Builder{}
	sb.Grow(n)
	for range n {
		sb.WriteByte(digitVals[rand.IntN(len(digitVals))]) // #nosec G404 -- load testing only
	}
	return sb.String()
}

func randomHexString(n int) string {
	const hexVals = "0123456789abcdef"
	sb := strings.Builder{}
	sb.Grow(n)
	for range n {
		sb.WriteByte(hexVals[rand.IntN(len(hexVals))]) // #nosec G404 -- load testing only
	}
	return sb.String()
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
	vitalsChurnProb float64,
	stats *osquery_perf.Stats,
) *androidAgent {
	enterpriseSpecificID := strings.ToUpper(uuid.New().String())
	deviceID := "fake" + strings.ReplaceAll(uuid.New().String()[:28], "-", "")
	serialNumber := fmt.Sprintf("AND%s", randomString(10))

	// Drawn from the vitals maps so a brand or release can only be simulated if
	// the manufacturer / API level / kernel line it needs is defined for it.
	brands := slices.Sorted(maps.Keys(androidManufacturers))
	models := []string{"Pixel 8 Pro", "Pixel 7a", "Galaxy S24", "Galaxy A54", "Nord CE 3", "Edge 40", "X30"}
	hardwareTypes := []string{"qcom", "exynos", "tensor", "dimensity"}

	brand := brands[rand.IntN(len(brands))]                  // #nosec G404 -- load testing only
	model := models[rand.IntN(len(models))]                  // #nosec G404 -- load testing only
	hardware := hardwareTypes[rand.IntN(len(hardwareTypes))] // #nosec G404 -- load testing only

	androidVersions := slices.Sorted(maps.Keys(androidAPILevels))
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

	agent := &androidAgent{
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
		vitalsChurnProb:      vitalsChurnProb,
		policyEditProb:       defaultPolicyEditProb,
	}
	// Depends on the brand, hardware and Android version picked above.
	agent.generateStableVitals()
	agent.generateVolatileVitals()

	return agent
}

// runLoop is the main loop for the Android agent.
// It registers with the mock proxy, sends enrollment to Fleet, then periodically sends status reports.
func (a *androidAgent) runLoop() {
	// Step 1: Register with mock AMAPI proxy (retry with backoff)
	for attempt := 1; ; attempt++ {
		if err := a.registerWithProxy(); err != nil {
			if attempt >= 5 {
				log.Printf("Android agent %d: failed to register with proxy after %d attempts: %v", a.agentIndex, attempt, err)
				return
			}
			log.Printf("Android agent %d: register attempt %d failed, retrying: %v", a.agentIndex, attempt, err)
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			continue
		}
		break
	}

	// Step 2: Send ENROLLMENT PubSub to Fleet (retry with backoff)
	for attempt := 1; ; attempt++ {
		if err := a.sendEnrollment(); err != nil {
			if attempt >= 5 {
				log.Printf("Android agent %d: enrollment failed after %d attempts: %v", a.agentIndex, attempt, err)
				return
			}
			log.Printf("Android agent %d: enrollment attempt %d failed, retrying: %v", a.agentIndex, attempt, err)
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			continue
		}
		break
	}
	a.stats.IncrementAndroidEnrollments()

	// Step 2b: Orbit enrollment (retry with backoff, non-fatal)
	for attempt := 1; ; attempt++ {
		if err := a.orbitEnroll(); err != nil {
			if attempt >= 3 {
				log.Printf("Android agent %d: orbit enrollment failed after %d attempts: %v", a.agentIndex, attempt, err)
				break // Non-fatal — certificate flow won't work but status reports will
			}
			log.Printf("Android agent %d: orbit enrollment attempt %d failed, retrying: %v", a.agentIndex, attempt, err)
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			continue
		}
		break
	}

	// Step 3: Periodic status reports + command ack + certificate verification loop
	statusTicker := time.NewTicker(a.statusReportInterval)
	defer statusTicker.Stop()

	// Track which certificate templates we've already verified so we don't re-verify
	verifiedCerts := make(map[uint]struct{})

	for range statusTicker.C {
		// Poll proxy for current state (policy version, pending commands)
		state, stale, err := a.currentState()
		if errors.Is(err, errProxyDeviceDeleted) {
			// Fleet unenrolled this host, so there is nothing left to simulate.
			log.Printf("Android agent %d: device was deleted, stopping", a.agentIndex)
			return
		}
		if err != nil {
			log.Printf("Android agent %d: failed to poll proxy: %v", a.agentIndex, err)
			a.stats.IncrementAndroidErrors()
			continue
		}
		if stale {
			// Still report, so the device doesn't look dead to Fleet's reconciler, but count
			// it: the state being reported is fabricated.
			a.stats.IncrementAndroidErrors()
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

		// Process certificate templates from the proxy state.
		if a.orbitNodeKey != "" {
			for _, certID := range state.PendingCertificates {
				// Always GET the cert to check its current status — renewals reuse the same template ID
				cert, err := a.getCertificateTemplate(certID)
				if err != nil {
					log.Printf("Android agent %d: get certificate %d failed: %v", a.agentIndex, certID, err)
					a.stats.IncrementAndroidErrors()
					continue
				}

				// If status went back to non-delivered (renewal in progress), clear our tracking
				if cert.Status != "delivered" {
					delete(verifiedCerts, certID)
					continue
				}

				// Skip if we already verified this delivery
				if _, ok := verifiedCerts[certID]; ok {
					continue
				}

				// PUT the certificate status as verified (simulates SCEP enrollment completion)
				if err := a.updateCertificateStatus(certID, "verified", "install"); err != nil {
					log.Printf("Android agent %d: update certificate %d status failed: %v", a.agentIndex, certID, err)
					a.stats.IncrementAndroidErrors()
					continue
				}

				verifiedCerts[certID] = struct{}{}
				a.stats.IncrementAndroidCertVerifications()
			}
		}
	}
}

// proxyDeviceState is the response from the mock proxy's coordination API.
type proxyDeviceState struct {
	PolicyVersion       int64    `json:"policy_version"`
	PolicyName          string   `json:"policy_name"`
	PendingCommands     []string `json:"pending_commands"`
	PendingCertificates []uint   `json:"pending_certificates"`
}

// certTemplateResponse is the response from GET /api/fleetd/certificates/{id}
type certTemplateResponse struct {
	Certificate *certTemplateInfo `json:"certificate"`
}

type certTemplateInfo struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

// registerWithProxy registers this fake device with the mock AMAPI proxy.
func (a *androidAgent) registerWithProxy() error {
	body := struct {
		EnterpriseSpecificID string `json:"enterprise_specific_id"`
		DeviceName           string `json:"device_name"`
		EnterpriseID         string `json:"enterprise_id"`
		PolicyName           string `json:"policy_name,omitempty"`
		PolicyVersion        int64  `json:"policy_version,omitempty"`
	}{
		EnterpriseSpecificID: a.enterpriseSpecificID,
		DeviceName:           a.deviceName,
		EnterpriseID:         a.enterpriseID,
	}
	// When registering again after the proxy lost its in-memory state, hand back the policy
	// we last observed so the proxy doesn't report a regressed policy to Fleet.
	if a.lastState != nil {
		body.PolicyName = a.lastState.PolicyName
		body.PolicyVersion = a.lastState.PolicyVersion
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

	if resp.StatusCode == http.StatusGone {
		return errProxyDeviceDeleted
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// errProxyDeviceUnknown means the mock proxy has no registration for this device, which
// happens when the proxy is restarted since it only keeps devices in memory.
var errProxyDeviceUnknown = errors.New("device not registered with proxy")

// errProxyDeviceDeleted means Fleet deleted this device through AMAPI, i.e. the host was
// unenrolled. Unlike errProxyDeviceUnknown this is terminal: the device is gone on purpose
// and must not register again.
var errProxyDeviceDeleted = errors.New("device was deleted from the proxy")

// maxStaleStateReports bounds how many consecutive status reports may be sent from a stale
// cached state, so a permanently broken agent stops looking healthy.
const maxStaleStateReports = 10

// pollProxyState asks the mock proxy for the current state this device should report.
func (a *androidAgent) pollProxyState() (*proxyDeviceState, error) {
	resp, err := http.Get(a.proxyAddress + "/mock/devices/" + a.enterpriseSpecificID + "/state")
	if err != nil {
		return nil, fmt.Errorf("poll state request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errProxyDeviceUnknown
	}
	if resp.StatusCode == http.StatusGone {
		return nil, errProxyDeviceDeleted
	}
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

// currentState polls the mock proxy for this device's state, recovering from a proxy that
// lost its in-memory registration by registering again. If the state still can't be fetched,
// the last known state is reused so the agent keeps sending status reports. Fleet's hourly
// device reconciler unenrolls hosts the proxy no longer lists — which is every fake device
// until it registers again after a proxy restart — and a status report is what re-enrolls
// one, so an agent that goes quiet stays unenrolled.
//
// The returned bool reports whether the state is stale (reused rather than freshly polled),
// so the caller can still count the failure instead of reporting a healthy load test.
// errProxyDeviceDeleted is returned as-is: that device is gone on purpose.
func (a *androidAgent) currentState() (*proxyDeviceState, bool, error) {
	state, err := a.pollProxyState()
	if errors.Is(err, errProxyDeviceUnknown) {
		log.Printf("Android agent %d: proxy lost our registration, registering again", a.agentIndex)
		if rerr := a.registerWithProxy(); rerr != nil {
			if errors.Is(rerr, errProxyDeviceDeleted) {
				return nil, false, rerr
			}
			return a.lastKnownState(fmt.Errorf("re-register with proxy: %w", rerr))
		}
		state, err = a.pollProxyState()
	}
	if errors.Is(err, errProxyDeviceDeleted) {
		return nil, false, err
	}
	if err != nil {
		return a.lastKnownState(err)
	}
	a.lastState = state
	a.staleStateReports = 0
	return state, false, nil
}

// lastKnownState returns the previously polled state, or err if there is none yet or the
// state has been reused too many times in a row.
func (a *androidAgent) lastKnownState(err error) (*proxyDeviceState, bool, error) {
	if a.lastState == nil {
		return nil, false, err
	}
	if a.staleStateReports >= maxStaleStateReports {
		return nil, false, fmt.Errorf("proxy state stale for %d consecutive reports: %w", a.staleStateReports, err)
	}
	a.staleStateReports++
	log.Printf("Android agent %d: reusing last known proxy state (%d in a row): %v", a.agentIndex, a.staleStateReports, err)
	// Pending commands were already acked against the earlier state; re-acking them would
	// send duplicate operation results to Fleet.
	stale := *a.lastState
	stale.PendingCommands = nil
	// Certificates can't have changed while the proxy is unreachable, so there is nothing to
	// act on; the agent re-checks them as usual once a real state comes back.
	stale.PendingCertificates = nil
	return &stale, true, nil
}

// deviceInfo builds the parts of the AMAPI device payload that describe the
// device itself, which every message about it carries. Enrollment and status
// reports must agree here: Fleet writes host vitals from both, so a section
// present in one and absent from the other would make a host's vitals appear or
// disappear as the messages arrive.
func (a *androidAgent) deviceInfo() androidmanagement.Device {
	device := androidmanagement.Device{
		Name:                a.deviceName,
		Ownership:           "COMPANY_OWNED",
		ApiLevel:            a.apiLevel,
		EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, a.enrollSecret),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: a.enterpriseSpecificID,
			SerialNumber:         a.serialNumber,
			Brand:                a.brand,
			Model:                a.model,
			Hardware:             a.hardware,
			Manufacturer:         a.manufacturer,
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidVersion:      a.androidVersion,
			AndroidBuildNumber:  a.androidBuildNumber,
			SecurityPatchLevel:  a.securityPatchLevel,
			DeviceKernelVersion: a.deviceKernelVersion,
			BootloaderVersion:   a.bootloaderVersion,
			SystemUpdateInfo: &androidmanagement.SystemUpdateInfo{
				UpdateStatus: a.systemUpdateStatus,
			},
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             a.totalRAM,
			TotalInternalStorage: a.totalInternalStorage,
		},
		MemoryEvents: a.generateMemoryEvents(),
		// AMAPI only reports telephonyInfos, imei and meid for a fully managed
		// device, which every simulated device is (COMPANY_OWNED above).
		NetworkInfo: &androidmanagement.NetworkInfo{
			Imei:           a.imei,
			Meid:           a.meid,
			TelephonyInfos: a.simSnapshot(),
		},
	}

	// A section the applied policy does not collect is absent from the payload
	// entirely, rather than present and empty.
	if a.reportsDeviceSettings {
		device.DeviceSettings = &androidmanagement.DeviceSettings{
			AdbEnabled:        a.adbEnabled,
			IsDeviceSecure:    a.deviceSecure,
			VerifyAppsEnabled: a.verifyAppsEnabled,
			EncryptionStatus:  a.encryptionStatus,
			// Wire fidelity only: without this, omitempty drops the false
			// booleans and AMAPI always sends them explicitly. Fleet reads them
			// back as false either way, since it takes the zero value of a
			// deviceSettings that is present at all.
			ForceSendFields: []string{"AdbEnabled", "IsDeviceSecure", "VerifyAppsEnabled"},
		}
	}
	if a.reportsSecurityPosture {
		device.SecurityPosture = &androidmanagement.SecurityPosture{
			DevicePosture:  a.securityPosture,
			PostureDetails: a.postureDetails,
		}
	}

	return device
}

// simSnapshot copies the device's SIM cards so a payload is a snapshot of them.
// rollSIMActivation mutates them in place, and a message that shared the agent's
// pointers could otherwise be marshaled with a half-updated SIM — an activation
// state from after the re-roll beside a config mode from before it.
func (a *androidAgent) simSnapshot() []*androidmanagement.TelephonyInfo {
	sims := make([]*androidmanagement.TelephonyInfo, 0, len(a.telephonyInfos))
	for _, sim := range a.telephonyInfos {
		sims = append(sims, new(*sim))
	}
	return sims
}

// sendEnrollment sends an ENROLLMENT PubSub message to Fleet.
func (a *androidAgent) sendEnrollment() error {
	return a.sendPubSubMessage(android.PubSubEnrollment, a.deviceInfo())
}

// sendStatusReport sends a STATUS_REPORT PubSub message to Fleet.
func (a *androidAgent) sendStatusReport(state *proxyDeviceState) error {
	now := time.Now().UTC()

	if rand.Float64() < a.vitalsChurnProb { // #nosec G404 -- load testing only
		a.generateVolatileVitals()
	}

	device := a.deviceInfo()
	device.AppliedState = "ACTIVE"
	device.ApplicationReports = a.installedApps
	device.AppliedPolicyVersion = state.PolicyVersion
	device.AppliedPolicyName = state.PolicyName
	device.LastPolicySyncTime = now.Format(time.RFC3339)
	device.LastStatusReportTime = now.Format(time.RFC3339)

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

func (a *androidAgent) orbitEnroll() error {
	body := struct {
		EnrollSecret   string `json:"enroll_secret"`
		HardwareUUID   string `json:"hardware_uuid"`
		HardwareSerial string `json:"hardware_serial"`
		Platform       string `json:"platform"`
		ComputerName   string `json:"computer_name"`
	}{
		EnrollSecret:   a.enrollSecret,
		HardwareUUID:   a.enterpriseSpecificID,
		HardwareSerial: a.serialNumber,
		Platform:       "android",
		ComputerName:   fmt.Sprintf("%s %s", a.brand, a.model),
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal orbit enroll: %w", err)
	}

	url := fmt.Sprintf("%s/api/fleet/orbit/enroll", a.serverAddress)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) // #nosec G107 -- load testing
	if err != nil {
		return fmt.Errorf("orbit enroll request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("orbit enroll returned %d: %s", resp.StatusCode, string(respBody))
	}

	var enrollResp struct {
		OrbitNodeKey string `json:"orbit_node_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return fmt.Errorf("decode orbit enroll response: %w", err)
	}
	if enrollResp.OrbitNodeKey == "" {
		return errors.New("empty orbit_node_key in response")
	}

	a.orbitNodeKey = enrollResp.OrbitNodeKey
	return nil
}

// getCertificateTemplate fetches a certificate template from Fleet.
func (a *androidAgent) getCertificateTemplate(certID uint) (*certTemplateInfo, error) {
	url := fmt.Sprintf("%s/api/fleetd/certificates/%d", a.serverAddress, certID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Node key "+a.orbitNodeKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get certificate returned %d: %s", resp.StatusCode, string(respBody))
	}

	var certResp certTemplateResponse
	if err := json.NewDecoder(resp.Body).Decode(&certResp); err != nil {
		return nil, fmt.Errorf("decode certificate: %w", err)
	}
	return certResp.Certificate, nil
}

// updateCertificateStatus reports a certificate template's status to Fleet.
func (a *androidAgent) updateCertificateStatus(certID uint, status, operationType string) error {
	now := time.Now().UTC()
	notBefore := now.Add(-1 * time.Hour)
	notAfter := now.Add(365 * 24 * time.Hour)

	// Generate a random serial number
	serialBytes := make([]byte, 16)
	if _, err := cryptorand.Read(serialBytes); err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}
	serial := new(big.Int).SetBytes(serialBytes).Text(16)

	body := struct {
		Status         string     `json:"status"`
		OperationType  string     `json:"operation_type"`
		NotValidBefore *time.Time `json:"not_valid_before"`
		NotValidAfter  *time.Time `json:"not_valid_after"`
		Serial         *string    `json:"serial"`
	}{
		Status:         status,
		OperationType:  operationType,
		NotValidBefore: &notBefore,
		NotValidAfter:  &notAfter,
		Serial:         &serial,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}

	url := fmt.Sprintf("%s/api/fleetd/certificates/%d/status", a.serverAddress, certID)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Node key "+a.orbitNodeKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("update certificate status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update certificate status returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
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
	url := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/pubsub?token=%s", a.serverAddress, neturl.QueryEscape(a.pubSubToken))
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
