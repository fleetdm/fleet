package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	carvestorectx "github.com/fleetdm/fleet/v4/server/contexts/carvestore"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

// List Carves
func listCarvesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListCarvesRequest)
	carves, err := svc.ListCarves(ctx, req.ListOptions)
	if err != nil {
		return fleet.ListCarvesResponse{Err: err}, nil
	}

	resp := fleet.ListCarvesResponse{}
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

// Get Carve
func getCarveEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetCarveRequest)
	carve, err := svc.GetCarve(ctx, req.ID)
	if err != nil {
		return fleet.GetCarveResponse{Err: err}, nil
	}

	return fleet.GetCarveResponse{Carve: *carve}, nil
}

func (svc *Service) GetCarve(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.carveStore.Carve(ctx, id)
}

// Get Carve Block
func getCarveBlockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetCarveBlockRequest)
	data, err := svc.GetBlock(ctx, req.ID, req.BlockId)
	if err != nil {
		return fleet.GetCarveBlockResponse{Err: err}, nil
	}

	return fleet.GetCarveBlockResponse{Data: data}, nil
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

// Begin File Carve
func carveBeginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CarveBeginRequest)

	payload := fleet.CarveBeginPayload{
		BlockCount: req.BlockCount,
		BlockSize:  req.BlockSize,
		CarveSize:  req.CarveSize,
		CarveId:    req.CarveId,
		RequestId:  req.RequestId,
	}

	carve, err := svc.CarveBegin(ctx, payload)
	if err != nil {
		return fleet.CarveBeginResponse{Err: err}, nil
	}

	return fleet.CarveBeginResponse{SessionId: carve.SessionId, Success: true}, nil
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

	// sessionId generated here is overriden if the carve store is S3 (in svc.carveStore.NewCarve).
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

// Receive Block for File Carve
// DecodeRequest for the /api/v1/osquery/carve/block endpoint performs raw JSON parsing
// to prevent DoS attacks on this unauthenticated endpoint.
// Carve block requests are authenticated by their "session_id" and "request_id".
// The osquery API sends the "session_id" and "request_id" in the JSON object in the body that
// also includes the "data" field with the actual "block". If Fleet parses the full JSON to extract
// the "session_id" and "request_id" then attackers could DoS Fleet by sending big JSON documents.
// To prevent such an attack, we rely on the stability of the osquery carve endpoints (they have been
// stable for many years) and parse the body field by field. The "session_id" and "request_id" always
// come before the "data" field; thus Fleet will extract "session_id" and "request_id", perform authentication
// and if the credentials are valid parse and decode the "data" field.

type decodeCarveBlockRequest struct{}

