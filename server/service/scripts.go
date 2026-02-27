package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/gorilla/mux"
)

// Run Script on a Host (async)
func runScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.RunScriptRequest)

	var noWait time.Duration
	result, err := svc.RunHostScript(ctx, &fleet.HostScriptRequestPayload{
		HostID:         req.HostID,
		ScriptID:       req.ScriptID,
		ScriptContents: req.ScriptContents,
		ScriptName:     req.ScriptName,
		TeamID:         req.TeamID,
	}, noWait)
	if err != nil {
		return fleet.RunScriptResponse{Err: err}, nil
	}
	return fleet.RunScriptResponse{HostID: result.HostID, ExecutionID: result.ExecutionID}, nil
}

// Run Script on a Host (sync)
// this is to be used only by tests, to be able to use a shorter timeout.
var testRunScriptWaitForResult time.Duration

func runScriptSyncEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	waitForResult := scripts.MaxServerWaitTime
	if testRunScriptWaitForResult != 0 {
		waitForResult = testRunScriptWaitForResult
	}

	req := request.(*fleet.RunScriptSyncRequest)
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
			return fleet.RunScriptSyncResponse{Err: err}, nil
		}
		// We should still return the execution id and host id in this timeout case,
		// so the user knows what script request to look at in the UI. We cannot
		// return an error (field Err) in this case, as the Errorer interface's
		// rendering logic would take over and only render the error part of the
		// response struct.
		hostTimeout = true
	}
	result.Message = result.UserMessage(hostTimeout, result.Timeout)
	return fleet.RunScriptSyncResponse{
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
			return nil, fleet.NewInvalidArgumentError("script_id", `The script does not belong to the same fleet (or "Unassigned") as the host.`)
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

func getScriptResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetScriptResultRequest)
	scriptResult, err := svc.GetScriptResult(ctx, req.ExecutionID)
	if err != nil {
		return fleet.GetScriptResultResponse{Err: err}, nil
	}

	return setUpGetScriptResultResponse(scriptResult), nil
}

func setUpGetScriptResultResponse(scriptResult *fleet.HostScriptResult) *fleet.GetScriptResultResponse {
	hostTimeout := scriptResult.HostTimeout(scripts.MaxServerWaitTime)
	scriptResult.Message = scriptResult.UserMessage(hostTimeout, scriptResult.Timeout)

	return &fleet.GetScriptResultResponse{
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
	}
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

// Create a (saved) script (via a multipart file upload)
type decodeCreateScriptRequest struct{}

func (decodeCreateScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var decoded fleet.CreateScriptRequest

	err := parseMultipartForm(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	val := r.MultipartForm.Value["fleet_id"]
	if len(val) > 0 {
		fleetID, err := strconv.ParseUint(val[0], 10, 64)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode fleet_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = ptr.Uint(uint(fleetID)) // nolint:gosec // ignore G115
	}

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

func createScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.CreateScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return &fleet.CreateScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	script, err := svc.NewScript(ctx, req.TeamID, filepath.Base(req.Script.Filename), scriptFile)
	if err != nil {
		return fleet.CreateScriptResponse{Err: err}, nil
	}
	return fleet.CreateScriptResponse{ScriptID: script.ID}, nil
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
			err = fleet.NewInvalidArgumentError("team_id/fleet_id", "The fleet does not exist.").WithStatus(http.StatusNotFound)
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

// Delete a (saved) script
func deleteScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteScriptRequest)
	err := svc.DeleteScript(ctx, req.ScriptID)
	if err != nil {
		return fleet.DeleteScriptResponse{Err: err}, nil
	}
	return fleet.DeleteScriptResponse{}, nil
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

// List (saved) scripts (paginated)
func listScriptsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListScriptsRequest)
	scripts, meta, err := svc.ListScripts(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return fleet.ListScriptsResponse{Err: err}, nil
	}
	return fleet.ListScriptsResponse{
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

// Get/download a (saved) script
func getScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetScriptRequest)

	downloadRequested := req.Alt == "media"
	script, content, err := svc.GetScript(ctx, req.ScriptID, downloadRequested)
	if err != nil {
		return fleet.GetScriptResponse{Err: err}, nil
	}

	if downloadRequested {
		return fleet.DownloadFileResponse{
			Content:  content,
			Filename: fmt.Sprintf("%s %s", time.Now().Format(time.DateOnly), script.Name),
		}, nil
	}
	return fleet.GetScriptResponse{Script: script}, nil
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

// Update Script Contents
type decodeUpdateScriptRequest struct{}

func (decodeUpdateScriptRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var decoded fleet.UpdateScriptRequest

	err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	vars := mux.Vars(r)
	scriptIDStr, ok := vars["script_id"]
	if !ok {
		return nil, &fleet.BadRequestError{Message: "missing script id"}
	}
	scriptID, err := strconv.ParseUint(scriptIDStr, 10, 64)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "invalid script id"}
	}
	// Check if scriptID exceeds the maximum value for uint, code linter
	if scriptID > uint64(^uint(0)) {
		return nil, &fleet.BadRequestError{Message: "script id out of bounds"}
	}

	decoded.ScriptID = uint(scriptID)

	fhs, ok := r.MultipartForm.File["script"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for script"}
	}
	decoded.Script = fhs[0]

	return &decoded, nil
}

func updateScriptEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UpdateScriptRequest)

	scriptFile, err := req.Script.Open()
	if err != nil {
		return &fleet.UpdateScriptResponse{Err: err}, nil
	}
	defer scriptFile.Close()

	script, err := svc.UpdateScript(ctx, req.ScriptID, scriptFile)
	if err != nil {
		return fleet.UpdateScriptResponse{Err: err}, nil
	}
	return fleet.UpdateScriptResponse{ScriptID: script.ID}, nil
}

func (svc *Service) UpdateScript(ctx context.Context, scriptID uint, r io.Reader) (*fleet.Script, error) {
	script, err := svc.ds.Script(ctx, scriptID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
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

	if err := fleet.ValidateHostScriptContents(scriptContents, true); err != nil {
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

// Get Host Script Details
func getHostScriptDetailsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetHostScriptDetailsRequest)
	scripts, meta, err := svc.GetHostScriptDetails(ctx, req.HostID, req.ListOptions)
	if err != nil {
		return fleet.GetHostScriptDetailsResponse{Err: err}, nil
	}
	return fleet.GetHostScriptDetailsResponse{
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

// Batch Replace Scripts
// TODO - remove these once we retire batch script summary endpoint and code.
type (
	batchScriptExecutionSummaryRequest  fleet.BatchScriptExecutionStatusRequest
	batchScriptExecutionSummaryResponse struct {
		ScriptID    uint      `json:"script_id" db:"script_id"`
		ScriptName  string    `json:"script_name" db:"script_name"`
		TeamID      *uint     `json:"team_id" db:"team_id" renameto:"fleet_id"`
		CreatedAt   time.Time `json:"created_at" db:"created_at"`
		NumTargeted *uint     `json:"targeted" db:"num_targeted"`
		NumPending  *uint     `json:"pending" db:"num_pending"`
		NumRan      *uint     `json:"ran" db:"num_ran"`
		NumErrored  *uint     `json:"errored" db:"num_errored"`
		NumCanceled *uint     `json:"canceled" db:"num_canceled"`
		Err         error     `json:"error,omitempty"`
	}
)

func batchSetScriptsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchSetScriptsRequest)
	scriptList, err := svc.BatchSetScripts(ctx, req.TeamID, req.TeamName, req.Scripts, req.DryRun)
	if err != nil {
		return fleet.BatchSetScriptsResponse{Err: err}, nil
	}
	return fleet.BatchSetScriptsResponse{Scripts: scriptList}, nil
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

func (r batchScriptExecutionSummaryResponse) Error() error { return r.Err }

// Deprecated summary endpoint, to be removed in favor of the status endpoint
// once the batch script details page is ready.
func batchScriptExecutionSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*batchScriptExecutionSummaryRequest)
	summary, err := svc.BatchScriptExecutionSummary(ctx, req.BatchExecutionID)
	if err != nil {
		return batchScriptExecutionSummaryResponse{Err: err}, nil
	}
	return batchScriptExecutionSummaryResponse{
		ScriptID:    *summary.ScriptID,
		ScriptName:  summary.ScriptName,
		TeamID:      summary.TeamID,
		CreatedAt:   summary.CreatedAt,
		NumTargeted: summary.NumTargeted,
		NumPending:  summary.NumPending,
		NumRan:      summary.NumRan,
		NumErrored:  summary.NumErrored,
		NumCanceled: summary.NumCanceled,
	}, nil
}

func batchScriptExecutionHostResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchScriptExecutionHostResultsRequest)
	hosts, meta, count, err := svc.BatchScriptExecutionHostResults(ctx, req.BatchExecutionID, req.BatchExecutionStatus, req.ListOptions)
	if err != nil {
		return fleet.BatchScriptExecutionHostResultsResponse{Err: err}, nil
	}
	return fleet.BatchScriptExecutionHostResultsResponse{Hosts: hosts, Meta: *meta, Count: count}, nil
}

func batchScriptExecutionStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchScriptExecutionStatusRequest)
	status, err := svc.BatchScriptExecutionStatus(ctx, req.BatchExecutionID)
	if err != nil {
		return fleet.BatchScriptExecutionStatusResponse{Err: err}, nil
	}
	return fleet.BatchScriptExecutionStatusResponse{BatchActivity: *status}, nil
}

func batchScriptExecutionListEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchScriptExecutionListRequest)

	page := 0
	pageSize := 0
	if req.Page != nil {
		page = int(*req.Page) //nolint:gosec // dismiss G115
	}
	if req.PerPage != nil {
		pageSize = int(*req.PerPage) //nolint:gosec // dismiss G115
	}
	// Set query offset based on the specified page and page size.
	offset := uint(page * pageSize) //nolint:gosec // dismiss G115
	filter := fleet.BatchExecutionStatusFilter{
		TeamID: &req.TeamID,
		Status: req.Status,
		Offset: &offset,
		Limit:  req.PerPage,
	}
	list, count, err := svc.BatchScriptExecutionList(ctx, filter)
	if err != nil {
		return fleet.BatchScriptExecutionStatusResponse{Err: err}, nil
	}
	// Get the # of results returned by this query.
	listSize := len(list)
	// We have previous results if we're not on the first page.
	hasPreviousResults := req.Page != nil && *req.Page > 0
	// Calculate the number of results on this page + all previous pages.
	resultsSeen := (page * pageSize) + listSize
	// If it's less than the total count, we have more results.
	hasNextResults := resultsSeen < int(count)
	return fleet.BatchScriptExecutionListResponse{
		BatchScriptExecutions: list,
		Count:                 uint(count), //nolint:gosec // dismiss G115
		Meta: fleet.PaginationMetadata{
			HasNextResults:     hasNextResults,
			HasPreviousResults: hasPreviousResults,
		},
	}, nil
}

func (svc *Service) BatchScriptExecutionSummary(ctx context.Context, batchExecutionID string) (*fleet.BatchActivity, error) {
	summary, err := svc.ds.BatchExecuteSummary(ctx, batchExecutionID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get batch script summary")
	}

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: summary.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return summary, nil
}

func batchScriptCancelEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchScriptCancelRequest)
	if err := svc.BatchScriptCancel(ctx, req.BatchExecutionID); err != nil {
		return fleet.BatchScriptCancelResponse{Err: err}, nil
	}

	return fleet.BatchScriptCancelResponse{}, nil
}

