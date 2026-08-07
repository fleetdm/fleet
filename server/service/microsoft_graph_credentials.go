package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
)

// NoopMicrosoftGraphClient accepts any credential without touching the network.
//
// Credential verification runs on every config write that carries a new or changed credential, so a test server built
// without an injected factory would reach the real login.microsoftonline.com and graph.microsoft.com. Test harnesses
// inject this instead.
type NoopMicrosoftGraphClient struct{}

func (NoopMicrosoftGraphClient) VerifyCredential(context.Context) error { return nil }

func (NoopMicrosoftGraphClient) ListWindowsAutopilotDevices(context.Context) ([]msgraph.WindowsAutopilotDevice, error) {
	return nil, nil
}

// NoopMicrosoftGraphClientFactory builds a NoopMicrosoftGraphClient for any credential.
func NoopMicrosoftGraphClientFactory(*fleet.MicrosoftGraphCredential) (msgraph.Client, error) {
	return NoopMicrosoftGraphClient{}, nil
}

// maxMicrosoftGraphCredentials caps how many Graph credentials may be configured.
//
// Fleet's data model, sync loop, and storage are all built for the multi-tenant case, because an Autopilot device
// registry is scoped to a single Entra tenant and surfacing several tenants needs one credential each. Only this cap
// limits it to one for now, so raising it is a one-line change with no rename, no scalar-to-list type change, no API
// break, and no migration.
const maxMicrosoftGraphCredentials = 1

