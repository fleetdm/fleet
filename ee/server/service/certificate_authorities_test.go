package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestDeleteCertificateAuthority(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	ctx := context.Background()
	svc := newTestService(t, ds)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	t.Run("successfully deletes certificate", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID int64) error {
			return nil
		}
		err := svc.DeleteCertificateAuthority(ctx, 1)
		require.NoError(t, err)
	})

	t.Run("returns not found error if certificate authority does not exist", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID int64) error {
			return common_mysql.NotFound("certificate authority")
		}
		err := svc.DeleteCertificateAuthority(ctx, 999)
		require.Error(t, err)
		require.Contains(t, err.Error(), "certificate authority was not found")
	})
}
