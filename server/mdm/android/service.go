package android

import (
	"context"
)

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, signupToken string, enterpriseToken string) error
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	DeleteEnterprise(ctx context.Context) error
	EnterpriseSignupSSE(ctx context.Context) (chan string, error)

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

type EnterpriseSignupResponse struct {
	Url string `json:"android_enterprise_signup_url"`
	DefaultResponse
}
