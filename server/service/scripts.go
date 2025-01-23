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
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
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
	ScriptName     string `json:"script_name"`
	TeamID         uint   `json:"team_id"`
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
		ScriptName:     req.ScriptName,
		TeamID:         req.TeamID,
	}, noWait)
	if err != nil {
		return runScriptResponse{Err: err}, nil
	}
	return runScriptResponse{HostID: result.HostID, ExecutionID: result.ExecutionID}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Run Script on a Host (sync)
////////////////////////////////////////////////////////////////////////////////

type runScriptSyncRequest struct {
	HostID         uint   `json:"host_id"`
	ScriptID       *uint  `json:"script_id"`
	ScriptContents string `json:"script_contents"`
	ScriptName     string `json:"script_name"`
	TeamID         uint   `json:"team_id"`
}

type runScriptSyncResponse struct {
	Err error `json:"error,omitempty"`
	*fleet.HostScriptResult
	HostTimeout bool `json:"host_timeout"`
}

func (r runScriptSyncResponse) error() error { return r.Err }
func (r runScriptSyncResponse) Status() int {
	if r.HostTimeout {
		// The more proper response for a timeout on the server would be: StatusGatewayTimeout = 504
		// However, as described in https://github.com/fleetdm/fleet/issues/15430 we will send:
		// StatusRequestTimeout = 408 // RFC 9110, 15.5.9
		// See: https://github.com/fleetdm/fleet/issues/15430#issuecomment-1847345617
		return http.StatusRequestTimeout
	}
	return http.StatusOK
}

// this is to be used only by tests, to be able to use a shorter timeout.
var testRunScriptWaitForResult time.Duration

func runScriptSyncEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	waitForResult := scripts.MaxServerWaitTime
	if testRunScriptWaitForResult != 0 {
		waitForResult = testRunScriptWaitForResult
	}

	req := request.(*runScriptSyncRequest)
	result, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         req.HostID,
		ScriptID:       req.ScriptID,
		ScriptContents: req.ScriptContents,
		ScriptName:     req.ScriptName,
		TeamID:         req.TeamID,
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
	result.Message = result.UserMessage(hostTimeout, result.Timeout)
	return runScriptSyncResponse{
		HostScriptResult: result,
		HostTimeout:      hostTimeout,
	}, nil
}

func (svc *Service) GetScriptIDByName(ctx context.Context, scriptName string, teamID *uint) (uint, error) {
	// TODO: confirm auth level
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionRead); err != nil {
		return 0, err
	}

	id, err := svc.ds.GetScriptIDByName(ctx, scriptName, teamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return 0, fleet.NewInvalidArgumentError("script_name", fmt.Sprintf(`Script '%s' doesn’t exist.`, scriptName))
		}
		return 0, err
	}
	return id, nil
}

const maxPendingScripts = 1000

