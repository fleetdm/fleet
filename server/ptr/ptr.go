// Package ptr includes functions for creating pointers from values.
package ptr

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
