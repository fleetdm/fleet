package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestVPPSelfServiceRestrictions(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	t.Run("AddAppStoreApp", func(t *testing.T) {
		t.Run("fails for iOS with self-service", func(t *testing.T) {
			_, err := svc.AddAppStoreApp(ctx, ptr.Uint(1), fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					Platform: fleet.IOSPlatform,
				},
				SelfService: true,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "Self-service is not supported for iOS and iPadOS apps")
		})

		t.Run("fails for iPadOS with self-service", func(t *testing.T) {
			_, err := svc.AddAppStoreApp(ctx, ptr.Uint(1), fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					Platform: fleet.IPadOSPlatform,
				},
				SelfService: true,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "Self-service is not supported for iOS and iPadOS apps")
		})
	})

	t.Run("UpdateAppStoreApp", func(t *testing.T) {
		ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
			return &fleet.VPPAppStoreApp{
				VPPAppID: fleet.VPPAppID{
					Platform: fleet.IOSPlatform,
				},
				SelfService: false,
			}, nil
		}
		ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			return &fleet.Team{ID: 1, Name: "team"}, nil
		}

		t.Run("fails for iOS with self-service", func(t *testing.T) {
			_, err := svc.UpdateAppStoreApp(ctx, 1, ptr.Uint(1), ptr.Bool(true), nil, nil, nil, nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "Self-service is not supported for iOS and iPadOS apps")
		})
	})
}
