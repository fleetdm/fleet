package test

import (
	"reflect"
	"runtime"
	"strings"
)

// FunctionName returns the name of the function provided as the argument.
// Behavior is undefined if a non-function is passed.
func FunctionName(f interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}

// MakeTestChecksum creates a 16-byte checksum for testing purposes.
// It returns a byte slice with 15 zeros followed by the provided value as the last byte.
// This is commonly used in MDM profile tests to generate unique checksums.
func MakeTestChecksum(value byte) []byte {
	return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, value}
}

// MakeTestBytes creates a 16-byte array with sequential values from 1 to 16.
// This is commonly used in tests for checksums, tokens, and other 16-byte identifiers.
func MakeTestBytes() []byte {
	return []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
}
