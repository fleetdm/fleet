package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
)

// maxMicrosoftGraphCredentials caps how many Graph credentials may be configured. Fleet's data model, sync loop, and
// storage are all built for the multi-tenant case. We limit it now for initial rollout.
const maxMicrosoftGraphCredentials = 1

// ListMicrosoftGraphCredentials returns the stored credentials with their per-tenant sync status. Client secrets are
// never decrypted on this path.
func (svc *Service) ListMicrosoftGraphCredentials(ctx context.Context) ([]*fleet.MicrosoftGraphCredentialMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MicrosoftGraphCredential{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	creds, err := svc.ds.ListMicrosoftGraphCredentialMetadata(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credential metadata")
	}
	return creds, nil
}

// ApplyMicrosoftGraphCredentials reconciles the stored credentials to match the supplied list. It is declarative: a
// tenant absent from the list is deleted. Every credential that is new or whose values changed is verified against Graph
// before anything is written, so a bad credential is rejected at write time instead of failing silently on the next
// sync. Unchanged credentials are skipped entirely: re-applying an identical GitOps config makes no network call,
// performs no write, and emits no activity.
func (svc *Service) ApplyMicrosoftGraphCredentials(ctx context.Context, incoming []fleet.MicrosoftGraphCredential, dryRun bool) error {
	if err := svc.authz.Authorize(ctx, &fleet.MicrosoftGraphCredential{}, fleet.ActionWrite); err != nil {
		return err
	}

	invalid := &fleet.InvalidArgumentError{}

	var err error
	var stored map[string]*fleet.MicrosoftGraphCredential
	if len(incoming) > 0 {
		// Read the stored credentials once.
		if stored, err = svc.storedMicrosoftGraphCredentialsByTenant(ctx); err != nil {
			return err
		}
	}

	resolved, err := svc.resolveMicrosoftGraphCredentials(incoming, invalid, stored)
	if err != nil {
		return err
	}
	if !invalid.HasErrors() {
		if err := svc.verifyMicrosoftGraphCredentials(ctx, resolved, invalid, stored); err != nil {
			return err
		}
	}
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}

	if dryRun {
		return nil
	}

	added, edited, deleted, err := svc.persistMicrosoftGraphCredentials(ctx, resolved, stored)
	if err != nil {
		return err
	}

	if len(added)+len(edited)+len(deleted) == 0 {
		return nil
	}

	// A credential that was just verified is healthy, and a deleted one can no longer be unhealthy, so the aggregate is
	// recomputed after a change.
	// Reads the credentials just persisted above, so it cannot be served by a replica.
	if err := svc.ds.UpdateMicrosoftGraphCredentialInvalidAggregate(ctxdb.RequirePrimary(ctx, true)); err != nil {
		return ctxerr.Wrap(ctx, err, "refresh microsoft graph credential invalid aggregate")
	}

	for _, act := range newMicrosoftGraphCredentialActivities(added, edited, deleted) {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
			return ctxerr.Wrap(ctx, err, "create microsoft graph credential activity")
		}
	}

	return nil
}

