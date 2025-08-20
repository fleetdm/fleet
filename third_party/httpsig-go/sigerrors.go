package httpsig

import (
	"errors"
	"fmt"
)

// ErrCode enumerates the reasons a signing or verification can fail
type ErrCode string

const (
	// Error Codes

	// Errors related to not being able to extract a valid signature.
	ErrNoSigInvalidHeader     ErrCode = "nosig_invalid_header"
	ErrNoSigUnsupportedDigest ErrCode = "nosig_unsupported_digest"
	ErrNoSigWrongDigest       ErrCode = "nosig_wrong_digest"
	ErrNoSigMissingSignature  ErrCode = "nosig_missing_signature"
	ErrNoSigInvalidSignature  ErrCode = "nosig_invalid_signature"
	ErrNoSigMessageBody       ErrCode = "nosig_message_body" // Could not read message body

	// Errors related to an individual signature.
	ErrSigInvalidSignature     ErrCode = "sig_invalid_signature"        // The signature is unparseable or in the wrong format.
	ErrSigKeyFetch             ErrCode = "sig_key_fetch"                // Failed to the key for a signature
	ErrSigVerification         ErrCode = "sig_failed_algo_verification" // The signature did not verify according to the algorithm.
	ErrSigPublicKey            ErrCode = "sig_public_key"               // The public key for the signature is invalid or missing.
	ErrSigSecretKey            ErrCode = "sig_secret_key"               // The secret key for the signature is invalid or missing.
	ErrSigUnsupportedAlgorithm ErrCode = "sig_unsupported_algorithm"    // unsupported or invalid algorithm.
	ErrSigProfile              ErrCode = "sig_failed_profile"           // The signature was valid but failed the verify profile check

	// Signing
	ErrInvalidSignatureOptions ErrCode = "invalid_signature_options"
	ErrInvalidComponent        ErrCode = "invalid_component"
	ErrInvalidMetadata         ErrCode = "invalid_metadata"

	// Accept Signature
	ErrInvalidAcceptSignature ErrCode = "invalid_accept_signature"
	ErrMissingAcceptSignature ErrCode = "missing_accept_signature" // The Accept-Signature field was present but had an empty value.

	// General
	ErrInternal    ErrCode = "internal_error"
	ErrUnsupported ErrCode = "unsupported" // A particular feature of the spec is not supported
)

type SignatureError struct {
	Cause   error // may be nil
	Code    ErrCode
	Message string
}

func (se *SignatureError) Error() string {
	return se.Message
}

func (se *SignatureError) Unwrap() error {
	return se.Cause
}

func (se *SignatureError) GoString() string {
	cause := ""
	if se.Cause != nil {
		cause = fmt.Sprintf("Cause: %s\n", se.Cause)
	}
	return fmt.Sprintf("Code: %s\nMessage: %s\n%s", se.Code, se.Message, cause)
}

func newError(code ErrCode, msg string, cause ...error) *SignatureError {
	var rootErr error
	if len(cause) > 0 {
		rootErr = cause[0]
	}
	return &SignatureError{
		Cause:   rootErr,
		Code:    code,
		Message: msg,
	}
}

func errCode(err error) (ec ErrCode) {
	if err == nil {
		return ""
	}
	var se *SignatureError
	if errors.As(err, &se) {
		return se.Code
	}

	return ""
}
