package google_cloud_identity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Syncer drives per-host PATCHes to Cloud Identity. One Syncer is constructed
// per Fleet server startup and reused across host check-ins.
type Syncer struct {
	ds         fleet.Datastore
	client     *Client
	cfg        config.GoogleCloudIdentityConfig
	logger     *slog.Logger
	customerID string // cached: "{C-id-without-C}" (the leading C stripped)
}

// NewSyncer builds a Syncer. The provided Client must already be authenticated
// against the customer's Workspace tenant (caller validated against
// customers/my_customer at startup; mismatch = hard error).
func NewSyncer(ds fleet.Datastore, client *Client, cfg config.GoogleCloudIdentityConfig, logger *slog.Logger) *Syncer {
	// customerID in PATCH is "{C-id-without-C}" — strip the leading C if
	// present, since AppConfig stores the full Cxxxxxxx form per Google's
	// directory format but the partner segment of the ClientState resource
	// name omits the C.
	cust := strings.TrimSpace(cfg.CustomerID)
	cust = strings.TrimPrefix(cust, "C")
	cust = strings.TrimPrefix(cust, "c")
	return &Syncer{
		ds:         ds,
		client:     client,
		cfg:        cfg,
		logger:     logger,
		customerID: cust,
	}
}

// SyncHost computes the desired ClientState for every (host, deviceUser)
// pair Fleet has staged for the host, lazily resolves any unresolved
// raw_resource_ids, diffs against the last-known state, and PATCHes Cloud
// Identity only when something changed.
//
// Called from processConditionalAccess after the Microsoft branch.
//
// `managed` = MDM enrolled, `compliant` = all CA-flagged policies passing,
// `scoreReason` = comma-joined failing-policy names (or "" when compliant).
func (s *Syncer) SyncHost(
	ctx context.Context,
	host *fleet.Host,
	managed bool,
	compliant bool,
	scoreReason string,
) error {
	rows, err := s.ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	if err != nil {
		return fmt.Errorf("load clientstates: %w", err)
	}
	if len(rows) == 0 {
		// No EV-resolved Workspace identities on the host — nothing to PATCH.
		// This is the normal state for hosts without Endpoint Verification.
		return nil
	}

	for _, row := range rows {
		if err := s.syncRow(ctx, host, row, managed, compliant, scoreReason); err != nil {
			// One row failing shouldn't drop the others. Log and continue.
			s.logger.ErrorContext(ctx, "google_cloud_identity: sync row failed",
				"host_id", host.ID,
				"raw_resource_id", row.RawResourceID,
				"err", err,
			)
		}
	}
	return nil
}

func (s *Syncer) syncRow(
	ctx context.Context,
	host *fleet.Host,
	row *fleet.HostGoogleCloudIdentityClientState,
	managed bool,
	compliant bool,
	scoreReason string,
) error {
	// Step 1: lazy-resolve raw_resource_id -> canonical deviceUser name.
	deviceUserResource, err := s.ensureDeviceUserResolved(ctx, host, row)
	if err != nil {
		return fmt.Errorf("resolve deviceUser: %w", err)
	}
	if deviceUserResource == "" {
		// Lookup returned no matches — Google doesn't know about this device.
		// Could be transient (EV just installed, propagation pending) or
		// permanent (resource_id stale after a wipe). Skip; next ingest will
		// retry.
		s.logger.DebugContext(ctx, "google_cloud_identity: lookup returned no deviceUser",
			"host_id", host.ID,
			"raw_resource_id", row.RawResourceID,
		)
		return nil
	}

	// Step 2: diff against last-known state.
	if row.LastManaged != nil && row.LastCompliant != nil &&
		*row.LastManaged == managed && *row.LastCompliant == compliant &&
		strDeref(row.LastScoreReason) == scoreReason {
		// No change. Nothing to PATCH.
		return nil
	}

	// Step 3: build desired ClientState and PATCH.
	desired := buildClientState(host, managed, compliant, scoreReason)
	// On second-and-later PATCHes, send the etag for optimistic concurrency.
	if row.LastEtag != nil {
		desired.Etag = *row.LastEtag
	}

	partner := s.partnerSegment(row.PartnerSuffix)
	result, err := s.client.PatchClientState(ctx, PatchClientStateRequest{
		DeviceUserResource: deviceUserResource,
		Partner:            partner,
		Customer:           "customers/my_customer",
		State:              desired,
		UpdateMask:         "complianceState,managed,scoreReason,customId,assetTags",
	})
	if err != nil {
		if IsPermissionDenied(err) {
			s.logger.ErrorContext(ctx, "google_cloud_identity: PERMISSION_DENIED — verify customer has Cloud Identity Premium / Workspace Enterprise edition",
				"host_id", host.ID,
			)
		}
		return fmt.Errorf("PATCH clientstate: %w", err)
	}

	// Step 4: record last-known state in DB so next sync diffs against it.
	if err := s.ds.SetHostGoogleCloudIdentityClientState(
		ctx, host.ID, row.RawResourceID, row.PartnerSuffix,
		managed, compliant, scoreReason, result.Etag,
	); err != nil {
		return fmt.Errorf("save last-known state: %w", err)
	}
	return nil
}

