package fleet

import "context"

// ValidateEmbeddedSecretsAndCustomHostVitals validates the database-backed
// variables a script or profile can embed: $FLEET_SECRET_* secrets and
// $FLEET_HOST_VITAL_<id> custom host vitals. Callers run it on upload so a
// document referencing a non-existent secret or vital is rejected up front. It
// lives in the fleet package (rather than a secrets- or vitals-specific file)
// because it spans both domains.
func ValidateEmbeddedSecretsAndCustomHostVitals(ctx context.Context, ds Datastore, documents []string) error {
	if err := ds.ValidateEmbeddedSecrets(ctx, documents); err != nil {
		return err
	}
	return ds.ValidateReferencedCustomHostVitals(ctx, documents)
}
