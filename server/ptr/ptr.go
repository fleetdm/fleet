// Package ptr includes functions for creating pointers from values.
package ptr

import (
	"encoding/json"
	"time"
)

// String returns a pointer to the provided string.
func String(x string) *string {
	return &x
}

// Int returns a pointer to the provided int.
func Int(x int) *int {
	return &x
}

// Uint returns a pointer to the provided uint.
func Uint(x uint) *uint {
	return &x
}

// Bool returns a pointer to the provided bool.
func Bool(x bool) *bool {
	return &x
}

// BoolPtr returns a double pointer to the provided bool.
func BoolPtr(x bool) **bool {
	p := Bool(x)
	return &p
}

func StringPtr(x string) **string {
	p := String(x)
	return &p
}

// Time returns a pointer to the provided time.Time.
func Time(x time.Time) *time.Time {
	return &x
}

// TimePtr returns a *time.Time Pointer (**time.Time) for the provided time.
func TimePtr(x time.Time) **time.Time {
	t := Time(x)
	return &t
}

// RawMessage returns a pointer to the provided json.RawMessage.
func RawMessage(x json.RawMessage) *json.RawMessage {
	return &x
}

// Float64 returns a pointer to a float64.
func Float64(x float64) *float64 {
	return &x
}

// Float64Ptr returns a pointer to a *float64.
func Float64Ptr(x float64) **float64 {
	p := Float64(x)
	return &p
}

func Int64(x int64) *int64 {
	return &x
}

func Duration(x time.Duration) *time.Duration {
	return &x
}
