package launcher

import (
	"bytes"
	"encoding/json"
	"strconv"

	pb "github.com/kolide/agent-api"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

var errNotImplmented = errors.New("not implemented")

// agentBinding implements ApiClient interface and maps gRPC domain functions to the application.
type agentBinding struct {
	service kolide.OsqueryService
}

func newAgentBinding(svc kolide.OsqueryService) pb.ApiServer {
	return &agentBinding{
		service: svc,
	}
}

type enrollmentError interface {
	NodeInvalid() bool
	Error() string
}

// Attempt to enroll a host with kolide/cloud
func (b *agentBinding) RequestEnrollment(ctx context.Context, req *pb.EnrollmentRequest) (*pb.EnrollmentResponse, error) {
	var resp pb.EnrollmentResponse
	nodeKey, err := b.service.EnrollAgent(newCtx(ctx), req.EnrollSecret, req.HostIdentifier)
	if err != nil {
		if errEnroll, ok := err.(enrollmentError); ok {
			resp.NodeInvalid = errEnroll.NodeInvalid()
			resp.ErrorCode = errEnroll.Error()
			return &resp, nil
		}
		return nil, err
	}
	resp.NodeKey = nodeKey
	return &resp, nil
}

// RequestConfig requests an updated configuration
func (b *agentBinding) RequestConfig(ctx context.Context, req *pb.AgentApiRequest) (*pb.ConfigResponse, error) {
	config, err := b.service.GetClientConfig(newCtx(ctx))
	if err != nil {
		return nil, err
	}
	// Launcher manages plugins so remove them from configuration if they exist.
	for _, optionName := range []string{"distributed_plugin", "logger_plugin"} {
		if _, ok := config.Options[optionName]; ok {
			delete(config.Options, optionName)
		}
	}
	var writer bytes.Buffer
	if err = json.NewEncoder(&writer).Encode(config); err != nil {
		return nil, err
	}
	return &pb.ConfigResponse{ConfigJsonBlob: writer.String()}, nil
}

// RequestQueries request/pull distributed queries
func (b *agentBinding) RequestQueries(ctx context.Context, _ *pb.AgentApiRequest) (*pb.QueryCollection, error) {
	queryMap, _, err := b.service.GetDistributedQueries(newCtx(ctx))
	if err != nil {
		return nil, err
	}
	var result pb.QueryCollection
	for id, query := range queryMap {
		result.Queries = append(result.Queries, &pb.QueryCollection_Query{Id: id, Query: query})
	}
	return &result, nil
}

// StatusLog handles osquery logging messages
type StatusLog struct {
	Severity string `json:"s"`
	Filename string `json:"f"`
	Line     string `json:"i"`
	Message  string `json:"m"`
}

// convert the json from grpc client to an object suitable
// for consumption by fleet
func toKolideLog(jsn string) (*kolide.OsqueryStatusLog, error) {
	var status StatusLog
	err := json.NewDecoder(bytes.NewBufferString(jsn)).Decode(&status)
	if err != nil {
		return nil, err
	}
	result := &kolide.OsqueryStatusLog{
		Severity: status.Severity,
		Filename: status.Filename,
		Line:     status.Line,
		Message:  status.Message,
	}
	return result, nil
}

// PublishLogs publish logs from osqueryd
func (b *agentBinding) PublishLogs(ctx context.Context, coll *pb.LogCollection) (*pb.AgentApiResponse, error) {
	handler := func(_ context.Context, _ *pb.LogCollection) error { return nil }
	switch coll.LogType {
	case pb.LogCollection_RESULT:
		handler = b.handleResultLogs
	case pb.LogCollection_STATUS:
		handler = b.handleStatusLogs
	}
	if err := handler(ctx, coll); err != nil {
		return nil, err
	}
	return &pb.AgentApiResponse{}, nil
}

func (b *agentBinding) handleResultLogs(ctx context.Context, coll *pb.LogCollection) error {
	var results []kolide.OsqueryResultLog
	for _, log := range coll.Logs {
		var result kolide.OsqueryResultLog
		if err := json.Unmarshal([]byte(log.Data), &result); err != nil {
			return errors.Wrap(err, "unmarshaling result log")
		}
		results = append(results, result)
	}
	if err := b.service.SubmitResultLogs(newCtx(ctx), results); err != nil {
		return errors.Wrap(err, "submitting status logs")
	}
	return nil
}

func (b *agentBinding) handleStatusLogs(ctx context.Context, coll *pb.LogCollection) error {
	var statuses []kolide.OsqueryStatusLog
	for _, record := range coll.Logs {
		status, err := toKolideLog(record.Data)
		if err != nil {
			return errors.Wrap(err, "decoding status log")
		}
		statuses = append(statuses, *status)
	}
	if err := b.service.SubmitStatusLogs(newCtx(ctx), statuses); err != nil {
		return errors.Wrap(err, "submitting status logs")
	}
	return nil
}

// PublishResults publish distributed query results
func (b *agentBinding) PublishResults(ctx context.Context, coll *pb.ResultCollection) (*pb.AgentApiResponse, error) {
	results := kolide.OsqueryDistributedQueryResults{}
	statuses := map[string]string{}
	for _, result := range coll.Results {
		statuses[result.Id] = strconv.Itoa(int(result.Status))
		rows := []map[string]string{}
		for _, row := range result.Rows {
			cols := map[string]string{}
			for _, colVal := range row.Columns {
				cols[colVal.Name] = colVal.Value
			}
			if len(cols) == 0 {
				continue
			}
			rows = append(rows, cols)
		}
		results[result.Id] = rows
	}
	if err := b.service.SubmitDistributedQueryResults(newCtx(ctx), results, statuses); err != nil {
		return nil, errors.Wrap(err, "submitting distributed query results")
	}
	return &pb.AgentApiResponse{}, nil
}

func (svc *agentBinding) CheckHealth(ctx context.Context, coll *pb.AgentApiRequest) (*pb.HealthCheckResponse, error) {
	return nil, errNotImplmented
}
