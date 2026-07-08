package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/require"
)

func TestListAppleDDMAssets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	t.Run("Observer cannot list DDM assets", func(t *testing.T) {
		ds.ListAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.DDMAsset, error) {
			return []*fleet.DDMAsset{}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})

		_, err := svc.ListAppleDDMAssets(ctx, nil)
		require.Error(t, err)
		var forbiddenErr *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenErr)
		require.False(t, ds.ListAppleDDMAssetsFuncInvoked)

		ds.ListAppleDDMAssetsFuncInvoked = false
	})

	t.Run("Global admin can list DDM assets", func(t *testing.T) {
		ds.ListAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.DDMAsset, error) {
			return []*fleet.DDMAsset{}, nil
		}

		_, err := svc.ListAppleDDMAssets(ctx, nil)
		require.NoError(t, err)
		require.True(t, ds.ListAppleDDMAssetsFuncInvoked)

		ds.ListAppleDDMAssetsFuncInvoked = false
	})

	t.Run("Team admin can list DDM assets for their team", func(t *testing.T) {
		ds.ListAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.DDMAsset, error) {
			return []*fleet.DDMAsset{}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin), Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})

		_, err := svc.ListAppleDDMAssets(ctx, new(uint(1)))
		require.NoError(t, err)
		require.True(t, ds.ListAppleDDMAssetsFuncInvoked)

		ds.ListAppleDDMAssetsFuncInvoked = false
	})

	t.Run("Team admin cannot list DDM assets for other teams", func(t *testing.T) {
		ds.ListAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.DDMAsset, error) {
			return []*fleet.DDMAsset{}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})

		_, err := svc.ListAppleDDMAssets(ctx, new(uint(2)))
		require.Error(t, err)
		var forbiddenErr *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenErr)
		require.False(t, ds.ListAppleDDMAssetsFuncInvoked)

		ds.ListAppleDDMAssetsFuncInvoked = false
	})
}

func TestGetAppleDDMAsset(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	t.Run("Observer cannot get DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})
		_, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team Observer cannot get DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}})
		_, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Global admin can get DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		asset, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.Equal(t, "some-asset-uuid", asset.AssetUUID)
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin can get DDM asset for their team", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(1))}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		asset, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.Equal(t, "some-asset-uuid", asset.AssetUUID)
		require.Equal(t, uint(1), *asset.TeamID)
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin cannot get DDM asset for other teams", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(2))}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		_, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))

		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Not found asset returns not found error", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return nil, common_mysql.NotFound("asset")
		}

		_, err := svc.GetAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})
}

func TestDownloadAppleDDMAsset(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	t.Run("Observer cannot download DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return &fleet.DownloadableDDMAsset{DDMAsset: fleet.DDMAsset{AssetUUID: assetUUID}, Data: []byte("some data")}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})
		_, _, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})

	t.Run("Team Observer cannot download DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return &fleet.DownloadableDDMAsset{DDMAsset: fleet.DDMAsset{AssetUUID: assetUUID}, Data: []byte("some data")}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}})
		_, _, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})

	t.Run("Global admin can download DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return &fleet.DownloadableDDMAsset{DDMAsset: fleet.DDMAsset{AssetUUID: assetUUID, Name: assetUUID}, Data: []byte("some data")}, nil
		}
		name, data, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.Equal(t, "some-asset-uuid.json", name)
		require.Equal(t, []byte("some data"), data)
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})

	t.Run("Team admin can download DDM asset for their team", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return &fleet.DownloadableDDMAsset{DDMAsset: fleet.DDMAsset{AssetUUID: assetUUID, Name: assetUUID, TeamID: new(uint(1))}, Data: []byte("some data")}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		name, data, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.Equal(t, "some-asset-uuid.json", name)
		require.Equal(t, []byte("some data"), data)
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})

	t.Run("Team admin cannot download DDM asset for other teams", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return &fleet.DownloadableDDMAsset{DDMAsset: fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(2))}, Data: []byte("some data")}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		_, _, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})

	t.Run("Not found asset returns not found error", func(t *testing.T) {
		ds.GetAppleDDMAssetForDownloadFunc = func(ctx context.Context, assetUUID string) (*fleet.DownloadableDDMAsset, error) {
			return nil, common_mysql.NotFound("asset")
		}

		_, _, err := svc.DownloadAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetForDownloadFuncInvoked)

		ds.GetAppleDDMAssetForDownloadFuncInvoked = false
	})
}

func TestDeleteAppleDDMAsset(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	t.Run("Observer cannot delete DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})
		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team Observer cannot delete DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}})
		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
	})

	t.Run("Global admin can delete DDM asset", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID}, nil
		}
		ds.DeleteAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) error {
			return nil
		}

		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)
		require.True(t, ds.DeleteAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
		ds.DeleteAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin can delete DDM asset for their team", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(1))}, nil
		}
		ds.DeleteAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) error {
			return nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})

		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.NoError(t, err)
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)
		require.True(t, ds.DeleteAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
		ds.DeleteAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin cannot delete DDM asset for other teams", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(2))}, nil
		}
		ds.DeleteAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) error {
			return nil
		}
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})

		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)
		require.False(t, ds.DeleteAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
		ds.DeleteAppleDDMAssetFuncInvoked = false
	})

	t.Run("Not found asset returns not found error", func(t *testing.T) {
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return nil, common_mysql.NotFound("asset")
		}
		ds.DeleteAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) error {
			return nil
		}

		err := svc.DeleteAppleDDMAsset(ctx, "some-asset-uuid")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err))
		require.True(t, ds.GetAppleDDMAssetFuncInvoked)
		require.False(t, ds.DeleteAppleDDMAssetFuncInvoked)

		ds.GetAppleDDMAssetFuncInvoked = false
		ds.DeleteAppleDDMAssetFuncInvoked = false
	})
}
