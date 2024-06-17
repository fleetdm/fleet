package webhooks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestTriggerVulnerabilitiesWebhook(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	mapper := Mapper{}

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

	recentVulns := []fleet.SoftwareVulnerability{
		{SoftwareID: 1, CVE: "CVE-2012-1234"},
		{SoftwareID: 2, CVE: "CVE-2012-1234"},
	}

	t.Run("disabled", func(t *testing.T) {
		appCfg := *appCfg
		appCfg.WebhookSettings.VulnerabilitiesWebhook.Enable = false
		args := VulnArgs{
			Vulnerablities: recentVulns,
			Meta:           nil,
			AppConfig:      &appCfg,
			Time:           time.Now(),
		}
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, args, &mapper)
		require.NoError(t, err)
	})

	t.Run("invalid server url", func(t *testing.T) {
		appCfg := *appCfg
		appCfg.ServerSettings.ServerURL = ":nope:"
		args := VulnArgs{
			Vulnerablities: recentVulns,
			Meta:           nil,
			AppConfig:      &appCfg,
			Time:           time.Now(),
		}
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, args, &mapper)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid server")
	})

	t.Run("empty recent vulns", func(t *testing.T) {
		args := VulnArgs{
			Vulnerablities: nil,
			Meta:           nil,
			AppConfig:      appCfg,
			Time:           time.Now(),
		}
		err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, args, &mapper)
		require.NoError(t, err)
	})

	t.Run("trigger requests", func(t *testing.T) {
		now := time.Now()

		hosts := []fleet.HostVulnerabilitySummary{
			{ID: 1, Hostname: "h1", DisplayName: "d1"},
			{ID: 2, Hostname: "h2", DisplayName: "d2"},
			{ID: 3, Hostname: "h3", DisplayName: "d3"},
			{ID: 4, Hostname: "h4", DisplayName: "d4"},
		}
		jsonH1 := fmt.Sprintf(`{"id":1,"hostname":"h1","display_name":"d1","url":"%s/hosts/1"}`, appCfg.ServerSettings.ServerURL)
		jsonH2 := fmt.Sprintf(`{"id":2,"hostname":"h2","display_name":"d2","url":"%s/hosts/2"}`, appCfg.ServerSettings.ServerURL)
		jsonH3 := fmt.Sprintf(`{"id":3,"hostname":"h3","display_name":"d3","url":"%s/hosts/3"}`, appCfg.ServerSettings.ServerURL)
		jsonH4 := fmt.Sprintf(`{"id":4,"hostname":"h4","display_name":"d4","url":"%s/hosts/4"}`, appCfg.ServerSettings.ServerURL)

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
			vulns []fleet.SoftwareVulnerability
			meta  map[string]fleet.CVEMeta
			hosts []fleet.HostVulnerabilitySummary
			want  string
		}{
			{
				"1 vuln, 1 host",
				[]fleet.SoftwareVulnerability{{CVE: cves[0], SoftwareID: 1}},
				nil,
				hosts[:1],
				fmt.Sprintf("%s[%s]}}", jsonCVE1, jsonH1),
			},
			{
				"1 vuln in multiple software, 1 host",
				[]fleet.SoftwareVulnerability{
					{CVE: cves[0], SoftwareID: 1},
					{CVE: cves[0], SoftwareID: 1},
					{CVE: cves[0], SoftwareID: 2},
				},
				nil,
				hosts[:1],
				fmt.Sprintf("%s[%s]}}", jsonCVE1, jsonH1),
			},
			{
				"1 vuln, 2 hosts",
				[]fleet.SoftwareVulnerability{
					{CVE: cves[0], SoftwareID: 1},
				},
				nil,
				hosts[:2],
				fmt.Sprintf("%s[%s,%s]}}", jsonCVE1, jsonH1, jsonH2),
			},
			{
				"1 vuln, 3 hosts",
				[]fleet.SoftwareVulnerability{
					{CVE: cves[0], SoftwareID: 1},
				},
				nil,
				hosts[:3],
				fmt.Sprintf("%s[%s,%s]}}\n%s[%s]}}", jsonCVE1, jsonH1, jsonH2, jsonCVE1, jsonH3), // 2 requests, batch of 2 max
			},
			{
				"1 vuln, 4 hosts",
				[]fleet.SoftwareVulnerability{
					{CVE: cves[0], SoftwareID: 1},
				},
				nil,
				hosts[:4],
				fmt.Sprintf("%s[%s,%s]}}\n%s[%s,%s]}}", jsonCVE1, jsonH1, jsonH2, jsonCVE1, jsonH3, jsonH4), // 2 requests, batch of 2 max
			},
			{
				"2 vulns, 1 host each",
				[]fleet.SoftwareVulnerability{
					{CVE: cves[0], SoftwareID: 1},
					{CVE: cves[1], SoftwareID: 2},
				},
				nil,
				hosts[:1],
				fmt.Sprintf("%s[%s]}}\n%s[%s]}}", jsonCVE1, jsonH1, jsonCVE2, jsonH1),
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var requests []string

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					requests = append(requests, string(b))
					_, err = w.Write(nil)
					assert.NoError(t, err)
				}))
				defer srv.Close()

				ds.HostVulnSummariesBySoftwareIDsFunc = func(ctx context.Context, softwareIDs []uint) ([]fleet.HostVulnerabilitySummary, error) {
					return c.hosts, nil
				}

				appCfg := *appCfg
				appCfg.WebhookSettings.VulnerabilitiesWebhook.DestinationURL = srv.URL
				args := VulnArgs{
					Vulnerablities: c.vulns,
					Meta:           c.meta,
					AppConfig:      &appCfg,
					Time:           now,
				}

				err := TriggerVulnerabilitiesWebhook(ctx, ds, logger, args, &mapper)
				require.NoError(t, err)

				assert.True(t, ds.HostVulnSummariesBySoftwareIDsFuncInvoked)
				ds.HostVulnSummariesBySoftwareIDsFuncInvoked = false

				want := strings.Split(c.want, "\n")
				assert.ElementsMatch(t, want, requests)
			})
		}
	})
}
