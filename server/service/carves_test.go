package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCarves(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListCarvesFunc = func(ctx context.Context, opts fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
		return []*fleet.CarveMetadata{
			{ID: 1},
			{ID: 2},
		}, nil
	}

	// admin user
	carves, err := svc.ListCarves(test.UserContext(ctx, test.UserAdmin), fleet.CarveListOptions{})
	require.NoError(t, err)
	require.Len(t, carves, 2)

	// only global admin can read carves
	_, err = svc.ListCarves(test.UserContext(ctx, test.UserNoRoles), fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.ListCarves(ctx, fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestGetCarve(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.CarveFunc = func(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
		return &fleet.CarveMetadata{
			ID: id,
		}, nil
	}

	// admin user
	carve, err := svc.GetCarve(test.UserContext(ctx, test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), carve.ID)

	// only global admin can read carves
	_, err = svc.GetCarve(test.UserContext(ctx, test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.GetCarve(ctx, 1)
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

	data, err := svc.GetBlock(test.UserContext(context.Background(), test.UserAdmin), metadata.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, []byte("foobar"), data)

	// only global admin can read carves
	_, err = svc.GetBlock(test.UserContext(context.Background(), test.UserNoRoles), metadata.ID, 2)
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
	_, err := svc.GetBlock(test.UserContext(context.Background(), test.UserAdmin), metadata.ID, 7)
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
	_, err := svc.GetBlock(test.UserContext(context.Background(), test.UserAdmin), metadata.ID, 3)
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
	_, err := svc.GetBlock(test.UserContext(context.Background(), test.UserAdmin), metadata.ID, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired carve")
}

func TestCarveBegin(t *testing.T) {
	host := fleet.Host{ID: 3}
	payload := fleet.CarveBeginPayload{
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}
	expectedMetadata := fleet.CarveMetadata{
		ID:         7,
		HostId:     host.ID,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms.NewCarveFunc = func(ctx context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
		metadata.ID = 7
		return metadata, nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if host.ID != id {
			return nil, errors.New("not found")
		}
		return &fleet.Host{
			Hostname: host.Hostname,
		}, nil
	}

	ctx := hostctx.NewContext(context.Background(), &host)

	metadata, err := svc.CarveBegin(ctx, payload)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.SessionId)
	metadata.SessionId = ""          // Clear this before comparison
	metadata.Name = ""               // Clear this before comparison
	metadata.CreatedAt = time.Time{} // Clear this before comparison
	assert.Equal(t, expectedMetadata, *metadata)
}

func TestCarveBeginNewCarveError(t *testing.T) {
	host := fleet.Host{ID: 3}
	payload := fleet.CarveBeginPayload{
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}
	ms.NewCarveFunc = func(ctx context.Context, metadata *fleet.CarveMetadata) (*fleet.CarveMetadata, error) {
		return nil, errors.New("ouch!")
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if host.ID != id {
			return nil, errors.New("not found")
		}
		return &fleet.Host{
			Hostname: host.Hostname,
		}, nil
	}

	ctx := hostctx.NewContext(context.Background(), &host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ouch!")
}

func TestCarveBeginEmptyError(t *testing.T) {
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}
	ctx := hostctx.NewContext(context.Background(), &fleet.Host{ID: 1})

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 {
			return nil, errors.New("not found")
		}
		return &fleet.Host{}, nil
	}

	_, err := svc.CarveBegin(ctx, fleet.CarveBeginPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size must be greater than 0")
}

func TestCarveBeginMissingHostError(t *testing.T) {
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}

	_, err := svc.CarveBegin(context.Background(), fleet.CarveBeginPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestCarveBeginBlockSizeMaxError(t *testing.T) {
	host := fleet.Host{ID: 3}
	payload := fleet.CarveBeginPayload{
		BlockCount: 10,
		BlockSize:  1024 * 1024 * 1024 * 1024,      // 1TB
		CarveSize:  10 * 1024 * 1024 * 1024 * 1024, // 10TB
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if host.ID != id {
			return nil, errors.New("not found")
		}
		return &fleet.Host{
			Hostname: host.Hostname,
		}, nil
	}

	ctx := hostctx.NewContext(context.Background(), &host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_size exceeds max")
}

func TestCarveBeginCarveSizeMaxError(t *testing.T) {
	host := fleet.Host{ID: 3}
	payload := fleet.CarveBeginPayload{
		BlockCount: 1024 * 1024,
		BlockSize:  10 * 1024 * 1024,               // 1TB
		CarveSize:  10 * 1024 * 1024 * 1024 * 1024, // 10TB
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if host.ID != id {
			return nil, errors.New("not found")
		}
		return &fleet.Host{
			Hostname: host.Hostname,
		}, nil
	}

	ctx := hostctx.NewContext(context.Background(), &host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size exceeds max")
}

func TestCarveBeginCarveSizeError(t *testing.T) {
	host := fleet.Host{ID: 3}
	payload := fleet.CarveBeginPayload{
		BlockCount: 7,
		BlockSize:  13,
		CarveSize:  7*13 + 1,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	ds := new(mock.Store)
	svc := &Service{
		carveStore: ms,
		ds:         ds,
	}
	ctx := hostctx.NewContext(context.Background(), &host)

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if host.ID != id {
			return nil, errors.New("not found")
		}
		return &fleet.Host{
			Hostname: host.Hostname,
		}, nil
	}

	// Too big
	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size does not match")

	// Too small
	payload.CarveSize = 6 * 13
	_, err = svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size does not match")
}

func TestCarveCarveBlockGetCarveError(t *testing.T) {
	sessionId := "foobar"
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		return nil, errors.New("ouch!")
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		SessionId: sessionId,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ouch!")
}

func TestCarveCarveBlockRequestIdError(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "not_matching",
		SessionId: sessionId,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request_id does not match")
}

func TestCarveCarveBlockBlockCountExceedError(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.UpdateCarveFunc = func(ctx context.Context, carve *fleet.CarveMetadata) error {
		assert.NotNil(t, carve.Error)
		assert.Equal(t, *carve.Error, "block_id exceeds expected max (22): 23")
		return nil
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   23,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_id exceeds expected max")
}

func TestCarveBlockCountMatchError(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
		MaxBlock:   3,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.UpdateCarveFunc = func(ctx context.Context, carve *fleet.CarveMetadata) error {
		assert.NotNil(t, carve.Error)
		assert.Equal(t, *carve.Error, "block_id does not match expected block (4): 7")
		return nil
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   7,
	}

	err := svc.CarveBlock(context.Background(), payload)
	var be *fleet.BadRequestError
	require.ErrorAs(t, err, &be)
	assert.Contains(t, err.Error(), "block_id does not match")
}

func TestCarveCarveBlockBlockSizeError(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  16,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
		MaxBlock:   3,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.UpdateCarveFunc = func(ctx context.Context, carve *fleet.CarveMetadata) error {
		assert.NotNil(t, carve.Error)
		assert.Equal(t, *carve.Error, "exceeded declared block size 16: 37")
		return nil
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :) TOO LONG!!!"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   4,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded declared block size")
}

func TestCarveCarveBlockNewBlockError(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
		MaxBlock:   3,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.NewBlockFunc = func(ctx context.Context, carve *fleet.CarveMetadata, blockId int64, data []byte) error {
		return errors.New("kaboom!")
	}
	ms.UpdateCarveFunc = func(ctx context.Context, carve *fleet.CarveMetadata) error {
		assert.NotNil(t, carve.Error)
		assert.Equal(t, *carve.Error, "kaboom!")
		return nil
	}

	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   4,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kaboom!")
}

func TestCarveCarveBlock(t *testing.T) {
	sessionId := "foobar"
	metadata := &fleet.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
		MaxBlock:   3,
	}
	payload := fleet.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   4,
	}
	ms := new(mock.Store)
	svc := &Service{carveStore: ms}
	ms.CarveBySessionIdFunc = func(ctx context.Context, sessionId string) (*fleet.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.NewBlockFunc = func(ctx context.Context, carve *fleet.CarveMetadata, blockId int64, data []byte) error {
		assert.Equal(t, metadata, carve)
		assert.Equal(t, int64(4), blockId)
		assert.Equal(t, payload.Data, data)
		return nil
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.NoError(t, err)
	assert.True(t, ms.NewBlockFuncInvoked)
}
