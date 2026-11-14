package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveQueryRestPeriod(t *testing.T) {
	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "")
	d, err := LiveQueryRestPeriod()
	require.NoError(t, err)
	assert.Equal(t, 25*time.Second, d)

	t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "40s")
	d, err = LiveQueryRestPeriod()
	require.NoError(t, err)
	assert.Equal(t, 40*time.Second, d)

	for _, invalid := range []string{"bogus", "0s", "-45s"} {
		t.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", invalid)
		d, err = LiveQueryRestPeriod()
		require.Error(t, err, invalid)
		assert.Equal(t, 25*time.Second, d, invalid)
	}
}
