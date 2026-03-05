package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	strconv "strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	carvestorectx "github.com/fleetdm/fleet/v4/server/contexts/carvestore"
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

// MockCarveStore for testing
type mockCarveStore struct {
	carves map[string]*fleet.CarveMetadata
	err    error
}

func (m *mockCarveStore) CarveBySessionId(ctx context.Context, sessionID string) (*fleet.CarveMetadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	c, ok := m.carves[sessionID]
	if !ok {
		return nil, errors.New("carve not found")
	}
	return c, nil
}

func TestCarveBlockDecodeRequest(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		ctxSetup       func(ctx context.Context) context.Context
		wantErr        bool
		wantErrType    string // e.g., "AuthFailedError", "ctxerr", ""
		wantErrMessage string
		wantResult     *carveBlockRequest
	}{
		{
			name: "valid request MySQL session IDs",
			body: `{"block_id":123,"session_id":"23bbbcf6-6b8a-4f3a-9924-bdd084f31097","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"23bbbcf6-6b8a-4f3a-9924-bdd084f31097": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   123,
				SessionId: "23bbbcf6-6b8a-4f3a-9924-bdd084f31097",
				RequestId: "req123",
				Data:      []byte("database64"),
			},
		},
		{
			name: "valid request AWS like session IDs",
			body: `{"block_id":123,"session_id":"JUMHLnWZ.A7y5ns2jUODzG8eTr5m9lvFKDD3nBN.hJ8mwr2szW0iUSNrusaE41__.wrtsNokzejFLQyNJTTqY_QN1grwAT0yXGi8A77Kf9ZJlvSiWggmncDAhVev4QXxx2PyN_GtTRPC71WGKPN2YxBFfWBjZlCZBXmPCtc4zrQ","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"JUMHLnWZ.A7y5ns2jUODzG8eTr5m9lvFKDD3nBN.hJ8mwr2szW0iUSNrusaE41__.wrtsNokzejFLQyNJTTqY_QN1grwAT0yXGi8A77Kf9ZJlvSiWggmncDAhVev4QXxx2PyN_GtTRPC71WGKPN2YxBFfWBjZlCZBXmPCtc4zrQ": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   123,
				SessionId: "JUMHLnWZ.A7y5ns2jUODzG8eTr5m9lvFKDD3nBN.hJ8mwr2szW0iUSNrusaE41__.wrtsNokzejFLQyNJTTqY_QN1grwAT0yXGi8A77Kf9ZJlvSiWggmncDAhVev4QXxx2PyN_GtTRPC71WGKPN2YxBFfWBjZlCZBXmPCtc4zrQ",
				RequestId: "req123",
				Data:      []byte("database64"),
			},
		},
		{
			name: "valid request rustfs like session IDs",
			body: `{"block_id":123,"session_id":"ZGVhZDYwYTctZTVlOC00MzE1LWFhOWMtZDIzMzc5MTI4NGUyLjUzM2MxZjhhLTFiODktNDQ1YS04NTE0LTBjMWE0NDVlNjkwMXgxNzcwMTQ1Njc5NDgyNDg3NzE3","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"ZGVhZDYwYTctZTVlOC00MzE1LWFhOWMtZDIzMzc5MTI4NGUyLjUzM2MxZjhhLTFiODktNDQ1YS04NTE0LTBjMWE0NDVlNjkwMXgxNzcwMTQ1Njc5NDgyNDg3NzE3": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   123,
				SessionId: "ZGVhZDYwYTctZTVlOC00MzE1LWFhOWMtZDIzMzc5MTI4NGUyLjUzM2MxZjhhLTFiODktNDQ1YS04NTE0LTBjMWE0NDVlNjkwMXgxNzcwMTQ1Njc5NDgyNDg3NzE3",
				RequestId: "req123",
				Data:      []byte("database64"),
			},
		},
		{
			name: "valid max-sized session_id",
			body: fmt.Sprintf(`{"block_id":123,"session_id":"%s","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`, strings.Repeat("F", 255)),
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						strings.Repeat("F", 255): {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   123,
				SessionId: strings.Repeat("F", 255),
				RequestId: "req123",
				Data:      []byte("database64"),
			},
		},
		{
			name:           "missing carve store",
			body:           `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup:       func(ctx context.Context) context.Context { return ctx },
			wantErr:        true,
			wantErrMessage: "missing carve store from context",
		},
		{
			name: "invalid start delimiter",
			body: `["block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "Authentication failed",
		},
		{
			name: "short non-ending session_id",
			body: `{"block_id":123,"session_id":"sess123`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "Authentication failed",
		},
		{
			name: "max non-ending session_id",
			body: fmt.Sprintf(`{"block_id":123,"session_id":"%s`, strings.Repeat("F", 256)),
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "Authentication failed",
		},
		{
			name: "invalid block_id key",
			body: `{"blockid":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "block_id":, got "blockid":`,
		},
		{
			name: "non-ending block_id key",
			body: `{"block_id`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "end character not found",
		},
		{
			name: "invalid block_id too long",
			body: `{"block_id":12345678901234567890,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "end character not found",
		},
		{
			name: "invalid block_id not number",
			body: `{"block_id":"abc","session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "invalid \"block_id\" format",
		},
		{
			name: "missing session_id key",
			body: `{"block_id":123,"request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "session_id":", got "request_id":"`,
		},
		{
			name: "missing request_id key",
			body: `{"block_id":123,"session_id":"sess123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "session_id":", got "data":"ZGF0YW`,
		},
		{
			name: "invalid session_id key",
			body: `{"block_id":123,"sess_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "session_id":", got "sess_id":"`,
		},
		{
			name: "invalid session_id empty",
			body: `{"block_id":123,"session_id":"","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "empty session_id",
		},
		{
			name: "invalid session_id too long",
			body: `{"block_id":123,"session_id":"` + strings.Repeat("a", 256) + `","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "end character not found",
		},
		{
			name: "missing session_id key, terminated body",
			body: `{"block_id":123,`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "end character not found",
		},
		{
			name: "invalid request_id key",
			body: `{"block_id":123,"session_id":"sess123","req_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected ,"request_id":", got ,"req_id":"`,
		},
		{
			name: "invalid request_id empty",
			body: `{"block_id":123,"session_id":"sess123","request_id":"","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "empty request_id",
		},
		{
			name: "invalid request_id too long",
			body: `{"block_id":123,"session_id":"sess123","request_id":"` + strings.Repeat("F", 65) + `","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "end character not found",
		},
		{
			name: "max non-ending request_id",
			body: fmt.Sprintf(`{"block_id":123,"session_id":"sess123","request_id":"%s`, strings.Repeat("F", 65)),
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "Authentication failed",
		},
		{
			name: "missing request_id key, terminated body",
			body: `{"block_id":123,"session_id":"sess123"`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "Authentication failed",
		},
		{
			name: "valid max-sized request_id",
			body: fmt.Sprintf(`{"block_id":123,"session_id":"sess123","request_id":"%s","data":"ZGF0YWJhc2U2NA=="}`, strings.Repeat("F", 64)),
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: strings.Repeat("F", 64)},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   123,
				SessionId: "sess123",
				RequestId: strings.Repeat("F", 64),
				Data:      []byte("database64"),
			},
		},
		{
			name: "auth failure carve not found",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "carve by session ID: carve not found",
		},
		{
			name: "auth failure request_id mismatch",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "wrongreq"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "request_id does not match session",
		},
		{
			name: "auth failure store error",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					err: errors.New("store error"),
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "carve by session ID: store error",
		},
		{
			name: "invalid data key",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","datum":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `expected ,"data":", got ,"datum":`,
		},
		{
			name: "missing data key",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123"}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `expected ,"data":", got }`,
		},
		{
			name: "missing data key, terminated body",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123"`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `failed to read "data" key`,
		},
		{
			name: "missing data value, terminated body (ending length=0)",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `invalid "data" ending length`,
		},
		{
			name: "missing data value, terminated body (ending length=1)",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"a`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `invalid "data" ending length`,
		},
		{
			name: "empty data key", // empty block is a valid block.
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":""}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
		},
		{
			name: "invalid data not base64",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"notbase64!!"}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: "base64 decode block data: illegal base64 data",
		},
		{
			name: "invalid ending",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `invalid "data" ending: ="`,
		},
		{
			name: "short body - after request_id",
			body: `{"block_id":123,"session_id":"sess123","request_id":"req123"`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr:        true,
			wantErrMessage: `failed to read "data" key: EOF`,
		},
		{
			name: "empty body",
			body: ``,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "failed to read object start: EOF",
		},
		{
			name: "empty JSON",
			body: `{}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "block_id":, got }`,
		},
		{
			name: "unending JSON",
			body: `{`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: `expected "block_id":, got }`,
		},
		{
			name: "string is a valid JSON",
			body: `"foobar"`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "failed to read object start: EOF",
		},
		{
			name: "max block_id digits",
			body: `{"block_id":` + strconv.FormatInt(1<<63-1, 10) + `,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				store := &mockCarveStore{
					carves: map[string]*fleet.CarveMetadata{
						"sess123": {RequestId: "req123"},
					},
				}
				return carvestorectx.NewContext(ctx, store)
			},
			wantErr: false,
			wantResult: &carveBlockRequest{
				BlockId:   1<<63 - 1,
				SessionId: "sess123",
				RequestId: "req123",
				Data:      []byte("database64"),
			},
		},
		{
			name: "negative block_id",
			body: `{"block_id":-123,"session_id":"sess123","request_id":"req123","data":"ZGF0YWJhc2U2NA=="}`,
			ctxSetup: func(ctx context.Context) context.Context {
				return carvestorectx.NewContext(ctx, &mockCarveStore{})
			},
			wantErr:        true,
			wantErrType:    "AuthFailedError",
			wantErrMessage: "invalid \"block_id\" format",
		},
		// Add more edge cases as needed, e.g., special characters in strings, zero block_id, etc.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ctxSetup != nil {
				ctx = tt.ctxSetup(ctx)
			}
			req := &http.Request{
				Body: io.NopCloser(bytes.NewReader([]byte(tt.body))),
			}
			var r carveBlockRequest
			result, err := r.DecodeRequest(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				// Check error type and message
				if tt.wantErrType == "AuthFailedError" {
					var afe *fleet.AuthFailedError
					require.ErrorAs(t, err, &afe)
				} else if tt.wantErrMessage != "" && !strings.Contains(err.Error(), tt.wantErrMessage) {
					t.Errorf("error message = %v, want containing %s", err, tt.wantErrMessage)
				}
				return
			}
			got, ok := result.(*carveBlockRequest)
			if !ok {
				t.Errorf("result not *carveBlockRequest")
				return
			}
			if tt.wantResult != nil {
				if got.BlockId != tt.wantResult.BlockId {
					t.Errorf("BlockId = %d, want %d", got.BlockId, tt.wantResult.BlockId)
				}
				if got.SessionId != tt.wantResult.SessionId {
					t.Errorf("SessionId = %s, want %s", got.SessionId, tt.wantResult.SessionId)
				}
				if got.RequestId != tt.wantResult.RequestId {
					t.Errorf("RequestId = %s, want %s", got.RequestId, tt.wantResult.RequestId)
				}
				if !bytes.Equal(got.Data, tt.wantResult.Data) {
					t.Errorf("Data = %v, want %v", got.Data, tt.wantResult.Data)
				}
			}
		})
	}
}
