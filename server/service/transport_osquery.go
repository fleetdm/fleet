package service

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func decodeEnrollAgentRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req enrollAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return req, nil
}

func decodeGetClientConfigRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req getClientConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return req, nil
}

func decodeGetDistributedQueriesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req getDistributedQueriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	return req, nil
}

func decodeSubmitDistributedQueryResultsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// When a distributed query has no results, the JSON schema is
	// inconsistent, so we use this shim and massage into a consistent
	// schema. For example (simplified from actual osqueryd 1.8.2 output):
	// {
	// "queries": {
	//   "query_with_no_results": "", // <- Note string instead of array
	//   "query_with_results": [{"foo":"bar","baz":"bang"}]
	//  },
	// "node_key":"IGXCXknWQ1baTa8TZ6rF3kAPZ4\/aTsui"
	// }

	type distributedQueryResultsShim struct {
		NodeKey  string                     `json:"node_key"`
		Results  map[string]json.RawMessage `json:"queries"`
		Statuses map[string]interface{}     `json:"statuses"`
		Messages map[string]string          `json:"messages"`
	}

	var shim distributedQueryResultsShim
	if err := json.NewDecoder(r.Body).Decode(&shim); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	results := fleet.OsqueryDistributedQueryResults{}
	for query, raw := range shim.Results {
		queryResults := []map[string]string{}
		// No need to handle error because the empty array is what we
		// want if there was an error parsing the JSON (the error
		// indicates that osquery sent us incosistently schemaed JSON)
		_ = json.Unmarshal(raw, &queryResults)
		results[query] = queryResults
	}

	// Statuses were represented by strings in osquery < 3.0 and now
	// integers in osquery > 3.0. Massage to string for compatibility with
	// the service definition.
	statuses := map[string]fleet.OsqueryStatus{}
	for query, status := range shim.Statuses {
		switch s := status.(type) {
		case string:
			sint, err := strconv.Atoi(s)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "parse status to int")
			}
			statuses[query] = fleet.OsqueryStatus(sint)
		case float64:
			statuses[query] = fleet.OsqueryStatus(s)
		default:
			return nil, ctxerr.Errorf(ctx, "query status should be string or number, got %T", s)
		}
	}

	req := SubmitDistributedQueryResultsRequest{
		NodeKey:  shim.NodeKey,
		Results:  results,
		Statuses: statuses,
		Messages: shim.Messages,
	}

	return req, nil
}

func decodeSubmitLogsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var err error
	body := r.Body
	if r.Header.Get("content-encoding") == "gzip" {
		body, err = gzip.NewReader(body)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decoding gzip")
		}
		defer body.Close()
	}

	var req submitLogsRequest
	if err = json.NewDecoder(body).Decode(&req); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding JSON")
	}
	defer r.Body.Close()

	return req, nil
}
