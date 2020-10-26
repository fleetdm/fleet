package service

import (
	"context"
	"fmt"
	"io"

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

func (svc service) ListCarves(ctx context.Context, opt kolide.ListOptions) ([]*kolide.CarveMetadata, error) {
	return svc.ds.ListCarves(opt)
}

type carveReader struct {
	metadata  kolide.CarveMetadata
	ds        kolide.Datastore
	bytesRead int64
	curBlock  int64
	buffer    []byte
}

func newCarveReader(metadata kolide.CarveMetadata, ds kolide.Datastore) *carveReader {
	return &carveReader{
		metadata:  metadata,
		ds:        ds,
		bytesRead: 0,
		curBlock:  0,
	}
}

func (r *carveReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.bytesRead >= r.metadata.CarveSize {
		return 0, io.EOF
	}

	// Load data from the database if necessary
	if len(r.buffer) == 0 {
		var err error
		r.buffer, err = r.ds.GetBlock(r.metadata.ID, r.curBlock)
		if err != nil {
			return 0, errors.Wrapf(err, "get block %d", r.curBlock)
		}
		r.curBlock++
	}

	// Calculate length we can copy
	copyLen := len(p)
	if copyLen > len(r.buffer) {
		copyLen = len(r.buffer)
	}

	// Perform copy and clear copied contents from buffer
	copy(p, r.buffer[:copyLen])
	r.buffer = r.buffer[copyLen:]

	r.bytesRead += int64(copyLen)

	return copyLen, nil
}
