package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDebugConnectionCommand(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.VerifyEnrollSecretFunc = func(secret string) (*fleet.EnrollSecret, error) {
		return nil, errors.New("invalid")
	}

	output := runAppForTest(t, []string{"debug", "connection"})
	// 3 successes: resolve host, dial address, check api endpoint
	require.Equal(t, 3, strings.Count(output, "Success:"))
}

func TestDebugConnectionChecks(t *testing.T) {
	const timeout = 100 * time.Millisecond

	t.Run("resolveHostname", func(t *testing.T) {
		// resolves host name
		err := resolveHostname(context.Background(), timeout, "localhost")
		require.NoError(t, err)

		// resolves ip4 address
		err = resolveHostname(context.Background(), timeout, "127.0.0.1")
		require.NoError(t, err)

		// resolves ip6 address
		err = resolveHostname(context.Background(), timeout, "::1")
		require.NoError(t, err)

		// fails on invalid host
		randBytes := make([]byte, 8)
		_, err = rand.Read(randBytes)
		require.NoError(t, err)
		noSuchHost := "no_such_host" + hex.EncodeToString(randBytes)

		err = resolveHostname(context.Background(), timeout, noSuchHost)
		require.Error(t, err)
	})

	t.Run("checkAPIEndpoint", func(t *testing.T) {
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