func (svc *Service) RunHostScript(ctx context.Context, request *fleet.HostScriptRequestPayload, waitForResult time.Duration) (*fleet.HostScriptResult, error) {
	// First check if scripts are disabled globally. If so, no need for further processing.
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, err
	}

	if cfg.ServerSettings.ScriptsDisabled {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewUserMessageError(errors.New(fleet.RunScriptScriptsDisabledGloballyErrMsg), http.StatusForbidden)
	}

	// Must check for presence of mutually exclusive parameters before
	// authorization, as the permissions are not the same in all cases.
	// There's no harm in returning the error if this validation fails,
	// since all values are user-provided it doesn't leak any internal
	// information.
	if err := request.ValidateParams(waitForResult); err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, err
	}

	if request.TeamID > 0 {
		lic, _ := license.FromContext(ctx)
		if !lic.IsPremium() {
			svc.authz.SkipAuthorization(ctx)
			return nil, fleet.ErrMissingLicense
		}
	}

	if request.ScriptContents != "" {
		if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{request.ScriptContents}); err != nil {
			svc.authz.SkipAuthorization(ctx)
			return nil, fleet.NewInvalidArgumentError("script", err.Error())
		}
	}

	if request.ScriptName != "" {
		scriptID, err := svc.GetScriptIDByName(ctx, request.ScriptName, &request.TeamID)
		if err != nil {
			return nil, err
		}
		request.ScriptID = &scriptID
	}

	// must load the host to get the team (cannot use lite, the last seen time is
	// required to check if it is online) to authorize with the proper team id.
	// We cannot first authorize if the user can list hosts, in case we
	// eventually allow a write-only role (e.g. gitops).
	host, err := svc.ds.Host(ctx, request.HostID)
	if err != nil {
		// if error is because the host does not exist, check first if the user
		// had access to run a script (to prevent leaking valid host ids).
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{}, fleet.ActionWrite); err != nil {
				return nil, err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get host lite")
	}

	if host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" {
		// fleetd is required to run scripts so if the host is enrolled via plain osquery we return
		// an error
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewUserMessageError(errors.New(fleet.RunScriptDisabledErrMsg), http.StatusUnprocessableEntity)
	}

	// If scripts are disabled (according to the last detail query), we return an error.
	// host.ScriptsEnabled may be nil for older orbit versions.
	if host.ScriptsEnabled != nil && !*host.ScriptsEnabled {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewUserMessageError(errors.New(fleet.RunScriptsOrbitDisabledErrMsg), http.StatusUnprocessableEntity)
	}

	maxPending := maxPendingScripts

	// authorize with the host's team
	if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	var isSavedScript bool
	if request.ScriptID != nil {
		script, err := svc.ds.Script(ctx, *request.ScriptID)
		if err != nil {
			if fleet.IsNotFound(err) {
				return nil, fleet.NewInvalidArgumentError("script_id", `No script exists for the provided "script_id".`).
					WithStatus(http.StatusNotFound)
			}
			return nil, err
		}
		var scriptTmID, hostTmID uint
		if script.TeamID != nil {
			scriptTmID = *script.TeamID
		}
		if host.TeamID != nil {
			hostTmID = *host.TeamID
		}
		if scriptTmID != hostTmID {
			return nil, fleet.NewInvalidArgumentError("script_id", `The script does not belong to the same team (or no team) as the host.`)
		}

		isQueued, err := svc.ds.IsExecutionPendingForHost(ctx, request.HostID, *request.ScriptID)
		if err != nil {
			return nil, err
		}

		if isQueued {
			return nil, fleet.NewInvalidArgumentError("script_id", `The script is already queued on the given host.`).WithStatus(http.StatusConflict)
		}

		contents, err := svc.ds.GetScriptContents(ctx, *request.ScriptID)
		if err != nil {
			if fleet.IsNotFound(err) {
				return nil, fleet.NewInvalidArgumentError("script_id", `No script exists for the provided "script_id".`).
					WithStatus(http.StatusNotFound)
			}
			return nil, err
		}
		request.ScriptContents = string(contents)
		request.ScriptContentID = script.ScriptContentID
		isSavedScript = true
	}

	if err := fleet.ValidateHostScriptContents(request.ScriptContents, isSavedScript); err != nil {
		return nil, fleet.NewInvalidArgumentError("script_contents", err.Error())
	}

	asyncExecution := waitForResult <= 0

	if !asyncExecution && host.Status(time.Now()) != fleet.StatusOnline {
		return nil, fleet.NewInvalidArgumentError("host_id", fleet.RunScriptHostOfflineErrMsg)
	}

	pending, err := svc.ds.ListPendingHostScriptExecutions(ctx, request.HostID, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host pending script executions")
	}
	if len(pending) >= maxPending {
		return nil, fleet.NewInvalidArgumentError(
			"script_id", "cannot queue more than 1000 scripts per host",
		).WithStatus(http.StatusConflict)
	}

	if !asyncExecution && len(pending) > 0 {
		return nil, fleet.NewInvalidArgumentError("script_id", fleet.RunScriptAlreadyRunningErrMsg).WithStatus(http.StatusConflict)
	}

	// create the script execution request, the host will be notified of the
	// script execution request via the orbit config's Notifications mechanism.
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		request.UserID = &ctxUser.ID
	}
	request.SyncRequest = !asyncExecution
	script, err := svc.ds.NewHostScriptExecutionRequest(ctx, request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create script execution request")
	}
	script.Hostname = host.DisplayName()

	if asyncExecution {
		// async execution, return
		return script, nil
	}

	ctx, cancel := context.WithTimeout(ctx, waitForResult)
	defer cancel()

	// if waiting for a result times out, we still want to return the script's
	// execution request information along with the error, so that the caller can
	// use the execution id for later checks.
	timeoutResult := script
	checkInterval := time.Second
	after := time.NewTimer(checkInterval)
	for {
		select {
		case <-ctx.Done():
			return timeoutResult, ctx.Err()
		case <-after.C:
			result, err := svc.ds.GetHostScriptExecutionResult(ctx, script.ExecutionID)
			if err != nil {
				// is that due to the context being canceled during the DB access?
				if ctxErr := ctx.Err(); ctxErr != nil {
					return timeoutResult, ctxErr
				}
				return nil, ctxerr.Wrap(ctx, err, "get script execution result")
			}
			if result.ExitCode != nil {
				// a result was received from the host, return
				result.Hostname = host.DisplayName()
				return result, nil
			}

			// at a second to every attempt, until it reaches 5s (then check every 5s)
			if checkInterval < 5*time.Second {
				checkInterval += time.Second
			}
			after.Reset(checkInterval)
		}
	}
}

