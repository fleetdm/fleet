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
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (async)
////////////////////////////////////////////////////////////////////////////////

type runScriptRequest struct {
	HostID         uint   `json:"host_id"`
	ScriptID       *uint  `json:"script_id"`
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
		ScriptID:       req.ScriptID,
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
		ScriptID:       req.ScriptID,
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
	ScriptID       *uint  `json:"script_id"`
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
		ScriptID:       scriptResult.ScriptID,
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

////////////////////////////////////////////////////////////////////////////////
// Delete a (saved) script
////////////////////////////////////////////////////////////////////////////////

type deleteScriptRequest struct {
	ScriptID uint `url:"script_id"`
}

type deleteScriptResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteScriptResponse) error() error { return r.Err }
func (r deleteScriptResponse) Status() int  { return http.StatusNoContent }

func deleteScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteScriptRequest)
	err := svc.DeleteScript(ctx, req.ScriptID)
	if err != nil {
		return deleteScriptResponse{Err: err}, nil
	}
	return deleteScriptResponse{}, nil
}

func (svc *Service) DeleteScript(ctx context.Context, scriptID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// List (saved) scripts (paginated)
////////////////////////////////////////////////////////////////////////////////

type listScriptsRequest struct {
	TeamID      *uint             `query:"team_id,optional"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listScriptsResponse struct {
	Meta    *fleet.PaginationMetadata `json:"meta"`
	Scripts []*fleet.Script           `json:"scripts"`
	Err     error                     `json:"error,omitempty"`
}

func (r listScriptsResponse) error() error { return r.Err }

func listScriptsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listScriptsRequest)
	scripts, meta, err := svc.ListScripts(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listScriptsResponse{Err: err}, nil
	}
	return listScriptsResponse{
		Meta:    meta,
		Scripts: scripts,
	}, nil
}

func (svc *Service) ListScripts(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.Script, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get/download a (saved) script
////////////////////////////////////////////////////////////////////////////////

type getScriptRequest struct {
	ScriptID uint   `url:"script_id"`
	Alt      string `query:"alt,optional"`
}

type getScriptResponse struct {
	*fleet.Script
	Err error `json:"error,omitempty"`
}

func (r getScriptResponse) error() error { return r.Err }

type downloadScriptResponse struct {
	Err      error `json:"error,omitempty"`
	filename string
	content  []byte
}

func (r downloadScriptResponse) error() error { return r.Err }

func (r downloadScriptResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.content)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.filename))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.content); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func getScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getScriptRequest)

	downloadRequested := req.Alt == "media"
	script, content, err := svc.GetScript(ctx, req.ScriptID, downloadRequested)
	if err != nil {
		return getScriptResponse{Err: err}, nil
	}

	if downloadRequested {
		return downloadScriptResponse{
			content:  content,
			filename: fmt.Sprintf("%s %s", time.Now().Format(time.DateOnly), script.Name),
		}, nil
	}
	return getScriptResponse{Script: script}, nil
}

func (svc *Service) GetScript(ctx context.Context, scriptID uint, withContent bool) (*fleet.Script, []byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}