// resolveMicrosoftGraphCredentials validates the incoming credential list and returns it with every client secret
// resolved to a usable value. Omitting the secret for an already-stored credential means "keep the stored secret"; the
// standard masked placeholder is accepted as the same thing.
//
// Validation failures are accumulated on invalid rather than returned, so one bad entry does not mask another. The
// returned slice only contains entries that validated.
func (svc *Service) resolveMicrosoftGraphCredentials(
	incoming []fleet.MicrosoftGraphCredential,
	invalid *fleet.InvalidArgumentError,
	storedByTenant map[string]*fleet.MicrosoftGraphCredential,
) ([]fleet.MicrosoftGraphCredential, error) {
	if len(incoming) == 0 {
		return nil, nil
	}

	if len(incoming) > maxMicrosoftGraphCredentials {
		invalid.Append("microsoft_graph_credentials",
			fmt.Sprintf("Only %d Microsoft Graph credential can be configured.", maxMicrosoftGraphCredentials))
		return nil, nil
	}

	seen := make(map[string]struct{}, len(incoming))
	resolved := make([]fleet.MicrosoftGraphCredential, 0, len(incoming))

	for _, cred := range incoming {
		// Entra emits these lower-cased but admins paste them either way. Normalizing keeps the unique key on
		// tenant_id meaningful and makes the stored-credential lookup below reliable.
		cred.TenantID = strings.ToLower(strings.TrimSpace(cred.TenantID))
		cred.ClientID = strings.ToLower(strings.TrimSpace(cred.ClientID))
		cred.ClientSecret = strings.TrimSpace(cred.ClientSecret)

		if !fleet.IsValidEntraGUID(cred.TenantID) {
			invalid.Append("microsoft_graph_credentials.tenant_id", fmt.Sprintf("Invalid Entra tenant ID: %s", cred.TenantID))
			continue
		}
		if !fleet.IsValidEntraGUID(cred.ClientID) {
			invalid.Append("microsoft_graph_credentials.client_id", fmt.Sprintf("Invalid Entra client ID: %s", cred.ClientID))
			continue
		}
		if _, dup := seen[cred.TenantID]; dup {
			// Two credentials for one tenant would read an identical Autopilot list, because the registry is scoped to
			// the tenant and not to the application.
			invalid.Append("microsoft_graph_credentials.tenant_id",
				fmt.Sprintf("Duplicate Entra tenant ID: %s. Only one credential per tenant is supported.", cred.TenantID))
			continue
		}
		seen[cred.TenantID] = struct{}{}

		if cred.ClientSecret == "" || cred.ClientSecret == fleet.MaskedPassword {
			// The mask means "keep the secret for this credential", and a credential's identity is the app registration:
			// tenant plus client.
			existing, ok := storedByTenant[cred.TenantID]
			if !ok || existing.ClientSecret == "" || !strings.EqualFold(existing.ClientID, cred.ClientID) {
				invalid.Append("microsoft_graph_credentials.client_secret",
					"client_secret must be provided when adding a Microsoft Graph credential or changing its tenant or client ID")
				continue
			}
			cred.ClientSecret = existing.ClientSecret
		} else if svc.config.Server.PrivateKey == "" {
			invalid.Append("microsoft_graph_credentials",
				"Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
			continue
		}

		resolved = append(resolved, cred)
	}

	return resolved, nil
}

// verifyMicrosoftGraphCredentials mints a token and reads one page for every credential that is new or whose values
// changed, so a bad credential is rejected at write time instead of failing silently on the next sync. Unchanged
// credentials are skipped: re-applying an identical GitOps config should not make a network call.
func (svc *Service) verifyMicrosoftGraphCredentials(
	ctx context.Context,
	creds []fleet.MicrosoftGraphCredential,
	invalid *fleet.InvalidArgumentError,
	storedByTenant map[string]*fleet.MicrosoftGraphCredential,
) error {
	for _, cred := range creds {
		if existing, ok := storedByTenant[cred.TenantID]; ok && existing.Equal(cred) {
			continue
		}

		client, err := svc.msGraphClientFactory(&cred)
		if err != nil {
			invalid.Append("microsoft_graph_credentials", fmt.Sprintf("Couldn't use Microsoft Graph credential: %s", err))
			continue
		}
		if err := client.VerifyCredential(ctx); err != nil {
			invalid.Append("microsoft_graph_credentials", microsoftGraphVerifyMessage(err))
			continue
		}
	}

	return nil
}

// microsoftGraphVerifyMessage turns a save-time Graph failure into something an admin can act on. A failure that never
// reached Graph is described here rather than in msgraph, because only the caller knows what it was attempting.
func microsoftGraphVerifyMessage(err error) string {
	if msg := msgraph.UserFacingMessage(err); msg != "" {
		// Someone is waiting on this save, so retrying by hand is real advice.
		if graphErr, ok := msgraph.AsError(err); ok && graphErr.IsTransient() {
			return msg + " Please try again."
		}
		return msg
	}
	// Unwrapped for the same reason the sync path stores a classified message: the wrap chain is internal plumbing, and
	// the innermost error is the part an admin can act on (a DNS or TLS failure, say).
	return fmt.Sprintf("Couldn't connect to Microsoft Graph: %s", ctxerr.Cause(err))
}

// persistMicrosoftGraphCredentials reconciles stored credentials to match the supplied list, and reports which tenants
// were added, edited, or deleted so the caller can emit one activity apiece.
//
// storedByTenant distinguishes nil from empty, and the difference is load-bearing: nil means "the caller did not read
// the stored credentials", so this function reads them itself to work out what to delete. A non-nil empty map means
// "the caller read them and there are none". Passing an empty map when the caller has not actually read would silently
// skip every delete, so callers must pass nil rather than an empty map when they skipped the read.
func (svc *Service) persistMicrosoftGraphCredentials(
	ctx context.Context,
	creds []fleet.MicrosoftGraphCredential,
	storedByTenant map[string]*fleet.MicrosoftGraphCredential,
) (added, edited, deleted []string, err error) {
	storedTenants := make(map[string]struct{}, len(storedByTenant))
	if storedByTenant != nil {
		for tenantID := range storedByTenant {
			storedTenants[tenantID] = struct{}{}
		}
	} else {
		// The caller skipped the read (empty incoming list), so read what is stored to work out the deletes.
		storedMeta, err := svc.ds.ListMicrosoftGraphCredentialMetadata(ctx)
		if err != nil {
			return nil, nil, nil, ctxerr.Wrap(ctx, err, "list microsoft graph credential metadata")
		}
		for _, cred := range storedMeta {
			storedTenants[strings.ToLower(cred.TenantID)] = struct{}{}
		}
	}

	incomingByTenant := make(map[string]struct{}, len(creds))
	var toUpsert []*fleet.MicrosoftGraphCredential
	for _, cred := range creds {
		incomingByTenant[cred.TenantID] = struct{}{}

		existing, ok := storedByTenant[cred.TenantID]
		switch {
		case !ok:
			added = append(added, cred.TenantID)
		case existing.Equal(cred):
			// Nothing changed: skip the write so re-applying an identical GitOps config emits no activity.
			continue
		default:
			edited = append(edited, cred.TenantID)
		}

		toUpsert = append(toUpsert, &cred)
	}

	for tenantID := range storedTenants {
		if _, ok := incomingByTenant[tenantID]; ok {
			continue
		}
		deleted = append(deleted, tenantID)
	}

	if err := svc.ds.ReplaceMicrosoftGraphCredentials(ctx, toUpsert, deleted); err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "replace microsoft graph credentials")
	}

	return added, edited, deleted, nil
}

