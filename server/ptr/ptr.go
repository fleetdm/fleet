// Package ptr includes functions for creating pointers from values.
package ptr

import (
	"encoding/json"
	"time"
)

// String returns a pointer to the provided string.
//
// Deprecated: Use new instead.
func String(x string) *string {
	val := new(string)
	*val = x
	return val
}

// Int returns a pointer to the provided int.
//
// Deprecated: Use new instead.
func Int(x int) *int {
	return &x
}

// Uint returns a pointer to the provided uint.
//
// Deprecated: Use new instead.
func Uint(x uint) *uint {
	return &x
}

// UintOrNilIfZero returns nil if the supplied value is zero, else a pointer to the provided uint.
// This is useful for cases that expect nil to be supplied for "No team" instead of zero, and allows for
// a quick way to sidestep e.g. https://github.com/fleetdm/fleet/issues/37729 (which ptr.Uint() would cause).
func UintOrNilIfZero(x uint) *uint {
	if x > 0 {
		return &x
	}
	return nil
}

// Bool returns a pointer to the provided bool.
//
// Deprecated: Use new instead.
func Bool(x bool) *bool {
	return &x
}

// BoolPtr returns a double pointer to the provided bool.
//
// Deprecated: Use new instead.
func BoolPtr(x bool) **bool {
	p := Bool(x)
	return &p
}

// StringPtr returns a double pointer to the provided string.
//
// Deprecated: Use new instead.
func StringPtr(x string) **string {
	p := String(x)
	return &p
}

// Time returns a pointer to the provided time.Time.
//
// Deprecated: Use new instead.
func Time(x time.Time) *time.Time {
	val := new(time.Time)
	*val = x
	return val
}

// TimePtr returns a *time.Time Pointer (**time.Time) for the provided time.
//
// Deprecated: Use new instead.
func TimePtr(x time.Time) **time.Time {
	t := Time(x)
	return &t
}

// RawMessage returns a pointer to the provided json.RawMessage.
//
// Deprecated: Use new instead.
func RawMessage(x json.RawMessage) *json.RawMessage {
	return &x
}

// Float64 returns a pointer to a float64.
//
// Deprecated: Use new instead.
func Float64(x float64) *float64 {
	return &x
}

// Float64Ptr returns a pointer to a *float64.
//
// Deprecated: Use new instead.
func Float64Ptr(x float64) **float64 {
	p := Float64(x)
	return &p
}

// Int64 returns a pointer to the provided int64.
//
// Deprecated: Use new instead.
func Int64(x int64) *int64 {
	return &x
}

// Duration returns a pointer to the provided time.Duration.
//
// Deprecated: Use new instead.
func Duration(x time.Duration) *time.Duration {
	return &x
}

// Equal returns true if both pointers are nil, or both are non-nil and
// point to equal values.
func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ValOrZero returns the value of x if x is not nil, and the zero value
// for T otherwise.
func ValOrZero[T any](x *T) T {
	var ret T

	if x != nil {
		return *x
	}

	return ret
}
