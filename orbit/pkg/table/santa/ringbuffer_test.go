//go:build darwin

package santa

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func mk(n int) logEntry {
	return logEntry{Timestamp: string(rune('A' + n))}
}

func tsSlice(entries []logEntry) []string {
	out := make([]string, len(entries))
	for i := range entries {
		out[i] = entries[i].Timestamp
	}
	return out
}

// TestRingBuffer_KeepsNewestInOrder verifies the core contract at every fill
// level: the buffer holds the last cap entries added, oldest first.
func TestRingBuffer_KeepsNewestInOrder(t *testing.T) {
	tests := []struct {
		name string
		cap  int
		adds int
		want []string
	}{
		{"empty", 2, 0, []string{}},
		{"below capacity", 3, 2, []string{"A", "B"}},
		{"at capacity", 2, 2, []string{"A", "B"}},
		{"wraps keeping the newest", 3, 6, []string{"D", "E", "F"}},
		{"zero capacity holds nothing", 0, 4, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := newRingBuffer(tt.cap)
			for i := range tt.adds {
				rb.Add(mk(i))
			}
			require.Equal(t, tt.want, tsSlice(rb.SliceChrono()))
			require.Equal(t, len(tt.want), rb.Len())
		})
	}
}

// TestRingBuffer_GrowsOnDemand verifies that the backing array is sized to the
// entries actually added, not to the cap: the santa tables allocate a buffer on
// every query and most hosts have far fewer events than the cap allows.
func TestRingBuffer_GrowsOnDemand(t *testing.T) {
	rb := newRingBuffer(10_000)
	require.Empty(t, rb.buf)

	for i := range 5 {
		rb.Add(mk(i))
	}
	require.Less(t, len(rb.buf), rb.cap)
	require.Equal(t, []string{"A", "B", "C", "D", "E"}, tsSlice(rb.SliceChrono()))

	// Filling past the cap keeps the last cap entries and grows no further.
	rb = newRingBuffer(3)
	for i := range 100 {
		rb.Add(mk(i))
	}
	require.Len(t, rb.buf, 3)
	require.Equal(t, 3, rb.Len())
}

func TestRingBuffer_Reset(t *testing.T) {
	rb := newRingBuffer(3)
	// Wrap the buffer so the reset has to clear a non-zero start offset.
	for i := range 5 {
		rb.Add(mk(i))
	}
	rb.Reset()
	require.Equal(t, 0, rb.Len())
	require.Empty(t, rb.SliceChrono())

	rb.Add(mk(0))
	require.Equal(t, []string{"A"}, tsSlice(rb.SliceChrono()))
}
