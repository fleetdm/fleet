package fleet

import "context"

// ACMEWriteService is the subset of the ACME service modiule service
// used by the legacy service layer for write operations.
type ACMEWriteService interface {
	// UpsertEnrollment upserts the acme_enrollments table with the specified
	// host_uuid and returns a new path_identifier for the upserted row.
	UpsertEnrollment(ctx context.Context, hostIdentifier string) (string, error)
}
