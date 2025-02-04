package android

import "context"

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, enterpriseID uint, enterpriseToken string) error

	CreateOrUpdatePolicy(ctx context.Context, enterpriseID uint) error

	CreateEnrollmentToken(ctx context.Context, fleetEnterpriseID uint) (*EnrollmentToken, error)
}
