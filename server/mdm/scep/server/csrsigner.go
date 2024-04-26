package scepserver

import (
	"crypto/subtle"
	"crypto/x509"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
)

// CSRSigner is a handler for CSR signing by the CA/RA
//
// SignCSR should take the CSR in the CSRReqMessage and return a
// Certificate signed by the CA.
type CSRSigner interface {
	SignCSR(*scep.CSRReqMessage) (*x509.Certificate, error)
}

// CSRSignerFunc is an adapter for CSR signing by the CA/RA
type CSRSignerFunc func(*scep.CSRReqMessage) (*x509.Certificate, error)

// SignCSR calls f(m)
func (f CSRSignerFunc) SignCSR(m *scep.CSRReqMessage) (*x509.Certificate, error) {
	return f(m)
}

// NopCSRSigner does nothing
func NopCSRSigner() CSRSignerFunc {
	return func(m *scep.CSRReqMessage) (*x509.Certificate, error) {
		return nil, nil
	}
}

// ChallengeMiddleware wraps next in a CSRSigner that validates the challenge from the CSR
func ChallengeMiddleware(challenge string, next CSRSigner) CSRSignerFunc {
	challengeBytes := []byte(challenge)
	return func(m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// TODO: compare challenge only for PKCSReq?
		if subtle.ConstantTimeCompare(challengeBytes, []byte(m.ChallengePassword)) != 1 {
			return nil, errors.New("invalid challenge")
		}
		return next.SignCSR(m)
	}
}
