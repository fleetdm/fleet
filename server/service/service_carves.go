package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

const (
	maxCarveSize = 8 * 1024 * 1024 * 1024 // 8GB
	maxBlockSize = 256 * 1024 * 1024      // 256MB
)

func (svc *Service) CarveBegin(ctx context.Context, payload fleet.CarveBeginPayload) (*fleet.CarveMetadata, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

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

	now := time.Now().UTC()
	carve := &fleet.CarveMetadata{
		Name:       fmt.Sprintf("%s-%s-%s", host.Hostname, now.Format(time.RFC3339), payload.RequestId),
		HostId:     host.ID,
		BlockCount: payload.BlockCount,
		BlockSize:  payload.BlockSize,
		CarveSize:  payload.CarveSize,
		CarveId:    payload.CarveId,
		RequestId:  payload.RequestId,
		SessionId:  sessionId.String(),
		CreatedAt:  now,
	}

	carve, err = svc.carveStore.NewCarve(ctx, carve)
	if err != nil {
		return nil, osqueryError{message: "internal error: new carve: " + err.Error()}
	}

	return carve, nil
}

func (svc *Service) CarveBlock(ctx context.Context, payload fleet.CarveBlockPayload) error {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	// Note host did not authenticate via node key. We need to authenticate them
	// by the session ID and request ID
	carve, err := svc.carveStore.CarveBySessionId(ctx, payload.SessionId)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find carve by session_id")
	}

	if payload.RequestId != carve.RequestId {
		return errors.New("request_id does not match")
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

	if err := svc.carveStore.NewBlock(ctx, carve, payload.BlockId, payload.Data); err != nil {
		return ctxerr.Wrap(ctx, err, "save block data")
	}

	return nil
}