func (decodeCarveBlockRequest) DecodeRequest(ctx context.Context, req *http.Request) (any, error) {
	carveStore := carvestorectx.FromContext(ctx)
	if carveStore == nil {
		return nil, ctxerr.New(ctx, "missing carve store from context")
	}

	newAuthRequiredError := func(err error) error {
		// We don't want to return details to clients.
		return ctxerr.Wrap(ctx, fleet.NewAuthFailedError(err.Error()), "authentication error")
	}

	readUntil := func(maxToRead int, endChar byte) (string, error) {
		var s strings.Builder
		endCharFound := false
		for i := 0; i <= maxToRead; i++ {
			character := make([]byte, 1)
			if _, err := req.Body.Read(character); err != nil {
				return "", fmt.Errorf("failed to read character: %w", err)
			}
			if character[0] == endChar {
				endCharFound = true
				break
			}
			s.Write(character)
		}
		if !endCharFound {
			return "", fmt.Errorf(`end character not found: %q`, s.String())
		}
		return s.String(), nil
	}

	// 1. Must start with {
	delimiter := make([]byte, 1)
	if _, err := req.Body.Read(delimiter); err != nil {
		return nil, newAuthRequiredError(fmt.Errorf("failed to read object start: %w", err))
	}
	if string(delimiter) != "{" {
		return nil, newAuthRequiredError(fmt.Errorf("expected '{', got %q", string(delimiter)))
	}
	// 2. Must continue with "block_id":.
	blockIDKey := make([]byte, 11)
	if _, err := req.Body.Read(blockIDKey); err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`failed to read "block_id" key: %w`, err))
	}
	if string(blockIDKey) != `"block_id":` {
		return nil, newAuthRequiredError(fmt.Errorf(`expected "block_id":, got %q`, string(blockIDKey)))
	}
	// 3. Must continue with a number.
	const maxNumberOfDigits = 19
	blockIDStr, err := readUntil(maxNumberOfDigits, ',')
	if err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`invalid "block_id" field: %w`, err))
	}
	blockID, err := strconv.ParseInt(blockIDStr, 10, 64)
	if err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`invalid "block_id" format: %w`, err))
	}
	// 4. Must continue with "session_id":".
	sessionIDKey := make([]byte, 14)
	if _, err := req.Body.Read(sessionIDKey); err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`failed to read "session_id" key: %w`, err))
	}
	if string(sessionIDKey) != `"session_id":"` {
		return nil, newAuthRequiredError(fmt.Errorf(`expected "session_id":", got %q`, string(sessionIDKey)))
	}
	// 5. Must continue with a string (up to 255 chars).
	const maxSizeSessionID = 255 // defined in DB
	sessionID, err := readUntil(maxSizeSessionID, '"')
	if err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`invalid "session_id" field: %w`, err))
	}
	if sessionID == "" {
		return nil, newAuthRequiredError(errors.New("empty session_id"))
	}
	// 6. Must continue with ,"request_id":".
	requestIDKey := make([]byte, 15)
	if _, err := req.Body.Read(requestIDKey); err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`failed to read "request_id" key: %w`, err))
	}
	if string(requestIDKey) != `,"request_id":"` {
		return nil, newAuthRequiredError(fmt.Errorf(`expected ,"request_id":", got %q`, string(requestIDKey)))
	}
	// 7. Must continue with a string (up to 64 chars).
	const maxSizeRequestID = 64 // defined in DB.
	requestID, err := readUntil(maxSizeRequestID, '"')
	if err != nil {
		return nil, newAuthRequiredError(fmt.Errorf(`invalid "request_id" field: %w`, err))
	}
	if requestID == "" {
		return nil, newAuthRequiredError(errors.New("empty request_id"))
	}

	//
	// 8. Perform authentication before continuing with the read and parse of the "data" field.
	//

	carve, err := carveStore.CarveBySessionId(ctx, sessionID)
	if err != nil {
		return nil, newAuthRequiredError(fmt.Errorf("carve by session ID: %w", err))
	}
	if requestID != carve.RequestId {
		return nil, newAuthRequiredError(errors.New("request_id does not match session"))
	}

	//
	// 9. At this point the request is authenticated.
	//

	// Must continue with ,"data":".
	dataKey := make([]byte, 9)
	if _, err := req.Body.Read(dataKey); err != nil {
		return nil, ctxerr.Wrap(ctx, err, `failed to read "data" key`)
	}
	if string(dataKey) != `,"data":"` {
		return nil, ctxerr.New(ctx, fmt.Sprintf(`expected ,"data":", got %s`, dataKey))
	}

	// 10. Must continue with a string with the base64 encoded data.
	encodedData, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, `read "data" field`)
	}
	if len(encodedData) < 2 {
		return nil, ctxerr.New(ctx, `invalid "data" ending length`)
	}
	if ending := string(encodedData[len(encodedData)-2:]); ending != `"}` {
		return nil, ctxerr.New(ctx, fmt.Sprintf(`invalid "data" ending: %s`, ending))
	}
	// 11. Skip ending `"}`
	encodedData = encodedData[:len(encodedData)-2]
	// 12. Decode the base64-encoded field.
	data := make([]byte, base64.RawStdEncoding.DecodedLen(len(encodedData)))
	n, err := base64.StdEncoding.Decode(data, encodedData)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "base64 decode block data")
	}
	data = data[:n]

	return &fleet.CarveBlockRequest{
		BlockId:   blockID,
		SessionId: sessionID,
		RequestId: requestID,
		Data:      data,
	}, nil
}

func carveBlockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CarveBlockRequest)

	payload := fleet.CarveBlockPayload{
		SessionId: req.SessionId,
		RequestId: req.RequestId,
		BlockId:   req.BlockId,
		Data:      req.Data,
	}

	err := svc.CarveBlock(ctx, payload)
	if err != nil {
		return fleet.CarveBlockResponse{Err: err}, nil
	}

	return fleet.CarveBlockResponse{Success: true}, nil
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
