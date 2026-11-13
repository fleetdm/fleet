package service

import (
	"bytes"
	"cmp"
	"context"
	"crypto/md5" //nolint:gosec // checksum for change detection only
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
)

const policyEnforcementRulesKey = "policyEnforcementRules"

// buildFleetDefaultPolicy builds the policy a fleet's hosts enroll with from the fleet's label-less profiles,
// so the end user sets the passcode during setup. Only the passcode settings are copied, and each key goes
// to the profile whose name sorts last, as in the host's own policy, so the passcode asked for during setup
// matches the one enforced afterwards. policyEnforcementRules is copied only when a passcode profile wins
// it. Profiles with variables (only possible if saved before passcode profiles rejected them) are skipped
// and their names returned.
func buildFleetDefaultPolicy(profiles []*fleet.MDMAndroidConfigProfile) (*androidmanagement.Policy, []string, error) {
	sorted := slices.Clone(profiles)
	slices.SortFunc(sorted, func(a, b *fleet.MDMAndroidConfigProfile) int {
		return cmp.Compare(a.Name, b.Name)
	})

	type winner struct {
		value      json.RawMessage
		isPasscode bool
	}
	winners := make(map[string]winner)
	var skipped []string
	for _, p := range sorted {
		if err := fleet.ValidateAndroidPasscodeProfileVariables(p.RawJSON); err != nil {
			skipped = append(skipped, p.Name)
			continue
		}
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(p.RawJSON, &keys); err != nil {
			return nil, nil, fmt.Errorf("unmarshal profile %s: %w", p.ProfileUUID, err)
		}
		isPasscode := fleet.IsAndroidPasscodeProfile(p.RawJSON)
		for _, k := range append(slices.Clone(fleet.AndroidPasscodeKeys), policyEnforcementRulesKey) {
			if v, ok := keys[k]; ok {
				winners[k] = winner{value: v, isPasscode: isPasscode}
			}
		}
	}

	settings := make(map[string]json.RawMessage, len(winners))
	for k, w := range winners {
		if k == policyEnforcementRulesKey && !w.isPasscode {
			continue
		}
		settings[k] = w.value
	}

	var policy androidmanagement.Policy
	if len(settings) > 0 {
		b, err := json.Marshal(settings)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal default policy settings: %w", err)
		}
		if err := json.Unmarshal(b, &policy); err != nil {
			return nil, nil, fmt.Errorf("unmarshal default policy settings: %w", err)
		}
	}
	applyFleetEnforcedSettings(&policy)
	return &policy, skipped, nil
}

func fleetDefaultPolicyChecksum(policy *androidmanagement.Policy) ([]byte, error) {
	b, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}
	sum := md5.Sum(b) //nolint:gosec // checksum for change detection only
	return sum[:], nil
}

func fleetDefaultPolicyName(enterpriseName string, teamID uint) string {
	return fmt.Sprintf("%s/policies/%s", enterpriseName, android.FleetDefaultPolicyID(teamID))
}

// prepareFleetDefaultPolicy builds a fleet's default policy and its checksum.
func prepareFleetDefaultPolicy(ctx context.Context, logger *slog.Logger, teamID uint, profiles []*fleet.MDMAndroidConfigProfile,
) (*androidmanagement.Policy, []byte, error) {
	policy, skipped, err := buildFleetDefaultPolicy(profiles)
	if err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "build default policy for fleet %d", teamID)
	}
	if len(skipped) > 0 {
		logger.WarnContext(ctx, "android passcode profiles with variables are left out of the fleet's default policy",
			"fleet_id", teamID, "profiles", skipped)
	}
	checksum, err := fleetDefaultPolicyChecksum(policy)
	if err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "checksum default policy for fleet %d", teamID)
	}
	return policy, checksum, nil
}

// sendFleetDefaultPolicy patches a fleet's default policy in AMAPI. If AMAPI rejects the passcode settings,
// it sends the base settings instead, so the policy that enrollment tokens point at always exists; the
// passcode then arrives with the host's own policy, which reports the rejection on the profile.
func sendFleetDefaultPolicy(ctx context.Context, client androidmgmt.Client, logger *slog.Logger,
	enterpriseName string, teamID uint, policy *androidmanagement.Policy,
) error {
	policyName := fleetDefaultPolicyName(enterpriseName, teamID)
	patch := func(p *androidmanagement.Policy) error {
		_, err := client.EnterprisesPoliciesPatch(ctx, policyName, p, androidmgmt.PoliciesPatchOpts{ExcludeApps: true})
		if err != nil && !androidmgmt.IsNotModifiedError(err) {
			return err
		}
		return nil
	}
	err := patch(policy)
	if err == nil {
		return nil
	}
	if !androidmgmt.IsBadRequestError(err) {
		return ctxerr.Wrapf(ctx, err, "patch default policy %s", policyName)
	}
	logger.ErrorContext(ctx, "android rejected the passcode settings of the fleet's default policy, sending it without them",
		"fleet_id", teamID, "err", err)
	var base androidmanagement.Policy
	applyFleetEnforcedSettings(&base)
	if err := patch(&base); err != nil {
		return ctxerr.Wrapf(ctx, err, "patch base default policy %s", policyName)
	}
	return nil
}

