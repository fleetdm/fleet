package service

// This file exports internal functions and methods only for testing purposes.
// Those are used by mdm_external_test.go which runs the tests as an external
// package to avoid import cycles, and as such needs to be able to call these
// unexported symbols.

type NotFoundError = notFoundError

var (
	TestCert = testCert
	TestKey  = testKey
)
