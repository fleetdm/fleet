package launcher

import (
	newctx "context"
	"errors"
	"testing"

	pb "github.com/kolide/agent-api"
	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

type mockEnrollError struct{}

func (ee *mockEnrollError) NodeInvalid() bool { return true }
func (ee *mockEnrollError) Error() string     { return "enroll failed" }

type mockOsqueryService struct {
	mock.Mock
}

func (m *mockOsqueryService) EnrollAgent(ctx newctx.Context, enrollSecret, hostIdentifier string) (string, error) {
	args := m.Called(ctx, enrollSecret, hostIdentifier)
	return args.String(0), args.Error(1)
}

func (m *mockOsqueryService) AuthenticateHost(ctx newctx.Context, nodeKey string) (*kolide.Host, error) {
	args := m.Called(ctx, nodeKey)
	return args.Get(0).(*kolide.Host), args.Error(1)
}
func (m *mockOsqueryService) GetClientConfig(ctx newctx.Context) (*kolide.OsqueryConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).(*kolide.OsqueryConfig), args.Error(1)

}
func (m *mockOsqueryService) GetDistributedQueries(ctx newctx.Context) (map[string]string, uint, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]string), args.Get(1).(uint), args.Error(2)
}
func (m *mockOsqueryService) SubmitDistributedQueryResults(ctx newctx.Context, results kolide.OsqueryDistributedQueryResults, statuses map[string]string) error {
	args := m.Called(ctx, results, statuses)
	return args.Error(0)
}
func (m *mockOsqueryService) SubmitStatusLogs(ctx newctx.Context, logs []kolide.OsqueryStatusLog) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}
func (m *mockOsqueryService) SubmitResultLogs(ctx newctx.Context, logs []kolide.OsqueryResultLog) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

var oldContext = context.Background()
var newContext = newctx.Background()
var errMockEnrollError = &mockEnrollError{}
var errTestError = errors.New("test error")

func TestRequestEnrollementHappyPath(t *testing.T) {
	request := &pb.EnrollmentRequest{
		EnrollSecret:   "supersecret",
		HostIdentifier: "somehost",
	}

	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"EnrollAgent",
		newctx.Background(),
		request.EnrollSecret,
		request.HostIdentifier,
	).Return(
		"nodekey",
		nil,
	)
	agent := agentBinding{
		service: mockSvc,
	}

	resp, err := agent.RequestEnrollment(oldContext, request)
	mockSvc.AssertExpectations(t)
	assert.Nil(t, err)
	assert.Equal(t, "nodekey", resp.NodeKey)
	assert.False(t, resp.NodeInvalid)
}

func TestRequestEnrollmentFailed(t *testing.T) {
	request := &pb.EnrollmentRequest{
		EnrollSecret:   "supersecret",
		HostIdentifier: "somehost",
	}

	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"EnrollAgent",
		newctx.Background(),
		request.EnrollSecret,
		request.HostIdentifier,
	).Return(
		"",
		errMockEnrollError,
	)
	agent := agentBinding{
		service: mockSvc,
	}

	resp, err := agent.RequestEnrollment(oldContext, request)
	mockSvc.AssertExpectations(t)
	assert.Nil(t, err)
	assert.True(t, resp.NodeInvalid)
}

func TestRequestEnrollmentError(t *testing.T) {
	request := &pb.EnrollmentRequest{
		EnrollSecret:   "supersecret",
		HostIdentifier: "somehost",
	}

	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"EnrollAgent",
		newctx.Background(),
		request.EnrollSecret,
		request.HostIdentifier,
	).Return(
		"",
		errTestError,
	)
	agent := agentBinding{
		service: mockSvc,
	}

	_, err := agent.RequestEnrollment(oldContext, request)
	mockSvc.AssertExpectations(t)
	require.NotNil(t, err)
	assert.Equal(t, errTestError, err)
}

