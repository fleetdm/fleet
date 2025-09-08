package log

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	var o options

	// Ensure a zero start.
	require.Zero(t, o)

	// Set the 'WithCaller' option via the method, ensure it's set.
	o.SetWithCaller()
	require.True(t, o&OptWithCaller == OptWithCaller)

	// Set the 'WithLevel' option via the method, ensure both it and 'WithCaller'
	// are set.
	o.SetWithLevel()
	require.True(t, o&OptWithCaller == OptWithCaller)
	require.True(t, o&OptWithLevel == OptWithLevel)

	// Unset the first ('WithCaller') option, and ensure both it is unset and
	// 'WithLevel' remains set.
	o.UnsetWithCaller()
	require.False(t, o&OptWithCaller == OptWithCaller)
	require.True(t, o&OptWithLevel == OptWithLevel)

	// Unset the second ('WithLevel') option, and ensure the return to zero.
	o.UnsetWithLevel()
	require.Zero(t, o)
}
