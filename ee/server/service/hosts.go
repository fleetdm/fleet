package service

import (
	"bufio"
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

// anchored, so that it matches to the end of the line
var scriptHashbangValidation = regexp.MustCompile(`^#!\s*/bin/sh\s*$`)

func (svc *Service) RunHostScript(ctx context.Context, request *fleet.HostScriptRequestPayload, waitForResult time.Duration) (*fleet.HostScriptResult, error) {
	const (
		maxScriptRuneLen    = 10000
		maxPendingScriptAge = time.Minute // any script older than this is not considered pending anymore on that host
	)

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
	if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if request.ScriptContents == "" {
		return nil, fleet.NewInvalidArgumentError("script_contents", "a script to execute is required")
	}
	// look for the script length in bytes first, as rune counting a huge string
	// can be expensive.
	if len(request.ScriptContents) > utf8.UTFMax*maxScriptRuneLen {
		return nil, fleet.NewInvalidArgumentError("script_contents", "Error: Script is too large. It’s limited to 10,000 characters (approximately 125 lines).")
	}
	// now that we know that the script is at most 4*maxScriptRuneLen bytes long,
	// we can safely count the runes for a precise check.
	if utf8.RuneCountInString(request.ScriptContents) > maxScriptRuneLen {
		return nil, fleet.NewInvalidArgumentError("script_contents", "Error: Script is too large. It’s limited to 10,000 characters (approximately 125 lines).")
	}

	// script must be a "text file", but that's not so simple to validate, so we
	// assume that if it is valid utf8 encoding, it is a text file (binary files
	// will often have invalid utf8 byte sequences).
	if !utf8.ValidString(request.ScriptContents) {
		return nil, fleet.NewInvalidArgumentError("script_contents", "Error: Wrong data format. Only plain text allowed.")
	}
	if strings.HasPrefix(request.ScriptContents, "#!") {
		// read the first line in a portable way
		s := bufio.NewScanner(strings.NewReader(request.ScriptContents))
		// if a hashbang is present, it can only be `/bin/sh` for now
		if s.Scan() && !scriptHashbangValidation.MatchString(s.Text()) {
			return nil, fleet.NewInvalidArgumentError("script_contents", `Error: Interpreter not supported. Bash scripts must run in "#!/bin/sh”.`)
		}
	}

	// host must be online
	if host.Status(time.Now()) != fleet.StatusOnline {
		return nil, fleet.NewInvalidArgumentError("host_id", "Error: Script can’t run on offline host.")
	}

	pending, err := svc.ds.ListPendingHostScriptExecutions(ctx, request.HostID, maxPendingScriptAge)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list host pending script executions")
	}
	if len(pending) > 0 {
		return nil, fleet.NewInvalidArgumentError(
			"script_contents", "Error: A script is already running on this host. Please wait about 1 minute to let it finish.",
		).WithStatus(http.StatusConflict)
	}

	// create the script execution request, the host will be notified of the
	// script execution request via the orbit config's Notifications mechanism.
	script, err := svc.ds.NewHostScriptExecutionRequest(ctx, request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create script execution request")
	}
	script.Hostname = host.DisplayName()

	if waitForResult <= 0 {
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
			if result.ExitCode.Valid {
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
