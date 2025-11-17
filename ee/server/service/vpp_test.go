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
				},
			}, false)
			require.ErrorContains(t, err, "could not retrieve vpp token")
		})
	})
}
