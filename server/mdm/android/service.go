package android

import (
	"context"
	"net/http"
)

type Service interface {
	EnterpriseSignup(ctx context.Context) (*SignupDetails, error)
	EnterpriseSignupCallback(ctx context.Context, signupToken string, enterpriseToken string) error
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	DeleteEnterprise(ctx context.Context) error
	EnterpriseSignupSSE(ctx context.Context) (chan string, error)

	// CreateEnrollmentToken creates an enrollment token for a new Android device.
	CreateEnrollmentToken(ctx context.Context, enrollSecret, idpUUID string) (*EnrollmentToken, error)
	ProcessPubSubPush(ctx context.Context, token string, message *PubSubMessage) error

	// UnenrollAndroidHost triggers unenrollment (work profile removal) for the given Android host ID.
	UnenrollAndroidHost(ctx context.Context, hostID uint) error
}

// /////////////////////////////////////////////
// Android API request and response structs

type DefaultResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DefaultResponse) Error() error { return r.Err }

// StatusCode implements the go-kit http StatusCoder interface to preserve HTTP status codes from errors
func (r DefaultResponse) StatusCode() int {
	if r.Err != nil {
		// Check if the error has a custom status code (like errors created with .WithStatus())
		if sc, ok := r.Err.(interface{ StatusCode() int }); ok {
			return sc.StatusCode()
		}
	}
	// Default to 200 OK if no error or no custom status code
	return http.StatusOK
}

type GetEnterpriseResponse struct {
	EnterpriseID string `json:"android_enterprise_id"`
	DefaultResponse
}

type EnterpriseSignupResponse struct {
	Url string `json:"android_enterprise_signup_url"`
	DefaultResponse
}

type EnrollmentTokenResponse struct {
	*EnrollmentToken
	DefaultResponse
}
