package mock

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
)

func (p *Client) InitCommonMocks() {
	p.EnterpriseDeleteFunc = func(ctx context.Context, enterpriseID string) error {
		return nil
	}
	p.SignupURLsCreateFunc = func(serverURL, callbackURL string) (*android.SignupDetails, error) {
		return &android.SignupDetails{}, nil
	}
	p.EnterprisesCreateFunc = func(ctx context.Context, req androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName:    "enterprises/name",
			TopicName:         "",
			FleetServerSecret: "fleetServerSecret",
		}, nil
	}
	p.EnterprisesPoliciesPatchFunc = func(policyName string, policy *androidmanagement.Policy) error {
		return nil
	}
	p.SetAuthenticationSecretFunc = func(secret string) error { return nil }
}