// reconcileFleetDefaultPolicies sends the default policy of every fleet whose passcode profiles changed,
// including fleets whose last passcode profile was removed, which go back to the base settings. Fleets
// without passcode profiles get theirs when an enrollment token is first created for them.
func (r *profileReconciler) reconcileFleetDefaultPolicies(ctx context.Context) error {
	profiles, err := r.DS.ListAndroidFleetDefaultPolicyProfiles(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list android fleet default policy profiles")
	}
	checksums, err := r.DS.GetAndroidFleetDefaultPolicyChecksums(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android fleet default policy checksums")
	}

	profilesByTeam := make(map[uint][]*fleet.MDMAndroidConfigProfile)
	for _, p := range profiles {
		var teamID uint
		if p.TeamID != nil {
			teamID = *p.TeamID
		}
		profilesByTeam[teamID] = append(profilesByTeam[teamID], p)
	}
	teamIDs := make(map[uint]struct{}, len(profilesByTeam)+len(checksums))
	for teamID := range profilesByTeam {
		teamIDs[teamID] = struct{}{}
	}
	for teamID := range checksums {
		teamIDs[teamID] = struct{}{}
	}

	for teamID := range teamIDs {
		// One fleet's failure (e.g. AMAPI quota) must not hold back the others; it's retried next run.
		if err := r.syncFleetDefaultPolicy(ctx, teamID, profilesByTeam[teamID], checksums[teamID]); err != nil {
			r.Logger.ErrorContext(ctx, "failed to sync android fleet default policy", "fleet_id", teamID, "err", err)
		}
	}
	return nil
}

func (r *profileReconciler) syncFleetDefaultPolicy(ctx context.Context, teamID uint, profiles []*fleet.MDMAndroidConfigProfile,
	storedChecksum []byte,
) error {
	policy, checksum, err := prepareFleetDefaultPolicy(ctx, r.Logger, teamID, profiles)
	if err != nil {
		return err
	}
	if bytes.Equal(checksum, storedChecksum) {
		return nil
	}
	if err := sendFleetDefaultPolicy(ctx, r.Client, r.Logger, r.Enterprise.Name(), teamID, policy); err != nil {
		return err
	}
	// Only replaces what was read, so a policy that EnsureFleetDefaultPolicy marked unverified meanwhile
	// is sent again next run instead of being recorded as up to date.
	if err := r.DS.SetAndroidFleetDefaultPolicyChecksum(ctx, teamID, checksum, storedChecksum); err != nil {
		return ctxerr.Wrapf(ctx, err, "save default policy checksum for fleet %d", teamID)
	}
	return nil
}

// EnsureFleetDefaultPolicy makes sure the default policy of a fleet (0 is Unassigned) was sent to AMAPI
// and returns its name, for enrollment tokens to point at.
func (svc *Service) EnsureFleetDefaultPolicy(ctx context.Context, enterpriseName string, teamID uint) (string, error) {
	checksums, err := svc.fleetDS.GetAndroidFleetDefaultPolicyChecksums(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get android fleet default policy checksums")
	}
	if _, ok := checksums[teamID]; !ok {
		profiles, err := svc.fleetDS.ListAndroidFleetDefaultPolicyProfiles(ctx, &teamID)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "list android fleet default policy profiles")
		}
		policy, _, err := prepareFleetDefaultPolicy(ctx, svc.logger, teamID, profiles)
		if err != nil {
			return "", err
		}
		// The premium service calls this with its own client, so this one may not be authenticated yet.
		secret, err := svc.getClientAuthenticationSecret(ctx)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting client authentication secret")
		}
		_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

		if err := sendFleetDefaultPolicy(ctx, svc.androidAPIClient, svc.logger, enterpriseName, teamID, policy); err != nil {
			return "", err
		}
		// The reconciler may be sending newer profiles at the same time, and either patch can land last, so
		// this doesn't vouch for the content: the reconciler sends the policy again on its next run.
		if err := svc.fleetDS.MarkAndroidFleetDefaultPolicyUnverified(ctx, teamID); err != nil {
			return "", ctxerr.Wrapf(ctx, err, "save default policy of fleet %d", teamID)
		}
	}
	return fleetDefaultPolicyName(enterpriseName, teamID), nil
}
