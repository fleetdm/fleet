package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notFound(kind string) *common_mysql.NotFoundError {
	return &common_mysql.NotFoundError{
		ResourceType: kind,
	}
}

func TestGetSoftwareTitleIcon(t *testing.T) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	mockIconStore := s3.SetupTestSoftwareTitleIconStore(t, "software-title-icons-unit-test", "prefix")
	svc.softwareTitleIconStore = mockIconStore

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetSoftwareTitleIconFunc = func(ctx context.Context, teamID, titleID uint) (*fleet.SoftwareTitleIcon, error) {
		if titleID == 1 {
			return &fleet.SoftwareTitleIcon{
				TeamID:          teamID,
				SoftwareTitleID: titleID,
				StorageID:       "mock-storage-id",
				Filename:        "icon.png",
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, notFound("SoftwareTitleIcon"), "get software title icon")
	}
	ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
		if titleID == 3 {
			return &fleet.VPPAppStoreApp{
				IconURL: ptr.String("mock-vpp-icon-url"),
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, notFound("VPPAppMetadata"), "get VPP app metadata")
	}

	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			name: "non-existing software title icon",
			before: func() {
			},
			testFunc: func(t *testing.T) {
				_, _, _, err := svc.GetSoftwareTitleIcon(ctx, 1, 2)
				require.Error(t, err)
			},
		},
		{
			name: "existing software title icon",
			before: func() {
			},
			testFunc: func(t *testing.T) {
				imageBytes, size, filename, err := svc.GetSoftwareTitleIcon(ctx, 1, 1)
				require.NoError(t, err)
				assert.Equal(t, "icon.png", filename)
				assert.Equal(t, int64(14), size)
				assert.Equal(t, "mock-icon-data", string(imageBytes))
			},
		},
		{
			name: "vpp icon fall back",
			before: func() {
			},
			testFunc: func(t *testing.T) {
				_, _, _, err := svc.GetSoftwareTitleIcon(ctx, 1, 3)
				require.Error(t, err)
				var vppIconErr *fleet.VPPIconAvailable
				require.True(t, errors.As(err, &vppIconErr), "error should be VPPIconAvailable")
				require.Equal(t, "mock-vpp-icon-url", vppIconErr.IconURL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			tc.testFunc(t)
		})
	}
}
