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

func TestRingBuffer_Empty(t *testing.T) {
	rb := newRingBuffer(2)
	require.Empty(t, rb.SliceChrono())
}
