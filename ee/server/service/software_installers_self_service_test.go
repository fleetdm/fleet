package service

import (
	"context"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallerSelfServiceRestrictions(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	ctx = authz_ctx.NewContext(ctx, &authz_ctx.AuthorizationContext{})

	t.Run("UpdateSoftwareInstaller", func(t *testing.T) {
		ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, filter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
			return &fleet.SoftwareTitle{
				SoftwareInstallersCount: 1,
				Name:                    "Test App",
			}, nil
		}
		ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{
				Extension:   "ipa",
				SelfService: false,
			}, nil
		}
		ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, scripts []string) error {
			return nil
		}
		ds.TeamWithoutExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			return &fleet.Team{ID: 1, Name: "team"}, nil
		}

		t.Run("fails for ipa with self-service", func(t *testing.T) {
			_, err := svc.UpdateSoftwareInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{
				TitleID:     1,
				TeamID:      ptr.Uint(1),
				SelfService: ptr.Bool(true),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "Self-service is not supported for iOS and iPadOS apps")
		})
	})

	t.Run("updateInHouseAppInstaller", func(t *testing.T) {
		ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, filter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
			return &fleet.SoftwareTitle{
				InHouseAppCount: 1,
				Name:            "Test App",
			}, nil
		}
		ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{
				Extension:   "ipa",
				SelfService: false,
			}, nil
		}

		t.Run("fails for ipa with self-service", func(t *testing.T) {
			_, err := svc.UpdateSoftwareInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{
				TitleID:     1,
				TeamID:      ptr.Uint(1),
				SelfService: ptr.Bool(true),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "Self-service is not supported for iOS and iPadOS apps")
		})
	})
}
