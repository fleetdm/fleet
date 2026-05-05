// Package http provides HTTP types for bounded contexts.
package http

// Errorer is implemented by response types that may contain errors.
type Errorer interface {
	Error() error
}
