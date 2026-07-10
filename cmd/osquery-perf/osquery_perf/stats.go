package osquery_perf

import (
	"log"
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
	ddmActivationErrors            int
	ddmAssetErrors                 int
	ddmStatusErrors                int
	ddmDeclarationItemsSuccess     int
	ddmConfigurationSuccess        int
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

func (s *Stats) Log() {
	s.l.Lock()
	defer s.l.Unlock()

	log.Printf(
		"uptime: %s, error rate: %.2f, osquery enrolls: %d, orbit enrolls: %d, mdm enrolls: %d, mdm user enrolls: %d, distributed/reads: %d, distributed/writes: %d, config requests: %d, result log requests: %d, mdm sessions initiated: %d, mdm user sessions initiated: %d, mdm on-demand syncs: %d, mdm commands received: %d, mdm user commands received: %d, config errors: %d, distributed/read errors: %d, distributed/write errors: %d, log result errors: %d, orbit errors: %d, desktop errors: %d, mdm errors: %d, mdm user errors: %d, mdm scep requests: %d, mdm scep success: %d, mdm scep errors: %d, ddm tokens success: %d, ddm user tokens success: %d, ddm tokens errors: %d, ddm user tokens errors: %d, ddm declaration items success: %d, ddm user declaration items success: %d, ddm declaration items errors: %d, ddm user declaration items errors: %d, ddm activation success: %d, ddm user activation success: %d, ddm activation errors: %d, ddm user activation errors: %d, ddm configuration success: %d, ddm user configuration success: %d, ddm configuration errors: %d, ddm user configuration errors: %d, ddm asset success: %d, ddm user asset success: %d, ddm asset errors: %d, ddm user asset errors: %d, ddm status success: %d, ddm user status success: %d, ddm status errors: %d, ddm user status errors: %d, buffered logs: %d, script execs (errs): %d (%d), software installs (errs): %d (%d)",
		time.Since(s.StartTime).Round(time.Second),
		float64(s.errors)/float64(s.osqueryEnrollments),
		s.osqueryEnrollments,
		s.orbitEnrollments,
		s.mdmEnrollments,
		s.mdmUserEnrollments,
		s.distributedReads,
		s.distributedWrites,
		s.configRequests,
		s.resultLogRequests,
		s.mdmSessions,
		s.mdmUserSessions,
		s.mdmOnDemandSyncs,
		s.mdmCommandsReceived,
		s.mdmUserCommandsReceived,
		s.configErrors,
		s.distributedReadErrors,
		s.distributedWriteErrors,
		s.resultLogErrors,
		s.orbitErrors,
		s.desktopErrors,
		s.mdmErrors,
		s.mdmUserErrors,
		s.mdmSCEPRequests,
		s.mdmSCEPSuccess,
		s.mdmSCEPErrors,
		s.ddmTokensSuccess,
		s.ddmUserTokensSuccess,
		s.ddmTokensErrors,
		s.ddmUserTokensErrors,
		s.ddmDeclarationItemsSuccess,
		s.ddmUserDeclarationItemsSuccess,
		s.ddmDeclarationItemsErrors,
		s.ddmUserDeclarationItemsErrors,
		s.ddmActivationSuccess,
		s.ddmUserActivationSuccess,
		s.ddmActivationErrors,
		s.ddmUserActivationErrors,
		s.ddmConfigurationSuccess,
		s.ddmUserConfigurationSuccess,
		s.ddmConfigurationErrors,
		s.ddmUserConfigurationErrors,
		s.ddmAssetSuccess,
		s.ddmUserAssetSuccess,
		s.ddmAssetErrors,
		s.ddmUserAssetErrors,
		s.ddmStatusSuccess,
		s.ddmUserStatusSuccess,
		s.ddmStatusErrors,
		s.ddmUserStatusErrors,
		s.bufferedLogs,
		s.scriptExecs,
		s.scriptExecErrs,
		s.softwareInstalls,
		s.softwareInstallErrs,
	)
}

func (s *Stats) RunLoop() {
	ticker := time.Tick(10 * time.Second)
	for range ticker {
		s.Log()
	}
}