func (svc *Service) BatchScriptCancel(ctx context.Context, batchExecutionID string) error {
	summaryList, err := svc.ds.ListBatchScriptExecutions(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &batchExecutionID,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get batch script summary")
	}

	// If the list is empty, it means the batch execution does not exist.
	if len(summaryList) == 0 {
		// If the user can see a no-team script, we can return a 404 because they have global access.
		// Otherwise, we return a 403 to avoid leaking info about which IDs exist.
		if err := svc.authz.Authorize(ctx, &fleet.Script{}, fleet.ActionRead); err != nil {
			return err
		}
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "get batch script status")
	}

	if len(summaryList) > 1 {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("batch_execution_id", "expected a single batch execution status, got multiple"))
	}

	summary := (summaryList)[0]

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: summary.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}
	if err := svc.ds.CancelBatchScript(ctx, batchExecutionID); err != nil {
		return ctxerr.Wrap(ctx, err, "canceling batch script")
	}

	batchActivity, err := svc.ds.GetBatchActivity(ctx, batchExecutionID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting canceled activity stats")
	}

	ctxUser := authz.UserFromContext(ctx)

	targeted := uint(0)
	if batchActivity.NumTargeted != nil {
		// No nil dereference in case this is not set for some reason
		targeted = *batchActivity.NumTargeted
	}

	canceled := uint(0)
	if batchActivity.NumCanceled != nil {
		canceled = *batchActivity.NumCanceled
	}

	if err := svc.NewActivity(ctx, ctxUser, fleet.ActivityTypeBatchScriptCanceled{
		BatchExecutionID: batchExecutionID,
		ScriptName:       batchActivity.ScriptName,
		HostCount:        targeted,
		CanceledCount:    canceled,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for cancel batch script")
	}

	return nil
}

func (svc *Service) BatchScriptExecutionStatus(ctx context.Context, batchExecutionID string) (*fleet.BatchActivity, error) {
	summaryList, err := svc.ds.ListBatchScriptExecutions(ctx, fleet.BatchExecutionStatusFilter{
		ExecutionID: &batchExecutionID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get batch script summary")
	}

	// If the list is empty, it means the batch execution does not exist.
	if len(summaryList) == 0 {
		// If the user can see a no-team script, we can return a 404 because they have global access.
		// Otherwise, we return a 403 to avoid leaking info about which IDs exist.
		if err := svc.authz.Authorize(ctx, &fleet.Script{}, fleet.ActionRead); err != nil {
			return nil, err
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get batch script status")
	}

	if len(summaryList) > 1 {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("batch_execution_id", "expected a single batch execution status, got multiple"))
	}

	summary := (summaryList)[0]

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: summary.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return &summary, nil
}

func (svc *Service) BatchScriptExecutionHostResults(ctx context.Context, batchExecutionID string, status fleet.BatchScriptExecutionStatus, opt fleet.ListOptions) (hosts []fleet.BatchScriptHost, meta *fleet.PaginationMetadata, count uint, err error) {
	// Get the batch activity.
	batchActivity, err := svc.ds.GetBatchActivity(ctx, batchExecutionID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "getting batch activity")
	}
	if batchActivity.ScriptID == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "batch activity has no script ID")
	}

	// Get the script referred to by the batch activity.
	script, err := svc.ds.Script(ctx, *batchActivity.ScriptID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "getting script")
	}
	if script == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "script not found")
	}

	// Authorize based on the script's team ID.
	if err = svc.authz.Authorize(ctx, &fleet.Script{TeamID: script.TeamID}, fleet.ActionRead); err != nil {
		return nil, nil, 0, err
	}

	// Validate the supplied batch execution status.
	if !status.IsValid() {
		return nil, nil, 0, fleet.NewInvalidArgumentError("batch_execution_status", "invalid batch execution status")
	}

	// Always include pagination info.
	opt.IncludeMetadata = true
	// Default sort order is name ascending.
	if opt.OrderKey == "" {
		opt.OrderKey = "display_name"
		opt.OrderDirection = fleet.OrderAscending
	}

	hosts, meta, count, err = svc.ds.ListBatchScriptHosts(ctx, batchExecutionID, status, opt)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "list batch script hosts")
	}

	return hosts, meta, count, nil
}

