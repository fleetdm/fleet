package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCarves(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListCarvesFunc = func(ctx context.Context, opts fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
		return []*fleet.CarveMetadata{
			{ID: 1},
			{ID: 2},
		}, nil
	}

	// admin user
	carves, err := svc.ListCarves(test.UserContext(test.UserAdmin), fleet.CarveListOptions{})
	require.NoError(t, err)
	require.Len(t, carves, 2)

	// only global admin can read carves
	_, err = svc.ListCarves(test.UserContext(test.UserNoRoles), fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.ListCarves(context.Background(), fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestGetCarve(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.CarveFunc = func(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
		return &fleet.CarveMetadata{
			ID: id,
		}, nil
	}

	// admin user
	carve, err := svc.GetCarve(test.UserContext(test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), carve.ID)

	// only global admin can read carves
	_, err = svc.GetCarve(test.UserContext(test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.GetCarve(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestCarveGetBlock(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{carveStore: ds, authz: authz.Must()}

	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  "foobar",
		MaxBlock:   3,
	}

	ds.CarveFunc = func(ctx context.Context, carveId int64) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}
	ds.GetBlockFunc = func(ctx context.Context, carve *fleet.CarveMetadata, blockId int64) ([]byte, error) {
		assert.Equal(t, metadata.ID, carve.ID)
		assert.Equal(t, int64(3), blockId)
		return []byte("foobar"), nil
	}

	data, err := svc.GetBlock(test.UserContext(test.UserAdmin), metadata.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, []byte("foobar"), data)

	// only global admin can read carves
	_, err = svc.GetBlock(test.UserContext(test.UserNoRoles), metadata.ID, 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestCarveGetBlockNotAvailableError(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{carveStore: ds, authz: authz.Must()}

	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  "foobar",
		MaxBlock:   3,
	}

	ds.CarveFunc = func(ctx context.Context, carveId int64) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}

	// Block requested is greater than max block
	_, err := svc.GetBlock(test.UserContext(test.UserAdmin), metadata.ID, 7)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet available")
}

func TestCarveGetBlockGetBlockError(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{carveStore: ds, authz: authz.Must()}

	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  "foobar",
		MaxBlock:   3,
	}

	ds.CarveFunc = func(ctx context.Context, carveId int64) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}
	ds.GetBlockFunc = func(ctx context.Context, carve *fleet.CarveMetadata, blockId int64) ([]byte, error) {
		assert.Equal(t, metadata.ID, carve.ID)
		assert.Equal(t, int64(3), blockId)
		return nil, errors.New("yow!!")
	}

	// GetBlock failed
	_, err := svc.GetBlock(test.UserContext(test.UserAdmin), metadata.ID, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "yow!!")
}

func TestCarveGetBlockExpired(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{carveStore: ds, authz: authz.Must()}

	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  "foobar",
		MaxBlock:   3,
		Expired:    true,
	}

	ds.CarveFunc = func(ctx context.Context, carveId int64) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}

	// carve is expired
	_, err := svc.GetBlock(test.UserContext(test.UserAdmin), metadata.ID, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired carve")
}