// //////////////////////////////////////////////////////////////////////////////
// Get script result for a host
// //////////////////////////////////////////////////////////////////////////////
type getScriptResultRequest struct {
	ExecutionID string `url:"execution_id"`
}

type getScriptResultResponse struct {
	ScriptContents string    `json:"script_contents"`
	ScriptID       *uint     `json:"script_id"`
	ExitCode       *int64    `json:"exit_code"`
	Output         string    `json:"output"`
	Message        string    `json:"message"`
	HostName       string    `json:"hostname"`
	HostTimeout    bool      `json:"host_timeout"`
	HostID         uint      `json:"host_id"`
	ExecutionID    string    `json:"execution_id"`
	Runtime        int       `json:"runtime"`
	CreatedAt      time.Time `json:"created_at"`

	Err error `json:"error,omitempty"`
}

func (r getScriptResultResponse) error() error { return r.Err }

func getScriptResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getScriptResultRequest)
	scriptResult, err := svc.GetScriptResult(ctx, req.ExecutionID)
	if err != nil {
		return getScriptResultResponse{Err: err}, nil
	}

	// TODO: move this logic out of the endpoint function and consolidate in either the service
	// method or the fleet package
	hostTimeout := scriptResult.HostTimeout(scripts.MaxServerWaitTime)
	scriptResult.Message = scriptResult.UserMessage(hostTimeout, scriptResult.Timeout)

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
		CreatedAt:      scriptResult.CreatedAt,
	}, nil
}

