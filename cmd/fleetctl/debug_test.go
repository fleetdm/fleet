package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDebugResolveHostname(t *testing.T) {
	localIP4 := net.IPv4(127, 0, 0, 1)
	timeout := 100 * time.Millisecond

	// resolves host name
	ips, err := resolveHostname(context.Background(), timeout, "localhost")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(ips), 1)
	require.Contains(t, ips, localIP4)

	// resolves ip4 address
	ips, err = resolveHostname(context.Background(), timeout, "127.0.0.1")
	require.NoError(t, err)
	require.Len(t, ips, 1)
	require.Equal(t, localIP4, ips[0])

	// resolves ip6 address
	ips, err = resolveHostname(context.Background(), timeout, "::1")
	require.NoError(t, err)
	require.Len(t, ips, 1)
	require.Equal(t, net.IPv6loopback, ips[0])

	// fails on invalid host
	randBytes := make([]byte, 8)
	_, err = rand.Read(randBytes)
	require.NoError(t, err)
	noSuchHost := "no_such_host" + hex.EncodeToString(randBytes)

	_, err = resolveHostname(context.Background(), timeout, noSuchHost)
	require.Error(t, err)
}
