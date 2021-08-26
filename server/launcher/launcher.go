package launcher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/kolide/osquery-go/plugin/distributed"
	"github.com/kolide/osquery-go/plugin/logger"
	"github.com/pkg/errors"

	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
)

// launcherWrapper wraps the TLS interface.
type launcherWrapper struct {
	tls            fleet.OsqueryService
	logger         log.Logger
	healthCheckers map[string]health.Checker
}

func (svc *launcherWrapper) RequestEnrollment(ctx context.Context, enrollSecret, hostIdentifier string) (string, bool, error) {
	nodeKey, err := svc.tls.EnrollAgent(ctx, enrollSecret, hostIdentifier, map[string](map[string]string){})
	if err != nil {
		if authErr, ok := err.(nodeInvalidErr); ok {
			return "", authErr.NodeInvalid(), err
		}
		return "", false, err
	}
	return nodeKey, false, nil
}

func (svc *launcherWrapper) RequestConfig(ctx context.Context, nodeKey string) (string, bool, error) {
	newCtx, invalid, err := svc.authenticateHost(ctx, nodeKey)
	if err != nil {
		return "", invalid, err
	}

	config, err := svc.tls.GetClientConfig(newCtx)
	if err != nil {
		return "", false, errors.Wrap(err, "get config for launcher")
	}

	if options, ok := config["options"].(map[string]interface{}); ok {
		// Launcher manages plugins so remove them from configuration if they exist.
		for _, optionName := range []string{"distributed_plugin", "logger_plugin"} {
			delete(options, optionName)
		}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return "", false, errors.Wrap(err, "encoding config for launcher")
	}

	return string(configJSON), false, nil
}

func (svc *launcherWrapper) RequestQueries(ctx context.Context, nodeKey string) (*distributed.GetQueriesResult, bool, error) {
	newCtx, invalid, err := svc.authenticateHost(ctx, nodeKey)
	if err != nil {
		return nil, invalid, err
	}

	queryMap, accelerate, err := svc.tls.GetDistributedQueries(newCtx)
	if err != nil {
		return nil, false, errors.Wrap(err, "get queries for launcher")
	}

	result := &distributed.GetQueriesResult{
		Queries:           queryMap,
		AccelerateSeconds: int(accelerate),
	}

	return result, false, nil
}

func (svc *launcherWrapper) PublishLogs(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
	newCtx, invalid, err := svc.authenticateHost(ctx, nodeKey)
	if err != nil {
		return "", "", invalid, errors.Wrap(err, "authenticate launcher")
	}

	switch logType {
	case logger.LogTypeStatus:
		var statuses []json.RawMessage
		for _, log := range logs {
			statuses = append(statuses, []byte(log))
		}
		err = svc.tls.SubmitStatusLogs(newCtx, statuses)
		return "", "", false, errors.Wrap(err, "submit status logs from launcher")
	case logger.LogTypeSnapshot, logger.LogTypeString:
		var results []json.RawMessage
		for _, log := range logs {
			results = append(results, []byte(log))
		}
		err = svc.tls.SubmitResultLogs(newCtx, results)
		return "", "", false, errors.Wrap(err, "submit result logs from launcher")
	default:
		// We have a logTypeAgent which is not there in the osquery-go enum.
		// See https://github.com/kolide/launcher/issues/183
		panic(fmt.Sprintf("%s log type not implemented", logType))
	}
}

func (svc *launcherWrapper) PublishResults(ctx context.Context, nodeKey string, results []distributed.Result) (string, string, bool, error) {
	newCtx, invalid, err := svc.authenticateHost(ctx, nodeKey)
	if err != nil {
		return "", "", invalid, err
	}

	osqueryResults := make(fleet.OsqueryDistributedQueryResults, len(results))
	statuses := make(map[string]fleet.OsqueryStatus, len(results))

	for _, result := range results {
		statuses[result.QueryName] = fleet.OsqueryStatus(result.Status)
		osqueryResults[result.QueryName] = result.Rows
	}

	// TODO can Launcher expose the error messages?
	messages := make(map[string]string)
	err = svc.tls.SubmitDistributedQueryResults(newCtx, osqueryResults, statuses, messages)
	return "", "", false, errors.Wrap(err, "submit launcher results")
}

func (svc *launcherWrapper) CheckHealth(ctx context.Context) (int32, error) {
	healthy := health.CheckHealth(svc.logger, svc.healthCheckers)
	if !healthy {
		return 1, nil
	}
	return 0, nil
}

// authenticateHost verifies the host node key using the TLS API and returns back a
// context which includes the host as a context value.
// In the fleet.OsqueryService authentication is done via endpoint middleware, but all launcher endpoints require
// an explicit return for NodeInvalid, so we check in this helper method instead.
func (svc *launcherWrapper) authenticateHost(ctx context.Context, nodeKey string) (context.Context, bool, error) {
	node, err := svc.tls.AuthenticateHost(ctx, nodeKey)
	if err != nil {
		if authErr, ok := err.(nodeInvalidErr); ok {
			return ctx, authErr.NodeInvalid(), err
		}
		return ctx, false, err
	}

	ctx = host.NewContext(ctx, *node)
	return ctx, false, nil
}

type nodeInvalidErr interface {
	error
	NodeInvalid() bool
}
