// Package ptr includes functions for creating pointers from values.
package ptr

func String(x string) *string {
	return &x
}

func Int(x int) *int {
	return &x
}

func Uint(x uint) *uint {
	return &x
}
