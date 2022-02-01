package webhooks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestTriggerVulnerabilitiesWebhook(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()

	appCfg := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			VulnerabilitiesWebhook: fleet.VulnerabilitiesWebhookSettings{
				Enable:        true,
				HostBatchSize: 2,
			},
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://fleet.example.com",
		},
	}

	recentVulns := map[string][]string{
		"CVE-2012-1234": {"cpe1", "cpe2"},
	}

	t.Run("disabled", func(t *testing.T) {
		appCfg := *appCfg
		appCfg.WebhookSettings.VulnerabilitiesWebhook.Enable = false
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, recentVulns, &appCfg, time.Now())
		require.NoError(t, err)
	})

	t.Run("invalid server url", func(t *testing.T) {
		appCfg := *appCfg
		appCfg.ServerSettings.ServerURL = ":nope:"
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, recentVulns, &appCfg, time.Now())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server")
	})

	t.Run("empty recent vulns", func(t *testing.T) {
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, nil, appCfg, time.Now())
		require.NoError(t, err)
	})

	t.Run("trigger requests", func(t *testing.T) {
		now := time.Now()

		hosts := []*fleet.Host{
			{ID: 1, Hostname: "h1"},
			{ID: 2, Hostname: "h2"},
			{ID: 3, Hostname: "h3"},
			{ID: 4, Hostname: "h4"},
		}
		jsonH1 := fmt.Sprintf(`{"id":1,"hostname":"h1","url":"%s/hosts/1"}`, appCfg.ServerSettings.ServerURL)
		jsonH2 := fmt.Sprintf(`{"id":2,"hostname":"h2","url":"%s/hosts/2"}`, appCfg.ServerSettings.ServerURL)
		jsonH3 := fmt.Sprintf(`{"id":3,"hostname":"h3","url":"%s/hosts/3"}`, appCfg.ServerSettings.ServerURL)
		jsonH4 := fmt.Sprintf(`{"id":4,"hostname":"h4","url":"%s/hosts/4"}`, appCfg.ServerSettings.ServerURL)

		cves := []string{
			"CVE-2012-1234",
			"CVE-2012-4567",
		}
		jsonCVE1 := fmt.Sprintf(`{"timestamp":"%s","vulnerability":{"cve":%q,"details_link":"https://nvd.nist.gov/vuln/detail/%[2]s","hosts_affected":`,
			now.Format(time.RFC3339Nano), cves[0])
		jsonCVE2 := fmt.Sprintf(`{"timestamp":"%s","vulnerability":{"cve":%q,"details_link":"https://nvd.nist.gov/vuln/detail/%[2]s","hosts_affected":`,
			now.Format(time.RFC3339Nano), cves[1])

		cases := []struct {
			name  string
			vulns map[string][]string
			hosts []*fleet.Host
			want  string
		}{
			{
				"1 vuln, 1 host",
				map[string][]string{cves[0]: {"cpe1"}},
				hosts[:1],
				fmt.Sprintf("\n%s[%s]}}", jsonCVE1, jsonH1),
			},
			{
				"1 vuln, 2 hosts",
				map[string][]string{cves[0]: {"cpe1"}},
				hosts[:2],
				fmt.Sprintf("\n%s[%s,%s]}}", jsonCVE1, jsonH1, jsonH2),
			},
			{
				"1 vuln, 3 hosts",
				map[string][]string{cves[0]: {"cpe1"}},
				hosts[:3],
				fmt.Sprintf("\n%s[%s,%s]}}\n%s[%s]}}", jsonCVE1, jsonH1, jsonH2, jsonCVE1, jsonH3), // 2 requests, batch of 2 max
			},
			{
				"1 vuln, 4 hosts",
				map[string][]string{cves[0]: {"cpe1"}},
				hosts[:4],
				fmt.Sprintf("\n%s[%s,%s]}}\n%s[%s,%s]}}", jsonCVE1, jsonH1, jsonH2, jsonCVE1, jsonH3, jsonH4), // 2 requests, batch of 2 max
			},
			{
				"2 vulns, 1 host each",
				map[string][]string{cves[0]: {"cpe1"}, cves[1]: {"cpe2"}},
				hosts[:1],
				fmt.Sprintf("\n%s[%s]}}\n%s[%s]}}", jsonCVE1, jsonH1, jsonCVE2, jsonH1),
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var buf bytes.Buffer

				// each request is a new line
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					buf.WriteByte('\n')
					_, err := io.Copy(&buf, r.Body)
					assert.NoError(t, err)
					w.Write(nil)
				}))
				defer srv.Close()

				ds.HostsByCPEsFunc = func(ctx context.Context, cpes []string) ([]*fleet.Host, error) {
					return c.hosts, nil
				}

				appCfg := *appCfg
				appCfg.WebhookSettings.VulnerabilitiesWebhook.DestinationURL = srv.URL
				err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, c.vulns, &appCfg, now)
				require.NoError(t, err)

				assert.True(t, ds.HostsByCPEsFuncInvoked)
				ds.HostsByCPEsFuncInvoked = false

				assert.Equal(t, c.want, buf.String())
			})
		}
	})
}
