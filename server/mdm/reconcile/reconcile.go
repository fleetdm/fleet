// Package reconcile holds the platform-neutral core of MDM profile reconciliation: the include/exclude label handlers and the
// team+label applicability dispatcher. Every platform's batched reconciler (Apple profiles, Apple DDM declarations, Windows
// profiles) routes through this package so team / label semantics cannot drift between platforms.
//
// Platform eligibility is intentionally NOT part of this package: each caller applies its own platform gate (e.g.
// apple_mdm.IsEligiblePlatform) before, or together with, EntityAppliesToHost.
package reconcile

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// EntityAppliesToHost is the shared top-level dispatcher for MDM label-gated entities (profiles and declarations). It applies the
// team gate, then composes the include + exclude label handlers carried by the entity.
//
// hostEffectiveTeamID is the host's team with nil normalized to 0 (team_id=0 is its own "no team" scope, not a fallback for
// teamed hosts). hostLabelUpdatedAt is the host's labels-last-scanned timestamp, used by the exclude-any handler's dynamic-label
// timing rule. hostLabels is the set of label IDs the host is a member of.
func EntityAppliesToHost(
	e fleet.MDMLabeledEntity,
	hostEffectiveTeamID uint,
	hostLabelUpdatedAt time.Time,
	hostLabels map[uint]struct{},
) bool {
	if e.GetTeamID() != hostEffectiveTeamID {
		return false
	}

	if e.GetIncludeMode() != fleet.MDMProfileIncludeNone {
		var ok bool
		switch e.GetIncludeMode() {
		case fleet.MDMProfileIncludeAll:
			ok = HandlerIncludeAll(e.GetIncludeLabels(), hostLabels)
		case fleet.MDMProfileIncludeAny:
			ok = HandlerIncludeAny(e.GetIncludeLabels(), hostLabels)
		default:
			return false
		}
		if !ok {
			return false
		}
	}

	if exc := e.GetExcludeLabels(); len(exc) > 0 {
		if HandlerExcludeAny(exc, hostLabelUpdatedAt, hostLabels) {
			return false
		}
	}

	return true
}

// HandlerIncludeAll checks that host is a member of every (non-broken) include label. A broken label disqualifies the entity,
// mirroring the legacy SQL where include-* with a broken label produces no desired-state row.
func HandlerIncludeAll(labels []fleet.MDMProfileLabelRef, hostLabels map[uint]struct{}) bool {
	if len(labels) == 0 {
		return false
	}
	for _, l := range labels {
		if l.LabelID == nil {
			return false
		}
		if _, ok := hostLabels[*l.LabelID]; !ok {
			return false
		}
	}
	return true
}

// HandlerIncludeAny checks that host is a member of at least one include label. Broken labels can't match (host can't be a member
// of a deleted label) so they're silently skipped.
func HandlerIncludeAny(labels []fleet.MDMProfileLabelRef, hostLabels map[uint]struct{}) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			continue
		}
		if _, ok := hostLabels[*l.LabelID]; ok {
			return true
		}
	}
	return false
}

// HandlerExcludeAny checks that entity passes the exclude gate when the host is NOT a member of any referenced label, with two
// safety rules:
//
//   - Any broken exclude label disqualifies the entity entirely (we can't
//     prove the exclusion).
//   - Dynamic labels created after the host's last label scan
//     (hostLabelUpdatedAt) are treated as "results not yet reported" and
//     also disqualify, so we don't install a profile that the
//     not-yet-scanned label would exclude.
//     Manual labels (membership_type=1) skip this timing check.
//     Host vital labels run their own cron to associate, skip the timing check.
//
// Returns true if the host should be excluded, false if the host passes the exclude gate.
func HandlerExcludeAny(
	labels []fleet.MDMProfileLabelRef,
	hostLabelUpdatedAt time.Time,
	hostLabels map[uint]struct{},
) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			return true
		}
		if l.LabelMembershipType == int(fleet.LabelMembershipTypeDynamic) && !l.CreatedAt.IsZero() && hostLabelUpdatedAt.Before(l.CreatedAt) {
			return true
		}
		if _, isMember := hostLabels[*l.LabelID]; isMember {
			return true
		}
	}
	return false
}
