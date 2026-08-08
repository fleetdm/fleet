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

func TestRingBuffer_Len(t *testing.T) {
	rb := newRingBuffer(3)
	require.Equal(t, 0, rb.Len())
	rb.Add(mk(0))
	require.Equal(t, 1, rb.Len())
	rb.Add(mk(1))
	require.Equal(t, 2, rb.Len())
	rb.Add(mk(2))
	require.Equal(t, 3, rb.Len())
	rb.Add(mk(3))
	require.Equal(t, 3, rb.Len())
	rb.Add(mk(4))
	require.Equal(t, 3, rb.Len())
}

func TestRingBuffer_NoWrap(t *testing.T) {
	rb := newRingBuffer(3)
	rb.Add(mk(0)) // A
	rb.Add(mk(1)) // B
	require.Equal(t, []string{"A", "B"}, tsSlice(rb.SliceChrono()))
}

func TestRingBuffer_Wrap(t *testing.T) {
	rb := newRingBuffer(3)
	// Add 6: A B C D E F → keep last 3: D E F
	for i := range 6 {
		rb.Add(mk(i))
	}
	require.Equal(t, []string{"D", "E", "F"}, tsSlice(rb.SliceChrono()))
}

func TestRingBuffer_ExactCapacity(t *testing.T) {
	rb := newRingBuffer(2)
	rb.Add(mk(0)) // A
	rb.Add(mk(1)) // B

	require.Equal(t, []string{"A", "B"}, tsSlice(rb.SliceChrono()))
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

func TestRingBuffer_ZeroCapacity(t *testing.T) {
	rb := newRingBuffer(0)
	rb.Add(mk(0))
	require.Equal(t, 0, rb.Len())
	require.Empty(t, rb.SliceChrono())
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := newRingBuffer(2)
	require.Empty(t, rb.SliceChrono())
}
