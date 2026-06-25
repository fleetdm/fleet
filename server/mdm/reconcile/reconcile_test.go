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

func TestHandlerIncludeAll(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, HandlerIncludeAll(nil, map[uint]struct{}{1: {}}))
	})
	t.Run("broken label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), {LabelID: nil}}
		require.False(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host missing a label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.False(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host has all labels -> true", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)}
		require.True(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}, 2: {}}))
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
	hostLabelUpdatedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("empty labels -> false (nothing to exclude)", func(t *testing.T) {
		require.False(t, HandlerExcludeAny(nil, hostLabelUpdatedAt, map[uint]struct{}{}))
	})
	t.Run("broken label -> true (exclude)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{{LabelID: nil}}
		require.True(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{}))
	})
	t.Run("host is in an excluded label -> true", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1)}
		require.True(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{1: {}}))
	})
	t.Run("host is not in any excluded label -> false", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{labelRef(1)}
		require.False(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{2: {}}))
	})
	t.Run("dynamic label created after host's last scan -> true (exclude)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeDynamic),
			},
		}
		require.True(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{}))
	})
	t.Run("host vital label created after host's last scan -> false (include)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeHostVitals),
			},
		}
		require.False(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{}))
	})
	t.Run("manual label created after host's last scan -> still false (include)", func(t *testing.T) {
		labels := []fleet.MDMProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeManual),
			},
		}
		require.False(t, HandlerExcludeAny(labels, hostLabelUpdatedAt, map[uint]struct{}{}))
	})
}

func TestEntityAppliesToHost(t *testing.T) {
	hostLabelUpdatedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	t.Run("wrong team -> false", func(t *testing.T) {
		e := &testEntity{teamID: 5}
		require.False(t, EntityAppliesToHost(e, 3, hostLabelUpdatedAt, nil))
	})
	t.Run("matching team, no labels -> true", func(t *testing.T) {
		e := &testEntity{teamID: 5}
		require.True(t, EntityAppliesToHost(e, 5, hostLabelUpdatedAt, nil))
	})
	t.Run("team 0 is its own scope, not a fallback", func(t *testing.T) {
		e := &testEntity{teamID: 0}
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, nil))
		require.False(t, EntityAppliesToHost(e, 5, hostLabelUpdatedAt, nil))
	})
	t.Run("include_all gate", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}, 2: {}}))
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}}))
	})
	t.Run("include_any gate", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAny,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{2: {}}))
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{3: {}}))
	})
	t.Run("unknown include mode -> false", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeMode(99),
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1)},
		}
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}}))
	})
	t.Run("pure exclude: member -> false, non-member -> true", func(t *testing.T) {
		e := &testEntity{
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(7)},
		}
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{7: {}}))
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{8: {}}))
	})
	t.Run("combined include_any + exclude_any", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAny,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1)},
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(9)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}}))
		// In the include label but also in the exclude label -> excluded.
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}, 9: {}}))
	})
	t.Run("combined include_all + exclude_any", func(t *testing.T) {
		e := &testEntity{
			includeMode:   fleet.MDMProfileIncludeAll,
			includeLabels: []fleet.MDMProfileLabelRef{labelRef(1), labelRef(2)},
			excludeLabels: []fleet.MDMProfileLabelRef{labelRef(9)},
		}
		require.True(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}, 2: {}}))
		require.False(t, EntityAppliesToHost(e, 0, hostLabelUpdatedAt, map[uint]struct{}{1: {}, 2: {}, 9: {}}))
	})
}