func TestRequestConfigHappyPath(t *testing.T) {
	request := &pb.AgentApiRequest{
		NodeKey: "nodekey",
	}

	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"GetClientConfig",
		newctx.Background(),
	).Return(
		&kolide.OsqueryConfig{
			Options: map[string]interface{}{
				"option1":            "optionval",
				"distributed_plugin": "tls",
			},
			Decorators: kolide.Decorators{
				Load: []string{
					"SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid",
				},
			},
		},
		nil,
	)
	agent := agentBinding{
		service: mockSvc,
	}

	resp, err := agent.RequestConfig(oldContext, request)
	mockSvc.AssertExpectations(t)
	assert.Nil(t, err)
	// verify distributed_plugin was removed
	expectedJSON := "{\"options\":{\"option1\":\"optionval\"},\"decorators\":{\"load\":[\"SELECT * FROM users u JOIN groups g WHERE u.gid = g.gid\"]}}\n"
	require.NotNil(t, resp)
	assert.Equal(t, expectedJSON, resp.ConfigJsonBlob)
	assert.False(t, resp.NodeInvalid)
}

func TestRequestConfigError(t *testing.T) {
	request := &pb.AgentApiRequest{
		NodeKey: "nodekey",
	}
	var nilConfig *kolide.OsqueryConfig

	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"GetClientConfig",
		newctx.Background(),
	).Return(
		nilConfig,
		errTestError,
	)
	agent := agentBinding{
		service: mockSvc,
	}

	resp, err := agent.RequestConfig(oldContext, request)
	mockSvc.AssertExpectations(t)
	require.NotNil(t, err)
	assert.Equal(t, errTestError, err)
	assert.Nil(t, resp)
}

func TestRequestQueriesHappyPath(t *testing.T) {
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"GetDistributedQueries",
		newctx.Background(),
	).Return(
		map[string]string{
			"query1": "select * from foo;",
		},
		uint(0),
		nil,
	)
	agent := agentBinding{mockSvc}
	qc, err := agent.RequestQueries(oldContext, nil)
	mockSvc.AssertExpectations(t)
	require.Nil(t, err)
	require.NotNil(t, qc)
	assert.Len(t, qc.Queries, 1)
}

func TestToKolideLog(t *testing.T) {
	jsn := "{\"s\":\"0\",\"f\":\"scheduler.cpp\",\"i\":\"73\",\"m\":\"Executing scheduled query pack\\/xxx\\/services: select name, port, protocol from etc_services;\",\"h\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"c\":\"Fri Aug 11 22:32:27 2017 UTC\",\"u\":\"1502490747\"}"
	sl, err := toKolideLog(jsn)
	require.Nil(t, err, "unexpected error")
	assert.Equal(t, "0", sl.Severity, "severity mismatch")
	assert.Equal(t, "scheduler.cpp", sl.Filename, "file name mismatch")
	assert.Equal(t, "73", sl.Line, "line number mismatch")
	malformedJSON := "{\"s\":\"0,\"f\":\"scheduler.cpp\",\"i\":\"73\",\"m\":\"Executing scheduled query pack\\/xxx\\/services: select name, port, protocol from etc_services;\",\"h\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"c\":\"Fri Aug 11 22:32:27 2017 UTC\",\"u\":\"1502490747\"}"
	sl, err = toKolideLog(malformedJSON)
	assert.NotNil(t, err, "malformed json should have erred")
	assert.Nil(t, sl, "result should be nil on err")
}

func TestPublishStatusLogs(t *testing.T) {
	statusJSON := "{\"s\":\"0\",\"f\":\"scheduler.cpp\",\"i\":\"73\",\"m\":\"Executing scheduled query pack\\/xxx\\/services: select name, port, protocol from etc_services;\",\"h\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"c\":\"Fri Aug 11 22:32:27 2017 UTC\",\"u\":\"1502490747\"}"
	statusLogCol := &pb.LogCollection{
		LogType: pb.LogCollection_STATUS,
		Logs: []*pb.LogCollection_Log{
			&pb.LogCollection_Log{
				Data: statusJSON,
			},
		},
	}
	statuses := []kolide.OsqueryStatusLog{
		kolide.OsqueryStatusLog{
			Severity: "0",
			Filename: "scheduler.cpp",
			Line:     "73",
			Message:  `Executing scheduled query pack/xxx/services: select name, port, protocol from etc_services;`,
		},
	}
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"SubmitStatusLogs",
		newContext,
		statuses,
	).Return(
		nil,
	)
	agent := agentBinding{mockSvc}
	resp, err := agent.PublishLogs(oldContext, statusLogCol)
	mockSvc.AssertExpectations(t)
	mockSvc.AssertCalled(t, "SubmitStatusLogs", newContext, statuses)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.NodeInvalid)
}

