package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	carvestorectx "github.com/fleetdm/fleet/v4/server/contexts/carvestore"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

////////////////////////////////////////////////////////////////////////////////
// List Carves
////////////////////////////////////////////////////////////////////////////////

type listCarvesRequest struct {
	ListOptions fleet.CarveListOptions `url:"carve_options"`
}

type listCarvesResponse struct {
	Carves []fleet.CarveMetadata `json:"carves"`
	Err    error                 `json:"error,omitempty"`
}

func (r listCarvesResponse) Error() error { return r.Err }

func listCarvesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listCarvesRequest)
	carves, err := svc.ListCarves(ctx, req.ListOptions)
	if err != nil {
		return listCarvesResponse{Err: err}, nil
	}

	resp := listCarvesResponse{}
	for _, carve := range carves {
		resp.Carves = append(resp.Carves, *carve)
	}
	return resp, nil
}

func (svc *Service) ListCarves(ctx context.Context, opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.carveStore.ListCarves(ctx, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Get Carve
////////////////////////////////////////////////////////////////////////////////

type getCarveRequest struct {
	ID int64 `url:"id"`
}

type getCarveResponse struct {
	Carve fleet.CarveMetadata `json:"carve"`
	Err   error               `json:"error,omitempty"`
}

func (r getCarveResponse) Error() error { return r.Err }

func getCarveEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCarveRequest)
	carve, err := svc.GetCarve(ctx, req.ID)
	if err != nil {
		return getCarveResponse{Err: err}, nil
	}

	return getCarveResponse{Carve: *carve}, nil
}

func (svc *Service) GetCarve(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.carveStore.Carve(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Get Carve Block
////////////////////////////////////////////////////////////////////////////////

type getCarveBlockRequest struct {
	ID      int64 `url:"id"`
	BlockId int64 `url:"block_id"`
}

type getCarveBlockResponse struct {
	Data []byte `json:"data"`
	Err  error  `json:"error,omitempty"`
}

func (r getCarveBlockResponse) Error() error { return r.Err }

func getCarveBlockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCarveBlockRequest)
	data, err := svc.GetBlock(ctx, req.ID, req.BlockId)
	if err != nil {
		return getCarveBlockResponse{Err: err}, nil
	}

	return getCarveBlockResponse{Data: data}, nil
}

func (svc *Service) GetBlock(ctx context.Context, carveId, blockId int64) ([]byte, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	metadata, err := svc.carveStore.Carve(ctx, carveId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get carve by name")
	}

	if metadata.Expired {
		return nil, errors.New("cannot get block for expired carve")
	}

	if blockId > metadata.MaxBlock {
		return nil, fmt.Errorf("block %d not yet available", blockId)
	}

	data, err := svc.carveStore.GetBlock(ctx, metadata, blockId)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "get block %d", blockId)
	}

	return data, nil
}

////////////////////////////////////////////////////////////////////////////////
// Begin File Carve
////////////////////////////////////////////////////////////////////////////////

type carveBeginRequest struct {
	NodeKey    string `json:"node_key"`
	BlockCount int64  `json:"block_count"`
	BlockSize  int64  `json:"block_size"`
	CarveSize  int64  `json:"carve_size"`
	CarveId    string `json:"carve_id"`
	RequestId  string `json:"request_id"`
}

func (r *carveBeginRequest) hostNodeKey() string {
	return r.NodeKey
}

