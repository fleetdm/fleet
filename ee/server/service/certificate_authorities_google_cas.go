package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/ee/server/service/googlecas"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

const (
	// googleCASMaxFieldLength is the size of the varchar columns that hold the Google CAS project, location and CA pool.
	googleCASMaxFieldLength = 255
	// googleCASMaxKeySize bounds the service account key, which is a few KB, well below what the encrypted BLOB column holds.
	googleCASMaxKeySize = 16 << 10
)

// The project, location and pool become path segments of Google API URLs, which the client doesn't escape, so they are limited to the
// characters Google allows in those IDs. The project also allows digits first (project numbers) and ".:" (domain-scoped projects).
var (
	googleCASProjectRegexp  = regexp.MustCompile(`^[a-z0-9][a-z0-9.:-]*$`)
	googleCASLocationRegexp = regexp.MustCompile(`^[a-z0-9-]+$`)
	googleCASCAPoolRegexp   = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// validateGoogleCAS checks a new Google CAS CA and verifies that the key can read the CA pool.
func (svc *Service) validateGoogleCAS(ctx context.Context, ca *fleet.GoogleCASCA, errPrefix string) error {
	if err := validateGoogleCASLocally(ca, errPrefix); err != nil {
		return err
	}
	return svc.verifyGoogleCAS(ctx, *ca, errPrefix)
}

// validateGoogleCASLocally runs every check that doesn't need Google.
func validateGoogleCASLocally(ca *fleet.GoogleCASCA, errPrefix string) error {
	if err := validateCAName(ca.Name, errPrefix); err != nil {
		return err
	}
	if err := validateGoogleCASFields(ca, errPrefix); err != nil {
		return err
	}
	return validateGoogleCASAPIKeyJSON(string(ca.APIKeyJSON), errPrefix)
}

// validateGoogleCASFields checks the fields that identify the CA pool, whether the CA is new or the result of an update.
func validateGoogleCASFields(ca *fleet.GoogleCASCA, errPrefix string) error {
	for _, field := range []struct {
		name, label, value, allowed string
		re                          *regexp.Regexp
	}{
		{"project", "Google Cloud project ID", ca.Project, "lowercase letters, numbers, hyphens, periods and colons", googleCASProjectRegexp},
		{"location", "location", ca.Location, "lowercase letters, numbers and hyphens", googleCASLocationRegexp},
		{"ca_pool", "CA pool", ca.CAPool, "letters, numbers, hyphens and underscores", googleCASCAPoolRegexp},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fleet.NewInvalidArgumentError(field.name, fmt.Sprintf("%sThe %s cannot be empty.", errPrefix, field.label))
		}
		if utf8.RuneCountInString(field.value) > googleCASMaxFieldLength {
			return fleet.NewInvalidArgumentError(field.name,
				fmt.Sprintf("%sThe %s cannot be longer than %d characters.", errPrefix, field.label, googleCASMaxFieldLength))
		}
		if !field.re.MatchString(field.value) {
			return fleet.NewInvalidArgumentError(field.name,
				fmt.Sprintf("%sThe %s can only contain %s.", errPrefix, field.label, field.allowed))
		}
	}
	return nil
}

func validateGoogleCASAPIKeyJSON(keyJSON, errPrefix string) error {
	if strings.TrimSpace(keyJSON) == "" {
		return fleet.NewInvalidArgumentError("api_key_json", fmt.Sprintf("%sThe service account key cannot be empty.", errPrefix))
	}
	if keyJSON == fleet.MaskedPassword {
		return fleet.NewInvalidArgumentError("api_key_json", fmt.Sprintf("%sThe service account key must be provided in full, not masked.", errPrefix))
	}
	if len(keyJSON) > googleCASMaxKeySize {
		return fleet.NewInvalidArgumentError("api_key_json", fmt.Sprintf("%sThe service account key is too large.", errPrefix))
	}
	if err := googlecas.ValidateServiceAccountKey(keyJSON); err != nil {
		return fleet.NewInvalidArgumentError("api_key_json", errPrefix+err.Error())
	}
	return nil
}

// verifyGoogleCAS checks that the service account key can read the CA pool. The client logs the underlying failure.
func (svc *Service) verifyGoogleCAS(ctx context.Context, ca fleet.GoogleCASCA, errPrefix string) error {
	if err := svc.googleCASService.VerifyCAPool(ctx, ca); err != nil {
		return &fleet.BadRequestError{Message: errPrefix + ctxerr.Cause(err).Error()}
	}
	return nil
}

// googleCASToCertificateAuthority maps the Google CAS settings onto the flat CA record. The url column is NOT NULL and Google CAS has no
// URL, so it is stored empty.
func googleCASToCertificateAuthority(ca fleet.GoogleCASCA) *fleet.CertificateAuthority {
	return &fleet.CertificateAuthority{
		Type:                string(fleet.CATypeGoogleCAS),
		Name:                &ca.Name,
		URL:                 new(""),
		GoogleCASProject:    &ca.Project,
		GoogleCASLocation:   &ca.Location,
		GoogleCASCAPool:     &ca.CAPool,
		GoogleCASAPIKeyJSON: new(string(ca.APIKeyJSON)),
	}
}

