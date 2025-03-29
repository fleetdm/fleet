package service

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJitterForHost(t *testing.T) {
	jh := newJitterHashTable(30)

	histogram := make(map[int64]int)
	hostCount := 3000
	for i := 0; i < hostCount; i++ {
		hostID, err := crand.Int(crand.Reader, big.NewInt(10000))
		require.NoError(t, err)
		jitter := jh.jitterForHost(uint(hostID.Int64() + 10000)) //nolint:gosec // dismiss G115
		jitterMinutes := int64(jitter.Minutes())
		histogram[jitterMinutes]++
	}
	minVal, maxVal := math.MaxInt, 0
	for jitterMinutes, count := range histogram {
		if count < minVal {
			minVal = count
		}
		if count > maxVal {
			maxVal = count
		}
		t.Logf("jitterMinutes=%d \t count=%d\n", jitterMinutes, count)
	}
	variation := maxVal - minVal
	t.Logf("min=%d \t max=%d \t variation=%d\n", minVal, maxVal, variation)

	// check that variation is below 1% of the total amount of hosts
	require.Less(t, variation, int(float32(hostCount)/0.01))
}

func TestNoJitter(t *testing.T) {
	jh := newJitterHashTable(0)

	hostCount := 3000
	for i := 0; i < hostCount; i++ {
		hostID, err := crand.Int(crand.Reader, big.NewInt(10000))
		require.NoError(t, err)
		jitter := jh.jitterForHost(uint(hostID.Int64() + 10000)) //nolint:gosec // dismiss G115
		jitterMinutes := int64(jitter.Minutes())
		require.Equal(t, int64(0), jitterMinutes)
	}
}