func (svc *Service) GetScriptResult(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	scriptResult, err := svc.ds.GetHostScriptExecutionResult(ctx, execID)
	if err != nil {
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{}, fleet.ActionRead); err != nil {
				return nil, err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get script result")
	}

	if scriptResult.HostDeletedAt == nil {
		// host is not deleted, get it and authorize for the host's team
		host, err := svc.ds.HostLite(ctx, scriptResult.HostID)
		if err != nil {
			// if error is because the host does not exist, check first if the user
			// had access to run a script (to prevent leaking valid host ids).
			if fleet.IsNotFound(err) {
				if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{}, fleet.ActionRead); err != nil {
					return nil, err
				}
			}
			svc.authz.SkipAuthorization(ctx)
			return nil, ctxerr.Wrap(ctx, err, "get host lite")
		}
		if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{TeamID: host.TeamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
		scriptResult.Hostname = host.DisplayName()
	} else {
		// host was deleted, authorize for no-team as a fallback
		if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	return scriptResult, nil
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
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read script contents")
	}

	script := &fleet.Script{
		TeamID:         teamID,
		Name:           name,
		ScriptContents: file.Dos2UnixNewlines(string(b)),
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{script.ScriptContents}); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	if err := script.ValidateNewScript(); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	savedScript, err := svc.ds.NewScript(ctx, script)
	if err != nil {
		var (
			existsErr fleet.AlreadyExistsError
			fkErr     fleet.ForeignKeyError
		)
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("script", "A script with this name already exists.").WithStatus(http.StatusConflict)
		} else if errors.As(err, &fkErr) {
			err = fleet.NewInvalidArgumentError("team_id", "The team does not exist.").WithStatus(http.StatusNotFound)
		}
		return nil, ctxerr.Wrap(ctx, err, "create script")
	}

	var teamName *string
	if teamID != nil && *teamID != 0 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team name for create script activity")
		}
		teamName = &tm.Name
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAddedScript{
			TeamID:     teamID,
			TeamName:   teamName,
			ScriptName: script.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new activity for create script")
	}

	return savedScript, nil
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
	script, err := svc.authorizeScriptByID(ctx, scriptID, fleet.ActionWrite)
	if err != nil {
		return err
	}

	if err := svc.ds.DeleteScript(ctx, script.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete script")
	}

	var teamName *string
	if script.TeamID != nil && *script.TeamID != 0 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, script.TeamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team name for delete script activity")
		}
		teamName = &tm.Name
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedScript{
			TeamID:     script.TeamID,
			TeamName:   teamName,
			ScriptName: script.Name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "new activity for delete script")
	}

	return nil
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
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported for scripts
	opt.After = ""
	// custom ordering is not supported, always by name
	opt.OrderKey = "name"
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata for scripts
	opt.IncludeMetadata = true

	return svc.ds.ListScripts(ctx, teamID, opt)
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

type downloadFileResponse struct {
	Err         error `json:"error,omitempty"`
	filename    string
	content     []byte
	contentType string // optional, defaults to application/octet-stream
}

func (r downloadFileResponse) error() error { return r.Err }

func (r downloadFileResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.content)))
	if r.contentType == "" {
		r.contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", r.contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

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
		return downloadFileResponse{
			content:  content,
			filename: fmt.Sprintf("%s %s", time.Now().Format(time.DateOnly), script.Name),
		}, nil
	}
	return getScriptResponse{Script: script}, nil
}

func (svc *Service) GetScript(ctx context.Context, scriptID uint, withContent bool) (*fleet.Script, []byte, error) {
	script, err := svc.authorizeScriptByID(ctx, scriptID, fleet.ActionRead)
	if err != nil {
		return nil, nil, err
	}

	var content []byte
	if withContent {
		content, err = svc.ds.GetScriptContents(ctx, scriptID)
		if err != nil {
			return nil, nil, err
		}
	}
	return script, content, nil
}

////////////////////////////////////////////////////////////////////////////////
// Update Script Contents
////////////////////////////////////////////////////////////////////////////////

type updateScriptRequest struct {
	Script   *multipart.FileHeader
	ScriptID uint
}

func (updateScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var decoded updateScriptRequest

	err := r.ParseMultipartForm(512 * units.MiB) // same in-memory size as for other multipart requests we have
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["id"]
	if len(val) < 1 {
		return nil, &fleet.BadRequestError{Message: "no script id"}
	}
	scriptID, err := strconv.ParseUint(val[0], 10, 0)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode id in multipart form: %s", err.Error())}
	}
	decoded.ScriptID = uint(scriptID)

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

type updateScriptResponse struct {
	Err      error `json:"error,omitempty"`
	ScriptID uint  `json:"script_id,omitempty"`
}

func (r updateScriptResponse) error() error { return r.Err }

func updateScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return &updateScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	script, err := svc.UpdateScript(ctx, req.ScriptID, scriptFile)
	if err != nil {
		return updateScriptResponse{Err: err}, nil
	}
	return updateScriptResponse{ScriptID: script.ID}, nil
}