type carveBeginResponse struct {
	SessionId string `json:"session_id"`
	Success   bool   `json:"success,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r carveBeginResponse) Error() error { return r.Err }

func carveBeginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*carveBeginRequest)

	payload := fleet.CarveBeginPayload{
		BlockCount: req.BlockCount,
		BlockSize:  req.BlockSize,
		CarveSize:  req.CarveSize,
		CarveId:    req.CarveId,
		RequestId:  req.RequestId,
	}

	carve, err := svc.CarveBegin(ctx, payload)
	if err != nil {
		return carveBeginResponse{Err: err}, nil
	}

	return carveBeginResponse{SessionId: carve.SessionId, Success: true}, nil
}

const (
	maxCarveSize = 8 * 1024 * 1024 * 1024 // 8GB
	maxBlockSize = 256 * 1024 * 1024      // 256MB
)

func (svc *Service) CarveBegin(ctx context.Context, payload fleet.CarveBeginPayload) (*fleet.CarveMetadata, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, newOsqueryError("internal error: missing host from request context")
	}

	if payload.CarveSize == 0 {
		return nil, newOsqueryError("carve_size must be greater than 0")
	}

	if payload.BlockSize > maxBlockSize {
		return nil, newOsqueryError("block_size exceeds max")
	}
	if payload.CarveSize > maxCarveSize {
		return nil, newOsqueryError("carve_size exceeds max")
	}

	// The carve should have a total size that fits appropriately into the
	// number of blocks of the specified size.
	if payload.CarveSize <= (payload.BlockCount-1)*payload.BlockSize ||
		payload.CarveSize > payload.BlockCount*payload.BlockSize {
		return nil, newOsqueryError("carve_size does not match block_size and block_count")
	}

	sessionId, err := uuid.NewRandom()
	if err != nil {
		return nil, newOsqueryError("internal error: generate session ID for carve: " + err.Error())
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
		return nil, newOsqueryError("internal error: new carve: " + err.Error())
	}

	return carve, nil
}

////////////////////////////////////////////////////////////////////////////////
// Receive Block for File Carve
////////////////////////////////////////////////////////////////////////////////

type carveBlockRequest struct {
	BlockId   int64  `json:"block_id"`
	SessionId string `json:"session_id"`
	RequestId string `json:"request_id"`
	Data      []byte `json:"data"`
}

type carveBlockResponse struct {
	Success bool  `json:"success,omitempty"`
	Err     error `json:"error,omitempty"`
}

func (r carveBlockResponse) Error() error { return r.Err }

func (r carveBlockRequest) DecodeRequest(ctx context.Context, req *http.Request) (any, error) {
	carveStore := carvestorectx.FromContext(ctx)
	if carveStore == nil {
		return nil, ctxerr.New(ctx, "missing carve store from context")
	}

	decoder := json.NewDecoder(req.Body)

	newAuthRequiredError := func(err error) error {
		return ctxerr.Wrap(ctx, fleet.NewAuthFailedError(err.Error()), "authentication error")
	}

	// 1. Must start with {
	if t, err := decoder.Token(); err != nil {
		return nil, newAuthRequiredError(fmt.Errorf("expected object start: %w", err))
	} else if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, newAuthRequiredError(fmt.Errorf("expected '{', got %v", t))
	}

	// 2. Parse field by field.
	var (
		blockID   int64
		sessionID string
		requestID string
		data      []byte

		authenticated bool
	)
	for decoder.More() {
		t, err := decoder.Token()
		if err != nil {
			return nil, newAuthRequiredError(fmt.Errorf("reading field name: %w", err))
		}
		fieldName, ok := t.(string)
		if !ok {
			return nil, newAuthRequiredError(fmt.Errorf("expected string field name, got %T: %v", t, t))
		}
		switch fieldName {
		case "block_id":
			if err := decoder.Decode(&blockID); err != nil {
				return nil, newAuthRequiredError(fmt.Errorf("invalid block_id: %w", err))
			}
		case "session_id":
			if err := decoder.Decode(&sessionID); err != nil {
				return nil, newAuthRequiredError(fmt.Errorf("invalid session_id: %w", err))
			}
		case "request_id":
			if err := decoder.Decode(&requestID); err != nil {
				return nil, newAuthRequiredError(fmt.Errorf("invalid request_id: %w", err))
			}

			if sessionID == "" {
				return nil, newAuthRequiredError(errors.New("missing session_id"))
			}
			carve, err := carveStore.CarveBySessionId(ctx, sessionID)
			if err != nil {
				return nil, newAuthRequiredError(fmt.Errorf("carve by session ID: %w", err))
			}
			if requestID != carve.RequestId {
				return nil, newAuthRequiredError(errors.New("request_id does not match session"))
			}
			authenticated = true
		case "data":
			if !authenticated {
				return nil, newAuthRequiredError(errors.New("unauthenticated data"))
			}
			// Request is authenticated, thus we proceed to parse "data" field.
			if err := decoder.Decode(&data); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "invalid data")
			}
		default:
			// If a new field is added to the API we'll need to account for it here.
			return nil, newAuthRequiredError(fmt.Errorf("unexpected field: %q", fieldName))
		}
	}

	if !authenticated {
		return nil, newAuthRequiredError(errors.New("unauthenticated request"))
	}

	// 3. Expect closing }
	if t, err := decoder.Token(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expected object end")
	} else if delim, ok := t.(json.Delim); !ok || delim != '}' {
		return nil, ctxerr.Errorf(ctx, "expected '}', got %v", t)
	}

	return &carveBlockRequest{
		BlockId:   blockID,
		SessionId: sessionID,
		RequestId: requestID,
		Data:      data,
	}, nil
}

func carveBlockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*carveBlockRequest)

	payload := fleet.CarveBlockPayload{
		SessionId: req.SessionId,
		RequestId: req.RequestId,
		BlockId:   req.BlockId,
		Data:      req.Data,
	}

	err := svc.CarveBlock(ctx, payload)
	if err != nil {
		return carveBlockResponse{Err: err}, nil
	}

	return carveBlockResponse{Success: true}, nil
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

	if err := svc.validateCarveBlock(payload, carve); err != nil {
		carve.Error = ptr.String(err.Error())
		if errRecord := svc.carveStore.UpdateCarve(ctx, carve); errRecord != nil {
			logging.WithExtras(ctx, "validate_carve_error", errRecord, "carve_id", carve.ID)
		}

		return ctxerr.Wrap(ctx, badRequest("validate carve block"), err.Error())
	}

	if err := svc.carveStore.NewBlock(ctx, carve, payload.BlockId, payload.Data); err != nil {
		carve.Error = ptr.String(err.Error())
		if errRecord := svc.carveStore.UpdateCarve(ctx, carve); errRecord != nil {
			logging.WithExtras(ctx, "record_carve_error", errRecord, "carve_id", carve.ID)
		}

		return ctxerr.Wrap(ctx, err, "save carve block data")
	}

	return nil
}

func (svc *Service) validateCarveBlock(payload fleet.CarveBlockPayload, carve *fleet.CarveMetadata) error {
	if payload.BlockId > carve.BlockCount-1 {
		return fmt.Errorf("block_id exceeds expected max (%d): %d", carve.BlockCount-1, payload.BlockId)
	}

	if payload.BlockId != carve.MaxBlock+1 {
		return fmt.Errorf("block_id does not match expected block (%d): %d", carve.MaxBlock+1, payload.BlockId)
	}

	if int64(len(payload.Data)) > carve.BlockSize {
		return fmt.Errorf("exceeded declared block size %d: %d", carve.BlockSize, len(payload.Data))
	}

	return nil
}
