package osquery_perf

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	StartTime                      time.Time
	errors                         int
	osqueryEnrollments             int
	orbitEnrollments               int
	mdmEnrollments                 int
	mdmUserEnrollments             int
	mdmSessions                    int
	mdmUserSessions                int
	mdmOnDemandSyncs               int
	distributedWrites              int
	mdmCommandsReceived            int
	mdmUserCommandsReceived        int
	mdmSCEPRequests                int
	mdmSCEPSuccess                 int
	mdmSCEPErrors                  int
	distributedReads               int
	configRequests                 int
	configErrors                   int
	resultLogRequests              int
	orbitErrors                    int
	mdmErrors                      int
	mdmUserErrors                  int
	ddmTokensErrors                int
	ddmTokensSuccess               int
	ddmDeclarationItemsErrors      int
	ddmConfigurationErrors         int
	ddmManagementErrors            int
	ddmActivationErrors            int
	ddmAssetErrors                 int
	ddmStatusErrors                int
	ddmDeclarationItemsSuccess     int
	ddmConfigurationSuccess        int
	ddmManagementSuccess           int
	ddmActivationSuccess           int
	ddmAssetSuccess                int
	ddmStatusSuccess               int
	ddmUserTokensErrors            int
	ddmUserTokensSuccess           int
	ddmUserDeclarationItemsErrors  int
	ddmUserConfigurationErrors     int
	ddmUserActivationErrors        int
	ddmUserAssetErrors             int
	ddmUserStatusErrors            int
	ddmUserDeclarationItemsSuccess int
	ddmUserConfigurationSuccess    int
	ddmUserActivationSuccess       int
	ddmUserAssetSuccess            int
	ddmUserStatusSuccess           int
	desktopErrors                  int
	distributedReadErrors          int
	distributedWriteErrors         int
	resultLogErrors                int
	bufferedLogs                   int
	scriptExecs                    int
	scriptExecErrs                 int
	softwareInstalls               int
	softwareInstallErrs            int
	androidEnrollments             int
	androidStatusReports           int
	androidCommandAcks             int
	androidCertVerifications       int
	androidErrors                  int
	pssoRegistrations              int
	pssoLogins                     int
	pssoKeyRequests                int
	pssoKeyExchanges               int
	pssoErrors                     int
	websocketConnected             int // gauge: currently connected WebSocket transports
	websocketConnects              int
	websocketNotifications         int
	websocketErrors                int

	// ETag / conditional config stats
	configFullResponses       int64
	configNotModified         int64
	configConditionalRequests int64
	configResponseBodyBytes   int64
	configEstimatedSavedBytes int64
	configETagDrift           int64

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

func (s *Stats) IncrementMDMUserEnrollments() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmUserEnrollments++
}

func (s *Stats) IncrementMDMSessions() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmSessions++
}

func (s *Stats) IncrementMDMUserSessions() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmUserSessions++
}

// IncrementMDMOnDemandSyncs counts Windows MDM sessions that were triggered by an on-demand wake
// (WindowsMDMSyncRequest) rather than the poll ticker. This is a subset of mdmSessions, not a separate total.
func (s *Stats) IncrementMDMOnDemandSyncs() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmOnDemandSyncs++
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

func (s *Stats) IncrementMDMUserCommandsReceived() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmUserCommandsReceived++
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

func (s *Stats) IncrementMDMUserErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmUserErrors++
}

func (s *Stats) IncrementMDMSCEPRequests() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmSCEPRequests++
}

func (s *Stats) IncrementMDMSCEPSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmSCEPSuccess++
}

func (s *Stats) IncrementMDMSCEPErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmSCEPErrors++
}

func (s *Stats) IncrementDDMTokensErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmTokensErrors++
}

func (s *Stats) IncrementDDMTokensSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmTokensSuccess++
}

func (s *Stats) IncrementDDMDeclarationItemsErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmDeclarationItemsErrors++
}

func (s *Stats) IncrementDDMConfigurationErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmConfigurationErrors++
}

func (s *Stats) IncrementDDMManagementErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmManagementErrors++
}

func (s *Stats) IncrementDDMActivationErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmActivationErrors++
}

func (s *Stats) IncrementDDMAssetErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmAssetErrors++
}

