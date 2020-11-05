package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	hostctx "github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

const (
	maxCarveSize = 8 * 1024 * 1024 * 1024 // 8MB
	maxBlockSize = 256 * 1024 * 1024      // 256MB
)

func (svc service) CarveBegin(ctx context.Context, payload kolide.CarveBeginPayload) (*kolide.CarveMetadata, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	if payload.CarveSize == 0 {
		return nil, osqueryError{message: "carve_size must be greater than 0"}
	}

	if payload.BlockSize > maxBlockSize {
		return nil, osqueryError{message: "block_size exceeds max"}
	}
	if payload.CarveSize > maxCarveSize {
		return nil, osqueryError{message: "carve_size exceeds max"}
	}

	// The carve should have a total size that fits appropriately into the
	// number of blocks of the specified size.
	if payload.CarveSize <= (payload.BlockCount-1)*payload.BlockSize ||
		payload.CarveSize > payload.BlockCount*payload.BlockSize {
		return nil, osqueryError{message: "carve_size does not match block_size and block_count"}
	}

	sessionId, err := uuid.NewRandom()
	if err != nil {
		return nil, osqueryError{message: "internal error: generate session ID for carve: " + err.Error()}
	}

	carve := &kolide.CarveMetadata{
		Name:       fmt.Sprintf("%s-%s-%s", host.HostName, time.Now().Format(time.RFC3339), payload.RequestId),
		HostId:     host.ID,
		BlockCount: payload.BlockCount,
		BlockSize:  payload.BlockSize,
		CarveSize:  payload.CarveSize,
		CarveId:    payload.CarveId,
		RequestId:  payload.RequestId,
		SessionId:  sessionId.String(),
	}

	carve, err = svc.ds.NewCarve(carve)
	if err != nil {
		return nil, osqueryError{message: "internal error: new carve: " + err.Error()}
	}

	return carve, nil
}

func (svc service) CarveBlock(ctx context.Context, payload kolide.CarveBlockPayload) error {
	// Note host did not authenticate via node key. We need to authenticate them
	// by the session ID and request ID
	carve, err := svc.ds.CarveBySessionId(payload.SessionId)
	if err != nil {
		return errors.Wrap(err, "find carve by session_id")
	}

	if payload.RequestId != carve.RequestId {
		return fmt.Errorf("request_id does not match")
	}

	// Request is now authenticated

	if payload.BlockId > carve.BlockCount-1 {
		return fmt.Errorf("block_id exceeds expected max (%d): %d", carve.BlockCount-1, payload.BlockId)
	}

	if payload.BlockId != carve.MaxBlock+1 {
		return fmt.Errorf("block_id does not match expected block (%d): %d", carve.MaxBlock+1, payload.BlockId)
	}

	if int64(len(payload.Data)) > carve.BlockSize {
		return fmt.Errorf("exceeded declared block size %d: %d", carve.BlockSize, len(payload.Data))
	}

	if err := svc.ds.NewBlock(carve.ID, payload.BlockId, payload.Data); err != nil {
		return errors.Wrap(err, "save block data")
	}

	return nil
}

func (svc service) GetCarve(ctx context.Context, id int64) (*kolide.CarveMetadata, error) {
	return svc.ds.Carve(id)
}

func (svc service) ListCarves(ctx context.Context, opt kolide.CarveListOptions) ([]*kolide.CarveMetadata, error) {
	return svc.ds.ListCarves(opt)
}

func (svc service) GetBlock(ctx context.Context, carveId, blockId int64) ([]byte, error) {
	metadata, err := svc.ds.Carve(carveId)
	if err != nil {
		return nil, errors.Wrap(err, "get carve by name")
	}

	if metadata.Expired {
		return nil, fmt.Errorf("cannot get block for expired carve")
	}

	if blockId > metadata.MaxBlock {
		return nil, fmt.Errorf("block %d not yet available", blockId)
	}

	data, err := svc.ds.GetBlock(metadata.ID, blockId)
	if err != nil {
		return nil, errors.Wrapf(err, "get block %d", blockId)
	}

	return data, nil
}