func (svc *Service) storedMicrosoftGraphCredentialsByTenant(ctx context.Context) (map[string]*fleet.MicrosoftGraphCredential, error) {
	stored, err := svc.ds.ListMicrosoftGraphCredentials(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list microsoft graph credentials")
	}
	byTenant := make(map[string]*fleet.MicrosoftGraphCredential, len(stored))
	for _, cred := range stored {
		byTenant[strings.ToLower(cred.TenantID)] = cred
	}
	return byTenant, nil
}

// newMicrosoftGraphCredentialActivities builds the activities for one reconciliation pass.
func newMicrosoftGraphCredentialActivities(added, edited, deleted []string) []fleet.ActivityDetails {
	acts := make([]fleet.ActivityDetails, 0, len(added)+len(edited)+len(deleted))
	for _, tenantID := range added {
		acts = append(acts, fleet.ActivityTypeAddedMicrosoftGraphCredential{TenantID: tenantID})
	}
	for _, tenantID := range edited {
		acts = append(acts, fleet.ActivityTypeEditedMicrosoftGraphCredential{TenantID: tenantID})
	}
	for _, tenantID := range deleted {
		acts = append(acts, fleet.ActivityTypeDeletedMicrosoftGraphCredential{TenantID: tenantID})
	}
	return acts
}
