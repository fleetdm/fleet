package android

import "context"

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, enterpriseID uint, enterpriseToken string) error
	DeleteEnterprise(ctx context.Context) error

	// CreateEnrollmentToken creates an enrollment token for a new Android device.
	CreateEnrollmentToken(ctx context.Context, enrollSecret string) (*EnrollmentToken, error)
}
