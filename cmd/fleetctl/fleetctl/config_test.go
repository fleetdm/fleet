package fleetctl

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigCommand(t *testing.T) {
	t.Run("invalid config set", func(t *testing.T) {
		cases := []struct {
			desc     string
			setFlags []string
			wantErr  string
		}{
			{
				desc:     "invalid flag",
				setFlags: []string{"--nosuchoption", "xyz"},
				wantErr:  "flag provided but not defined: -nosuchoption",
			},
			{
				desc:     "invalid tls-skip-verify",
				setFlags: []string{"--tls-skip-verify=xyz"},
				wantErr:  `invalid boolean value "xyz" for -tls-skip-verify: parse error`,
			},
		}

		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				dir := t.TempDir()
				configFile := filepath.Join(dir, "config")

				baseFlags := []string{"config", "set", "--config", configFile}
				RunAppCheckErr(t, append(baseFlags, c.setFlags...), c.wantErr)
			})
		}
	})

	t.Run("valid config set", func(t *testing.T) {
		cases := []struct {
			desc     string
			context  string
			setFlags []string
			want     Context
		}{
			{
				desc:     "set email",
				context:  "",
				setFlags: []string{"--email", "a@b.c"},
				want:     Context{Email: "a@b.c"},
			},
			{
				desc:     "set address",
				context:  "",
				setFlags: []string{"--address", "http://localhost"},
				want:     Context{Address: "http://localhost"},
			},
			{
				desc:     "set token",
				context:  "",
				setFlags: []string{"--token", "abc"},
				want:     Context{Token: "abc"},
			},
			{
				desc:     "set tls-skip-verify",
				context:  "",
				setFlags: []string{"--tls-skip-verify"},
				want:     Context{TLSSkipVerify: true},
			},
			{
				desc:     "set rootca",
				context:  "",
				setFlags: []string{"--rootca", "./rootca"},
				want:     Context{RootCA: "./rootca"},
			},
			{
				desc:     "set url-prefix",
				context:  "",
				setFlags: []string{"--url-prefix", "/test"},
				want:     Context{URLPrefix: "/test"},
			},
			{
				desc:     "set custom-headers",
				context:  "",
				setFlags: []string{"--custom-header", "X-Test:1"},
				want:     Context{CustomHeaders: map[string]string{"X-Test": "1"}},
			},
			{
				desc:     "set custom-headers no value",
				context:  "",
				setFlags: []string{"--custom-header", "X-Test"},
				want:     Context{CustomHeaders: map[string]string{"X-Test": ""}},
			},
			{
				desc:     "set custom-headers multiple separators",
				context:  "",
				setFlags: []string{"--custom-header", "X-Test:1:2:3"},
				want:     Context{CustomHeaders: map[string]string{"X-Test": "1:2:3"}},
			},
			{
				desc:     "set multiple custom-headers",
				context:  "",
				setFlags: []string{"--custom-header", "X-Test:1", "--custom-header", "X-Test2:2"},
				want:     Context{CustomHeaders: map[string]string{"X-Test": "1", "X-Test2": "2"}},
			},
			{
				desc:     "set different options in distinct context",
				context:  "test",
				setFlags: []string{"--email", "b@c.d", "--address", "http://localhost", "--custom-header", "X-Test:1", "--custom-header", "X-Test2:2"},
				want: Context{
					Email:         "b@c.d",
					Address:       "http://localhost",
					CustomHeaders: map[string]string{"X-Test": "1", "X-Test2": "2"},
				},
			},
		}

		for _, c := range cases {
			t.Run(c.desc, func(t *testing.T) {
				dir := t.TempDir()
				configFile := filepath.Join(dir, "config")

				baseFlags := []string{"config", "set", "--config", configFile}
				if c.context != "" {
					baseFlags = append(baseFlags, "--context", c.context)
				} else {
					c.context = "default"
				}
				RunAppForTest(t, append(baseFlags, c.setFlags...))

				cfg, err := readConfig(configFile)
				require.NoError(t, err)
				cfgCtx, ok := cfg.Contexts[c.context]
				require.True(t, ok)
				require.Equal(t, c.want, cfgCtx)
			})
		}
	})
}

func TestCustomHeadersConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config")

	// start a server that will receive requests
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		require.Equal(t, "custom", r.Header.Get("X-Fleet-Test"))
		require.Equal(t, "another", r.Header.Get("X-Fleet-MoreTest"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("FLEET_SERVER_ADDRESS", srv.URL)

	RunAppForTest(t, []string{
		"config", "set",
		"--config", configFile,
		"--token", "abcd",
		"--custom-header", "X-Fleet-Test:custom",
		"--custom-header", "X-Fleet-MoreTest:another",
		"--address", srv.URL,
	})
	RunAppNoChecks([]string{"get", "packs", "--config", configFile}) //nolint:errcheck
	require.True(t, called)
}
