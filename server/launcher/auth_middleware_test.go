package launcher

import (
	"context"
	"testing"

	pb "github.com/kolide/agent-api"
	"github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var authTestHost = &kolide.Host{HostName: "jimmy"}
var nullHost *kolide.Host

func TestAuthRequestConfig(t *testing.T) {
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"AuthenticateHost",
		oldContext,
		"nodekey",
	).Return(
		authTestHost,
		nil,
	)
	mockSvc.On(
		"GetClientConfig",
		mock.MatchedBy(func(ctx context.Context) bool {
			if h, ok := host.FromContext(ctx); ok {
				return h.HostName == authTestHost.HostName
			}
			return false
		}),
	).Return(
		&kolide.OsqueryConfig{},
		nil,
	)
	svr := newAuthMiddleware(mockSvc)(&agentBinding{mockSvc})
	resp, err := svr.RequestConfig(oldContext, &pb.AgentApiRequest{NodeKey: "nodekey"})
	mockSvc.AssertExpectations(t)
	require.Nil(t, err)
	require.NotNil(t, resp)
}

func TestAuthFailRequestConfig(t *testing.T) {
	cxtMatcher := mock.MatchedBy(func(ctx context.Context) bool {
		if h, ok := hostFromContext(ctx); ok {
			return h.HostName == authTestHost.HostName
		}
		return false
	})
	mockSvc := new(mockOsqueryService)
	mockSvc.On(
		"AuthenticateHost",
		oldContext,
		"nodekey",
	).Return(
		nullHost,
		&mockEnrollError{},
	)
	mockSvc.On(
		"GetClientConfig",
		cxtMatcher,
	).Return(
		&kolide.OsqueryConfig{},
		nil,
	)
	svr := newAuthMiddleware(mockSvc)(&agentBinding{mockSvc})
	resp, err := svr.RequestConfig(oldContext, &pb.AgentApiRequest{NodeKey: "nodekey"})
	mockSvc.AssertNotCalled(t, "GetClientConfig", newContext)
	require.Nil(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.NodeInvalid)
}
