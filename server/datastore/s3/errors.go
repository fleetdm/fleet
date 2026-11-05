package s3

import (
	"errors"
	"net/http"

	"github.com/aws/smithy-go"
	"github.com/fleetdm/fleet/v4/server/datastore/cloudstorage"
)

// misconfiguredCodes are the S3 API error codes that mean the store is
// misconfigured (credentials, permissions, bucket, region or endpoint).
// BadRequest and MovedPermanently are what the SDK reports for body-less HEAD
// responses (e.g. Exists) with those statuses, which is how S3 answers a
// request for a bucket in another region or with invalid credentials.
var misconfiguredCodes = map[string]struct{}{
	"AccessDenied":                 {},
	"AccountProblem":               {},
	"AllAccessDisabled":            {},
	"AuthorizationHeaderMalformed": {},
	"BadRequest":                   {},
	"ExpiredToken":                 {},
	"Forbidden":                    {},
	"InvalidAccessKeyId":           {},
	"InvalidBucketName":            {},
	"InvalidSecurity":              {},
	"InvalidToken":                 {},
	"MovedPermanently":             {},
	"NoSuchBucket":                 {},
	"PermanentRedirect":            {},
	"SignatureDoesNotMatch":        {},
}

// wrapAPIError returns a *cloudstorage.MisconfiguredError if err is an S3 API
// error caused by the store's configuration, otherwise it returns err
// unchanged.
func (s *s3store) wrapAPIError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	var status int
	var statusErr interface{ HTTPStatusCode() int }
	if errors.As(err, &statusErr) {
		status = statusErr.HTTPStatusCode()
	}

	_, knownCode := misconfiguredCodes[apiErr.ErrorCode()]
	if !knownCode && status != http.StatusUnauthorized && status != http.StatusForbidden {
		return err
	}

	// Responses without a body (e.g. HEAD requests) get the SDK's placeholder
	// message, so fall back to the status text.
	msg := apiErr.ErrorMessage()
	if msg == "" || msg == "UnknownError" {
		msg = http.StatusText(status)
	}
	return cloudstorage.NewMisconfigured(err, "S3", apiErr.ErrorCode(), msg, "bucket: "+s.bucket)
}
