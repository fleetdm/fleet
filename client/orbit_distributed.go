package client

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// The types below follow the osquery TLS distributed protocol wire format.
// With the WebSocket transport active, orbit acts as osquery's distributed
// plugin and calls the distributed endpoints itself, authenticating with its
// orbit node key (accepted by the server as a fallback to the osquery node
// key). The key is sent both in the body ("node_key") and in the
// "Authorization: NodeKey <key>" header: the server consults exactly one of
// the two depending on its osquery.allow_body_auth_fallback setting, so
// sending both works in either mode.

// DistributedReadResponse is the response of the osquery distributed/read
// endpoint: the queries the host should run, keyed by query name.
type DistributedReadResponse struct {
	Queries    map[string]string `json:"queries"`
	Discovery  map[string]string `json:"discovery"`
	Accelerate uint              `json:"accelerate"`
}

type distributedReadRequest struct {
	NodeKey string `json:"node_key"`
}

func (r *distributedReadRequest) SetOrbitNodeKey(nodeKey string) { r.NodeKey = nodeKey }

func (r *distributedReadRequest) setRequestHeaders(req *http.Request) {
	setNodeKeyHeader(req, r.NodeKey)
}

// DistributedWriteRequest is the body of the osquery distributed/write
// endpoint: results of the queries previously returned by distributed/read.
type DistributedWriteRequest struct {
	NodeKey  string                               `json:"node_key"`
	Results  fleet.OsqueryDistributedQueryResults `json:"queries"`
	Statuses map[string]fleet.OsqueryStatus       `json:"statuses"`
	Messages map[string]string                    `json:"messages,omitempty"`
	Stats    map[string]*fleet.Stats              `json:"stats,omitempty"`
}

func (r *DistributedWriteRequest) SetOrbitNodeKey(nodeKey string) { r.NodeKey = nodeKey }

func (r *DistributedWriteRequest) setRequestHeaders(req *http.Request) {
	setNodeKeyHeader(req, r.NodeKey)
}

type distributedWriteResponse struct{}

// setNodeKeyHeader sets the "Authorization: NodeKey <key>" header used by the
// server's osquery header pre-auth (enabled when allow_body_auth_fallback is
// false).
func setNodeKeyHeader(req *http.Request, nodeKey string) {
	req.Header.Set("Authorization", "NodeKey "+nodeKey)
}

// DistributedRead fetches the distributed queries the host should run
// (live queries, and label/policy/detail queries when due), authenticating
// with the orbit node key.
func (oc *OrbitClient) DistributedRead() (*DistributedReadResponse, error) {
	var resp DistributedReadResponse
	if err := oc.authenticatedRequest(http.MethodPost, "/api/osquery/distributed/read", &distributedReadRequest{}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DistributedWrite submits distributed query results, authenticating with the
// orbit node key. The node key field of req is set by this method.
func (oc *OrbitClient) DistributedWrite(req *DistributedWriteRequest) error {
	var resp distributedWriteResponse
	return oc.authenticatedRequest(http.MethodPost, "/api/osquery/distributed/write", req, &resp)
}
