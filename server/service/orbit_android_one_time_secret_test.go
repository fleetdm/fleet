package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollOrbitAndroidOneTimeSecret(t *testing.T) {
	const esid = "esid-1"
	androidHostID := uint(42)
	androidInfo := fleet.OrbitHostInfo{HardwareUUID: esid, HardwareSerial: esid, Platform: "android", ComputerName: "Google Pixel"}

	newFixture := func(t *testing.T, androidOneTimeEnrollSecrets bool) (*oneTimeEnrollFixture, *[]softwareWorkerJobArgs) {
		f := newOneTimeEnrollFixtureWithMDM(t, func(mdmConfig *config.MDMConfig) {
			mdmConfig.AndroidOneTimeEnrollSecrets = androidOneTimeEnrollSecrets
			// Another platform's switch keeps the one-time lookup running when Android's is off.
			mdmConfig.AppleOneTimeEnrollSecrets = true
		})
		f.row = fleet.HostOneTimeEnrollSecret{
			ID: 8, Secret: "android-secret", HostID: &androidHostID, TeamID: new(uint(3)), Platform: "android", HardwareUUID: esid,
		}
		f.ds.GetHostOneTimeEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.HostOneTimeEnrollSecret, error) {
			if secret == f.row.Secret {
				r := f.row
				return &r, nil
			}
			return nil, newNotFoundError()
		}
		f.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.AndroidEnabledAndConfigured = true
			return ac, nil
		}
		f.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			return &fleet.Host{ID: id, UUID: esid, Platform: "android"}, nil
		}
		f.ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
			return &android.Enterprise{ID: 1, EnterpriseID: "LC0123"}, nil
		}
		var jobs []softwareWorkerJobArgs
		f.ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.Equal(t, "software_worker", job.Name)
			var args softwareWorkerJobArgs
			require.NoError(t, json.Unmarshal(*job.Args, &args))
			jobs = append(jobs, args)
			return job, nil
		}
		return f, &jobs
	}
	rejectWith := func(f *oneTimeEnrollFixture, reason string, hostID *uint) {
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return nil, &fleet.EnrollmentRejectedError{Reason: reason, HostID: hostID}
		}
	}
	wantResend := softwareWorkerJobArgs{Task: "send_android_fleet_agent_config", HostUUID: esid, EnterpriseName: "enterprises/LC0123"}

	t.Run("the host's secret enrolls into its fleet", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		var received *fleet.DatastoreEnrollOrbitConfig
		f.ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			received = orbitEnrollConfig(opts)
			return &fleet.Host{ID: androidHostID, UUID: esid, Platform: "android"}, nil
		}
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		require.NoError(t, err)
		require.Equal(t, &f.row.ID, received.OneTimeEnrollSecretID)
		require.Equal(t, f.row.TeamID, received.TeamID)
		require.Empty(t, *jobs)
	})

	t.Run("another device's ID is refused", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		info := androidInfo
		info.HardwareUUID = "esid-2"
		_, err := f.svc.EnrollOrbit(f.ctx, info, f.row.Secret, "")
		require.Error(t, err)
		require.Len(t, *f.rejections, 1)
		require.Equal(t, fleet.EnrollmentRejectedOneTimeSecretIdentifierMismatch, (*f.rejections)[0].Reason)
		require.Empty(t, *jobs)
	})

	t.Run("a refused shared secret resends the agent configuration once", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		for range 3 {
			_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
			var authErr *fleet.AuthFailedError
			require.ErrorAs(t, err, &authErr)
		}
		require.Equal(t, []softwareWorkerJobArgs{wantResend}, *jobs)
		require.Len(t, *f.rejections, 1)
	})

	t.Run("concurrent refusals queue one resend", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		var jobsMu sync.Mutex
		newJob := f.ds.NewJobFunc
		f.ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			jobsMu.Lock()
			defer jobsMu.Unlock()
			return newJob(ctx, job)
		}
		svc := f.svc.(validationMiddleware).Service.(*Service)
		atomicKV := &atomicKeyValueStore{
			values: map[string]string{}, barrierSuffix: ":android_agent_config", barrierReaders: 20, allArrived: make(chan struct{}),
		}
		svc.keyValueStore = atomicKV

		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
				assert.Error(t, err)
			})
		}
		wg.Wait()
		require.Equal(t, []softwareWorkerJobArgs{wantResend}, *jobs)
	})

	t.Run("no resend when the rate limit can't be checked", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		f.kv.GetFunc = func(ctx context.Context, key string) (*string, error) { return nil, errors.New("redis down") }
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
		require.Error(t, err)
		require.Empty(t, *jobs)
	})

	t.Run("no resend while Android MDM is off", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		f.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
		rejectWith(f, fleet.EnrollmentRejectedOneTimeSecretSpent, &androidHostID)
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		require.Error(t, err)
		require.Empty(t, *jobs)
		require.False(t, f.ds.GetEnterpriseFuncInvoked)
	})

	t.Run("a failed resend still waits out the window", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID)
		var enterpriseLookups int
		f.ds.GetEnterpriseFunc = func(ctx context.Context) (*android.Enterprise, error) {
			enterpriseLookups++
			return nil, errors.New("database unavailable")
		}
		for range 3 {
			_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
			require.Error(t, err)
		}
		require.Equal(t, 1, enterpriseLookups)
		require.Empty(t, *jobs)
	})

	t.Run("the host's secret is refused while the Android switch is off", func(t *testing.T) {
		// Another platform's switch is on, so the one-time lookup runs and finds the Android secret, but doesn't honor it.
		f, jobs := newFixture(t, false)
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		var authErr *fleet.AuthFailedError
		require.ErrorAs(t, err, &authErr)
		require.False(t, f.ds.EnrollOrbitFuncInvoked)
		require.Empty(t, *jobs)
	})

	t.Run("a spent secret resends the agent configuration", func(t *testing.T) {
		f, jobs := newFixture(t, true)
		rejectWith(f, fleet.EnrollmentRejectedOneTimeSecretSpent, &androidHostID)
		_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, f.row.Secret, "")
		require.Error(t, err)
		require.Equal(t, []softwareWorkerJobArgs{wantResend}, *jobs)
	})

	t.Run("no resend", func(t *testing.T) {
		for _, tc := range []struct {
			name                        string
			androidOneTimeEnrollSecrets bool
			reason                      string
			hostID                      *uint
			platform                    string
		}{
			{"switch off", false, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID, "android"},
			{"not an android host", true, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, &androidHostID, "windows"},
			{"no host, as for a deleted one", true, fleet.EnrollmentRejectedSharedSecretForMDMManagedHost, nil, "android"},
			{"another reason", true, fleet.EnrollmentRejectedHostIdentityCertRequired, &androidHostID, "android"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f, jobs := newFixture(t, tc.androidOneTimeEnrollSecrets)
				f.ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
					return &fleet.Host{ID: id, UUID: esid, Platform: tc.platform}, nil
				}
				rejectWith(f, tc.reason, tc.hostID)
				_, err := f.svc.EnrollOrbit(f.ctx, androidInfo, "shared-secret", "")
				require.Error(t, err)
				require.Empty(t, *jobs)
			})
		}
	})
}

