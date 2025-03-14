package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
)

func (p *Proxy) InitCommonMocks() {
	p.EnterpriseDeleteFunc = func(ctx context.Context, enterpriseID string) error {
		return nil
	}
	p.SignupURLsCreateFunc = func(callbackURL string) (*android.SignupDetails, error) {
		return &android.SignupDetails{}, nil
	}
	p.EnterprisesCreateFunc = func(ctx context.Context, req android.ProxyEnterprisesCreateRequest) (string, string, error) {
		return "enterpriseName", "projects/project/topics/topic", nil
	}
	p.EnterprisesPoliciesPatchFunc = func(enterpriseID string, policyName string, policy *androidmanagement.Policy) error {
		return nil
	}
}
