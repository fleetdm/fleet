package server

import (
	"encoding/json"
	"fmt"

	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) EnrollAgent(ctx context.Context, enrollSecret, hostIdentifier string) (string, error) {
	if enrollSecret != svc.osqueryEnrollSecret {
		return "", errors.New(
			"Node key invalid",
			fmt.Sprintf("Invalid node key provided: %s", enrollSecret),
		)
	}

	host, err := svc.ds.EnrollHost(hostIdentifier, "", "", "", svc.osqueryNodeKeySize)
	if err != nil {
		return "", err
	}

	return host.NodeKey, nil
}

func (svc service) GetClientConfig(ctx context.Context, action string, data *json.RawMessage) (*kolide.OsqueryConfig, error) {
	return nil, nil
}

func (svc service) Log(ctx context.Context, logType string, data *json.RawMessage) error {
	return nil
}

func (svc service) GetDistributedQueries(ctx context.Context) (map[string]string, error) {
	var queries map[string]string

	queries["id1"] = "select * from osquery_info"

	return queries, nil
}

func (svc service) LogDistributedQueryResults(ctx context.Context, queries map[string][]map[string]string) error {
	return nil
}
