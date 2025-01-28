package osquery_perf

import (
	"log"
	"sync"
	"time"
)

type Stats struct {
	StartTime                  time.Time
	errors                     int
	osqueryEnrollments         int
	orbitEnrollments           int
	mdmEnrollments             int
	mdmSessions                int
	distributedWrites          int
	mdmCommandsReceived        int
	distributedReads           int
	configRequests             int
	configErrors               int
	resultLogRequests          int
	orbitErrors                int
	mdmErrors                  int
	ddmDeclarationItemsErrors  int
	ddmConfigurationErrors     int
	ddmActivationErrors        int
	ddmStatusErrors            int
	ddmDeclarationItemsSuccess int
	ddmConfigurationSuccess    int
	ddmActivationSuccess       int
	ddmStatusSuccess           int
	desktopErrors              int
	distributedReadErrors      int
	distributedWriteErrors     int
	resultLogErrors            int
	bufferedLogs               int

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

func (s *Stats) IncrementMDMSessions() {
	s.l.Lock()
	defer s.l.Unlock()
	s.mdmSessions++
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

func (s *Stats) IncrementDDMStatusSuccess() {
	s.l.Lock()
	defer s.l.Unlock()
	s.ddmStatusSuccess++
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
		"uptime: %s, error rate: %.2f, osquery enrolls: %d, orbit enrolls: %d, mdm enrolls: %d, distributed/reads: %d, distributed/writes: %d, config requests: %d, result log requests: %d, mdm sessions initiated: %d, mdm commands received: %d, config errors: %d, distributed/read errors: %d, distributed/write errors: %d, log result errors: %d, orbit errors: %d, desktop errors: %d, mdm errors: %d, ddm declaration items success: %d, ddm declaration items errors: %d, ddm activation success: %d, ddm activation errors: %d, ddm configuration success: %d, ddm configuration errors: %d, ddm status success: %d, ddm status errors: %d, buffered logs: %d",
		time.Since(s.StartTime).Round(time.Second),
		float64(s.errors)/float64(s.osqueryEnrollments),
		s.osqueryEnrollments,
		s.orbitEnrollments,
		s.mdmEnrollments,
		s.distributedReads,
		s.distributedWrites,
		s.configRequests,
		s.resultLogRequests,
		s.mdmSessions,
		s.mdmCommandsReceived,
		s.configErrors,
		s.distributedReadErrors,
		s.distributedWriteErrors,
		s.resultLogErrors,
		s.orbitErrors,
		s.desktopErrors,
		s.mdmErrors,
		s.ddmDeclarationItemsSuccess,
		s.ddmDeclarationItemsErrors,
		s.ddmActivationSuccess,
		s.ddmActivationErrors,
		s.ddmConfigurationSuccess,
		s.ddmConfigurationErrors,
		s.ddmStatusSuccess,
		s.ddmStatusErrors,
		s.bufferedLogs,
	)
}

func (s *Stats) RunLoop() {
	ticker := time.Tick(10 * time.Second)
	for range ticker {
		s.Log()
	}
}
