package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestBatchAssociateVPPApps(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	t.Run("Fails if missing VPP token when payloads to associate", func(t *testing.T) {
		ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
			return nil, sql.ErrNoRows
		}
		t.Run("dry run", func(t *testing.T) {
			_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
				{
					AppStoreID:       "my-fake-app",
					LabelsExcludeAny: []string{},
					LabelsIncludeAny: []string{},
					Categories:       []string{},
					Platform:         fleet.MacOSPlatform,
				},
			}, true)
			require.ErrorContains(t, err, "could not retrieve vpp token")
		})
		t.Run("not dry run", func(t *testing.T) {
			_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
				{
					AppStoreID:       "my-fake-app",
					LabelsExcludeAny: []string{},
					LabelsIncludeAny: []string{},
					Categories:       []string{},
					Platform:         fleet.MacOSPlatform,
				},
			}, false)
			require.ErrorContains(t, err, "could not retrieve vpp token")
		})
	})

	t.Run("Fails for Fleet Agent Android apps via GitOps", func(t *testing.T) {
		ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
			return nil, nil
		}

		fleetAgentPackages := []string{
			"com.fleetdm.agent",
			"com.fleetdm.agent.pingali",
			"com.fleetdm.agent.private.testuser",
		}

		for _, pkg := range fleetAgentPackages {
			t.Run(pkg+" dry run", func(t *testing.T) {
				_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
					{
						AppStoreID:       pkg,
						LabelsExcludeAny: []string{},
						LabelsIncludeAny: []string{},
						Categories:       []string{},
						Platform:         fleet.AndroidPlatform,
					},
				}, true)
				require.ErrorContains(t, err, "The Fleet agent cannot be added manually")
			})
			t.Run(pkg+" not dry run", func(t *testing.T) {
				_, err := svc.BatchAssociateVPPApps(ctx, "", []fleet.VPPBatchPayload{
					{
						AppStoreID:       pkg,
						LabelsExcludeAny: []string{},
						LabelsIncludeAny: []string{},
						Categories:       []string{},
						Platform:         fleet.AndroidPlatform,
					},
				}, false)
				require.ErrorContains(t, err, "The Fleet agent cannot be added manually")
			})
		}
	})
}
