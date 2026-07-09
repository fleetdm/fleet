package service

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func TestCreateAppleDDMAsset(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	validData := []byte(`{"Type":"com.apple.asset.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"https://example.com/data"}}}`)

	ds.CreateAppleDDMAssetFunc = func(ctx context.Context, name, identifier string, data []byte, teamID *uint) (string, error) {
		return "some-asset-uuid", nil
	}
	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}

	reset := func() {
		ds.CreateAppleDDMAssetFuncInvoked = false
		ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked = false
	}

	t.Run("Observer cannot create DDM asset", func(t *testing.T) {
		defer reset()
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", validData)
		require.Error(t, err)
		var forbiddenErr *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenErr)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)

		ds.CreateAppleDDMAssetFuncInvoked = false
	})

	t.Run("Global admin can create DDM asset", func(t *testing.T) {
		defer reset()
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", validData)
		require.NoError(t, err)
		require.True(t, ds.CreateAppleDDMAssetFuncInvoked)

		ds.CreateAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin can create DDM asset for their team", func(t *testing.T) {
		defer reset()
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		_, err := svc.CreateAppleDDMAsset(ctx, new(uint(1)), "asset", validData)
		require.NoError(t, err)
		require.True(t, ds.CreateAppleDDMAssetFuncInvoked)

		ds.CreateAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin cannot create DDM asset for other teams", func(t *testing.T) {
		defer reset()
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: nil, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}})
		_, err := svc.CreateAppleDDMAsset(ctx, new(uint(2)), "asset", validData)
		require.Error(t, err)
		var forbiddenErr *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenErr)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)

		ds.CreateAppleDDMAssetFuncInvoked = false
	})

	t.Run("Malformed JSON is rejected", func(t *testing.T) {
		defer reset()
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", []byte(`{not json`))
		require.Error(t, err)
		require.False(t, ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Empty identifier is rejected", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"com.apple.asset.data","Identifier":"","Payload":{"Reference":{"DataURL":"https://example.com/data"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Invalid asset type is rejected", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"com.example.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"https://example.com/data"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Empty payload reference data URL is rejected", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"com.apple.asset.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":""}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Invalid payload reference data URL is rejected", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"com.apple.asset.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"notaurl"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Secret in type is rejected before expansion", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"$FLEET_SECRET_TYPE","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"https://example.com/data"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Secret in identifier is rejected before expansion", func(t *testing.T) {
		defer reset()
		data := []byte(`{"Type":"com.apple.asset.data","Identifier":"$FLEET_SECRET_ID","Payload":{"Reference":{"DataURL":"https://example.com/data"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.Error(t, err)
		require.False(t, ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Secret in payload data URL is expanded and allowed", func(t *testing.T) {
		defer reset()
		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
			return string(validData), nil, nil
		}
		defer func() {
			ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
				return document, nil, nil
			}
		}()
		data := []byte(`{"Type":"com.apple.asset.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"$FLEET_SECRET_URL"}}}`)
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", data)
		require.NoError(t, err)
		require.True(t, ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked)
		require.True(t, ds.CreateAppleDDMAssetFuncInvoked)
	})

	t.Run("Expanded payload with authentication key is rejected", func(t *testing.T) {
		defer reset()
		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
			return `{"Type":"com.apple.asset.data","Identifier":"com.example.asset","Payload":{"Reference":{"DataURL":"https://example.com/data"},"Authentication":{"Username":"u"}}}`, nil, nil
		}
		defer func() {
			ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
				return document, nil, nil
			}
		}()
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", validData)
		require.Error(t, err)
		require.True(t, ds.ExpandEmbeddedSecretsAndUpdatedAtFuncInvoked)
		require.False(t, ds.CreateAppleDDMAssetFuncInvoked)
	})
}

func TestBatchSetAppleDDMAssets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	assetData := func(identifier string) []byte {
		return []byte(fmt.Sprintf(`{"Type":"com.apple.asset.data","Identifier":%q,"Payload":{"Reference":{"DataURL":"https://example.com/%s"}}}`, identifier, identifier))
	}

	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}
	ds.BatchSetAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint, assets []*fleet.MDMAppleDDMAssetToSet) error {
		return nil
	}
	reset := func() { ds.BatchSetAppleDDMAssetsFuncInvoked = false }

	t.Run("Observer cannot batch set", func(t *testing.T) {
		defer reset()
		ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleObserver)}})
		err := svc.BatchSetAppleDDMAssets(ctx, nil, "", []fleet.MDMAppleDDMAssetBatchPayload{{Name: "a", Contents: assetData("id.a")}}, false)
		require.Error(t, err)
		var forbiddenErr *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenErr)
		require.False(t, ds.BatchSetAppleDDMAssetsFuncInvoked)
	})

	t.Run("Global admin can batch set", func(t *testing.T) {
		defer reset()
		err := svc.BatchSetAppleDDMAssets(ctx, nil, "", []fleet.MDMAppleDDMAssetBatchPayload{
			{Name: "a", Contents: assetData("id.a")},
			{Name: "b", Contents: assetData("id.b")},
		}, false)
		require.NoError(t, err)
		require.True(t, ds.BatchSetAppleDDMAssetsFuncInvoked)
	})

	t.Run("Duplicate identifier is rejected", func(t *testing.T) {
		defer reset()
		err := svc.BatchSetAppleDDMAssets(ctx, nil, "", []fleet.MDMAppleDDMAssetBatchPayload{
			{Name: "a", Contents: assetData("id.dup")},
			{Name: "b", Contents: assetData("id.dup")},
		}, false)
		require.Error(t, err)
		require.False(t, ds.BatchSetAppleDDMAssetsFuncInvoked)
	})

	t.Run("Duplicate name is rejected", func(t *testing.T) {
		defer reset()
		err := svc.BatchSetAppleDDMAssets(ctx, nil, "", []fleet.MDMAppleDDMAssetBatchPayload{
			{Name: "same", Contents: assetData("id.a")},
			{Name: "same", Contents: assetData("id.b")},
		}, false)
		require.Error(t, err)
		require.False(t, ds.BatchSetAppleDDMAssetsFuncInvoked)
	})

	t.Run("Dry run does not write", func(t *testing.T) {
		defer reset()
		err := svc.BatchSetAppleDDMAssets(ctx, nil, "", []fleet.MDMAppleDDMAssetBatchPayload{
			{Name: "a", Contents: assetData("id.a")},
		}, true)
		require.NoError(t, err)
		require.False(t, ds.BatchSetAppleDDMAssetsFuncInvoked)
	})
}
