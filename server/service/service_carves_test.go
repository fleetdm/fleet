package service

import (
	"context"
	"fmt"
	"testing"

	hostctx "github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCarveBegin(t *testing.T) {
	host := kolide.Host{ID: 3}
	payload := kolide.CarveBeginPayload{
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	expectedMetadata := kolide.CarveMetadata{
		ID:         7,
		HostId:     host.ID,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms.NewCarveFunc = func(metadata *kolide.CarveMetadata) (*kolide.CarveMetadata, error) {
		metadata.ID = 7
		return metadata, nil
	}

	ctx := hostctx.NewContext(context.Background(), host)

	metadata, err := svc.CarveBegin(ctx, payload)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.SessionId)
	metadata.SessionId = "" // Clear this before comparison
	metadata.Name = "" // Clear this before comparison
	assert.Equal(t, expectedMetadata, *metadata)
}

func TestCarveBeginNewCarveError(t *testing.T) {
	host := kolide.Host{ID: 3}
	payload := kolide.CarveBeginPayload{
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	ms.NewCarveFunc = func(metadata *kolide.CarveMetadata) (*kolide.CarveMetadata, error) {
		return nil, fmt.Errorf("ouch!")
	}

	ctx := hostctx.NewContext(context.Background(), host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ouch!")
}

func TestCarveBeginEmptyError(t *testing.T) {
	ms := new(mock.Store)
	svc := service{ds: ms}
	ctx := hostctx.NewContext(context.Background(), kolide.Host{})

	_, err := svc.CarveBegin(ctx, kolide.CarveBeginPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size must be greater than 0")
}

func TestCarveBeginMissingHostError(t *testing.T) {
	ms := new(mock.Store)
	svc := service{ds: ms}

	_, err := svc.CarveBegin(context.Background(), kolide.CarveBeginPayload{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing host")
}

func TestCarveBeginBlockSizeMaxError(t *testing.T) {
	host := kolide.Host{ID: 3}
	payload := kolide.CarveBeginPayload{
		BlockCount: 10,
		BlockSize:  1024 * 1024 * 1024 * 1024,      // 1TB
		CarveSize:  10 * 1024 * 1024 * 1024 * 1024, // 10TB
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	svc := service{ds: ms}

	ctx := hostctx.NewContext(context.Background(), host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_size exceeds max")
}

func TestCarveBeginCarveSizeMaxError(t *testing.T) {
	host := kolide.Host{ID: 3}
	payload := kolide.CarveBeginPayload{
		BlockCount: 1024 * 1024,
		BlockSize:  10 * 1024 * 1024,               // 1TB
		CarveSize:  10 * 1024 * 1024 * 1024 * 1024, // 10TB
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	svc := service{ds: ms}

	ctx := hostctx.NewContext(context.Background(), host)

	_, err := svc.CarveBegin(ctx, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "carve_size exceeds max")
}

func TestCarveBeginCarveSizeError(t *testing.T) {
	host := kolide.Host{ID: 3}
	payload := kolide.CarveBeginPayload{
		BlockCount: 7,
		BlockSize:  13,
		CarveSize:  7*13 + 1,
		RequestId:  "carve_request",
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	ctx := hostctx.NewContext(context.Background(), host)

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
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		return nil, fmt.Errorf("ouch!")
	}

	payload := kolide.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		SessionId: sessionId,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ouch!")
}

func TestCarveCarveBlockRequestIdError(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}

	payload := kolide.CarveBlockPayload{
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
	metadata := &kolide.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}

	payload := kolide.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   23,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_id exceeds expected max")
}

func TestCarveCarveBlockBlockCountMatchError(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}

	payload := kolide.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   7,
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block_id does not match")
}

func TestCarveCarveBlockBlockSizeError(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}

	payload := kolide.CarveBlockPayload{
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
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.NewBlockFunc = func(carveId int64, blockId int64, data []byte) error {
		return fmt.Errorf("kaboom!")
	}

	payload := kolide.CarveBlockPayload{
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
	metadata := &kolide.CarveMetadata{
		ID:         2,
		HostId:     3,
		BlockCount: 23,
		BlockSize:  64,
		CarveSize:  23 * 64,
		RequestId:  "carve_request",
		SessionId:  sessionId,
		MaxBlock:   3,
	}
	payload := kolide.CarveBlockPayload{
		Data:      []byte("this is the carve data :)"),
		RequestId: "carve_request",
		SessionId: sessionId,
		BlockId:   4,
	}
	ms := new(mock.Store)
	svc := service{ds: ms}
	ms.CarveBySessionIdFunc = func(sessionId string) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.SessionId, sessionId)
		return metadata, nil
	}
	ms.NewBlockFunc = func(carveId int64, blockId int64, data []byte) error {
		assert.Equal(t, metadata.ID, carveId)
		assert.Equal(t, int64(4), blockId)
		assert.Equal(t, payload.Data, data)
		return nil
	}

	err := svc.CarveBlock(context.Background(), payload)
	require.NoError(t, err)
	assert.True(t, ms.NewBlockFuncInvoked)
}

func TestCarveGetBlock(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveFunc = func(carveId int64) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}
	ms.GetBlockFunc = func(metadataId int64, blockId int64) ([]byte, error) {
		assert.Equal(t, metadata.ID, metadataId)
		assert.Equal(t, int64(3), blockId)
		return []byte("foobar"), nil
	}

	data, err := svc.GetBlock(context.Background(), metadata.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, []byte("foobar"), data)
}

func TestCarveGetBlockNotAvailableError(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveFunc = func(carveId int64) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}

	// Block requested is great than max block
	_, err := svc.GetBlock(context.Background(), metadata.ID, 7)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet available")
}

func TestCarveGetBlockGetBlockError(t *testing.T) {
	sessionId := "foobar"
	metadata := &kolide.CarveMetadata{
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
	svc := service{ds: ms}
	ms.CarveFunc = func(carveId int64) (*kolide.CarveMetadata, error) {
		assert.Equal(t, metadata.ID, carveId)
		return metadata, nil
	}
	ms.GetBlockFunc = func(metadataId int64, blockId int64) ([]byte, error) {
		assert.Equal(t, metadata.ID, metadataId)
		assert.Equal(t, int64(3), blockId)
		return nil, fmt.Errorf("yow!!")
	}

	// Block requested is great than max block
	_, err := svc.GetBlock(context.Background(), metadata.ID, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "yow!!")
}


