package android

import "context"

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, enterpriseID uint, enterpriseToken string) error
	DeleteEnterprise(ctx context.Context) error

	CreateEnrollmentToken(ctx context.Context) (*EnrollmentToken, error)
}
