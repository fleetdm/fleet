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
// teamed hosts). hostLabelUpdatedAt is the host's labels-last-scanned timestamp, used by the include-all and exclude-any
// handlers' dynamic-label unknown-membership rule. hostLabels is the set of label IDs the host is a member of.
//
// entityOnHost reports whether the entity is currently on the host: an install-operation row exists for it in the platform's
// host_mdm_*_profiles table, whatever its status (including failed — Fleet still intends the entity to be there). No row, or a
// remove-operation row, means not on host. It drives the unknown-membership rules so that a dynamic label the host hasn't
// evaluated yet preserves the entity's current state instead of forcing a removal.
func EntityAppliesToHost(
	e fleet.MDMLabeledEntity,
	hostEffectiveTeamID uint,
	hostLabelUpdatedAt time.Time,
	hostLabels map[uint]struct{},
	entityOnHost bool,
) bool {
	if e.GetTeamID() != hostEffectiveTeamID {
		return false
	}

	if e.GetIncludeMode() != fleet.MDMProfileIncludeNone {
		var ok bool
		switch e.GetIncludeMode() {
		case fleet.MDMProfileIncludeAll:
			ok = HandlerIncludeAll(e.GetIncludeLabels(), hostLabelUpdatedAt, hostLabels, entityOnHost)
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
		if HandlerExcludeAny(exc, hostLabelUpdatedAt, hostLabels, entityOnHost) {
			return false
		}
	}

	return true
}

// membershipUnknown reports whether the host's membership in the label cannot be known yet: a dynamic label created after the
// host's last label-query report (hostLabelUpdatedAt) has never been evaluated by the host, so the absence of a membership row
// carries no signal. Manual and host-vitals labels are server-populated, so their membership is always considered known.
func membershipUnknown(l fleet.MDMProfileLabelRef, hostLabelUpdatedAt time.Time) bool {
	return l.LabelMembershipType == int(fleet.LabelMembershipTypeDynamic) &&
		!l.CreatedAt.IsZero() &&
		hostLabelUpdatedAt.Before(l.CreatedAt)
}

// HandlerIncludeAll checks that host is a member of every (non-broken) include label. A broken label disqualifies the entity,
// mirroring the legacy SQL where include-* with a broken label produces no desired-state row.
//
// A label with unknown membership (see membershipUnknown) counts as a member only when the entity is already on the host, so
// adding a label to an entity's scope doesn't remove it from hosts that haven't evaluated the label yet; hosts without the
// entity keep waiting for confirmed membership. A confirmed non-member label still disqualifies regardless of entityOnHost.
func HandlerIncludeAll(labels []fleet.MDMProfileLabelRef, hostLabelUpdatedAt time.Time, hostLabels map[uint]struct{}, entityOnHost bool) bool {
	if len(labels) == 0 {
		return false
	}
	for _, l := range labels {
		if l.LabelID == nil {
			return false
		}
		if _, ok := hostLabels[*l.LabelID]; ok {
			continue
		}
		if entityOnHost && membershipUnknown(l, hostLabelUpdatedAt) {
			continue
		}
		return false
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
//   - A label with unknown membership (see membershipUnknown) preserves the
//     entity's current state: it disqualifies only when the entity is not
//     already on the host, so we don't install an entity the not-yet-evaluated
//     label might exclude, and we don't remove one the label might allow.
//
// Returns true if the host should be excluded, false if the host passes the exclude gate.
func HandlerExcludeAny(
	labels []fleet.MDMProfileLabelRef,
	hostLabelUpdatedAt time.Time,
	hostLabels map[uint]struct{},
	entityOnHost bool,
) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			return true
		}
		if _, isMember := hostLabels[*l.LabelID]; isMember {
			return true
		}
		if !entityOnHost && membershipUnknown(l, hostLabelUpdatedAt) {
			return true
		}
	}
	return false
}
