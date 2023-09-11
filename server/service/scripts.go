package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (async)
////////////////////////////////////////////////////////////////////////////////

type runScriptRequest struct {
	HostID         uint   `json:"host_id"`
	ScriptContents string `json:"script_contents"`
}

type runScriptResponse struct {
	Err         error  `json:"error,omitempty"`
	HostID      uint   `json:"host_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
}

func (r runScriptResponse) error() error { return r.Err }
func (r runScriptResponse) Status() int  { return http.StatusAccepted }

func runScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runScriptRequest)

	var noWait time.Duration
	result, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         req.HostID,
		ScriptContents: req.ScriptContents,
	}, noWait)
	if err != nil {
		return runScriptResponse{Err: err}, nil
	}
	return runScriptResponse{HostID: result.HostID, ExecutionID: result.ExecutionID}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (sync)
////////////////////////////////////////////////////////////////////////////////

type runScriptSyncResponse struct {
	Err error `json:"error,omitempty"`
	*fleet.HostScriptResult
	HostTimeout bool `json:"host_timeout"`
}

func (r runScriptSyncResponse) error() error { return r.Err }
func (r runScriptSyncResponse) Status() int {
	if r.HostTimeout {
		return http.StatusGatewayTimeout
	}
	return http.StatusOK
}

// this is to be used only by tests, to be able to use a shorter timeout.
var testRunScriptWaitForResult time.Duration

// waitForResultTime is the default timeout for the synchronous script execution.
const waitForResultTime = time.Minute

func runScriptSyncEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	waitForResult := waitForResultTime
	if testRunScriptWaitForResult != 0 {
		waitForResult = testRunScriptWaitForResult
	}

	req := request.(*runScriptRequest)
	result, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         req.HostID,
		ScriptContents: req.ScriptContents,
	}, waitForResult)
	var hostTimeout bool
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			return runScriptSyncResponse{Err: err}, nil
		}
		// We should still return the execution id and host id in this timeout case,
		// so the user knows what script request to look at in the UI. We cannot
		// return an error (field Err) in this case, as the errorer interface's
		// rendering logic would take over and only render the error part of the
		// response struct.
		hostTimeout = true
	}
	result.Message = result.UserMessage(hostTimeout)
	return runScriptSyncResponse{
		HostScriptResult: result,
		HostTimeout:      hostTimeout,
	}, nil
}

func (svc *Service) RunHostScript(ctx context.Context, request *fleet.HostScriptRequestPayload, waitForResult time.Duration) (*fleet.HostScriptResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// //////////////////////////////////////////////////////////////////////////////
// Get script result for a host
// //////////////////////////////////////////////////////////////////////////////
type getScriptResultRequest struct {
	ExecutionID string `url:"execution_id"`
}

type getScriptResultResponse struct {
	ScriptContents string `json:"script_contents"`
	ExitCode       *int64 `json:"exit_code"`
	Output         string `json:"output"`
	Message        string `json:"message"`
	HostName       string `json:"hostname"`
	HostTimeout    bool   `json:"host_timeout"`
	HostID         uint   `json:"host_id"`
	ExecutionID    string `json:"execution_id"`
	Runtime        int    `json:"runtime"`

	Err error `json:"error,omitempty"`
}

func (r getScriptResultResponse) error() error { return r.Err }

func getScriptResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getScriptResultRequest)
	scriptResult, err := svc.GetScriptResult(ctx, req.ExecutionID)
	if err != nil {
		return getScriptResultResponse{Err: err}, nil
	}

	// check if a minute has passed since the script was created at
	hostTimeout := scriptResult.HostTimeout(waitForResultTime)
	scriptResult.Message = scriptResult.UserMessage(hostTimeout)

	return &getScriptResultResponse{
		ScriptContents: scriptResult.ScriptContents,
		ExitCode:       scriptResult.ExitCode,
		Output:         scriptResult.Output,
		Message:        scriptResult.Message,
		HostName:       scriptResult.Hostname,
		HostTimeout:    hostTimeout,
		HostID:         scriptResult.HostID,
		ExecutionID:    scriptResult.ExecutionID,
		Runtime:        scriptResult.Runtime,
	}, nil
}

func (svc *Service) GetScriptResult(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Create a (saved) script (via a multipart file upload)
////////////////////////////////////////////////////////////////////////////////

type createScriptRequest struct {
	TeamID *uint
	Script *multipart.FileHeader
}

func (createScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var decoded createScriptRequest

	err := r.ParseMultipartForm(512 * units.MiB) // same in-memory size as for other multipart requests we have
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["team_id"]
	if len(val) > 0 {
		teamID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = ptr.Uint(uint(teamID))
	}

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

type createScriptResponse struct {
	Err      error `json:"error,omitempty"`
	ScriptID uint  `json:"script_id,omitempty"`
}

func (r createScriptResponse) error() error { return r.Err }

func createScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return &createScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	script, err := svc.NewScript(ctx, req.TeamID, filepath.Base(req.Script.Filename), scriptFile)
	if err != nil {
		return createScriptResponse{Err: err}, nil
	}
	return createScriptResponse{ScriptID: script.ID}, nil
}

func (svc *Service) NewScript(ctx context.Context, teamID *uint, name string, r io.Reader) (*fleet.Script, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
