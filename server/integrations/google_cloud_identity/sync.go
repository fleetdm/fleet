package google_cloud_identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	cloudidentity "google.golang.org/api/cloudidentity/v1beta1"
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
// against the customer's Workspace tenant.
func NewSyncer(ds fleet.Datastore, client *Client, cfg config.GoogleCloudIdentityConfig, logger *slog.Logger) *Syncer {
	// customerID in PATCH is "{C-id-without-C}" — strip the leading C since
	// AppConfig stores the full Cxxxxxxx form per Google's directory format
	// but the partner segment of the ClientState resource name omits the C.
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
// workspace_emails by querying Cloud Identity, diffs against last-known
// state, and PATCHes Cloud Identity only when something changed.
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
		return nil
	}

	for _, row := range rows {
		if err := s.syncRow(ctx, host, row, managed, compliant, scoreReason); err != nil {
			// One row failing shouldn't drop the others. Log and continue.
			s.logger.ErrorContext(ctx, "google_cloud_identity: sync row failed",
				"host_id", host.ID,
				"workspace_email", row.WorkspaceEmail,
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
	// Step 1: lazy-resolve workspace_email -> canonical deviceUser name via
	// Cloud Identity. Cached after first success.
	deviceUserResource, err := s.ensureDeviceUserResolved(ctx, host, row)
	if err != nil {
		return fmt.Errorf("resolve deviceUser: %w", err)
	}
	if deviceUserResource == "" {
		s.logger.DebugContext(ctx, "google_cloud_identity: no matching deviceUser",
			"host_id", host.ID,
			"workspace_email", row.WorkspaceEmail,
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
	if row.LastEtag != nil {
		desired.Etag = *row.LastEtag
	}

	partner := s.partnerSegment(row.PartnerSuffix)
	op, err := s.client.PatchClientState(ctx,
		deviceUserResource,
		partner,
		desired,
		"complianceState,managed,scoreReason,customId,assetTags",
	)
	if err != nil {
		if IsPermissionDenied(err) {
			s.logger.ErrorContext(ctx, "google_cloud_identity: PERMISSION_DENIED — verify customer has Cloud Identity Premium / Workspace Enterprise edition",
				"host_id", host.ID,
			)
		}
		return fmt.Errorf("PATCH clientstate: %w", err)
	}

	// Step 4: record last-known state in DB so next sync diffs against it.
	newEtag := etagFromOperation(op)
	if err := s.ds.SetHostGoogleCloudIdentityClientState(
		ctx, host.ID, row.WorkspaceEmail, row.PartnerSuffix,
		managed, compliant, scoreReason, newEtag,
	); err != nil {
		return fmt.Errorf("save last-known state: %w", err)
	}
	return nil
}

// ensureDeviceUserResolved fills in row.device_user_resource lazily by
// querying Cloud Identity via FindDeviceBySerial + ListDeviceUsers and
// matching by workspace_email. Returns the canonical deviceUser resource
// name, or empty string if no match.
//
// This is the corrected resolution flow per the rework: the rawResourceId
// lookup endpoint requires end-user creds, but the admin-side
// devices.list + deviceUsers.list endpoints are reachable from a DWD
// service account.
func (s *Syncer) ensureDeviceUserResolved(
	ctx context.Context,
	host *fleet.Host,
	row *fleet.HostGoogleCloudIdentityClientState,
) (string, error) {
	if row.DeviceUserResource != nil && *row.DeviceUserResource != "" {
		return *row.DeviceUserResource, nil
	}
	if host.HardwareSerial == "" {
		// No serial → can't resolve. This shouldn't happen for an
		// orbit-managed host, but skip safely if it does.
		return "", nil
	}

	device, err := s.client.FindDeviceBySerial(ctx, host.HardwareSerial)
	if err != nil {
		return "", err
	}
	if device == nil {
		// Host has a serial but Google has never seen it (no EV / GMM /
		// Drive for Desktop signed in yet). Try again on the next sync.
		return "", nil
	}

	users, err := s.client.ListDeviceUsers(ctx, device.Name)
	if err != nil {
		return "", err
	}

	target := strings.ToLower(row.WorkspaceEmail)
	var matched *cloudidentity.DeviceUser
	for _, u := range users {
		if strings.EqualFold(u.UserEmail, target) {
			matched = u
			break
		}
	}
	if matched == nil {
		// Device exists in Cloud Identity but the user we're tracking isn't
		// signed in there. Could mean EV is installed but the user hasn't
		// signed in yet, or they're signed in via a different surface.
		return "", nil
	}

	if err := s.ds.SetHostGoogleCloudIdentityResolvedDeviceUser(
		ctx, host.ID, row.WorkspaceEmail, row.PartnerSuffix, matched.Name,
	); err != nil {
		return "", fmt.Errorf("persist resolved deviceUser: %w", err)
	}
	return matched.Name, nil
}

// partnerSegment assembles the partner portion of the ClientState resource
// name. Non-Alliance form: `{suffix}-{customer_id_without_C}`.
func (s *Syncer) partnerSegment(suffix string) string {
	if suffix == "" {
		suffix = "fleet"
	}
	return fmt.Sprintf("%s-%s", suffix, s.customerID)
}

// buildClientState assembles the desired ClientState to write.
func buildClientState(host *fleet.Host, managed, compliant bool, scoreReason string) *cloudidentity.ClientState {
	state := &cloudidentity.ClientState{
		CustomId: host.UUID,
	}
	if managed {
		state.Managed = "MANAGED"
	} else {
		state.Managed = "UNMANAGED"
	}
	if compliant {
		state.ComplianceState = "COMPLIANT"
	} else {
		state.ComplianceState = "NON_COMPLIANT"
	}
	if scoreReason != "" {
		state.ScoreReason = scoreReason
	}
	if host.TeamID != nil {
		state.AssetTags = []string{fmt.Sprintf("fleet_team_id:%d", *host.TeamID)}
	}
	return state
}

// etagFromOperation extracts the etag from the response embedded in a
// long-running Operation. ClientStates PATCH appears synchronous in
// practice — Done=true with a ClientState in Response — but we accept
// either shape since the SDK types the return as Operation.
func etagFromOperation(op *cloudidentity.Operation) string {
	if op == nil || len(op.Response) == 0 {
		return ""
	}
	var cs cloudidentity.ClientState
	if err := json.Unmarshal(op.Response, &cs); err != nil {
		return ""
	}
	return cs.Etag
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