func (s *Stats) IncrementDDMStatusErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmStatusErrors++
}

func (s *Stats) IncrementDDMDeclarationItemsSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmDeclarationItemsSuccess++
}

func (s *Stats) IncrementDDMConfigurationSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmConfigurationSuccess++
}

func (s *Stats) IncrementDDMManagementSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmManagementSuccess++
}

func (s *Stats) IncrementDDMActivationSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmActivationSuccess++
}

func (s *Stats) IncrementDDMAssetSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmAssetSuccess++
}

func (s *Stats) IncrementDDMStatusSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmStatusSuccess++
}

func (s *Stats) IncrementUserDDMTokensErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserTokensErrors++
}

func (s *Stats) IncrementUserDDMTokensSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserTokensSuccess++
}

func (s *Stats) IncrementUserDDMDeclarationItemsErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserDeclarationItemsErrors++
}

func (s *Stats) IncrementUserDDMConfigurationErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserConfigurationErrors++
}

func (s *Stats) IncrementUserDDMActivationErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserActivationErrors++
}

func (s *Stats) IncrementUserDDMAssetErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserAssetErrors++
}

func (s *Stats) IncrementUserDDMStatusErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserStatusErrors++
}

func (s *Stats) IncrementUserDDMDeclarationItemsSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserDeclarationItemsSuccess++
}

func (s *Stats) IncrementUserDDMConfigurationSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserConfigurationSuccess++
}

func (s *Stats) IncrementUserDDMActivationSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserActivationSuccess++
}

func (s *Stats) IncrementUserDDMAssetSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserAssetSuccess++
}

func (s *Stats) IncrementUserDDMStatusSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmUserStatusSuccess++
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

func (s *Stats) IncrementScriptExecs() {
	s.l.Lock()
	defer s.l.Unlock()
	s.scriptExecs++
}

func (s *Stats) IncrementScriptExecErrs() {
	s.l.Lock()
	defer s.l.Unlock()
	s.scriptExecErrs++
}

func (s *Stats) IncrementSoftwareInstalls() {
	s.l.Lock()
	defer s.l.Unlock()
	s.softwareInstalls++
}

func (s *Stats) IncrementSoftwareInstallErrs() {
	s.l.Lock()
	defer s.l.Unlock()
	s.softwareInstallErrs++
}

func (s *Stats) IncrementAndroidEnrollments() {
	s.l.Lock()
	defer s.l.Unlock()
	s.androidEnrollments++
}

func (s *Stats) IncrementAndroidStatusReports() {
	s.l.Lock()
	defer s.l.Unlock()
	s.androidStatusReports++
}

func (s *Stats) IncrementAndroidCommandAcks() {
	s.l.Lock()
	defer s.l.Unlock()
	s.androidCommandAcks++
}

func (s *Stats) IncrementAndroidCertVerifications() {
	s.l.Lock()
	defer s.l.Unlock()
	s.androidCertVerifications++
}

func (s *Stats) IncrementAndroidErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.androidErrors++
}

func (s *Stats) IncrementPSSORegistrations() {
	s.l.Lock()
	defer s.l.Unlock()
	s.pssoRegistrations++
}

func (s *Stats) IncrementPSSOLogins() {
	s.l.Lock()
	defer s.l.Unlock()
	s.pssoLogins++
}

func (s *Stats) IncrementPSSOKeyRequests() {
	s.l.Lock()
	defer s.l.Unlock()
	s.pssoKeyRequests++
}

func (s *Stats) IncrementPSSOKeyExchanges() {
	s.l.Lock()
	defer s.l.Unlock()
	s.pssoKeyExchanges++
}

func (s *Stats) IncrementPSSOErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.pssoErrors++
}

func (s *Stats) RecordFullConfigResponse(bodyBytes int64, conditional bool) {
	s.l.Lock()
	defer s.l.Unlock()
	s.configFullResponses++
	s.configResponseBodyBytes += bodyBytes
	if conditional {
		s.configConditionalRequests++
	}
}

