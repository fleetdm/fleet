package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
)

func (p *Client) InitCommonMocks() {
	p.EnterpriseDeleteFunc = func(_ context.Context, enterpriseID string) error {
		return nil
	}
	p.SignupURLsCreateFunc = func(_ context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
		return &android.SignupDetails{}, nil
	}
	p.EnterprisesCreateFunc = func(ctx context.Context, req androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName:    "enterprises/name",
			TopicName:         "",
			FleetServerSecret: "fleetServerSecret",
		}, nil
	}
	p.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}
	p.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		// Default implementation returns a single enterprise with a standard name
		return []*androidmanagement.Enterprise{
			{
				Name: "enterprises/test-enterprise-id",
			},
		}, nil
	}
	p.SetAuthenticationSecretFunc = func(secret string) error { return nil }
}
