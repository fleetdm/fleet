package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

// TODO: Revisit this test, as it seems rather useless (at least the success case) due to it's simplicity
// and not being possible atm. to mock/call free service methods.
func TestDeleteCertificateAuthority(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	ctx := context.Background()
	svc := newTestService(t, ds)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	t.Run("successfully deletes certificate", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthority, error) {
			return nil, errors.New("forced error to short-circuit activity creation")
		}
		err := svc.DeleteCertificateAuthority(ctx, 1)
		require.Error(t, err)
		require.Equal(t, "forced error to short-circuit activity creation", err.Error())
	})

	t.Run("returns not found error if certificate authority does not exist", func(t *testing.T) {
		ds.DeleteCertificateAuthorityFunc = func(ctx context.Context, certificateAuthorityID uint) (*fleet.CertificateAuthority, error) {
			return nil, common_mysql.NotFound("certificate authority")
		}
		err := svc.DeleteCertificateAuthority(ctx, 999)
		require.Error(t, err)
		require.Contains(t, err.Error(), "certificate authority was not found")
	})
}