// RecordConfigNotModified records an unchanged response. bodyBytes is what the
// server actually sent (the constant unchanged body), lastBodyBytes the full
// config it replaced; only the difference was avoided. Counting the whole prior
// body as avoided and the unchanged body as nothing would overstate the savings
// percentage on every unchanged response.
func (s *Stats) RecordConfigNotModified(bodyBytes, lastBodyBytes int64) {
	s.l.Lock()
	defer s.l.Unlock()
	s.configNotModified++
	s.configConditionalRequests++
	s.configResponseBodyBytes += bodyBytes
	if avoided := lastBodyBytes - bodyBytes; avoided > 0 {
		s.configEstimatedSavedBytes += avoided
	}
}

func (s *Stats) IncrementConfigETagDrift() {
	s.l.Lock()
	defer s.l.Unlock()
	s.configETagDrift++
}

// Getters for ETag stats (used by tests)
func (s *Stats) ConfigFullResponses() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configFullResponses
}

func (s *Stats) ConfigNotModified() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configNotModified
}

func (s *Stats) ConfigConditionalRequests() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configConditionalRequests
}

func (s *Stats) ConfigResponseBodyBytes() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configResponseBodyBytes
}

func (s *Stats) ConfigEstimatedSavedBytes() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configEstimatedSavedBytes
}

func (s *Stats) ConfigETagDrift() int64 {
	s.l.Lock()
	defer s.l.Unlock()
	return s.configETagDrift
}

// UpdateWebSocketConnected adjusts the gauge of currently connected WebSocket
// transports by delta (+1 on connect, -1 on disconnect).
func (s *Stats) UpdateWebSocketConnected(delta int) {
	s.l.Lock()
	defer s.l.Unlock()
	s.websocketConnected += delta
	if s.websocketConnected < 0 {
		s.websocketConnected = 0
	}
}

func (s *Stats) IncrementWebSocketConnects() {
	s.l.Lock()
	defer s.l.Unlock()
	s.websocketConnects++
}

func (s *Stats) IncrementWebSocketNotifications() {
	s.l.Lock()
	defer s.l.Unlock()
	s.websocketNotifications++
}

func (s *Stats) IncrementWebSocketErrors() {
	s.l.Lock()
	defer s.l.Unlock()
	s.websocketErrors++
}

