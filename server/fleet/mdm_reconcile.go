package fleet

import "time"

// This file holds the platform-neutral types used by the batched MDM profile reconcilers to compute desired state in memory. The
// matching applicability logic lives in server/mdm/reconcile so label / team semantics cannot drift between platforms.

// MDMProfileIncludeMode indicates how a profile's include-labels gate applicability to a host. Independent of exclude-labels,
// which always have "exclude any" semantics. A single profile may carry both include labels (with one consistent mode) and
// exclude labels.
type MDMProfileIncludeMode int

const (
	// MDMProfileIncludeNone means the profile has no include labels; applicability is determined entirely by team, platform, and any
	// exclude labels present.
	MDMProfileIncludeNone MDMProfileIncludeMode = iota
	// MDMProfileIncludeAll requires the host to be a member of every (non-broken) include label.
	MDMProfileIncludeAll
	// MDMProfileIncludeAny requires the host to be a member of at least one include label.
	MDMProfileIncludeAny
)

// MDMProfileLabelRef is a single label reference attached to a profile. A nil LabelID means the label was deleted (the assignment
// is "broken").
type MDMProfileLabelRef struct {
	LabelID   *uint
	CreatedAt time.Time
	// LabelMembershipType mirrors labels.label_membership_type: 0=dynamic, 1=manual, 2=host_vitals (see LabelMembershipType in
	// labels.go). Needed by the exclude-any handler so dynamic labels that were created after a host's last label_updated_at are
	// treated as "results not yet reported" instead of "host is not a member"; manual and host-vitals labels skip that timing check.
	LabelMembershipType int
}

// MDMLabeledEntity is the minimal view of a label-gated MDM entity (profile or declaration) that the team/label dispatcher and
// the per-mode handlers in server/mdm/reconcile need. Every platform's reconciler entities implement this one interface so the
// applicability rules cannot drift between platforms.
type MDMLabeledEntity interface {
	GetTeamID() uint
	GetIncludeMode() MDMProfileIncludeMode
	GetIncludeLabels() []MDMProfileLabelRef
	GetExcludeLabels() []MDMProfileLabelRef
	HasBrokenLabel() bool
}

// anyMDMLabelBroken reports whether any label reference has a nil LabelID (the label was deleted, leaving the assignment
// "broken").
func anyMDMLabelBroken(labels []MDMProfileLabelRef) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			return true
		}
	}
	return false
}