func (svc *Service) UpdateScript(ctx context.Context, scriptID uint, r io.Reader) (*fleet.Script, error) {
	script, err := svc.ds.Script(ctx, scriptID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding original script to update")
	}

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: script.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read script contents")
	}

	scriptContents := file.Dos2UnixNewlines(string(b))

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{scriptContents}); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	// Update the script
	savedScript, err := svc.ds.UpdateScriptContents(ctx, scriptID, scriptContents)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating script contents")
	}

	var teamName *string
	if script.TeamID != nil && *script.TeamID != 0 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, script.TeamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team name for create script activity")
		}
		teamName = &tm.Name
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeUpdatedScript{
			TeamID:     script.TeamID,
			TeamName:   teamName,
			ScriptName: script.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new activity for update script")
	}

	return savedScript, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host Script Details
////////////////////////////////////////////////////////////////////////////////

type getHostScriptDetailsRequest struct {
	HostID      uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type getHostScriptDetailsResponse struct {
	Scripts []*fleet.HostScriptDetail `json:"scripts"`
	Meta    *fleet.PaginationMetadata `json:"meta"`
	Err     error                     `json:"error,omitempty"`
}

func (r getHostScriptDetailsResponse) error() error { return r.Err }

func getHostScriptDetailsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostScriptDetailsRequest)
	scripts, meta, err := svc.GetHostScriptDetails(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return getHostScriptDetailsResponse{Err: err}, nil
	}
	return getHostScriptDetailsResponse{
		Scripts: scripts,
		Meta:    meta,
	}, nil
}

func (svc *Service) GetHostScriptDetails(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.HostScriptDetail, *fleet.PaginationMetadata, error) {
	h, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// if error is because the host does not exist, check first if the user
			// had global access (to prevent leaking valid host ids).
			if err := svc.authz.Authorize(ctx, &fleet.Script{}, fleet.ActionRead); err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, err
	}

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: h.TeamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported for scripts
	opt.After = ""
	// custom ordering is not supported, always by name
	opt.OrderKey = "name"
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata for scripts
	opt.IncludeMetadata = true

	return svc.ds.GetHostScriptDetails(ctx, h.ID, h.TeamID, opt, h.Platform)
}

////////////////////////////////////////////////////////////////////////////////
// Batch Replace Scripts
////////////////////////////////////////////////////////////////////////////////

