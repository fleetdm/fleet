package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	scep_depot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	fleet_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/fleetdm/fleet/v4/server/service/async"
)

// This file holds test-helper types and constants that do NOT import the
// "testing" package. By keeping these here (instead of in
// testing_utils.go / testing_client.go), the production binary doesn't pull
// in the testing package transitively.
//
// Functions and methods that take *testing.T live in
// server/service/svctest, which is only ever imported from test code.

const (
	TestAdminUserEmail      = "admin1@example.com"
	TestMaintainerUserEmail = "user1@example.com"
	TestObserverUserEmail   = "user2@example.com"
)

// TestNewScheduleFunc is the signature used by tests that want to start a
// cron schedule against a real datastore.
type TestNewScheduleFunc func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc

// HostIdentity combines host identity-related test options.
type HostIdentity struct {
	SCEPStorage                 scep_depot.Depot
	RequireHTTPMessageSignature bool
}

// ConditionalAccess combines conditional access-related test options.
type ConditionalAccess struct {
	SCEPStorage scep_depot.Depot
}

// TestServerOpts configures the test fleet service / server. It is used by
// internal tests in this package and by external test packages (see
// svctest.RunServerForTestsWithDS, svctest.NewTestService).
type TestServerOpts struct {
	Logger                          *slog.Logger
	License                         *fleet.LicenseInfo
	SkipCreateTestUsers             bool
	Rs                              fleet.QueryResultStore
	Lq                              fleet.LiveQueryStore
	Pool                            fleet.RedisPool
	FailingPolicySet                fleet.FailingPolicySet
	Clock                           clock.Clock
	Task                            *async.Task
	EnrollHostLimiter               fleet.EnrollHostLimiter
	Is                              fleet.InstallerStore
	FleetConfig                     *config.FleetConfig
	MDMStorage                      fleet.MDMAppleStore
	DEPStorage                      nanodep_storage.AllDEPStorage
	SCEPStorage                     scep_depot.Depot
	MDMPusher                       nanomdm_push.Pusher
	HTTPServerConfig                *http.Server
	StartCronSchedules              []TestNewScheduleFunc
	UseMailService                  bool
	APNSTopic                       string
	ProfileMatcher                  fleet.ProfileMatcher
	EnableCachedDS                  bool
	NoCacheDatastore                bool
	SoftwareInstallStore            fleet.SoftwareInstallerStore
	BootstrapPackageStore           fleet.MDMBootstrapPackageStore
	SoftwareTitleIconStore          fleet.SoftwareTitleIconStore
	KeyValueStore                   fleet.KeyValueStore
	EnableSCEPProxy                 bool
	WithDEPWebview                  bool
	FeatureRoutes                   []endpointer.HandlerRoutesFunc
	SCEPConfigService               fleet.SCEPConfigService
	DigiCertService                 fleet.DigiCertService
	EnableSCIM                      bool
	ConditionalAccessMicrosoftProxy ConditionalAccessMicrosoftProxy
	HostIdentity                    *HostIdentity
	AndroidMockClient               *android_mock.Client
	AndroidModule                   android.Service
	ConditionalAccess               *ConditionalAccess
	DBConns                         *common_mysql.DBConnections

	// ActivityMock is populated automatically when a test service is built.
	// After setup, tests can use it to intercept or assert on activity creation.
	ActivityMock *fleet_mock.MockActivityService

	// MicrosoftGraphClientFactory overrides the Graph client used to verify a credential on config write.
	MicrosoftGraphClientFactory msgraph.ClientFactory

	ACMECertCA  *x509.Certificate
	ACMECertKey *ecdsa.PrivateKey
}

// memFailingPolicySet is an in-memory fleet.FailingPolicySet used by tests.
type memFailingPolicySet struct {
	mMu sync.RWMutex
	m   map[uint][]fleet.PolicySetHost
}

var _ fleet.FailingPolicySet = (*memFailingPolicySet)(nil)

// NewMemFailingPolicySet returns a new in-memory FailingPolicySet.
func NewMemFailingPolicySet() *memFailingPolicySet {
	return &memFailingPolicySet{
		m: make(map[uint][]fleet.PolicySetHost),
	}
}

