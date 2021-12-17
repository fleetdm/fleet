package service

import "testing"

func TestMemFailingPolicySet(t *testing.T) {
	m := NewMemFailingPolicySet()
	RunFailingPolicySetTests(t, m)
}
