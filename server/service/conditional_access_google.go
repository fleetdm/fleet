package service

import (
	"context"
	"fmt"
	"sort"


	google_cloud_identity "github.com/fleetdm/fleet/v4/server/integrations/google_cloud_identity"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"google.golang.org/api/option"
)

// processGoogleCloudIdentityForNewlyFailingPolicies is the Google analog of
// processConditionalAccessForNewlyFailingPolicies. Triggered from the osquery
// distributed-query write pipeline whenever a host's policy results change.
//
// Semantics:
//   - Vanilla osquery hosts skipped (require orbit, same as Microsoft).
//   - Server config + per-team enable flag must both be set.
//   - Per-team CA-flagged policy set is read; only those policies count toward
//     `compliant`.
//   - `managed` is mdmEnrolled.
//   - The actual lookup + PATCH is delegated to a *Syncer that's
//     lazy-constructed on first invocation.
//   - All Cloud Identity I/O runs in a goroutine so the osquery distributed-
//     query write path never blocks on Google network calls.
func (svc *Service) processGoogleCloudIdentityForNewlyFailingPolicies(
	ctx context.Context,
	host *fleet.Host,
	incomingPolicyResults map[uint]*bool,
) error {
	if host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" {
		// Vanilla osquery hosts can't drive conditional access — no orbit
		// extension = no endpoint_verification_accounts table.
		return nil
	}

	configured, enabledForTeam, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(ctx, host.TeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check google cloud identity enabled")
	}
	if !configured || !enabledForTeam {
		return nil
	}

	// Compute desired state from the same per-team CA-flagged policies the
	// Microsoft branch uses. Reusing the existing flag (no separate
	// `cloud_identity_enabled` column on policies) means admins flag a policy
	// for CA once and both providers honor it.
	var policyTeamID uint
	if host.TeamID == nil {
		policyTeamID = fleet.PolicyNoTeamID
	} else {
		policyTeamID = *host.TeamID
	}

	caPolicyIDs, err := svc.ds.GetPoliciesForConditionalAccess(ctx, policyTeamID, host.Platform)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "google cloud identity: get CA policies")
	}

	caSet := make(map[uint]struct{}, len(caPolicyIDs))
	for _, id := range caPolicyIDs {
		caSet[id] = struct{}{}
	}

	totalCAPolicies := len(caSet)

	var failingIDs []uint
	for incomingID, result := range incomingPolicyResults {
		if _, ok := caSet[incomingID]; !ok {
			continue
		}
		if result != nil && !*result {
			failingIDs = append(failingIDs, incomingID)
		}
	}

	// Resolve failing policy IDs to human-readable names for scoreReason.
	// Admins triaging "why is this host blocked" should see real names, not
	// internal IDs.
	var failingNames []string
	if len(failingIDs) > 0 {
		policiesByID, err := svc.ds.PoliciesByID(ctx, failingIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "google cloud identity: load policy names")
		}
		failingNames = make([]string, 0, len(failingIDs))
		for _, id := range failingIDs {
			p, ok := policiesByID[id]
			if !ok || p == nil {
				failingNames = append(failingNames, fmt.Sprintf("policy_%d", id))
				continue
			}
			failingNames = append(failingNames, p.Name)
		}
		// Stable ordering so scoreReason is deterministic across runs.
		sort.Strings(failingNames)
	}

	var mdmEnrolled bool
	hostMDM, err := svc.ds.GetHostMDM(ctx, host.ID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "google cloud identity: get host MDM")
	}
	if hostMDM != nil {
		mdmEnrolled = hostMDM.Enrolled
	}

	// Hand off to the Syncer in a goroutine so the osquery write path
	// returns quickly. The Syncer is lazy-constructed and reused.
	syncer, err := svc.googleCloudIdentitySyncerOrNil(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "google cloud identity: build syncer")
	}
	if syncer == nil {
		return nil
	}

	go func() {
		bg := context.Background()
		if err := syncer.SyncHost(bg, host, mdmEnrolled, totalCAPolicies, failingNames); err != nil {
			svc.logger.ErrorContext(bg, "google_cloud_identity: SyncHost failed",
				"host_id", host.ID,
				"err", err,
			)
		}
	}()
	return nil
}

// googleCloudIdentitySyncerOrNil returns the lazy-initialized Syncer, or nil
// if the integration's credentials aren't set / auth construction failed.
// Errors during the once-init are memoized; subsequent calls return the same
// state without re-attempting until process restart.
func (svc *Service) googleCloudIdentitySyncerOrNil(ctx context.Context) (*google_cloud_identity.Syncer, error) {
	if !svc.config.GoogleCloudIdentity.IsSet() {
		return nil, nil
	}

	svc.googleCloudIdentitySyncerOnce.Do(func() {
		ts, err := google_cloud_identity.NewTokenSource(ctx, svc.config.GoogleCloudIdentity)
		if err != nil {
			svc.googleCloudIdentitySyncerErr = fmt.Errorf("google cloud identity token source: %w", err)
			return
		}
		client, err := google_cloud_identity.NewClient(ctx, option.WithTokenSource(ts))
		if err != nil {
			svc.googleCloudIdentitySyncerErr = fmt.Errorf("google cloud identity client: %w", err)
			return
		}
		svc.googleCloudIdentitySyncerMu.Lock()
		svc.googleCloudIdentitySyncer = google_cloud_identity.NewSyncer(
			svc.ds, client, svc.config.GoogleCloudIdentity, svc.logger,
		)
		svc.googleCloudIdentitySyncerMu.Unlock()
	})

	if svc.googleCloudIdentitySyncerErr != nil {
		return nil, svc.googleCloudIdentitySyncerErr
	}
	svc.googleCloudIdentitySyncerMu.Lock()
	defer svc.googleCloudIdentitySyncerMu.Unlock()
	return svc.googleCloudIdentitySyncer, nil
}
