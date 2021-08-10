package insecure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	proxy, err := NewTLSProxy("localhost")
	require.NoError(t, err)
	assert.NotZero(t, proxy.Port)
}