// AddHost adds the given host to the policy set.
func (m *memFailingPolicySet) AddHost(policyID uint, host fleet.PolicySetHost) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	m.m[policyID] = append(m.m[policyID], host)
	return nil
}

// ListHosts returns the list of hosts present in the policy set.
func (m *memFailingPolicySet) ListHosts(policyID uint) ([]fleet.PolicySetHost, error) {
	m.mMu.RLock()
	defer m.mMu.RUnlock()

	hosts := make([]fleet.PolicySetHost, len(m.m[policyID]))
	copy(hosts, m.m[policyID])
	return hosts, nil
}

// RemoveHosts removes the hosts from the policy set.
func (m *memFailingPolicySet) RemoveHosts(policyID uint, hosts []fleet.PolicySetHost) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	if _, ok := m.m[policyID]; !ok {
		return nil
	}
	hostsSet := make(map[uint]struct{})
	for _, host := range hosts {
		hostsSet[host.ID] = struct{}{}
	}
	n := 0
	for _, host := range m.m[policyID] {
		if _, ok := hostsSet[host.ID]; !ok {
			m.m[policyID][n] = host
			n++
		}
	}
	m.m[policyID] = m.m[policyID][:n]
	return nil
}

// RemoveSet removes a policy set.
func (m *memFailingPolicySet) RemoveSet(policyID uint) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	delete(m.m, policyID)
	return nil
}

// ListSets lists all the policy sets.
func (m *memFailingPolicySet) ListSets() ([]uint, error) {
	m.mMu.RLock()
	defer m.mMu.RUnlock()

	var policyIDs []uint
	for policyID := range m.m {
		policyIDs = append(policyIDs, policyID)
	}
	return policyIDs, nil
}

// memSoftwareInstallAttemptStore is an in-memory fleet.SoftwareInstallAttemptStore used by tests.
type memSoftwareInstallAttemptStore struct {
	mMu sync.RWMutex
	m   map[string]map[string]time.Time
}

var _ fleet.SoftwareInstallAttemptStore = (*memSoftwareInstallAttemptStore)(nil)

// NewMemSoftwareInstallAttemptStore returns a new in-memory SoftwareInstallAttemptStore.
func NewMemSoftwareInstallAttemptStore() *memSoftwareInstallAttemptStore {
	return &memSoftwareInstallAttemptStore{
		m: make(map[string]map[string]time.Time),
	}
}

func memInstallAttemptKey(hostID uint, softwareInstallerID uint) string {
	return fmt.Sprintf("%d:%d", hostID, softwareInstallerID)
}

// RecordAttempt records an attempt and returns how many fall within the window.
func (m *memSoftwareInstallAttemptStore) RecordAttempt(ctx context.Context, hostID uint, softwareInstallerID uint,
	executionID string, now time.Time, window time.Duration,
) (int, error) {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	key := memInstallAttemptKey(hostID, softwareInstallerID)
	if m.m[key] == nil {
		m.m[key] = make(map[string]time.Time)
	}
	cutoff := now.Add(-window)
	for execID, at := range m.m[key] {
		if !at.After(cutoff) {
			delete(m.m[key], execID)
		}
	}
	m.m[key][executionID] = now
	return len(m.m[key]), nil
}

// CountAttempts returns how many recorded attempts fall within the window.
func (m *memSoftwareInstallAttemptStore) CountAttempts(ctx context.Context, hostID uint, softwareInstallerID uint,
	now time.Time, window time.Duration,
) (int, error) {
	m.mMu.RLock()
	defer m.mMu.RUnlock()

	cutoff := now.Add(-window)
	var count int
	for _, at := range m.m[memInstallAttemptKey(hostID, softwareInstallerID)] {
		if at.After(cutoff) {
			count++
		}
	}
	return count, nil
}

// ClearAttempts drops every recorded attempt for this host and installer.
func (m *memSoftwareInstallAttemptStore) ClearAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	delete(m.m, memInstallAttemptKey(hostID, softwareInstallerID))
	return nil
}