type batchSetScriptsRequest struct {
	TeamID   *uint                 `json:"-" query:"team_id,optional"`
	TeamName *string               `json:"-" query:"team_name,optional"`
	DryRun   bool                  `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Scripts  []fleet.ScriptPayload `json:"scripts"`
}

type batchSetScriptsResponse struct {
	Scripts []fleet.ScriptResponse `json:"scripts"`
	Err     error                  `json:"error,omitempty"`
}

func (r batchSetScriptsResponse) error() error { return r.Err }

func batchSetScriptsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetScriptsRequest)
	scriptList, err := svc.BatchSetScripts(ctx, req.TeamID, req.TeamName, req.Scripts, req.DryRun)
	if err != nil {
		return batchSetScriptsResponse{Err: err}, nil
	}
	return batchSetScriptsResponse{Scripts: scriptList}, nil
}

func (svc *Service) BatchSetScripts(ctx context.Context, maybeTmID *uint, maybeTmName *string, payloads []fleet.ScriptPayload, dryRun bool) ([]fleet.ScriptResponse, error) {
	if maybeTmID != nil && maybeTmName != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}

	var teamID *uint
	var teamName *string

	if maybeTmID != nil || maybeTmName != nil {
		team, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, maybeTmID, maybeTmName)
		if err != nil {
			// If this is a dry run, the team may not have been created yet
			if dryRun && fleet.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		teamID = &team.ID
		teamName = &team.Name
	}

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// any duplicate name in the provided set results in an error
	scripts := make([]*fleet.Script, 0, len(payloads))
	byName := make(map[string]bool, len(payloads))
	scriptContents := []string{}
	for i, p := range payloads {
		script := &fleet.Script{
			ScriptContents: string(p.ScriptContents),
			Name:           p.Name,
			TeamID:         teamID,
		}

		if err := script.ValidateNewScript(); err != nil {
			return nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("scripts[%d]", i), err.Error()))
		}

		if byName[script.Name] {
			return nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("scripts[%d]", i), fmt.Sprintf("Couldn’t edit scripts. More than one script has the same file name: %q", script.Name)),
				"duplicate script by name")
		}
		byName[script.Name] = true
		scriptContents = append(scriptContents, script.ScriptContents)
		scripts = append(scripts, script)
	}

	if dryRun {
		return nil, nil
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, scriptContents); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	scriptResponses, err := svc.ds.BatchSetScripts(ctx, teamID, scripts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "batch saving scripts")
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedScript{
			TeamID:   teamID,
			TeamName: teamName,
		}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for edited scripts")
	}
	return scriptResponses, nil
}

func (svc *Service) authorizeScriptByID(ctx context.Context, scriptID uint, authzAction string) (*fleet.Script, error) {
	// first, get the script because we don't know which team id it is for.
	script, err := svc.ds.Script(ctx, scriptID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// couldn't get the script to have its team, authorize with a no-team
			// script as a fallback - the requested script does not exist so there's
			// no way to know what team it would be for, and returning a 404 without
			// authorization would leak the existing/non existing ids.
			if err := svc.authz.Authorize(ctx, &fleet.Script{}, authzAction); err != nil {
				return nil, err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get script")
	}

	// do the actual authorization with the script's team id
	if err := svc.authz.Authorize(ctx, script, authzAction); err != nil {
		return nil, err
	}
	return script, nil
}

////////////////////////////////////////////////////////////////////////////////
// Lock host
////////////////////////////////////////////////////////////////////////////////

type lockHostRequest struct {
	HostID  uint `url:"id"`
	ViewPin bool `query:"view_pin,optional"`
}

type lockHostResponse struct {
	Err        error  `json:"error,omitempty"`
	UnlockPIN  string `json:"unlock_pin,omitempty"`
	StatusCode int    `json:"-"`
}

func (r lockHostResponse) Status() int {
	if r.StatusCode != 0 {
		return r.StatusCode
	}
	return http.StatusNoContent
}
func (r lockHostResponse) error() error { return r.Err }

func lockHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*lockHostRequest)
	unlockPIN, err := svc.LockHost(ctx, req.HostID, req.ViewPin)
	if err != nil {
		return lockHostResponse{Err: err}, nil
	}
	if req.ViewPin && unlockPIN != "" {
		return lockHostResponse{UnlockPIN: unlockPIN, StatusCode: http.StatusOK}, nil
	}
	return lockHostResponse{}, nil
}

func (svc *Service) LockHost(ctx context.Context, _ uint, _ bool) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Unlock host
////////////////////////////////////////////////////////////////////////////////

type unlockHostRequest struct {
	HostID uint `url:"id"`
}

type unlockHostResponse struct {
	HostID    *uint  `json:"host_id,omitempty"`
	UnlockPIN string `json:"unlock_pin,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r unlockHostResponse) Status() int {
	if r.HostID != nil {
		// there is a response body
		return http.StatusOK
	}
	// no response body
	return http.StatusNoContent
}
func (r unlockHostResponse) error() error { return r.Err }

func unlockHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*unlockHostRequest)
	pin, err := svc.UnlockHost(ctx, req.HostID)
	if err != nil {
		return unlockHostResponse{Err: err}, nil
	}

	var resp unlockHostResponse
	// only macOS hosts return an unlock PIN, for other platforms the UnlockHost
	// call triggers the unlocking without further user action.
	if pin != "" {
		resp.HostID = &req.HostID
		resp.UnlockPIN = pin
	}
	return resp, nil
}

func (svc *Service) UnlockHost(ctx context.Context, hostID uint) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Wipe host
////////////////////////////////////////////////////////////////////////////////

type wipeHostRequest struct {
	HostID uint `url:"id"`
}

type wipeHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r wipeHostResponse) Status() int  { return http.StatusNoContent }
func (r wipeHostResponse) error() error { return r.Err }

func wipeHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*wipeHostRequest)
	if err := svc.WipeHost(ctx, req.HostID); err != nil {
		return wipeHostResponse{Err: err}, nil
	}
	return wipeHostResponse{}, nil
}

func (svc *Service) WipeHost(ctx context.Context, hostID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
