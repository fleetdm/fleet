package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	hostctx "github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) CarveBegin(ctx context.Context, payload kolide.CarveBeginPayload) (*kolide.CarveMetadata, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, osqueryError{message: "internal error: missing host from request context"}
	}

	sessionId, err := uuid.NewRandom()
	if err != nil {
		return nil, osqueryError{message: "internal error: generate session ID for carve: " + err.Error()}
	}

	// TODO Fleet should enforce some kind of limit on carve sizes

	carve := &kolide.CarveMetadata{
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
		return errors.Wrap(err, "find carve by session ID")
	}

	if payload.RequestId != carve.RequestId {
		return fmt.Errorf("request ID does not match")
	}

	// Request is now authenticated

	if payload.BlockId > carve.BlockCount-1 {
		return fmt.Errorf("block ID exceeds expected max (%d): %d", carve.BlockCount-1, payload.BlockId)
	}

	if payload.BlockId != carve.MaxBlock+1 {
		return fmt.Errorf("block ID does not match expected block (%d): %d", carve.MaxBlock+1, payload.BlockId)
	}

	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return errors.Wrap(err, "base64 decode data")
	}

	if err := svc.ds.NewBlock(carve.ID, payload.BlockId, data); err != nil {
		return errors.Wrap(err, "save block data")
	}

	return nil
}

func (svc service) ListCarves(ctx context.Context, opt kolide.ListOptions) ([]*kolide.CarveMetadata, error) {
	return svc.ds.ListCarves(opt)
}
