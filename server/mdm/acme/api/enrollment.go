package api

import "context"

// EnrollmentService upserts records in the acme_enrollments table.
type EnrollmentService interface {
	// UpsertEnrollment upserts the acme_enrollments table with the specified
	// host_uuid and returns a new path_identifier for the upserted row.
	UpsertEnrollment(ctx context.Context, hostIdentifier string) (string, error)
}
