package test

import "testing"

type quiet struct {
	*testing.T
}

// Quiet returns a wrapper around testing.T that silences Logf calls
func Quiet(t *testing.T) *quiet {
	return &quiet{t}
}

func (q *quiet) Logf(format string, args ...interface{}) {
	// No logging
	return
}
