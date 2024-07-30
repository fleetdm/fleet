// Package challenge defines an interface for a dynamic challenge password cache.
package challenge

import (
	"crypto/x509"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
)

// Store is a dynamic challenge password cache.
type Store interface {
	SCEPChallenge() (string, error)
	HasChallenge(pw string) (bool, error)
}

// Middleware wraps next in a CSRSigner that verifies and invalidates the challenge
func Middleware(store Store, next scepserver.CSRSigner) scepserver.CSRSignerFunc {
	return func(m *scep.CSRReqMessage) (*x509.Certificate, error) {
		// TODO: compare challenge only for PKCSReq?
		valid, err := store.HasChallenge(m.ChallengePassword)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, errors.New("invalid challenge")
		}
		return next.SignCSR(m)
	}
}
