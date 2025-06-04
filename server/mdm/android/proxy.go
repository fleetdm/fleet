package android

import (
	"context"

	"google.golang.org/api/androidmanagement/v1"
)

type Proxy interface {
	SignupURLsCreate(callbackURL string) (*SignupDetails, error)
	EnterprisesCreate(ctx context.Context, req ProxyEnterprisesCreateRequest) (string, string, error)
	EnterprisesPoliciesPatch(enterpriseID string, policyName string, policy *androidmanagement.Policy) error
	EnterprisesEnrollmentTokensCreate(enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error)
	EnterpriseDelete(ctx context.Context, enterpriseID string) error
}

type ProxyEnterprisesCreateRequest struct {
	androidmanagement.Enterprise
	EnterpriseToken string
	SignupUrlName   string
	PubSubPushURL   string
}