// updateGoogleCAS validates an update against the stored CA and returns the changes to store, or nil when nothing changes. Fields the
// update doesn't set keep their stored values, and the result is validated as a whole. The CA pool is verified again only when a
// field Google sees changes.
func (svc *Service) updateGoogleCAS(
	ctx context.Context, u *fleet.GoogleCASCAUpdatePayload, oldCA *fleet.CertificateAuthority, errPrefix string,
) (*fleet.CertificateAuthority, error) {
	u.Preprocess()
	stored := fleet.GoogleCASCA{
		Name:       ptr.ValOrZero(oldCA.Name),
		Project:    ptr.ValOrZero(oldCA.GoogleCASProject),
		Location:   ptr.ValOrZero(oldCA.GoogleCASLocation),
		CAPool:     ptr.ValOrZero(oldCA.GoogleCASCAPool),
		APIKeyJSON: fleet.GoogleCASKeyJSON(ptr.ValOrZero(oldCA.GoogleCASAPIKeyJSON)),
	}
	merged := stored
	for _, o := range []struct {
		field  *string
		update *string
	}{
		{&merged.Name, u.Name},
		{&merged.Project, u.Project},
		{&merged.Location, u.Location},
		{&merged.CAPool, u.CAPool},
	} {
		if o.update != nil {
			*o.field = *o.update
		}
	}
	// A masked value is what a read returns for the key, so it means "keep the stored one".
	if u.APIKeyJSON != nil && *u.APIKeyJSON != fleet.MaskedPassword {
		merged.APIKeyJSON = *u.APIKeyJSON
	}
	if merged.Equals(&stored) {
		return nil, nil
	}

	if merged.Name != stored.Name {
		if err := validateCAName(merged.Name, errPrefix); err != nil {
			return nil, err
		}
	}
	if err := validateGoogleCASFields(&merged, errPrefix); err != nil {
		return nil, err
	}
	keyChanged := !merged.APIKeyJSON.Equals(stored.APIKeyJSON)
	if keyChanged {
		if err := validateGoogleCASAPIKeyJSON(string(merged.APIKeyJSON), errPrefix); err != nil {
			return nil, err
		}
	}

	// The name only exists in Fleet, so a change to just the name doesn't need Google.
	if merged.Project != stored.Project || merged.Location != stored.Location || merged.CAPool != stored.CAPool || keyChanged {
		if err := svc.verifyGoogleCAS(ctx, merged, errPrefix); err != nil {
			return nil, err
		}
	}

	changes := &fleet.CertificateAuthority{Type: string(fleet.CATypeGoogleCAS)}
	if merged.Name != stored.Name {
		changes.Name = &merged.Name
	}
	if merged.Project != stored.Project {
		changes.GoogleCASProject = &merged.Project
	}
	if merged.Location != stored.Location {
		changes.GoogleCASLocation = &merged.Location
	}
	if merged.CAPool != stored.CAPool {
		changes.GoogleCASCAPool = &merged.CAPool
	}
	if keyChanged {
		changes.GoogleCASAPIKeyJSON = (*string)(&merged.APIKeyJSON)
	}
	return changes, nil
}

func (svc *Service) processGoogleCASCAs(ctx context.Context, batchOps *fleet.CertificateAuthoritiesBatchOperations, incomingCAs []fleet.GoogleCASCA, existingCAs []fleet.GoogleCASCA) error {
	incomingByName := make(map[string]*fleet.GoogleCASCA)
	for _, incoming := range incomingCAs {
		// Note: caller is responsible for ensuring incoming list has no duplicates
		incomingByName[caNameKey(incoming.Name)] = &incoming
	}

	existingByName := make(map[string]*fleet.GoogleCASCA)
	for _, existing := range existingCAs {
		// if existing CA isn't in the incoming list, we should delete it
		if _, ok := incomingByName[caNameKey(existing.Name)]; !ok {
			batchOps.Delete = append(batchOps.Delete, googleCASToCertificateAuthority(existing))
		}
		// Note: the datastore guarantees existing names are unique
		existingByName[caNameKey(existing.Name)] = &existing
	}

	const errPrefix = "certificate_authorities.google_cas: "
	// The slice, not the map, sets the order, so Google calls, the first error and the recorded activities are the same on every run.
	for _, incoming := range incomingCAs {
		if err := validateGoogleCASLocally(&incoming, errPrefix); err != nil {
			return err
		}
		existing, ok := existingByName[caNameKey(incoming.Name)]
		// An unchanged CA is left alone, so GitOps runs don't call Google or record an edited activity for it.
		if ok && existing.Equals(&incoming) {
			continue
		}
		if err := svc.verifyGoogleCAS(ctx, incoming, errPrefix); err != nil {
			return err
		}
		if ok {
			batchOps.Update = append(batchOps.Update, googleCASToCertificateAuthority(incoming))
		} else {
			batchOps.Add = append(batchOps.Add, googleCASToCertificateAuthority(incoming))
		}
	}

	return nil
}
