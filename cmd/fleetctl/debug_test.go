package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

func TestDebugConnection(t *testing.T) {
	t.Run("resolveHostname", func(t *testing.T) {
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
	})

	t.Run("checkAPIEndpoint", func(t *testing.T) {
		const timeout = 100 * time.Millisecond

		cases := [...]struct {
			code        int // == 0 panics, negative value waits for timeout, sets status code to absolute value
			body        string
			errContains string // empty if checkAPIEndpoint should not return an error
		}{
			{401, `{"error": "fail", "node_invalid": true}`, ""},
			{-401, `{"error": "fail", "node_invalid": true}`, "deadline exceeded"},
			{200, `{"error": "", "node_invalid": false}`, "unexpected 200 response"},
			{0, `panic`, "EOF"},
		}
		var callCount int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res := cases[callCount]

			switch {
			case res.code == 0:
				panic(res.body)
			case res.code < 0:
				time.Sleep(timeout + time.Millisecond)
				res.code = -res.code
			}
			w.WriteHeader(res.code)
			fmt.Fprint(w, res.body)
		}))
		defer srv.Close()

		cli, err := service.NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		for i, c := range cases {
			callCount = i
			t.Run(fmt.Sprint(c.code), func(t *testing.T) {
				err := checkAPIEndpoint(context.Background(), timeout, cli)
				if c.errContains == "" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), c.errContains)
				}
			})
		}
	})
}
