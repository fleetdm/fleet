package launcher

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/go-kit/log"
	"github.com/kolide/launcher/pkg/service"
	"github.com/osquery/osquery-go/plugin/distributed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLauncherEnrollment(t *testing.T) {
	launcher, tls := newTestService(t)
	ctx := context.Background()

	nodeKey, invalid, err := launcher.RequestEnrollment(ctx, "secret", "identifier", service.EnrollmentDetails{})
	require.Nil(t, err)
	assert.True(t, tls.EnrollAgentFuncInvoked)
	assert.False(t, invalid)
	assert.Equal(t, "noop", nodeKey)
}

func TestLauncherRequestConfig(t *testing.T) {
	launcher, tls := newTestService(t)
	ctx := context.Background()

	config, invalid, err := launcher.RequestConfig(ctx, "noop")
	require.Nil(t, err)
	assert.True(t, tls.AuthenticateHostFuncInvoked)
	assert.False(t, invalid)
	assert.JSONEq(t, `{"options":{"key":"value"},"decorators":{"deco":"foobar"}}`, config)
}

func TestLauncherRequestQueries(t *testing.T) {
	launcher, tls := newTestService(t)
	ctx := context.Background()

	result, invalid, err := launcher.RequestQueries(ctx, "noop")
	require.Nil(t, err)
	assert.True(t, tls.AuthenticateHostFuncInvoked)
	assert.False(t, invalid)
	assert.Equal(t, map[string]string{"noop": `{"key": "value"}`}, result.Queries)
}

func TestLauncherPublishResults(t *testing.T) {
	launcher, tls := newTestService(t)
	ctx := context.Background()

	_, _, invalid, err := launcher.PublishResults(
		ctx,
		"noop",
		[]distributed.Result{},
	)
	require.Nil(t, err)
	assert.True(t, tls.AuthenticateHostFuncInvoked)
	assert.False(t, invalid)

	// test with result
	result := map[string]string{"key": "value"}
	tls.SubmitDistributedQueryResultsFunc = func(
		ctx context.Context,
		results fleet.OsqueryDistributedQueryResults,
		statuses map[string]fleet.OsqueryStatus,
		messages map[string]string,
		stats map[string]*fleet.Stats,
	) (err error) {
		assert.Equal(t, results["query"][0], result)
		return nil
	}

	_, _, invalid, err = launcher.PublishResults(
		ctx,
		"noop",
		[]distributed.Result{
			{
				QueryName: "query",
				Status:    1,
				Rows:      []map[string]string{result},
			},
		},
	)
	require.Nil(t, err)
	assert.False(t, invalid)
}

func newTestService(t *testing.T) (*launcherWrapper, *mock.TLSService) {
	tls := newTLSService(t)
	launcher := &launcherWrapper{
		tls:    tls,
		logger: log.NewNopLogger(),
		healthCheckers: map[string]health.Checker{
			"noop": health.Nop(),
		},
	}
	return launcher, tls
}

// NewTLS service returns a mock TLS service where all the methods have a noop implementation.
// To test additional behaviors, override the funcs on the TLSService struct.
func newTLSService(t *testing.T) *mock.TLSService {
	return &mock.TLSService{
		EnrollAgentFunc: func(
			ctx context.Context,
			enrollSecret string,
			hostIdentifier string,
			hostDetails map[string](map[string]string),
		) (nodeKey string, err error) {
			nodeKey = "noop"
			return
		},

		AuthenticateHostFunc: func(
			ctx context.Context,
			nodeKey string,
		) (host *fleet.Host, debug bool, err error) {
			return &fleet.Host{
				NodeKey: &nodeKey,
			}, false, nil
		},
		GetClientConfigFunc: func(
			ctx context.Context,
		) (config map[string]interface{}, err error) {
			return map[string]interface{}{
				"options": map[string]interface{}{
					"key": "value",
				},
				"decorators": map[string]interface{}{
					"deco": "foobar",
				},
			}, nil
		},

		GetDistributedQueriesFunc: func(
			ctx context.Context,
		) (queries map[string]string, discovery map[string]string, accelerate uint, err error) {
			queries = map[string]string{
				"noop": `{"key": "value"}`,
			}
			discovery = map[string]string{
				"noop": `select 1`,
			}
			return
		},
		SubmitDistributedQueryResultsFunc: func(
			ctx context.Context,
			results fleet.OsqueryDistributedQueryResults,
			statuses map[string]fleet.OsqueryStatus,
			messages map[string]string,
			stats map[string]*fleet.Stats,
		) (err error) {
			return
		},

		SubmitStatusLogsFunc: func(ctx context.Context, logs []json.RawMessage) (err error) {
			return
		},
		SubmitResultLogsFunc: func(ctx context.Context, logs []json.RawMessage) (err error) {
			return
		},
	}
}
