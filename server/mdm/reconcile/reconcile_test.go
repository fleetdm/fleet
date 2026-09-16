package reconcile

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// testEntity is a minimal fleet.MDMLabeledEntity for dispatcher tests.
type testEntity struct {
	teamID        uint
	includeMode   fleet.MDMProfileIncludeMode
	includeLabels []fleet.MDMProfileLabelRef
	excludeLabels []fleet.MDMProfileLabelRef
}

func (e *testEntity) GetTeamID() uint                              { return e.teamID }
func (e *testEntity) GetIncludeMode() fleet.MDMProfileIncludeMode  { return e.includeMode }
func (e *testEntity) GetIncludeLabels() []fleet.MDMProfileLabelRef { return e.includeLabels }
func (e *testEntity) GetExcludeLabels() []fleet.MDMProfileLabelRef { return e.excludeLabels }
func (e *testEntity) HasBrokenLabel() bool {
	// Iterate the two slices separately: append(e.includeLabels, e.excludeLabels...) can write into e.includeLabels' backing array
	// when it has spare capacity, leaking state across assertions.
	for _, l := range e.includeLabels {
		if l.LabelID == nil {
			return true
		}
	}
	for _, l := range e.excludeLabels {
		if l.LabelID == nil {
			return true
		}
	}
	return false
}

func labelRef(id uint) fleet.MDMProfileLabelRef {
	return fleet.MDMProfileLabelRef{LabelID: new(id)}
}

// hostScannedAt is the reference "host last reported label results" time; labels created after it have unknown membership.
var hostScannedAt = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

// unknownLabelRef returns a label of the given membership type (an int per fleet.MDMProfileLabelRef.LabelMembershipType)
// created after hostScannedAt, i.e. one the host has not evaluated yet (membership unknown for dynamic labels;
// manual/host-vitals are always considered known).
func unknownLabelRef(id uint, membershipType int) fleet.MDMProfileLabelRef {
	return fleet.MDMProfileLabelRef{
		LabelID:             new(id),
		CreatedAt:           hostScannedAt.Add(24 * time.Hour),
		LabelMembershipType: membershipType,
	}
}

func TestHandlerIncludeAll(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, HandlerIncludeAll(nil, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("broken label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), {LabelID: nil}}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("host missing a label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("host has all labels -> true", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.True(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{1: {}, 2: {}}, false))
	})
	t.Run("unknown dynamic label, entity on host -> counts as member", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))}
		require.True(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{1: {}}, true))
	})
	t.Run("unknown dynamic label, entity not on host -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("unknown dynamic label on host but confirmed non-member of another -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{3: {}}, true))
	})
	t.Run("evaluated dynamic label with no membership row is a confirmed non-member even on host", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: hostScannedAt.Add(-24 * time.Hour), LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)},
		}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
	t.Run("manual label created after host's last scan gets no unknown treatment", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeManual))}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
	t.Run("host-vitals label created after host's last scan gets no unknown treatment", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeHostVitals))}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
	t.Run("broken label is not preserved by the unknown rule", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{{LabelID: nil}}
		require.False(t, HandlerIncludeAll(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
}

func TestHandlerIncludeAny(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, HandlerIncludeAny(nil, map[uint]struct{}{}))
	})
	t.Run("broken labels are ignored", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{{LabelID: nil}, labelRef(2)}
		require.True(t, HandlerIncludeAny(labels, map[uint]struct{}{2: {}}))
	})
	t.Run("all labels broken -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{{LabelID: nil}, {LabelID: nil}}
		require.False(t, HandlerIncludeAny(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host in no label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.False(t, HandlerIncludeAny(labels, map[uint]struct{}{3: {}}))
	})
	t.Run("host in one label -> true", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.True(t, HandlerIncludeAny(labels, map[uint]struct{}{1: {}}))
	})
}

func TestHandlerExcludeAny(t *testing.T) {
	t.Run("empty labels -> false (nothing to exclude)", func(t *testing.T) {
		require.False(t, HandlerExcludeAny(nil, hostScannedAt, map[uint]struct{}{}, false))
	})
	t.Run("broken label -> true (exclude), even when entity on host", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{{LabelID: nil}}
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, false))
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
	t.Run("host is in an excluded label -> true, even when entity on host", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1)}
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{1: {}}, false))
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{1: {}}, true))
	})
	t.Run("host is not in any excluded label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1)}
		require.False(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{2: {}}, false))
	})
	t.Run("unknown dynamic label, entity not on host -> true (withhold)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeDynamic))}
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, false))
	})
	t.Run("unknown dynamic label, entity on host -> false (keep)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeDynamic))}
		require.False(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, true))
	})
	t.Run("mixed known and unknown: confirmed member always excludes", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))}
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{1: {}}, true))
	})
	t.Run("mixed known and unknown: known non-member, unknown preserved on host", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))}
		require.False(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, true))
		require.True(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, false))
	})
	t.Run("host vital label created after host's last scan -> false (include)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeHostVitals))}
		require.False(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, false))
	})
	t.Run("manual label created after host's last scan -> still false (include)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{unknownLabelRef(1, int(fleet.LabelMembershipTypeManual))}
		require.False(t, HandlerExcludeAny(labels, hostScannedAt, map[uint]struct{}{}, false))
	})
}

func TestEntityAppliesToHost(t *testing.T) {
	t.Run("wrong team -> false", func(t *testing.T) {
		e := &testEntity{teamID: 5}
		require.False(t, EntityAppliesToHost(e, 3, hostScannedAt, nil, false))
	})
	t.Run("matching team, no labels -> true", func(t *testing.T) {
		e := &testEntity{teamID: 5}
		require.True(t, EntityAppliesToHost(e, 5, hostScannedAt, nil, false))
	})
	t.Run("team 0 is its own scope, not a fallback", func(t *testing.T) {
		e := &testEntity{teamID: 0}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, nil, false))
		require.False(t, EntityAppliesToHost(e, 5, hostScannedAt, nil, false))
	})
	t.Run("include_all gate", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}, 2: {}}, false))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("include_any gate", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAny,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{2: {}}, false))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{3: {}}, false))
	})
	t.Run("unknown include mode -> false", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeMode(99),
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1)},
		}
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("pure exclude: member -> false, non-member -> true", func(t *testing.T) {
		e := &testEntity{
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(7)},
		}
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{7: {}}, false))
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{8: {}}, false))
	})
	t.Run("combined include_any + exclude_any", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAny,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1)},
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(9)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, false))
		// In the include label but also in the exclude label -> excluded.
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}, 9: {}}, false))
	})
	t.Run("combined include_all + exclude_any", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(9)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}, 2: {}}, false))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}, 2: {}, 9: {}}, false))
	})
	t.Run("new exclude label unknown: entity stays on hosts that have it, withheld from those that don't", func(t *testing.T) {
		e := &testEntity{
			excludeLabels: []fleet.MDMProfileLabelRef{unknownLabelRef(9, int(fleet.LabelMembershipTypeDynamic))},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{}, true))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{}, false))
	})
	t.Run("new include_all label unknown: entity stays on hosts that have it, withheld from those that don't", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, true))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
	t.Run("unknown labels in both gates preserve current state", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), unknownLabelRef(2, int(fleet.LabelMembershipTypeDynamic))},
			excludeLabels: []fleet.MDMProfileLabelRef{unknownLabelRef(9, int(fleet.LabelMembershipTypeDynamic))},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, true))
		require.False(t, EntityAppliesToHost(e, 0, hostScannedAt, map[uint]struct{}{1: {}}, false))
	})
}
