package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestListAllAndroidDeviceNames(t *testing.T) {
	const enterpriseName = "enterprises/LC123"

	t.Run("aggregates device names across pages", func(t *testing.T) {
		client := &mock.Client{}
		var calls int
		client.EnterprisesDevicesListPartialFunc = func(_ context.Context, gotEnterprise, pageToken string) (*androidmanagement.ListDevicesResponse, error) {
			assert.Equal(t, enterpriseName, gotEnterprise)
			calls++
			switch pageToken {
			case "":
				return &androidmanagement.ListDevicesResponse{
					Devices:       []*androidmanagement.Device{{Name: "a"}, {Name: "b"}},
					NextPageToken: "page2",
				}, nil
			case "page2":
				return &androidmanagement.ListDevicesResponse{
					Devices:       []*androidmanagement.Device{{Name: "c"}},
					NextPageToken: "",
				}, nil
			default:
				t.Fatalf("unexpected page token %q", pageToken)
				return nil, nil
			}
		}

		names, err := listAllAndroidDeviceNames(t.Context(), client, testLogger(), enterpriseName)
		require.NoError(t, err)
		assert.Equal(t, 2, calls)
		assert.Equal(t, map[string]struct{}{"a": {}, "b": {}, "c": {}}, names)
	})

	t.Run("propagates AMAPI errors", func(t *testing.T) {
		client := &mock.Client{}
		client.EnterprisesDevicesListPartialFunc = func(context.Context, string, string) (*androidmanagement.ListDevicesResponse, error) {
			return nil, assert.AnError
		}

		names, err := listAllAndroidDeviceNames(t.Context(), client, testLogger(), enterpriseName)
		require.Error(t, err)
		assert.Nil(t, names)
	})

	t.Run("aborts on cycling page token instead of looping forever", func(t *testing.T) {
		client := &mock.Client{}
		var calls int
		// Always return a non-empty NextPageToken to simulate a malformed/cycling response.
		client.EnterprisesDevicesListPartialFunc = func(_ context.Context, _, _ string) (*androidmanagement.ListDevicesResponse, error) {
			calls++
			return &androidmanagement.ListDevicesResponse{
				Devices:       []*androidmanagement.Device{{Name: fmt.Sprintf("dev-%d", calls)}},
				NextPageToken: "never-ends",
			}, nil
		}

		names, err := listAllAndroidDeviceNames(t.Context(), client, testLogger(), enterpriseName)
		require.Error(t, err)
		assert.Nil(t, names)
		assert.Contains(t, err.Error(), "exceeded max pages")
		// The loop is bounded: it stops after exactly androidReconcileMaxPages calls.
		assert.Equal(t, androidReconcileMaxPages, calls)
	})
}
