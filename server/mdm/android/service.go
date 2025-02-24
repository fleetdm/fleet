package android

import (
	"context"
)

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, enterpriseID uint, enterpriseToken string) error
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	DeleteEnterprise(ctx context.Context) error

	// CreateEnrollmentToken creates an enrollment token for a new Android device.
	CreateEnrollmentToken(ctx context.Context, enrollSecret string) (*EnrollmentToken, error)
	ProcessPubSubPush(ctx context.Context, token string, message *PubSubMessage) error
}

// /////////////////////////////////////////////
// Android API request and response structs

type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DefaultResponse) Error() error { return r.Err }

type GetEnterpriseResponse struct {
	EnterpriseID string `json:"android_enterprise_id"`
	DefaultResponse
}
