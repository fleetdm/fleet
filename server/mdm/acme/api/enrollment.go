package api

import "context"

// EnrollmentService stores records in the acme_enrollments table.
type EnrollmentService interface {
	// NewACMEEnrollment creates a new enrollment in the acme_enrollments table with the specified
	// host_uuid and returns a new path_identifier for the created row.
	NewACMEEnrollment(ctx context.Context, hostIdentifier string) (string, error)
}
