package android

// ConflictError is returned when there is a conflict with the current resource state,
// such as trying to create a resource that already exists.
// It implements the conflictErrorInterface from transport_error.go
type ConflictError struct {
	error
}

func (e *ConflictError) IsConflict() bool {
	return true
}

// IsClientError ensures that this error will be logged with info/debug level (and not error level) in the server logs.
func (e *ConflictError) IsClientError() bool {
	return true
}

func NewConflictError(err error) *ConflictError {
	return &ConflictError{err}
}
