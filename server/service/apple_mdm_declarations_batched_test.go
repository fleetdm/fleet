package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/stretchr/testify/require"
)

func TestResolveUserChannelDeliveries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.Background()

	recentlyEnrolled := time.Now().Add(-30 * time.Minute)
	pastGrace := time.Now().Add(-(apple_mdm.HoursToWaitForUserEnrollmentAfterDeviceEnrollment + 1) * time.Hour)

	// One user-scoped install row per host, so we can observe how its status is
	// mutated by the delivery decision.
	newRows := func(hostUUIDs ...string) map[string]*fleet.MDMAppleHostDeclaration {
		rows := make(map[string]*fleet.MDMAppleHostDeclaration, len(hostUUIDs))
		for _, h := range hostUUIDs {
			pending := fleet.MDMDeliveryPending
			rows[h] = &fleet.MDMAppleHostDeclaration{
				HostUUID:      h,
				Scope:         fleet.PayloadScopeUser,
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        &pending,
			}
		}
		return rows
	}
	toSlice := func(m map[string]*fleet.MDMAppleHostDeclaration) []*fleet.MDMAppleHostDeclaration {
		out := make([]*fleet.MDMAppleHostDeclaration, 0, len(m))
		for _, r := range m {
			out = append(out, r)
		}
		return out
	}

	t.Run("user channel exists -> deliver, row stays pending", func(t *testing.T) {
		host := &fleet.AppleHostReconcileInfo{UUID: "h1", Platform: "darwin", DeviceEnrolledAt: &recentlyEnrolled}
		rows := newRows("h1")
		resolver := func(hostUUID string) (string, error) { return "h1:user", nil }

		send, failed, toDelete, err := resolveUserChannelDeliveries(ctx, logger, []*fleet.AppleHostReconcileInfo{host}, []string{"h1"}, toSlice(rows), resolver)
		require.NoError(t, err)
		require.Equal(t, []string{"h1:user"}, send)
		require.Empty(t, failed)
		require.Empty(t, toDelete)
		require.NotNil(t, rows["h1"].Status)
		require.Equal(t, fleet.MDMDeliveryPending, *rows["h1"].Status)
	})

	t.Run("no user channel within grace -> hold (nil status), no send", func(t *testing.T) {
		host := &fleet.AppleHostReconcileInfo{UUID: "h2", Platform: "darwin", DeviceEnrolledAt: &recentlyEnrolled}
		rows := newRows("h2")
		resolver := func(hostUUID string) (string, error) { return "", nil }

		send, failed, toDelete, err := resolveUserChannelDeliveries(ctx, logger, []*fleet.AppleHostReconcileInfo{host}, []string{"h2"}, toSlice(rows), resolver)
		require.NoError(t, err)
		require.Empty(t, send)
		require.Empty(t, failed)
		require.Empty(t, toDelete)
		require.Nil(t, rows["h2"].Status, "held row should have a nil status so the next tick retries")
	})

	t.Run("no user channel past grace -> failed with detail", func(t *testing.T) {
		host := &fleet.AppleHostReconcileInfo{UUID: "h3", Platform: "darwin", DeviceEnrolledAt: &pastGrace}
		rows := newRows("h3")
		resolver := func(hostUUID string) (string, error) { return "", nil }

		send, failed, toDelete, err := resolveUserChannelDeliveries(ctx, logger, []*fleet.AppleHostReconcileInfo{host}, []string{"h3"}, toSlice(rows), resolver)
		require.NoError(t, err)
		require.Empty(t, send)
		require.Empty(t, toDelete)
		require.Len(t, failed, 1)
		require.Contains(t, failed[0].detail, "user channel doesn't exist")
		require.NotNil(t, rows["h3"].Status)
		require.Equal(t, fleet.MDMDeliveryFailed, *rows["h3"].Status)
	})

	t.Run("iOS/iPadOS -> failed with mobile-specific detail regardless of grace", func(t *testing.T) {
		for _, platform := range []string{"ios", "ipados"} {
			host := &fleet.AppleHostReconcileInfo{UUID: "h4", Platform: platform, DeviceEnrolledAt: &recentlyEnrolled}
			rows := newRows("h4")
			resolver := func(hostUUID string) (string, error) { return "", nil }

			send, failed, toDelete, err := resolveUserChannelDeliveries(ctx, logger, []*fleet.AppleHostReconcileInfo{host}, []string{"h4"}, toSlice(rows), resolver)
			require.NoError(t, err)
			require.Empty(t, send)
			require.Empty(t, toDelete)
			require.Len(t, failed, 1)
			require.Contains(t, failed[0].detail, "isn't available on iOS and iPadOS")
			require.Equal(t, fleet.MDMDeliveryFailed, *rows["h4"].Status)
		}
	})

	t.Run("no user channel -> user-scoped removes are returned for deletion, not left pending", func(t *testing.T) {
		host := &fleet.AppleHostReconcileInfo{UUID: "h5", Platform: "ios", DeviceEnrolledAt: &recentlyEnrolled}
		removeRow := &fleet.MDMAppleHostDeclaration{
			HostUUID:        "h5",
			DeclarationUUID: "d-remove",
			Scope:           fleet.PayloadScopeUser,
			OperationType:   fleet.MDMOperationTypeRemove,
			Status:          new(fleet.MDMDeliveryPending),
		}
		resolver := func(hostUUID string) (string, error) { return "", nil }

		send, failed, toDelete, err := resolveUserChannelDeliveries(ctx, logger, []*fleet.AppleHostReconcileInfo{host}, []string{"h5"},
			[]*fleet.MDMAppleHostDeclaration{removeRow}, resolver)
		require.NoError(t, err)
		require.Empty(t, send)
		require.Empty(t, failed)
		require.Equal(t, []*fleet.MDMAppleHostDeclaration{removeRow}, toDelete)
	})
}
