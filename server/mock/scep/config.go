// Automatically generated by mockimpl. DO NOT EDIT!

package mock

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var _ fleet.SCEPConfigService = (*SCEPConfigService)(nil)

type ValidateNDESSCEPAdminURLFunc func(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) error

type GetNDESSCEPChallengeFunc func(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) (string, error)

type ValidateSCEPURLFunc func(ctx context.Context, url string) error

type SCEPConfigService struct {
	ValidateNDESSCEPAdminURLFunc        ValidateNDESSCEPAdminURLFunc
	ValidateNDESSCEPAdminURLFuncInvoked bool

	GetNDESSCEPChallengeFunc        GetNDESSCEPChallengeFunc
	GetNDESSCEPChallengeFuncInvoked bool

	ValidateSCEPURLFunc        ValidateSCEPURLFunc
	ValidateSCEPURLFuncInvoked bool

	mu sync.Mutex
}

func (s *SCEPConfigService) ValidateNDESSCEPAdminURL(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) error {
	s.mu.Lock()
	s.ValidateNDESSCEPAdminURLFuncInvoked = true
	s.mu.Unlock()
	return s.ValidateNDESSCEPAdminURLFunc(ctx, proxy)
}

func (s *SCEPConfigService) GetNDESSCEPChallenge(ctx context.Context, proxy fleet.NDESSCEPProxyIntegration) (string, error) {
	s.mu.Lock()
	s.GetNDESSCEPChallengeFuncInvoked = true
	s.mu.Unlock()
	return s.GetNDESSCEPChallengeFunc(ctx, proxy)
}

func (s *SCEPConfigService) ValidateSCEPURL(ctx context.Context, url string) error {
	s.mu.Lock()
	s.ValidateSCEPURLFuncInvoked = true
	s.mu.Unlock()
	return s.ValidateSCEPURLFunc(ctx, url)
}
