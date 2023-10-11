package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
)

func (svc *Service) RunHostScript(ctx context.Context, request *fleet.HostScriptRequestPayload, waitForResult time.Duration) (*fleet.HostScriptResult, error) {
	const maxPendingScriptAge = time.Minute // any script older than this is not considered pending anymore on that host

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

	// must check that only one of script id or contents is provided before
	// authorization, as the permissions are not the same if a script id is
	// provided. There's no harm in returning the error if this validation fails,
	// since both values are user-provided it doesn't leak any internal
	// information.
	if request.ScriptID != nil && request.ScriptContents != "" {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("script_id", `Only one of "script_id" or "script_contents" can be provided.`)
	}

	// authorize with the host's team and the script id provided, as both affect
	// the permissions.
	if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{TeamID: host.TeamID, ScriptID: request.ScriptID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

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

		contents, err := svc.ds.GetScriptContents(ctx, *request.ScriptID)
		if err != nil {
			if fleet.IsNotFound(err) {
				return nil, fleet.NewInvalidArgumentError("script_id", `No script exists for the provided "script_id".`).
					WithStatus(http.StatusNotFound)
			}
			return nil, err
		}
		request.ScriptContents = string(contents)
	}

	if err := fleet.ValidateHostScriptContents(request.ScriptContents); err != nil {
		return nil, fleet.NewInvalidArgumentError("script_contents", err.Error())
	}

	// host must be online
	if host.Status(time.Now()) != fleet.StatusOnline {
		return nil, fleet.NewInvalidArgumentError("host_id", fleet.RunScriptHostOfflineErrMsg)
	}

	pending, err := svc.ds.ListPendingHostScriptExecutions(ctx, request.HostID, maxPendingScriptAge)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host pending script executions")
	}
	if len(pending) > 0 {
		return nil, fleet.NewInvalidArgumentError(
			"script_contents", fleet.RunScriptAlreadyRunningErrMsg,
		).WithStatus(http.StatusConflict)
	}

	// create the script execution request, the host will be notified of the
	// script execution request via the orbit config's Notifications mechanism.
	script, err := svc.ds.NewHostScriptExecutionRequest(ctx, request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create script execution request")
	}
	script.Hostname = host.DisplayName()

	asyncExecution := waitForResult <= 0

	err = svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeRanScript{
			HostID:            host.ID,
			HostDisplayName:   host.DisplayName(),
			ScriptExecutionID: script.ExecutionID,
			Async:             asyncExecution,
		},
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for script execution request")
	}

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

	return scriptResult, nil
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
		ScriptContents: string(b),
	}
	if err := script.Validate(); err != nil {
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
		tm, err := svc.teamByIDOrName(ctx, teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team name for create script activity")
		}
		teamName = &tm.Name
	}

	if err := svc.ds.NewActivity(
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
		tm, err := svc.teamByIDOrName(ctx, script.TeamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team name for delete script activity")
		}
		teamName = &tm.Name
	}

	if err := svc.ds.NewActivity(
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

	if h.Platform != "darwin" {
		// only darwin is supported for now, all other platforms return empty results
		level.Debug(svc.logger).Log("msg", "unsupported platform for host script details", "platform", h.Platform, "host_id", h.ID)
		return []*fleet.HostScriptDetail{}, &fleet.PaginationMetadata{}, nil
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

	return svc.ds.GetHostScriptDetails(ctx, h.ID, h.TeamID, opt)
}

func (svc *Service) BatchSetScripts(ctx context.Context, maybeTmID *uint, maybeTmName *string, payloads []fleet.ScriptPayload, dryRun bool) error {
	if maybeTmID != nil && maybeTmName != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}

	var teamID *uint
	var teamName *string

	if maybeTmID != nil || maybeTmName != nil {
		team, err := svc.teamByIDOrName(ctx, maybeTmID, maybeTmName)
		if err != nil {
			return err
		}
		teamID = &team.ID
		teamName = &team.Name
	}

	if err := svc.authz.Authorize(ctx, &fleet.Script{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// any duplicate name in the provided set results in an error
	scripts := make([]*fleet.Script, 0, len(payloads))
	byName := make(map[string]bool, len(payloads))
	for i, p := range payloads {
		script := &fleet.Script{
			ScriptContents: string(p.ScriptContents),
			Name:           p.Name,
			TeamID:         teamID,
		}

		if err := script.Validate(); err != nil {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("scripts[%d]", i), err.Error()))
		}

		if byName[script.Name] {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("scripts[%d]", i), fmt.Sprintf("Couldn’t edit scripts. More than one script has the same file name: %q", script.Name)),
				"duplicate script by name")
		}
		byName[script.Name] = true
		scripts = append(scripts, script)
	}

	if dryRun {
		return nil
	}

	if err := svc.ds.BatchSetScripts(ctx, teamID, scripts); err != nil {
		return ctxerr.Wrap(ctx, err, "batch saving scripts")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedScript{
		TeamID:   teamID,
		TeamName: teamName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited scripts")
	}
	return nil
}
