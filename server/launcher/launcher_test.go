package launcher

import (
	"context"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/kolide/fleet/server/health"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/mock"
	"github.com/kolide/osquery-go/plugin/distributed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLauncherEnrollment(t *testing.T) {
	launcher, tls := newTestService(t)
	ctx := context.Background()

	nodeKey, invalid, err := launcher.RequestEnrollment(ctx, "secret", "identifier")
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
	assert.Equal(t, `{"options":{"key":"value"},"decorators":{}}`, config)
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
	var result = map[string]string{"key": "value"}
	tls.SubmitDistributedQueryResultsFunc = func(
		ctx context.Context,
		results kolide.OsqueryDistributedQueryResults,
		statuses map[string]string) (err error) {
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
		) (nodeKey string, err error) {
			nodeKey = "noop"
			return

		},

		AuthenticateHostFunc: func(
			ctx context.Context,
			nodeKey string,
		) (host *kolide.Host, err error) {
			return &kolide.Host{
				NodeKey: nodeKey,
			}, nil
		},
		GetClientConfigFunc: func(
			ctx context.Context,
		) (config *kolide.OsqueryConfig, err error) {
			return &kolide.OsqueryConfig{
				Options: map[string]interface{}{
					"key": "value",
				},
			}, nil
		},

		GetDistributedQueriesFunc: func(
			ctx context.Context,
		) (queries map[string]string, accelerate uint, err error) {
			queries = map[string]string{
				"noop": `{"key": "value"}`,
			}
			return
		},
		SubmitDistributedQueryResultsFunc: func(
			ctx context.Context,
			results kolide.OsqueryDistributedQueryResults,
			statuses map[string]string,
		) (err error) {
			return
		},

		SubmitStatusLogsFunc: func(ctx context.Context, logs []kolide.OsqueryStatusLog) (err error) {
			return
		},
		SubmitResultLogsFunc: func(ctx context.Context, logs []kolide.OsqueryResultLog) (err error) {
			return
		},
	}
}