// ensureDeviceUserResolved fills in row.device_user_resource lazily by calling
// Cloud Identity's deviceUsers.lookup if it isn't set yet. Returns the
// canonical deviceUser resource name, or empty string if the lookup
// returned no matches.
func (s *Syncer) ensureDeviceUserResolved(
	ctx context.Context,
	host *fleet.Host,
	row *fleet.HostGoogleCloudIdentityClientState,
) (string, error) {
	if row.DeviceUserResource != nil && *row.DeviceUserResource != "" {
		return *row.DeviceUserResource, nil
	}
	resp, err := s.client.LookupDeviceUserByRawResourceID(ctx, row.RawResourceID)
	if err != nil {
		return "", err
	}
	if len(resp.Names) == 0 {
		return "", nil
	}
	// rawResourceId is unambiguous (one device → one deviceUser) so a
	// well-formed response has exactly one name.
	name := resp.Names[0]
	if err := s.ds.SetHostGoogleCloudIdentityResolvedDeviceUser(
		ctx, host.ID, row.RawResourceID, row.PartnerSuffix, name,
	); err != nil {
		return "", fmt.Errorf("persist resolved deviceUser: %w", err)
	}
	return name, nil
}

// partnerSegment assembles the partner portion of the ClientState resource
// name. Non-Alliance form: "{customer_id_without_C}-{suffix}". If suffix
// already contains a customer-ID-style prefix (in case a future Alliance
// rollout writes a global identifier directly), we leave it alone.
func (s *Syncer) partnerSegment(suffix string) string {
	if suffix == "" {
		suffix = "fleet"
	}
	return fmt.Sprintf("%s-%s", suffix, s.customerID)
}

// buildClientState assembles the desired ClientState to write. Fields are
// constrained to what's in the update mask above; the receiver populates
// the Name lazily from the PATCH URL.
func buildClientState(host *fleet.Host, managed, compliant bool, scoreReason string) *ClientState {
	state := &ClientState{
		CustomID: host.UUID,
	}
	if managed {
		state.Managed = ManagedStateManaged
	} else {
		state.Managed = ManagedStateUnmanaged
	}
	if compliant {
		state.ComplianceState = ComplianceStateCompliant
	} else {
		state.ComplianceState = ComplianceStateNonCompliant
	}
	if scoreReason != "" {
		state.ScoreReason = scoreReason
	}
	if host.TeamID != nil {
		state.AssetTags = []string{fmt.Sprintf("fleet_team_id:%d", *host.TeamID)}
	}
	return state
}

func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ErrNotConfigured is returned by syncer construction when the integration's
// server-side credentials aren't set. Callers should silently no-op.
var ErrNotConfigured = errors.New("google_cloud_identity: not configured")
