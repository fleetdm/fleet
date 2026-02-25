package fleet

import (
	"context"
	"encoding/json"
	"net/http"
)

type GetClientConfigRequest struct {
	NodeKey string `json:"node_key"`
}

func (r *GetClientConfigRequest) hostNodeKey() string {
	return r.NodeKey
}

type GetClientConfigResponse struct {
	Config map[string]interface{}
	Err    error `json:"error,omitempty"`
}

func (r GetClientConfigResponse) Error() error { return r.Err }

func (r GetClientConfigResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Config)
}

func (r *GetClientConfigResponse) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Config)
}

type GetDistributedQueriesRequest struct {
	NodeKey string `json:"node_key"`
}

func (r *GetDistributedQueriesRequest) hostNodeKey() string {
	return r.NodeKey
}

type GetDistributedQueriesResponse struct {
	Queries    map[string]string `json:"queries"`
	Discovery  map[string]string `json:"discovery"`
	Accelerate uint              `json:"accelerate,omitempty"`
	Err        error             `json:"error,omitempty"`
}

func (r GetDistributedQueriesResponse) Error() error { return r.Err }

type SubmitDistributedQueryResultsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SubmitDistributedQueryResultsResponse) Error() error { return r.Err }

type SubmitLogsRequest struct {
	NodeKey string            `json:"node_key"`
	LogType string            `json:"log_type"`
	Data    []json.RawMessage `json:"data"`
}

func (r *SubmitLogsRequest) hostNodeKey() string {
	return r.NodeKey
}

type SubmitLogsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SubmitLogsResponse) Error() error { return r.Err }

type GetYaraRequest struct {
	NodeKey string `json:"node_key"`
	Name    string `url:"name"`
}

func (r *GetYaraRequest) hostNodeKey() string {
	return r.NodeKey
}

type GetYaraResponse struct {
	Err     error `json:"error,omitempty"`
	Content string
}

func (r GetYaraResponse) Error() error { return r.Err }

func (r GetYaraResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(r.Content))
}
