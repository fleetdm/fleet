package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/assert"
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
	svc, mockSvc := newTestServiceWithMock(t, ds)
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
		mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
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
		mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
		ds.GetAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) (*fleet.DDMAsset, error) {
			return &fleet.DDMAsset{AssetUUID: assetUUID, TeamID: new(uint(1))}, nil
		}
		ds.DeleteAppleDDMAssetFunc = func(ctx context.Context, assetUUID string) error {
			return nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{ID: tid, Name: "team"}, nil
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
	svc, mockSvc := newTestServiceWithMock(t, ds)
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
		mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
		defer reset()
		_, err := svc.CreateAppleDDMAsset(ctx, nil, "asset", validData)
		require.NoError(t, err)
		require.True(t, ds.CreateAppleDDMAssetFuncInvoked)

		ds.CreateAppleDDMAssetFuncInvoked = false
	})

	t.Run("Team admin can create DDM asset for their team", func(t *testing.T) {
		mockSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{ID: tid, Name: "team"}, nil
		}
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
	ds.BatchSetAppleDDMAssetsFunc = func(ctx context.Context, teamID *uint, assets []*fleet.MDMAppleDDMAssetToSet) (*fleet.MDMAppleDDMAssetsBatchChanges, error) {
		return &fleet.MDMAppleDDMAssetsBatchChanges{}, nil
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

func appleHost(id uint, serial string, teamID *uint) *fleet.Host {
	return &fleet.Host{ID: id, Platform: "darwin", HardwareSerial: serial, TeamID: teamID, Hostname: fmt.Sprintf("host-%d", id)}
}

func depAssignment(hostID uint, serial string, tokenID *uint) *fleet.HostDEPAssignment {
	return &fleet.HostDEPAssignment{HostID: hostID, HardwareSerial: serial, ABMTokenID: tokenID}
}

// startDEPServer stands in for Apple's DEP API. sessionStatus/disownStatus of 0
// mean 200. serialStatus overrides the per-serial status echoed by /devices/disown
// (defaults to SUCCESS).
func startDEPServer(t *testing.T, sessionStatus, disownStatus int, serialStatus map[string]string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/session"):
			if sessionStatus != 0 && sessionStatus != http.StatusOK {
				w.WriteHeader(sessionStatus)
				_, _ = w.Write([]byte(`{"error":"FORBIDDEN"}`))
				return
			}
			_, _ = w.Write([]byte(`{"auth_session_token":"tok"}`))
		case strings.Contains(r.URL.Path, "/devices/disown"):
			if disownStatus != 0 && disownStatus != http.StatusOK {
				w.WriteHeader(disownStatus)
				return
			}
			var req struct {
				Devices []string `json:"devices"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			out := make(map[string]string, len(req.Devices))
			for _, s := range req.Devices {
				st := "SUCCESS"
				if v, ok := serialStatus[s]; ok {
					st = v
				}
				out[s] = st
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"devices": out}))
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

func setupReleaseABTest(t *testing.T) (*Service, *mock.Store, *nanodep_mock.Storage, *svcmock.Service) {
	ds := new(mock.Store)
	svc, base := newTestServiceWithMock(t, ds)
	svc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	base.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error { return nil }

	// Short-circuit the DEP client's terms-expired after-hook.
	ds.CountABMTokensWithTermsExpiredFunc = func(context.Context) (int, error) { return 0, nil }
	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
	ds.DeleteHostDEPAssignmentsFunc = func(context.Context, uint, []string) error { return nil }

	dep := &nanodep_mock.Storage{}
	dep.RetrieveAuthTokensFunc = func(context.Context, string) (*nanodep_client.OAuth1Tokens, error) {
		return &nanodep_client.OAuth1Tokens{ConsumerKey: "ck", ConsumerSecret: "cs", AccessToken: "at", AccessSecret: "as"}, nil
	}
	svc.depStorage = dep
	return svc, ds, dep, base
}

func byHostID(resp []*fleet.ABReleaseDeviceResponse) map[uint]*fleet.ABReleaseDeviceResponse {
	m := make(map[uint]*fleet.ABReleaseDeviceResponse, len(resp))
	for _, r := range resp {
		m[r.HostID] = r
	}
	return m
}

func adminCtx() context.Context {
	return viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
}

func TestReleaseABDevicesAuthorization(t *testing.T) {
	team1 := uint(1)
	team2 := uint(2)

	setup := func(t *testing.T) (*Service, *mock.Store) {
		svc, ds, _, _ := setupReleaseABTest(t)
		ds.ListHostsLiteByIDsFunc = func(_ context.Context, ids []uint) ([]*fleet.Host, error) {
			return []*fleet.Host{
				appleHost(1, "S1", &team1),
				appleHost(2, "S2", &team2),
			}, nil
		}
		// No DEP assignments so authorized calls short-circuit before hitting Apple.
		ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
			return nil, nil
		}
		ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
			return nil, nil
		}
		return svc, ds
	}

	cases := []struct {
		name    string
		user    *fleet.User
		wantErr bool
	}{
		{"global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, false},
		{"global observer", &fleet.User{GlobalRole: new(fleet.RoleObserver)}, true},
		{"global maintainer", &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}, true},
		{"global gitops", &fleet.User{GlobalRole: new(fleet.RoleGitOps)}, true},
		{"team admin missing one team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team1}, Role: fleet.RoleAdmin}}}, true},
		{"team admin all teams", &fleet.User{Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team1}, Role: fleet.RoleAdmin},
			{Team: fleet.Team{ID: team2}, Role: fleet.RoleAdmin},
		}}, false},
		{"team admin wrong role", &fleet.User{Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team1}, Role: fleet.RoleObserver},
			{Team: fleet.Team{ID: team2}, Role: fleet.RoleObserver},
		}}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _ := setup(t)
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: c.user})
			_, err := svc.ReleaseABDevices(ctx, []uint{1, 2})
			if c.wantErr {
				var forbidden *authz.Forbidden
				require.ErrorAs(t, err, &forbidden)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReleaseABDevicesTooManyHosts(t *testing.T) {
	svc, ds, _, _ := setupReleaseABTest(t)
	ids := make([]uint, 32_001)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	_, err := svc.ReleaseABDevices(adminCtx(), ids)
	var badReq *fleet.BadRequestError
	require.ErrorAs(t, err, &badReq)
	require.False(t, ds.ListHostsLiteByIDsFuncInvoked)
}

func TestReleaseABDevicesDatastoreError(t *testing.T) {
	svc, ds, _, _ := setupReleaseABTest(t)
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return nil, io.ErrUnexpectedEOF
	}
	_, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.Error(t, err)
}

func TestReleaseABDevicesPerHostErrors(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	tokenID := uint(10)
	otherToken := uint(99)

	// host 1: apple + assigned -> success
	// host 2: not returned by lite lookup -> not found
	// host 3: non-apple platform -> ineligible
	// host 4: apple but no DEP assignment -> not in AB
	// host 5: apple, assignment with nil token -> no ABM token
	// host 6: apple, assignment references unknown token -> token not found
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		h3 := appleHost(3, "S3", nil)
		h3.Platform = "windows"
		return []*fleet.Host{
			appleHost(1, "S1", nil),
			h3,
			appleHost(4, "S4", nil),
			appleHost(5, "S5", nil),
			appleHost(6, "S6", nil),
		}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{
			depAssignment(1, "S1", &tokenID),
			depAssignment(5, "S5", nil),
			depAssignment(6, "S6", &otherToken),
		}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	ts := startDEPServer(t, 0, 0, nil)
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1, 2, 3, 4, 5, 6})
	require.NoError(t, err)
	m := byHostID(resp)

	require.Equal(t, string(fleet.ABReleaseDeviceStatusSuccess), m[1].Status)
	require.Empty(t, m[1].Error)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusError), m[2].Status)
	require.Contains(t, m[2].Error, "Host not found")
	require.Contains(t, m[3].Error, "not an eligible Apple host")
	require.Contains(t, m[4].Error, "not found in Apple Business")
	require.Contains(t, m[5].Error, "no associated ABM token")
	require.Contains(t, m[6].Error, "ABM token not found")

	// Responses are sorted by host ID.
	for i := 1; i < len(resp); i++ {
		require.Less(t, resp[i-1].HostID, resp[i].HostID)
	}
}

func TestReleaseABDevicesDEPAuthError(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	tokenID := uint(10)
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{appleHost(1, "S1", nil)}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{depAssignment(1, "S1", &tokenID)}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	ts := startDEPServer(t, http.StatusForbidden, 0, nil)
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusError), resp[0].Status)
	require.Contains(t, resp[0].Error, "Apple rejected this request")
}

func TestReleaseABDevicesDEPGenericError(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	tokenID := uint(10)
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{appleHost(1, "S1", nil)}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{depAssignment(1, "S1", &tokenID)}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	ts := startDEPServer(t, 0, http.StatusInternalServerError, nil)
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusError), resp[0].Status)
	require.Contains(t, resp[0].Error, "Couldn't release host from Apple Business. Error:")
	require.NotContains(t, resp[0].Error, "Apple rejected this request")
}

func TestReleaseABDevicesNonSuccessStatus(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	tokenID := uint(10)
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{appleHost(1, "S1", nil), appleHost(2, "S2", nil)}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{depAssignment(1, "S1", &tokenID), depAssignment(2, "S2", &tokenID)}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	ts := startDEPServer(t, 0, 0, map[string]string{"S2": "NOT_ACCESSIBLE"})
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1, 2})
	require.NoError(t, err)
	m := byHostID(resp)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusSuccess), m[1].Status)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusError), m[2].Status)
	require.Contains(t, m[2].Error, "NOT_ACCESSIBLE")
}

func TestReleaseABDevicesActivityLogged(t *testing.T) {
	svc, ds, dep, base := setupReleaseABTest(t)
	tokenID := uint(10)
	var logged []fleet.ActivityTypeReleasedDeviceFromAB
	base.NewActivityFunc = func(_ context.Context, _ *fleet.User, act fleet.ActivityDetails) error {
		a, ok := act.(fleet.ActivityTypeReleasedDeviceFromAB)
		require.True(t, ok)
		logged = append(logged, a)
		return nil
	}
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{appleHost(1, "S1", nil)}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{depAssignment(1, "S1", &tokenID)}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	var deletedToken uint
	var deletedSerials []string
	ds.DeleteHostDEPAssignmentsFunc = func(_ context.Context, abmTokenID uint, serials []string) error {
		deletedToken = abmTokenID
		deletedSerials = serials
		return nil
	}
	ts := startDEPServer(t, 0, 0, nil)
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	_, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.NoError(t, err)
	require.True(t, base.NewActivityFuncInvoked)
	require.Len(t, logged, 1)
	require.Equal(t, uint(1), logged[0].HostID)
	require.Equal(t, "S1", logged[0].HostSerial)

	// Released devices have their DEP assignment cleared under the right token.
	require.True(t, ds.DeleteHostDEPAssignmentsFuncInvoked)
	require.Equal(t, tokenID, deletedToken)
	require.Equal(t, []string{"S1"}, deletedSerials)
}

// TestReleaseABDevicesMultiToken exercises devices spread across two ABM tokens,
// where one token's disown call is rejected by Apple and the other succeeds.
func TestReleaseABDevicesMultiToken(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	token1 := uint(10)
	token2 := uint(20)

	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{
			appleHost(1, "S1", nil),
			appleHost(2, "S2", nil),
			appleHost(3, "S3", nil),
		}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{
			depAssignment(1, "S1", &token1),
			depAssignment(2, "S2", &token1),
			depAssignment(3, "S3", &token2),
		}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{
			{ID: token1, OrganizationName: "Org1"},
			{ID: token2, OrganizationName: "Org2"},
		}, nil
	}

	deleted := map[uint][]string{}
	ds.DeleteHostDEPAssignmentsFunc = func(_ context.Context, abmTokenID uint, serials []string) error {
		deleted[abmTokenID] = serials
		return nil
	}

	// Org1 succeeds, Org2's session is rejected (auth error).
	okServer := startDEPServer(t, 0, 0, nil)
	authFailServer := startDEPServer(t, http.StatusForbidden, 0, nil)
	seen := map[string]bool{}
	dep.RetrieveConfigFunc = func(_ context.Context, name string) (*nanodep_client.Config, error) {
		seen[name] = true
		if name == "Org2" {
			return &nanodep_client.Config{BaseURL: authFailServer.URL}, nil
		}
		return &nanodep_client.Config{BaseURL: okServer.URL}, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1, 2, 3})
	require.NoError(t, err)
	m := byHostID(resp)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusSuccess), m[1].Status)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusSuccess), m[2].Status)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusError), m[3].Status)
	require.Contains(t, m[3].Error, "Apple rejected this request")

	// Each token was resolved by its own organization name.
	assert.True(t, seen["Org1"])
	assert.True(t, seen["Org2"])

	// Only the successful token clears its assignments; the failed token does not.
	require.ElementsMatch(t, []string{"S1", "S2"}, deleted[token1])
	require.NotContains(t, deleted, token2)
}

func TestReleaseABDevicesAllIneligibleSkipsDEP(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	h := appleHost(1, "S1", nil)
	h.Platform = "ubuntu"
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{h}, nil
	}
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		t.Fatal("DEP should not be contacted when there are no eligible hosts")
		return nil, nil
	}

	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Contains(t, resp[0].Error, "not an eligible Apple host")
	require.False(t, ds.GetHostDEPAssignmentsByHostIDsFuncInvoked)
}

func TestReleaseABDevicesDeleteAssignmentErrorIsNonFatal(t *testing.T) {
	svc, ds, dep, _ := setupReleaseABTest(t)
	tokenID := uint(10)
	ds.ListHostsLiteByIDsFunc = func(context.Context, []uint) ([]*fleet.Host, error) {
		return []*fleet.Host{appleHost(1, "S1", nil)}, nil
	}
	ds.GetHostDEPAssignmentsByHostIDsFunc = func(context.Context, []uint) ([]*fleet.HostDEPAssignment, error) {
		return []*fleet.HostDEPAssignment{depAssignment(1, "S1", &tokenID)}, nil
	}
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: tokenID, OrganizationName: "Org1"}}, nil
	}
	ds.DeleteHostDEPAssignmentsFunc = func(context.Context, uint, []string) error {
		return io.ErrUnexpectedEOF
	}
	ts := startDEPServer(t, 0, 0, nil)
	dep.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{BaseURL: ts.URL}, nil
	}

	// The device was released, so the call still succeeds even though clearing
	// the DEP assignment failed (that error is only logged).
	resp, err := svc.ReleaseABDevices(adminCtx(), []uint{1})
	require.NoError(t, err)
	require.Len(t, resp, 1)
	require.Equal(t, string(fleet.ABReleaseDeviceStatusSuccess), resp[0].Status)
	require.True(t, ds.DeleteHostDEPAssignmentsFuncInvoked)
}
