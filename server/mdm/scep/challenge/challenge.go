// Package challenge defines an interface for a dynamic challenge password cache.
package challenge

import (
	"context"
	"crypto/x509"
	"errors"

	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/smallstep/scep"
)

// Validator validates challenge passwords.
type Validator interface {
	// HasChallenge validates pw as valid.
	HasChallenge(pw string) (bool, error)
}

// Store is a dynamic challenge password cache.
type Store interface {
	// SCEPChallenge generates a new challenge password.
	SCEPChallenge() (string, error)
	Validator
}

// Middleware wraps next in a CSRSigner that verifies and invalidates the challenge.
func Middleware(store Validator, next scepserver.CSRSignerContext) scepserver.CSRSignerContextFunc {
	return func(ctx context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// TODO: compare challenge only for PKCSReq?
		valid, err := store.HasChallenge(m.ChallengePassword)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, errors.New("invalid challenge")
		}
		return next.SignCSRContext(ctx, m)
	}
}
