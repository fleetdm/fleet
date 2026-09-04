package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestAPNsSweep(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	newHost := func(i int, platform string) *fleet.Host {
		h, err := ds.NewHost(ctx, &fleet.Host{
			Hostname:        fmt.Sprintf("sweep-host-%02d", i),
			OsqueryHostID:   new(fmt.Sprintf("sweep-osq-%02d", i)),
			NodeKey:         new(fmt.Sprintf("sweep-node-%02d", i)),
			UUID:            fmt.Sprintf("sweep-uuid-%02d", i),
			Platform:        platform,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
		})
		require.NoError(t, err)
		return h
	}
	setHostMDMOn := func(h *fleet.Host) {
		err := ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, "", false)
		require.NoError(t, err)
	}
	setLastSeen := func(enrollmentID string, silentFor time.Duration) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE nano_enrollments SET last_seen_at = DATE_SUB(NOW(), INTERVAL ? SECOND) WHERE id = ?`,
				int(silentFor.Seconds()), enrollmentID)
			return err
		})
	}

	// silent > 24h, MDM on: eligible (device channel).
	hEligible := newHost(1, "darwin")
	nanoEnroll(t, ds, hEligible, false)
	setHostMDMOn(hEligible)
	setLastSeen(hEligible.UUID, 25*time.Hour)

	// silent > 24h, MDM on, with a user channel: both enrollments eligible.
	hEligibleUser := newHost(2, "darwin")
	nanoEnroll(t, ds, hEligibleUser, true)
	setHostMDMOn(hEligibleUser)
	userEnrollmentID := hEligibleUser.UUID + ":" + nanoenroll_useruuid_prefix + hEligibleUser.UUID
	setLastSeen(hEligibleUser.UUID, 25*time.Hour)
	setLastSeen(userEnrollmentID, 25*time.Hour)

	// seen recently: walked but not eligible (nanoEnroll sets last_seen_at ~now).
	hRecent := newHost(3, "ios")
	nanoEnroll(t, ds, hRecent, false)
	setHostMDMOn(hRecent)

	// disabled enrollment: not walked at all.
	hDisabled := newHost(4, "darwin")
	nanoEnroll(t, ds, hDisabled, false)
	setHostMDMOn(hDisabled)
	setLastSeen(hDisabled.UUID, 25*time.Hour)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE nano_enrollments SET enabled = 0 WHERE id = ?`, hDisabled.UUID)
		return err
	})

	// host_mdm says not enrolled (no row at all here): walked but not eligible.
	hMDMOff := newHost(5, "darwin")
	nanoEnroll(t, ds, hMDMOff, false)
	setLastSeen(hMDMOff.UUID, 25*time.Hour)

	// enrollment with no matching hosts row: walked but not eligible.
	const orphanID = "sweep-uuid-99-orphan"
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, orphanID); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `
			INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at)
			VALUES (?, ?, 'Device', 'topic', 'magic', 'aa', DATE_SUB(NOW(), INTERVAL 25 HOUR))`,
			orphanID, orphanID)
		return err
	})

	wantEligible := []string{hEligible.UUID, hEligibleUser.UUID, userEnrollmentID}
	// enabled rows walked: hEligible, hEligibleUser (device + user), hRecent, hMDMOff, orphan.
	const enabledRows = 6

	t.Run("eligibility matrix in one page", func(t *testing.T) {
		eligible, next, pageFull, err := ds.ListNanoEnrollmentIDsForAPNsSweep(ctx, "", 100, 24*time.Hour)
		require.NoError(t, err)
		require.ElementsMatch(t, wantEligible, eligible)
		require.False(t, pageFull)
		// the cursor advances past every walked row, eligible or not; the
		// orphan id sorts last among the seeded rows.
		require.Equal(t, orphanID, next)
	})

	t.Run("keyset pagination visits every enabled row exactly once", func(t *testing.T) {
		var allEligible []string
		cursor := ""
		pages := 0
		for {
			eligible, next, pageFull, err := ds.ListNanoEnrollmentIDsForAPNsSweep(ctx, cursor, 2, 24*time.Hour)
			require.NoError(t, err)
			allEligible = append(allEligible, eligible...)
			pages++
			require.Less(t, pages, 20, "walk must terminate")
			if !pageFull {
				break
			}
			require.Greater(t, next, cursor, "cursor must advance")
			cursor = next
		}
		// 6 enabled rows at batch size 2 = 3 full pages, then one empty page
		// that reports the pass complete.
		require.Equal(t, enabledRows/2+1, pages)
		require.ElementsMatch(t, wantEligible, allEligible)
	})

	t.Run("shorter silence window widens eligibility", func(t *testing.T) {
		// with a 1-second window even the recently-seen rows qualify,
		// proving the silentFor parameter drives the filter. Set the
		// timestamp explicitly rather than relying on nanoEnroll's seed.
		setLastSeen(hRecent.UUID, 2*time.Second)
		eligible, _, _, err := ds.ListNanoEnrollmentIDsForAPNsSweep(ctx, "", 100, time.Second)
		require.NoError(t, err)
		require.ElementsMatch(t, append(wantEligible, hRecent.UUID), eligible)
	})

	t.Run("count enabled only", func(t *testing.T) {
		count, err := ds.CountEnabledNanoEnrollments(ctx)
		require.NoError(t, err)
		require.Equal(t, enabledRows, count)
	})

	t.Run("bare datastore sweep state is a no-op", func(t *testing.T) {
		state, err := ds.GetMDMAppleAPNsSweepState(ctx)
		require.NoError(t, err)
		require.Nil(t, state)
		require.NoError(t, ds.SetMDMAppleAPNsSweepState(ctx, &fleet.MDMAppleAPNsSweepState{Cursor: "x", BatchSize: 1}))
	})
}
