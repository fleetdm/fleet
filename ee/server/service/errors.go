package service

type notFoundError struct{}

func (e notFoundError) Error() string {
	return "not found"
}

// IsNotFound implements the service.IsNotFound interface (from the non-premium
// service package) so that the handler returns 404 for this error.
func (e notFoundError) IsNotFound() bool {
	return true
}