// resolveMicrosoftGraphCredentials validates the incoming credential list and returns it with every client secret
// resolved to a usable value.
//
// A caller that already has a stored credential may send back the masked placeholder (the UI round-trips what
// GET /config returned) or omit the secret entirely; both mean "keep the stored secret". A genuinely new secret
// requires the server private key, since it is encrypted at rest.
//
// Validation failures are accumulated on invalid rather than returned, so one bad entry does not mask another. The
// returned slice only contains entries that validated.
func (svc *Service) resolveMicrosoftGraphCredentials(
	ctx context.Context,
	incoming []fleet.MicrosoftGraphCredential,
	lic *fleet.LicenseInfo,
	invalid *fleet.InvalidArgumentError,
) ([]fleet.MicrosoftGraphCredential, error) {
	if len(incoming) == 0 {
		return nil, nil
	}

	if lic == nil || !lic.IsPremium() {
		invalid.Append("mdm.microsoft_graph_credentials", ErrMissingLicense.Error())
		return nil, nil
	}

	if len(incoming) > maxMicrosoftGraphCredentials {
		invalid.Append("mdm.microsoft_graph_credentials",
			fmt.Sprintf("Only %d Microsoft Graph credential can be configured.", maxMicrosoftGraphCredentials))
		return nil, nil
	}

	storedByTenant, err := svc.storedMicrosoftGraphCredentialsByTenant(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(incoming))
	resolved := make([]fleet.MicrosoftGraphCredential, 0, len(incoming))

	for _, cred := range incoming {
		// Entra emits these lower-cased but admins paste them either way. Normalizing keeps the unique key on
		// tenant_id meaningful and makes the stored-credential lookup below reliable.
		cred.TenantID = strings.ToLower(strings.TrimSpace(cred.TenantID))
		cred.ClientID = strings.ToLower(strings.TrimSpace(cred.ClientID))
		cred.ClientSecret = strings.TrimSpace(cred.ClientSecret)

		if !windowsEntraGUIDRegex.MatchString(cred.TenantID) {
			invalid.Append("mdm.microsoft_graph_credentials.tenant_id", fmt.Sprintf("Invalid Entra tenant ID: %s", cred.TenantID))
			continue
		}
		if !windowsEntraGUIDRegex.MatchString(cred.ClientID) {
			invalid.Append("mdm.microsoft_graph_credentials.client_id", fmt.Sprintf("Invalid Entra client ID: %s", cred.ClientID))
			continue
		}
		if _, dup := seen[cred.TenantID]; dup {
			// Two credentials for one tenant would read an identical Autopilot list, because the registry is scoped to
			// the tenant and not to the application.
			invalid.Append("mdm.microsoft_graph_credentials.tenant_id",
				fmt.Sprintf("Duplicate Entra tenant ID: %s. Only one credential per tenant is supported.", cred.TenantID))
			continue
		}
		seen[cred.TenantID] = struct{}{}

		if cred.ClientSecret == "" || cred.ClientSecret == fleet.MaskedPassword {
			existing, ok := storedByTenant[cred.TenantID]
			if !ok || existing.ClientSecret == "" {
				invalid.Append("mdm.microsoft_graph_credentials.client_secret",
					"client_secret must be provided when adding a Microsoft Graph credential")
				continue
			}
			cred.ClientSecret = existing.ClientSecret
		} else if svc.config.Server.PrivateKey == "" {
			invalid.Append("mdm.microsoft_graph_credentials",
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
) error {
	if len(creds) == 0 {
		return nil
	}

	storedByTenant, err := svc.storedMicrosoftGraphCredentialsByTenant(ctx)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		if existing, ok := storedByTenant[cred.TenantID]; ok && existing.Equal(cred) {
			continue
		}

		client, err := svc.msGraphClientFactory(&cred)
		if err != nil {
			invalid.Append("mdm.microsoft_graph_credentials", fmt.Sprintf("Couldn't use Microsoft Graph credential: %s", err))
			continue
		}
		if err := client.VerifyCredential(ctx); err != nil {
			invalid.Append("mdm.microsoft_graph_credentials", microsoftGraphVerifyMessage(err))
			continue
		}
	}

	return nil
}

// microsoftGraphVerifyMessage turns a Graph failure into something an admin can act on. The three cases have genuinely
// different remedies, and Graph reports a missing permission under different error codes depending on the endpoint
// family, so the classification keys on the HTTP status rather than the code string.
func microsoftGraphVerifyMessage(err error) string {
	graphErr, ok := msgraph.AsError(err)
	if !ok {
		return fmt.Sprintf("Couldn't connect to Microsoft Graph: %s", err)
	}
	switch {
	case graphErr.IsPermissionError():
		return "Microsoft Graph denied the request. Grant the app registration the DeviceManagementServiceConfig.Read.All " +
			"application permission and grant admin consent for your tenant."
	case graphErr.IsAuthError():
		return "Microsoft Graph rejected the credential. Check the tenant ID, client ID, and client secret."
	case graphErr.IsTransient():
		return fmt.Sprintf("Microsoft Graph is temporarily unavailable (%d). Please try again.", graphErr.StatusCode)
	default:
		return fmt.Sprintf("Couldn't verify the Microsoft Graph credential: %s", graphErr)
	}
}

// persistMicrosoftGraphCredentials reconciles stored credentials to match the supplied list, and reports which tenants
// were added, edited, or deleted so the caller can emit one activity apiece.
//
// The credentials are not part of the AppConfig JSON, so a change here is invisible in the saved-config diff and has to
// be tracked explicitly.
func (svc *Service) persistMicrosoftGraphCredentials(
	ctx context.Context,
	creds []fleet.MicrosoftGraphCredential,
) (added, edited, deleted []string, err error) {
	storedByTenant, err := svc.storedMicrosoftGraphCredentialsByTenant(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	incomingByTenant := make(map[string]struct{}, len(creds))
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

		if err := svc.ds.UpsertMicrosoftGraphCredential(ctx, &cred); err != nil {
			return nil, nil, nil, ctxerr.Wrap(ctx, err, "upsert microsoft graph credential")
		}
	}

	for tenantID := range storedByTenant {
		if _, ok := incomingByTenant[tenantID]; ok {
			continue
		}
		if err := svc.ds.DeleteMicrosoftGraphCredential(ctx, tenantID); err != nil {
			return nil, nil, nil, ctxerr.Wrap(ctx, err, "delete microsoft graph credential")
		}
		deleted = append(deleted, tenantID)
	}

	return added, edited, deleted, nil
}

// hydrateMicrosoftGraphCredentials populates ac.MDM.MicrosoftGraphCredentials from the credentials table.
//
// Both the GET and PATCH config responses need this: the credentials are not part of the AppConfig JSON, so a config
// read (or the re-read that builds the PATCH response) carries nothing unless it is filled in here. Callers must still
// call Obfuscate afterwards to mask the secrets.
//
// When nothing is configured the field is left untouched rather than set to an empty list, mirroring the conditional
// hydration of MDM.WindowsEnrollment in AppConfigObfuscated. Unconditionally setting it would change the marshalled
// config for every deployment that has never configured the feature, which is the overwhelming majority.
func (svc *Service) hydrateMicrosoftGraphCredentials(ctx context.Context, ac *fleet.AppConfig) error {
	stored, err := svc.ds.ListMicrosoftGraphCredentialMetadata(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list microsoft graph credential metadata")
	}
	if len(stored) == 0 && !ac.MDM.MicrosoftGraphCredentials.Set {
		return nil
	}
	creds := make([]fleet.MicrosoftGraphCredential, 0, len(stored))
	for _, cred := range stored {
		creds = append(creds, *cred)
	}
	ac.MDM.MicrosoftGraphCredentials = optjson.SetSlice(creds)
	return nil
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