func (svc *Service) BatchScriptExecutionList(ctx context.Context, filter fleet.BatchExecutionStatusFilter) ([]fleet.BatchActivity, int64, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: filter.TeamID}, fleet.ActionRead); err != nil {
		return nil, 0, err
	}
	// Get the count first.
	count, err := svc.ds.CountBatchScriptExecutions(ctx, filter)
	if err != nil {
		return nil, 0, nil
	}

	summaryList, err := svc.ds.ListBatchScriptExecutions(ctx, filter)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "get batch script list")
	}

	return summaryList, count, nil
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

// Bulk script execution
func batchScriptRunEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchScriptRunRequest)
	batchID, err := svc.BatchScriptExecute(ctx, req.ScriptID, req.HostIDs, req.Filters, req.NotBefore)
	if err != nil {
		return fleet.BatchScriptRunResponse{Err: err}, nil
	}
	return fleet.BatchScriptRunResponse{BatchExecutionID: batchID}, nil
}

const MAX_BATCH_EXECUTION_HOSTS = 5000

func (svc *Service) BatchScriptExecute(ctx context.Context, scriptID uint, hostIDs []uint, filters *map[string]any, notBefore *time.Time) (string, error) {
	// If we are given both host IDs and filters, return an error
	if len(hostIDs) > 0 && filters != nil {
		return "", fleet.NewInvalidArgumentError("filters", "cannot specify both host_ids and filters")
	}

	// First check if scripts are disabled globally. If so, no need for further processing.
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return "", err
	}

	if cfg.ServerSettings.ScriptsDisabled {
		svc.authz.SkipAuthorization(ctx)
		return "", fleet.NewUserMessageError(errors.New(fleet.RunScriptScriptsDisabledGloballyErrMsg), http.StatusForbidden)
	}

	// Use the authorize script by ID to handle authz
	script, err := svc.authorizeScriptByID(ctx, scriptID, fleet.ActionWrite)
	if err != nil {
		return "", err
	}

	var userId *uint
	ctxUser := authz.UserFromContext(ctx)
	if ctxUser != nil {
		userId = &ctxUser.ID
	}

	var hosts []*fleet.Host

	// If we are given filters, we need to get the hosts matching those filters
	if filters != nil {
		opt, lid, err := hostListOptionsFromFilters(filters)
		if err != nil {
			return "", err
		}

		if opt == nil {
			return "", fleet.NewInvalidArgumentError("filters", "filters must be a valid set of host list options")
		}

		if opt.TeamFilter == nil {
			return "", fleet.NewInvalidArgumentError("filters", "filters must include a team filter")
		}

		filter := fleet.TeamFilter{User: ctxUser, IncludeObserver: true}

		// Load hosts, either from label if provided or from all hosts.
		if lid != nil {
			hosts, err = svc.ds.ListHostsInLabel(ctx, filter, *lid, *opt)
		} else {
			opt.DisableIssues = true // intentionally ignore failing policies
			hosts, err = svc.ds.ListHosts(ctx, filter, *opt)
		}

		if err != nil {
			return "", err
		}
	} else {
		// Get the hosts matching the host IDs
		hosts, err = svc.ds.ListHostsLiteByIDs(ctx, hostIDs)
		if err != nil {
			return "", err
		}
	}
	if len(hosts) == 0 {
		return "", &fleet.BadRequestError{Message: "no hosts match the specified host IDs"}
	}

	if len(hosts) > MAX_BATCH_EXECUTION_HOSTS {
		return "", fleet.NewInvalidArgumentError("filters", "too_many_hosts")
	}

	hostIDsToExecute := make([]uint, 0, len(hosts))
	for _, host := range hosts {
		hostIDsToExecute = append(hostIDsToExecute, host.ID)
		if host.TeamID == nil && script.TeamID == nil {
			continue
		}
		if host.TeamID == nil || script.TeamID == nil || *host.TeamID != *script.TeamID {
			return "", fleet.NewInvalidArgumentError("host_ids", "all hosts must be on the same fleet as the script")
		}
	}

	if notBefore == nil || notBefore.Before(time.Now()) {
		batchID, err := svc.ds.BatchExecuteScript(ctx, userId, scriptID, hostIDsToExecute)
		if err != nil {
			return "", fleet.NewUserMessageError(err, http.StatusBadRequest)
		}

		if err := svc.NewActivity(ctx, ctxUser, fleet.ActivityTypeRanScriptBatch{
			ScriptName:       script.Name,
			BatchExecutionID: batchID,
			HostCount:        uint(len(hostIDsToExecute)),
			TeamID:           script.TeamID,
		}); err != nil {
			return "", ctxerr.Wrap(ctx, err, "creating activity for batch run scripts")
		}

		return batchID, nil
	}

	notBeforeUTC := notBefore.UTC()
	batchID, err := svc.ds.BatchScheduleScript(ctx, userId, scriptID, hostIDsToExecute, notBeforeUTC)
	if err != nil {
		return "", fleet.NewUserMessageError(err, http.StatusBadRequest)
	}

	if err := svc.NewActivity(ctx, ctxUser, fleet.ActivityTypeBatchScriptScheduled{
		ScriptName:       &script.Name,
		BatchExecutionID: batchID,
		HostCount:        uint(len(hostIDsToExecute)),
		TeamID:           script.TeamID,
		NotBefore:        &notBeforeUTC,
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "creating activity for scheduled batch run scripts")
	}

	return batchID, nil
}