func (s *Stats) Log() {
	s.l.Lock()
	defer s.l.Unlock()

	var errorRate float64
	if s.osqueryEnrollments > 0 {
		errorRate = float64(s.errors) / float64(s.osqueryEnrollments)
	}

	// Estimated config body savings percentage from ETag 304s.
	var configSavingsPct float64
	baseline := s.configResponseBodyBytes + s.configEstimatedSavedBytes
	if baseline > 0 {
		configSavingsPct = float64(s.configEstimatedSavedBytes) / float64(baseline) * 100.0
	}

	var b strings.Builder

	// deviceUser formats a device/user metric pair as "device (user N)".
	deviceUser := func(device, user int) string {
		return fmt.Sprintf("%d (user %d)", device, user)
	}

	fmt.Fprintf(&b, "osquery-perf stats — uptime: %s\n", time.Since(s.StartTime).Round(time.Second))

	// --- Host / General -----------------------------------------------------
	b.WriteString("  [Host/General]\n")
	fmt.Fprintf(&b, "    error rate:          %.2f\n", errorRate)
	fmt.Fprintf(&b, "    osquery enrolls:     %d\n", s.osqueryEnrollments)
	fmt.Fprintf(&b, "    orbit enrolls:       %d\n", s.orbitEnrollments)
	fmt.Fprintf(&b, "    distributed:         reads=%d writes=%d (errs: reads=%d writes=%d)\n",
		s.distributedReads, s.distributedWrites, s.distributedReadErrors, s.distributedWriteErrors)
	fmt.Fprintf(&b, "    websocket:           connected=%d connects=%d notifications=%d (errs: %d)\n",
		s.websocketConnected, s.websocketConnects, s.websocketNotifications, s.websocketErrors)
	fmt.Fprintf(&b, "    config requests:     %d (errs: %d)\n", s.configRequests, s.configErrors)
	fmt.Fprintf(&b, "    config etags:        full=%d not_modified=%d conditional=%d drift=%d\n",
		s.configFullResponses, s.configNotModified, s.configConditionalRequests, s.configETagDrift)
	fmt.Fprintf(&b, "    config body bytes:   sent=%d avoided=%d (savings: %.1f%%)\n",
		s.configResponseBodyBytes, s.configEstimatedSavedBytes, configSavingsPct)
	fmt.Fprintf(&b, "    result log requests: %d (errs: %d)\n", s.resultLogRequests, s.resultLogErrors)
	fmt.Fprintf(&b, "    buffered logs:       %d\n", s.bufferedLogs)
	fmt.Fprintf(&b, "    script execs:        %d (errs: %d)\n", s.scriptExecs, s.scriptExecErrs)
	fmt.Fprintf(&b, "    software installs:   %d (errs: %d)\n", s.softwareInstalls, s.softwareInstallErrs)
	fmt.Fprintf(&b, "    orbit errors:        %d\n", s.orbitErrors)
	fmt.Fprintf(&b, "    desktop errors:      %d\n", s.desktopErrors)

	// --- MDM ----------------------------------------------------------------
	b.WriteString("  [MDM]\n")
	fmt.Fprintf(&b, "    enrolls:             %s\n", deviceUser(s.mdmEnrollments, s.mdmUserEnrollments))
	fmt.Fprintf(&b, "    sessions:            %s\n", deviceUser(s.mdmSessions, s.mdmUserSessions))
	fmt.Fprintf(&b, "    on-demand syncs:     %d\n", s.mdmOnDemandSyncs)
	fmt.Fprintf(&b, "    commands received:   %s\n", deviceUser(s.mdmCommandsReceived, s.mdmUserCommandsReceived))
	fmt.Fprintf(&b, "    errors:              %s\n", deviceUser(s.mdmErrors, s.mdmUserErrors))
	fmt.Fprintf(&b, "    scep:                requests=%d success=%d errors=%d\n",
		s.mdmSCEPRequests, s.mdmSCEPSuccess, s.mdmSCEPErrors)

	// DDM sub-types, formatted as "success / errors", each device (user N).
	b.WriteString("    ddm (success / errors):\n")
	fmt.Fprintf(&b, "      tokens:            %s / %s\n",
		deviceUser(s.ddmTokensSuccess, s.ddmUserTokensSuccess), deviceUser(s.ddmTokensErrors, s.ddmUserTokensErrors))
	fmt.Fprintf(&b, "      declaration items: %s / %s\n",
		deviceUser(s.ddmDeclarationItemsSuccess, s.ddmUserDeclarationItemsSuccess), deviceUser(s.ddmDeclarationItemsErrors, s.ddmUserDeclarationItemsErrors))
	fmt.Fprintf(&b, "      activation:        %s / %s\n",
		deviceUser(s.ddmActivationSuccess, s.ddmUserActivationSuccess), deviceUser(s.ddmActivationErrors, s.ddmUserActivationErrors))
	fmt.Fprintf(&b, "      configuration:     %s / %s\n",
		deviceUser(s.ddmConfigurationSuccess, s.ddmUserConfigurationSuccess), deviceUser(s.ddmConfigurationErrors, s.ddmUserConfigurationErrors))
	fmt.Fprintf(&b, "      management:        %d / %d\n", s.ddmManagementSuccess, s.ddmManagementErrors)
	fmt.Fprintf(&b, "      asset:             %s / %s\n",
		deviceUser(s.ddmAssetSuccess, s.ddmUserAssetSuccess), deviceUser(s.ddmAssetErrors, s.ddmUserAssetErrors))
	fmt.Fprintf(&b, "      status:            %s / %s\n",
		deviceUser(s.ddmStatusSuccess, s.ddmUserStatusSuccess), deviceUser(s.ddmStatusErrors, s.ddmUserStatusErrors))

	fmt.Fprintf(&b, "    android:             enrolls=%d status reports=%d command acks=%d cert verifications=%d errors=%d\n",
		s.androidEnrollments, s.androidStatusReports, s.androidCommandAcks, s.androidCertVerifications, s.androidErrors)
	fmt.Fprintf(&b, "    psso:                registrations=%d logins=%d key requests=%d key exchanges=%d errors=%d",
		s.pssoRegistrations, s.pssoLogins, s.pssoKeyRequests, s.pssoKeyExchanges, s.pssoErrors)

	log.Print(b.String())
}

func (s *Stats) RunLoop() {
	ticker := time.Tick(10 * time.Second)
	for range ticker {
		s.Log()
	}
}
