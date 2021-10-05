package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreview(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	os.Setenv("FLEET_SERVER_ADDRESS", "https://localhost:8412")
	testOverridePreviewDirectory = t.TempDir()

	t.Cleanup(func() {
		require.Equal(t, "", runAppForTest(t, []string{"preview", "stop"}))
	})

	require.Equal(t, "", runAppForTest(t, []string{"preview"}))
}
