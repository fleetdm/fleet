package api

import "context"

// EnrollmentService stores records in the acme_enrollments table.
type EnrollmentService interface {
	// NewEnrollment creates a new enrollment in the acme_enrollments table with the specified
	// host_uuid and returns a new path_identifier for the created row.
	NewEnrollment(ctx context.Context, hostIdentifier string) (string, error)
}
