package service

import (
	"encoding/json"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Enroll Agent
////////////////////////////////////////////////////////////////////////////////

type enrollAgentRequest struct {
	EnrollSecret   string `json:"enroll_secret"`
	HostIdentifier string `json:"host_identifier"`
}

type enrollAgentResponse struct {
	NodeKey string `json:"node_key,omitempty"`
	Err     error  `json:"error,omitempty"`
}

func (r enrollAgentResponse) error() error { return r.Err }

func makeEnrollAgentEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(enrollAgentRequest)
		nodeKey, err := svc.EnrollAgent(ctx, req.EnrollSecret, req.HostIdentifier)
		if err != nil {
			return enrollAgentResponse{Err: err}, nil
		}
		return enrollAgentResponse{NodeKey: nodeKey}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Client Config
////////////////////////////////////////////////////////////////////////////////

type getClientConfigRequest struct {
	NodeKey string `json:"node_key"`
}

type getClientConfigResponse struct {
	kolide.OsqueryConfig
	Err error `json:"error,omitempty"`
}

func (r getClientConfigResponse) error() error { return r.Err }

func makeGetClientConfigEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		config, err := svc.GetClientConfig(ctx)
		if err != nil {
			return getClientConfigResponse{Err: err}, nil
		}
		return getClientConfigResponse{OsqueryConfig: *config}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Distributed Queries
////////////////////////////////////////////////////////////////////////////////

type getDistributedQueriesRequest struct {
	NodeKey string `json:"node_key"`
}

type getDistributedQueriesResponse struct {
	Queries map[string]string `json:"queries"`
	Err     error             `json:"error,omitempty"`
}

func (r getDistributedQueriesResponse) error() error { return r.Err }

func makeGetDistributedQueriesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		queries, err := svc.GetDistributedQueries(ctx)
		if err != nil {
			return getDistributedQueriesResponse{Err: err}, nil
		}
		return getDistributedQueriesResponse{Queries: queries}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Write Distributed Query Results
////////////////////////////////////////////////////////////////////////////////

type submitDistributedQueryResultsRequest struct {
	NodeKey  string                                `json:"node_key"`
	Results  kolide.OsqueryDistributedQueryResults `json:"queries"`
	Statuses map[string]string                     `json:"statuses"`
}

type submitDistributedQueryResultsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitDistributedQueryResultsResponse) error() error { return r.Err }

func makeSubmitDistributedQueryResultsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(submitDistributedQueryResultsRequest)
		err := svc.SubmitDistributedQueryResults(ctx, req.Results, req.Statuses)
		if err != nil {
			return submitDistributedQueryResultsResponse{Err: err}, nil
		}
		return submitDistributedQueryResultsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Submit Logs
////////////////////////////////////////////////////////////////////////////////

type submitLogsRequest struct {
	NodeKey string          `json:"node_key"`
	LogType string          `json:"log_type"`
	Data    json.RawMessage `json:"data"`
}

type submitLogsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitLogsResponse) error() error { return r.Err }

func makeSubmitLogsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(submitLogsRequest)

		var err error
		switch req.LogType {
		case "status":
			var statuses []kolide.OsqueryStatusLog
			if err := json.Unmarshal(req.Data, &statuses); err != nil {
				err = osqueryError{message: "unmarshalling status logs: " + err.Error()}
				break
			}

			err = svc.SubmitStatusLogs(ctx, statuses)
			if err != nil {
				break
			}

		case "result":
			var results []kolide.OsqueryResultLog
			if err := json.Unmarshal(req.Data, &results); err != nil {
				err = osqueryError{message: "unmarshalling result logs: " + err.Error()}
				break
			}
			err = svc.SubmitResultLogs(ctx, results)
			if err != nil {
				break
			}

		default:
			err = osqueryError{message: "unknown log type: " + req.LogType}
		}

		return submitLogsResponse{Err: err}, nil
	}
}