// atomicKeyValueStore is an in-memory key-value store with an atomic set-if-not-exists, like the Redis one. Reads of keys
// with barrierSuffix return only once barrierReaders callers have read (or after a timeout), so concurrent callers that read
// before writing all read before anyone writes.
type atomicKeyValueStore struct {
	mu             sync.Mutex
	values         map[string]string
	barrierSuffix  string
	barrierReaders int
	arrived        int
	allArrived     chan struct{}
}

func (s *atomicKeyValueStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

func (s *atomicKeyValueStore) Get(_ context.Context, key string) (*string, error) {
	s.mu.Lock()
	var value *string
	if v, ok := s.values[key]; ok {
		value = &v
	}
	barrier := s.barrierSuffix != "" && strings.HasSuffix(key, s.barrierSuffix)
	if barrier {
		s.arrived++
		if s.arrived == s.barrierReaders {
			close(s.allArrived)
		}
	}
	s.mu.Unlock()
	if barrier {
		select {
		case <-s.allArrived:
		case <-time.After(2 * time.Second):
		}
	}
	return value, nil
}

func (s *atomicKeyValueStore) SetIfNotExists(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.values[key]; ok {
		return false, nil
	}
	s.values[key] = value
	return true, nil
}

// softwareWorkerJobArgs mirrors the fields of the worker's job arguments that these tests check.
type softwareWorkerJobArgs struct {
	Task           string `json:"task"`
	HostUUID       string `json:"host_uuid,omitempty"`
	EnterpriseName string `json:"enterprise_name,omitempty"`
}