func TestPublishStatusLogsUnhandledLogType(t *testing.T) {
	statusJSON := "{\"s\":\"0\",\"f\":\"scheduler.cpp\",\"i\":\"73\",\"m\":\"Executing scheduled query pack\\/xxx\\/services: select name, port, protocol from etc_services;\",\"h\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"c\":\"Fri Aug 11 22:32:27 2017 UTC\",\"u\":\"1502490747\"}"
	statusLogCol := &pb.LogCollection{
		LogType: pb.LogCollection_AGENT,
		Logs: []*pb.LogCollection_Log{
			&pb.LogCollection_Log{
				Data: statusJSON,
			},
		},
	}
	statuses := []kolide.OsqueryStatusLog{
		kolide.OsqueryStatusLog{
			Severity: "0",
			Filename: "scheduler.cpp",
			Line:     "73",
			Message:  `Executing scheduled query pack/xxx/services: select name, port, protocol from etc_services;`,
		},
	}
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"SubmitStatusLogs",
		newContext,
		statuses,
	).Return(
		nil,
	)
	agent := agentBinding{mockSvc}
	resp, err := agent.PublishLogs(oldContext, statusLogCol)
	mockSvc.AssertNotCalled(t, "SubmitStatusLogs", newContext, statuses)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.NodeInvalid)
}

func TestPublishResultLogs(t *testing.T) {
	resultJSON := "{\"name\":\"pack\\/xxx\\/services\",\"hostIdentifier\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"calendarTime\":\"Fri Aug 11 22:16:45 2017 UTC\",\"unixTime\":\"1502489805\",\"decorations\":{\"host_uuid\":\"DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C\",\"hostname\":\"Johns-MacBook-Pro.local\"},\"columns\":{\"name\":\"ms-dotnetster\",\"port\":\"3126\",\"protocol\":\"udp\"},\"action\":\"added\"}"
	resultLogColl := &pb.LogCollection{
		LogType: pb.LogCollection_RESULT,
		Logs: []*pb.LogCollection_Log{
			&pb.LogCollection_Log{
				Data: resultJSON,
			},
		},
	}
	results := []kolide.OsqueryResultLog{
		kolide.OsqueryResultLog{
			Name:           "pack/xxx/services",
			HostIdentifier: "DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C",
			UnixTime:       "1502489805",
			CalendarTime:   "Fri Aug 11 22:16:45 2017 UTC",
			Columns: map[string]string{
				"name":     "ms-dotnetster",
				"port":     "3126",
				"protocol": "udp",
			},
			Action: "added",
			Decorations: map[string]string{
				"host_uuid": "DE56C776-2F5A-56DF-81C7-F64EE1BBEC8C",
				"hostname":  "Johns-MacBook-Pro.local",
			},
		},
	}
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"SubmitResultLogs",
		newContext,
		results,
	).Return(
		nil,
	)
	agent := agentBinding{mockSvc}
	resp, err := agent.PublishLogs(oldContext, resultLogColl)
	mockSvc.AssertExpectations(t)
	require.Nil(t, err)
	require.NotNil(t, resp)
}

func TestPublishResults(t *testing.T) {
	coll := &pb.ResultCollection{
		NodeKey: "somekey",
		Results: []*pb.ResultCollection_Result{
			&pb.ResultCollection_Result{
				Id:     "myquery",
				Status: 0,
				Rows: []*pb.ResultCollection_Result_ResultRow{
					&pb.ResultCollection_Result_ResultRow{
						Columns: []*pb.ResultCollection_Result_ResultRow_Column{
							&pb.ResultCollection_Result_ResultRow_Column{
								Name:  "aColumn",
								Value: "aValue",
							},
						},
					},
				},
			},
		},
	}
	results := kolide.OsqueryDistributedQueryResults{
		"myquery": []map[string]string{
			map[string]string{
				"aColumn": "aValue",
			},
		},
	}
	statuses := map[string]string{
		"myquery": "0",
	}
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"SubmitDistributedQueryResults",
		newContext,
		results,
		statuses,
	).Return(
		nil,
	)
	agent := agentBinding{mockSvc}
	resp, err := agent.PublishResults(oldContext, coll)
	mockSvc.AssertExpectations(t)
	require.Nil(t, err)
	require.NotNil(t, resp)
}
