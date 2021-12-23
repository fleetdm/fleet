package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet/policytest"
)

func TestMemFailingPolicySet(t *testing.T) {
	m := NewMemFailingPolicySet()
	policytest.RunFailingPolicySetTests(t, m)
}
