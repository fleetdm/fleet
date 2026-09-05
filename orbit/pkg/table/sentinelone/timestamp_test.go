package sentinelone

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToUnixEpoch(t *testing.T) {
	t.Parallel()

	// All of these express the same local timestamp; the separator and padding
	// variants are what defeat the strict layouts and exercise the fallback.
	sameInstant := []string{
		"6/1/26, 10:09:51 AM",
		"6/1/26 10:09:51 AM",
		"06/01/2026, 10:09:51 AM",
		"6/1/26  10:09:51 AM",
		"6/1/26,\t10:09:51 AM",
		"6/1/26, 10:09:51\u00a0AM",
		"6/1/26 10:09:51 am",
	}
	want := localEpoch(t, "6/1/26, 10:09:51 AM")
	for _, in := range sameInstant {
		got, ok := toUnixEpoch(in)
		assert.True(t, ok, "toUnixEpoch(%q) should parse", in)
		assert.Equal(t, want, got, "toUnixEpoch(%q)", in)
	}
}

func TestToUnixEpochLayouts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want time.Time
	}{
		{name: "iso without zone", in: "2026-06-01 13:14:28", want: time.Date(2026, 6, 1, 13, 14, 28, 0, time.Local)},
		{name: "rfc3339 utc", in: "2026-06-01T13:14:28Z", want: time.Date(2026, 6, 1, 13, 14, 28, 0, time.UTC)},
		{name: "four digit year", in: "6/1/2026, 1:14:28 PM", want: time.Date(2026, 6, 1, 13, 14, 28, 0, time.Local)},
		{name: "pm without seconds", in: "6/1/26, 1:46 PM", want: time.Date(2026, 6, 1, 13, 46, 0, 0, time.Local)},
		{name: "midnight is 12 am", in: "6/1/26, 12:30:00 AM", want: time.Date(2026, 6, 1, 0, 30, 0, 0, time.Local)},
		{name: "noon is 12 pm", in: "6/1/26, 12:30:00 PM", want: time.Date(2026, 6, 1, 12, 30, 0, 0, time.Local)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := toUnixEpoch(tt.in)
			require.True(t, ok, "toUnixEpoch(%q) should parse", tt.in)
			assert.Equal(t, strconv.FormatInt(tt.want.Unix(), 10), got)
		})
	}
}

func TestToUnixEpochUnparseable(t *testing.T) {
	t.Parallel()

	// A day past the 12th in D/M/Y order is rejected rather than transposed
	// into a wrong month, and a date that doesn't exist is rejected rather than
	// rolled forward into the next month.
	unparseable := []string{
		"",
		"   ",
		"never",
		"n/a",
		"25/6/26, 10:09:51 AM",
		"6/1/26",
		"13/13/26, 10:09:51 AM",
		"2/31/26, 10:09:51 AM",
		"4/31/26, 1:00:00 PM",
		"2/30/26 1:00 PM",
	}
	for _, in := range unparseable {
		got, ok := toUnixEpoch(in)
		assert.False(t, ok, "toUnixEpoch(%q) should not parse", in)
		assert.Empty(t, got)
	}
}