// Lock host
func lockHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.LockHostRequest)
	unlockPIN, err := svc.LockHost(ctx, req.HostID, req.ViewPin)
	if err != nil {
		return fleet.LockHostResponse{Err: err}, nil
	}
	// We bail from locking if the host is locked or wiped, so we can assume the host is unlocked at this point
	response := &fleet.LockHostResponse{DeviceStatus: fleet.DeviceStatusUnlocked, PendingAction: fleet.PendingActionLock}

	if req.ViewPin && unlockPIN != "" {
		response.UnlockPIN = unlockPIN
	}
	return response, nil
}

func (svc *Service) LockHost(ctx context.Context, _ uint, _ bool) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

// Unlock host
func unlockHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UnlockHostRequest)
	pin, err := svc.UnlockHost(ctx, req.HostID)
	if err != nil {
		return fleet.UnlockHostResponse{Err: err}, nil
	}

	// We bail if a host is unlocked or wiped, so we can assume the host is locked at this point
	resp := fleet.UnlockHostResponse{HostID: &req.HostID, DeviceStatus: fleet.DeviceStatusLocked, PendingAction: fleet.PendingActionUnlock}
	// only macOS hosts return an unlock PIN, for other platforms the UnlockHost
	// call triggers the unlocking without further user action.
	if pin != "" {
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

// //////////////////////////////////////////////////////////////////////////////
// Wipe host
// //////////////////////////////////////////////////////////////////////////////

func wipeHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.WipeHostRequest)
	if err := svc.WipeHost(ctx, req.HostID, req.Metadata); err != nil {
		return fleet.WipeHostResponse{Err: err}, nil
	}
	// We bail if a host is locked or wiped, so we can assume the host is unlocked at this point
	return fleet.WipeHostResponse{DeviceStatus: fleet.DeviceStatusUnlocked, PendingAction: fleet.PendingActionWipe}, nil
}

func (svc *Service) WipeHost(ctx context.Context, _ uint, _ *fleet.MDMWipeMetadata) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
