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

func StringValueOrZero(x *string) string {
	if x == nil {
		return ""
	}
	return *x
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

// Time returns a pointer to the provided time.Time.
func Time(x time.Time) *time.Time {
	return &x
}

// RawMessage returns a pointer to the provided json.RawMessage.
func RawMessage(x json.RawMessage) *json.RawMessage {
	return &x
}
