package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error) {
	if enrollSecret != svc.config.Osquery.EnrollSecret {
		return "", errors.New(
			"Node key invalid",
			fmt.Sprintf("Invalid node key provided: %s", enrollSecret),
		)
	}

	host, err := svc.ds.EnrollHost(hostIdentifier, "", "", "", svc.config.Osquery.NodeKeySize)
	if err != nil {
		return "", err
	}

	return host.NodeKey, nil
}

func (svc service) GetClientConfig(ctx context.Context, action string, data json.RawMessage) (kolide.OsqueryConfig, error) {
	var config kolide.OsqueryConfig
	return config, nil
}

func (svc service) SubmitStatusLogs(ctx context.Context, logs []kolide.OsqueryResultLog) error {
	for _, log := range logs {
		err := json.NewEncoder(svc.osqueryStatusLogWriter).Encode(log)
		if err != nil {
			return errors.NewFromError(err, http.StatusInternalServerError, "error writing status log")
		}
	}
	return nil
}

func (svc service) SubmitResultsLogs(ctx context.Context, logs []kolide.OsqueryStatusLog) error {
	for _, log := range logs {
		err := json.NewEncoder(svc.osqueryResultsLogWriter).Encode(log)
		if err != nil {
			return errors.NewFromError(err, http.StatusInternalServerError, "error writing result log")
		}
	}
	return nil
}

func (svc service) GetDistributedQueries(ctx context.Context) (map[string]string, error) {
	var queries map[string]string

	queries["id1"] = "select * from osquery_info"

	return queries, nil
}

func (svc service) SubmitDistributedQueryResults(ctx context.Context, results kolide.OsqueryDistributedQueryResults) error {
	return nil
}
