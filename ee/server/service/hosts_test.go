package service

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHostManagedAccountPasswordAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)

	teamID := uint(1)

	verified := string(fleet.MDMDeliveryVerified)

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		return &fleet.Host{ID: hostID, UUID: "test-uuid", Platform: "darwin", TeamID: &teamID}, nil
	}
	ds.GetHostManagedLocalAccountStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
		return &fleet.HostMDMManagedLocalAccount{Status: &verified, PasswordAvailable: true}, nil
	}
	ds.GetHostManagedLocalAccountPasswordFunc = func(ctx context.Context, hostUUID string) (*fleet.HostManagedLocalAccountPassword, error) {
		return &fleet.HostManagedLocalAccountPassword{}, nil
	}
	ds.MarkManagedLocalAccountPasswordViewedFunc = func(ctx context.Context, hostUUID string) (time.Time, error) {
		return time.Now(), nil
	}
	baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			false,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			false,
		},
		{
			"team observer+, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			false,
		},
		{
			"team gitops, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.GetHostManagedAccountPassword(ctx, 1)
			checkAuthErr(t, tt.shouldFail, err)
		})
	}
}

// TestRotateWindowsManagedLocalAccountPassword covers the Windows branch of the rotate endpoint, which maps the
// datastore's typed errors rather than enqueuing an MDM command.
func TestRotateWindowsManagedLocalAccountPassword(t *testing.T) {
	t.Parallel()

	verified := string(fleet.MDMDeliveryVerified)
	admin := &fleet.User{ID: 42, GlobalRole: new(fleet.RoleAdmin)}

	setup := func(t *testing.T, platform string, acct *fleet.HostMDMManagedLocalAccount) (*mock.Store, *Service, *svcmock.Service, context.Context) {
		ds := new(mock.Store)
		svc, baseSvc := newTestServiceWithMock(t, ds)
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "test-uuid", Platform: platform}, nil
		}
		ds.GetHostManagedLocalAccountStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
			return acct, nil
		}
		ds.InitiateWindowsManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error { return nil }
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error { return nil }
		return ds, svc, baseSvc, viewer.NewContext(context.Background(), viewer.Viewer{User: admin})
	}

	available := &fleet.HostMDMManagedLocalAccount{Status: &verified, PasswordAvailable: true}

	t.Run("records the request and logs the activity against the requesting user", func(t *testing.T) {
		ds, svc, baseSvc, ctx := setup(t, "windows", available)
		var actor *fleet.User
		var logged fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(_ context.Context, u *fleet.User, a fleet.ActivityDetails) error {
			actor, logged = u, a
			return nil
		}

		require.NoError(t, svc.RotateManagedLocalAccountPassword(ctx, 1))

		require.True(t, ds.InitiateWindowsManagedLocalAccountRotationFuncInvoked)
		// No MDM command is involved on Windows, so nothing should reach the Apple rotation path.
		require.False(t, ds.GetManagedLocalAccountUUIDFuncInvoked)
		rotated, ok := logged.(fleet.ActivityTypeRotatedManagedLocalAccountPassword)
		require.True(t, ok)
		assert.False(t, rotated.FleetInitiated, "a manual rotation is the user's, not Fleet's")
		require.NotNil(t, actor)
		assert.Equal(t, admin.ID, actor.ID)
	})

	t.Run("maps the datastore's typed errors to bad requests", func(t *testing.T) {
		cases := []struct {
			name    string
			dsErr   error
			message string
		}{
			{
				"already outstanding",
				fleet.ErrManagedLocalAccountRotationPending,
				"Cannot rotate managed local account password while an operation is pending.",
			},
			{
				"no windows mdm enrollment to carry the notification",
				&testNotFoundError{},
				"Host does not have MDM turned on.",
			},
			{
				"row not eligible",
				fleet.ErrManagedLocalAccountNotEligible,
				"Couldn’t rotate managed local account password. Please try again.",
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ds, svc, _, ctx := setup(t, "windows", available)
				ds.InitiateWindowsManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error {
					return c.dsErr
				}

				err := svc.RotateManagedLocalAccountPassword(ctx, 1)
				require.Error(t, err)
				var badReq *fleet.BadRequestError
				require.ErrorAs(t, err, &badReq)
				assert.Equal(t, c.message, badReq.Message)
			})
		}
	})

	t.Run("rejects a platform that has no managed local account", func(t *testing.T) {
		_, svc, _, ctx := setup(t, "ubuntu", available)
		err := svc.RotateManagedLocalAccountPassword(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Equal(t, "Host is not a macOS or Windows device.", badReq.Message)
	})

	t.Run("rejects while a rotation is already in flight", func(t *testing.T) {
		ds, svc, _, ctx := setup(t, "windows", &fleet.HostMDMManagedLocalAccount{
			Status: &verified, PasswordAvailable: true, PendingRotation: true,
		})
		err := svc.RotateManagedLocalAccountPassword(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Equal(t, "Managed local account password rotation is already in progress for this host.", badReq.Message)
		assert.False(t, ds.InitiateWindowsManagedLocalAccountRotationFuncInvoked)
	})
}

type rotateFileVaultKeyCommander struct {
	fleet.MDMAppleCommandIssuer
	calls []string
	err   error
}

func (c *rotateFileVaultKeyCommander) RotateFileVaultKey(ctx context.Context, hostUUID, cmdUUID string, replyCertDER []byte) error {
	c.calls = append(c.calls, cmdUUID)
	return c.err
}

func TestRotateDiskEncryptionKey(t *testing.T) {
	t.Parallel()

	caCert, caKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	caCertPEM := tokenpki.PEMCertificate(caCert.Raw)
	caKeyPEM := tokenpki.PEMRSAPrivateKey(caKey)
	encrypted, err := pkcs7.Encrypt([]byte("ABCD-EFGH"), []*x509.Certificate{caCert})
	require.NoError(t, err)
	decryptableKey := func() *fleet.HostDiskEncryptionKey {
		return &fleet.HostDiskEncryptionKey{Base64Encrypted: base64.StdEncoding.EncodeToString(encrypted), Decryptable: new(true)}
	}

	teamID := uint(1)
	admin := &fleet.User{ID: 42, GlobalRole: new(fleet.RoleAdmin)}

	type env struct {
		ds        *mock.Store
		svc       *Service
		baseSvc   *svcmock.Service
		commander *rotateFileVaultKeyCommander
		host      *fleet.Host
		key       *fleet.HostDiskEncryptionKey
		keyErr    error
		escrow    bool
		marker    *string
		pending   bool
		acts      []fleet.ActivityDetails
	}
	setup := func(t *testing.T) *env {
		e := &env{
			ds:        new(mock.Store),
			commander: &rotateFileVaultKeyCommander{},
			host:      &fleet.Host{ID: 1, UUID: "host-uuid", Platform: "darwin", TeamID: &teamID},
			key:       decryptableKey(),
			escrow:    true,
		}
		e.svc, e.baseSvc = newTestServiceWithMock(t, e.ds)
		e.svc.mdmAppleCommander = e.commander
		e.svc.logger = slog.New(slog.DiscardHandler)
		e.baseSvc.VerifyMDMAppleConfiguredFunc = func(ctx context.Context) error { return nil }
		e.baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			require.Equal(t, admin.ID, user.ID)
			e.acts = append(e.acts, activity)
			return nil
		}
		e.ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
			if id != e.host.ID {
				return nil, &testNotFoundError{}
			}
			return e.host, nil
		}
		e.ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) { return true, nil }
		e.ds.GetConfigEnableDiskEncryptionFunc = func(ctx context.Context, teamID *uint) (fleet.DiskEncryptionConfig, error) {
			return fleet.DiskEncryptionConfig{MacOSEscrowEnabled: e.escrow}, nil
		}
		e.ds.GetHostDiskEncryptionKeyFunc = func(ctx context.Context, hostID uint) (*fleet.HostDiskEncryptionKey, error) {
			if e.keyErr != nil {
				return nil, e.keyErr
			}
			k := *e.key
			k.RotationCommandUUID = e.marker
			return &k, nil
		}
		e.ds.IsAppleMDMCommandPendingFunc = func(ctx context.Context, hostUUID, cmdUUID string) (bool, error) {
			return e.pending, nil
		}
		e.ds.ClearHostDiskEncryptionKeyRotationCommandFunc = func(ctx context.Context, hostID uint, cmdUUID string) error {
			if e.marker != nil && *e.marker == cmdUUID {
				e.marker = nil
			}
			return nil
		}
		e.ds.SetHostDiskEncryptionKeyRotationCommandFunc = func(ctx context.Context, hostID uint, cmdUUID string) (bool, error) {
			if e.marker != nil {
				return false, nil
			}
			e.marker = &cmdUUID
			return true, nil
		}
		e.ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext,
		) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetCACert: {Name: fleet.MDMAssetCACert, Value: caCertPEM},
				fleet.MDMAssetCAKey:  {Name: fleet.MDMAssetCAKey, Value: caKeyPEM},
			}, nil
		}
		e.ds.GetAllMDMConfigAssetsByNameIncludingDeletedFunc = func(ctx context.Context, _ []fleet.MDMAssetName) ([]fleet.MDMConfigAsset, error) {
			return []fleet.MDMConfigAsset{{Name: fleet.MDMAssetCACert, Value: caCertPEM}}, nil
		}
		return e
	}
	adminCtx := func() context.Context { return viewer.NewContext(context.Background(), viewer.Viewer{User: admin}) }
	requireStatus := func(t *testing.T, err error, status int, msg string) {
		t.Helper()
		var ume *fleet.UserMessageError
		require.ErrorAs(t, err, &ume)
		require.Equal(t, status, ume.StatusCode())
		require.Contains(t, ume.Error(), msg)
	}

	t.Run("enqueues the command, records the marker and the activity", func(t *testing.T) {
		e := setup(t)
		require.NoError(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1))
		require.Len(t, e.commander.calls, 1)
		require.NotNil(t, e.marker)
		require.Equal(t, e.commander.calls[0], *e.marker)
		require.Equal(t, []fleet.ActivityDetails{fleet.ActivityTypeRotatedDiskEncryptionKey{
			HostID: 1, HostDisplayName: e.host.DisplayName(),
		}}, e.acts)
	})

	t.Run("bad requests", func(t *testing.T) {
		for _, platform := range []string{"windows", "ubuntu", "ios"} {
			e := setup(t)
			e.host.Platform = platform
			var bre *fleet.BadRequestError
			require.ErrorAs(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), &bre, platform)
			require.Contains(t, bre.Message, "only supported on macOS")
		}

		e := setup(t)
		e.ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) { return false, nil }
		var bre *fleet.BadRequestError
		require.ErrorAs(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), &bre)
		require.Contains(t, bre.Message, "enrolled in Fleet MDM")
	})

	t.Run("MDM not configured", func(t *testing.T) {
		e := setup(t)
		e.baseSvc.VerifyMDMAppleConfiguredFunc = func(ctx context.Context) error { return fleet.ErrMDMNotConfigured }
		require.ErrorIs(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), fleet.ErrMDMNotConfigured)
	})

	t.Run("unprocessable preconditions", func(t *testing.T) {
		cases := []struct {
			name   string
			modify func(e *env)
			msg    string
		}{
			{"escrow off", func(e *env) { e.escrow = false }, "escrow is not turned on"},
			{"no key row", func(e *env) { e.keyErr = &testNotFoundError{} }, "does not have a disk encryption key"},
			{"empty key", func(e *env) { e.key.Base64Encrypted = "" }, "does not have a disk encryption key"},
			{"decryptable unknown", func(e *env) { e.key.Decryptable = nil }, "not decryptable"},
			{"decryptable false", func(e *env) { e.key.Decryptable = new(false) }, "not decryptable"},
			{"cms fails", func(e *env) { e.key.Base64Encrypted = base64.StdEncoding.EncodeToString([]byte("junk")) }, "not decryptable"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				e := setup(t)
				c.modify(e)
				requireStatus(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), http.StatusUnprocessableEntity, c.msg)
				require.Empty(t, e.commander.calls)
				require.Nil(t, e.marker, "a failed precondition leaves no marker")
				require.Empty(t, e.acts)
			})
		}
	})

	t.Run("pending rotation conflicts", func(t *testing.T) {
		e := setup(t)
		e.marker, e.pending = new("in-flight"), true
		var conflict *fleet.ConflictError
		require.ErrorAs(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), &conflict)
		require.Empty(t, e.commander.calls)
		require.Equal(t, "in-flight", *e.marker)
	})

	t.Run("lost race conflicts", func(t *testing.T) {
		e := setup(t)
		e.ds.SetHostDiskEncryptionKeyRotationCommandFunc = func(ctx context.Context, hostID uint, cmdUUID string) (bool, error) {
			return false, nil
		}
		var conflict *fleet.ConflictError
		require.ErrorAs(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1), &conflict)
		require.Empty(t, e.commander.calls)
	})

	t.Run("stale marker is replaced", func(t *testing.T) {
		e := setup(t)
		e.marker = new("stale")
		require.NoError(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1))
		require.Len(t, e.commander.calls, 1)
		require.Equal(t, e.commander.calls[0], *e.marker)
	})

	t.Run("enqueue failure clears the marker", func(t *testing.T) {
		e := setup(t)
		e.commander.err = errors.New("enqueue failed")
		require.Error(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1))
		require.Nil(t, e.marker)
		require.Empty(t, e.acts)
	})

	t.Run("push failure keeps the marker", func(t *testing.T) {
		e := setup(t)
		e.commander.err = ctxerr.Wrap(context.Background(), &apple_mdm.NotificationFailedError{}, "sending notifications")
		require.Error(t, e.svc.RotateDiskEncryptionKey(adminCtx(), 1))
		require.NotNil(t, e.marker)
	})

	t.Run("authorization", func(t *testing.T) {
		cases := []struct {
			name     string
			user     *fleet.User
			wantErr  bool
			notFound bool
		}{
			{"global admin", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleAdmin)}, false, false},
			{"global maintainer", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleMaintainer)}, false, false},
			// consistent with lock, wipe, and Recovery Lock rotation: gitops can't list hosts
			{"global gitops", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleGitOps)}, true, false},
			{"global observer", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleObserver)}, true, false},
			{"global observer+", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleObserverPlus)}, true, false},
			{"global technician", &fleet.User{ID: 42, GlobalRole: new(fleet.RoleTechnician)}, true, false},
			{"team admin", &fleet.User{ID: 42, Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleAdmin}}}, false, false},
			{"team maintainer", &fleet.User{ID: 42, Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleMaintainer}}}, false, false},
			{"team observer", &fleet.User{ID: 42, Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleObserver}}}, true, false},
			{"team technician", &fleet.User{ID: 42, Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleTechnician}}}, true, false},
			{"other team admin", &fleet.User{ID: 42, Teams: []fleet.UserTeam{{ID: 2, Role: fleet.RoleAdmin}}}, true, true},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				e := setup(t)
				e.baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error { return nil }
				err := e.svc.RotateDiskEncryptionKey(viewer.NewContext(context.Background(), viewer.Viewer{User: c.user}), 1)
				switch {
				case c.notFound:
					require.True(t, fleet.IsNotFound(err), "%v", err)
				case c.wantErr:
					checkAuthErr(t, true, err)
				default:
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestLockWipeRejectPersonalAppleHosts(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, _ := newTestServiceWithMock(t, ds)
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	for _, platform := range []string{"darwin", "ios", "ipados"} {
		for _, status := range []string{fleet.MDMEnrollmentStatusPersonal, fleet.MDMEnrollmentStatusManualPersonal} {
			t.Run(platform+" "+status, func(t *testing.T) {
				ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
					return &fleet.Host{ID: id, Platform: platform, MDM: fleet.MDMHostData{EnrollmentStatus: new(status)}}, nil
				}

				_, err := svc.LockHost(ctx, 1, false)
				require.ErrorContains(t, err, fleet.CantLockPersonalHostsMessage)

				err = svc.WipeHost(ctx, 1, nil)
				require.ErrorContains(t, err, fleet.CantWipePersonalHostsMessage)
			})
		}
	}
}
