package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet/policytest"
)

func TestMemFailingPolicySet(t *testing.T) {
	m := NewMemFailingPolicySet()
	policytest.RunFailing1000hosts(t, m)
	m = NewMemFailingPolicySet()
	policytest.RunFailingBasic(t, m)
}
