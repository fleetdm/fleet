package msgraph

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cloudTransport func(*http.Request) (*http.Response, error)

func (f cloudTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCloudEndpoints(t *testing.T) {
	for _, tc := range []struct {
		cloud fleet.MicrosoftGraphCloud
		login string
		graph string
	}{
		{"", "https://login.microsoftonline.com", "https://graph.microsoft.com"},
		{fleet.MicrosoftGraphCloudGlobal, "https://login.microsoftonline.com", "https://graph.microsoft.com"},
		{fleet.MicrosoftGraphCloudGCCHigh, "https://login.microsoftonline.us", "https://graph.microsoft.us"},
		{fleet.MicrosoftGraphCloudDoD, "https://login.microsoftonline.us", "https://dod-graph.microsoft.us"},
		{fleet.MicrosoftGraphCloudChina, "https://login.chinacloudapi.cn", "https://microsoftgraph.chinacloudapi.cn"},
	} {
		t.Run(string(tc.cloud), func(t *testing.T) {
			cred := &fleet.MicrosoftGraphCredential{
				MicrosoftGraphCredentialMetadata: fleet.MicrosoftGraphCredentialMetadata{
					TenantID: testTenantID, ClientID: testClientID, Cloud: tc.cloud,
				},
				ClientSecret: testSecret,
			}
			graphClient, err := NewClient(cred)
			require.NoError(t, err)
			c := graphClient.(*client)
			assert.Equal(t, tc.login+"/"+testTenantID+"/oauth2/v2.0/token", c.cfg.TokenURL)
			assert.Equal(t, []string{tc.graph + "/.default"}, c.cfg.Scopes)

			crossCloud := "https://graph.microsoft.com"
			if crossCloud == tc.graph {
				crossCloud = "https://graph.microsoft.us"
			}
			rejectNext := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/oauth2/v2.0/token") {
					assert.Equal(t, tc.login, r.Header.Get("X-Test-Origin"))
					assert.NoError(t, r.ParseForm())
					assert.Equal(t, tc.graph+"/.default", r.Form.Get("scope"))
					assert.Equal(t, testSecret, r.Form.Get("client_secret"))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3599}`))
					return
				}
				assert.Equal(t, tc.graph, r.Header.Get("X-Test-Origin"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				assert.Equal(t, autopilotDevicesPath, r.URL.Path)
				if r.URL.Query().Get("$top") == "1" {
					writeDevices(t, w, "", device("1", "serial-1", "tag"))
				} else if r.URL.Query().Get("$skiptoken") != "" {
					writeDevices(t, w, "", device("2", "serial-2", "tag"))
				} else {
					nextHost := tc.graph
					if rejectNext {
						nextHost = crossCloud
					}
					writeDevices(t, w, nextHost+autopilotDevicesPath+"?$skiptoken=page2", device("1", "serial-1", "tag"))
				}
			}))
			defer srv.Close()
			localURL, err := url.Parse(srv.URL)
			require.NoError(t, err)
			c.baseClient.Transport = cloudTransport(func(r *http.Request) (*http.Response, error) {
				origin := r.URL.Scheme + "://" + r.URL.Host
				if origin != tc.login && origin != tc.graph {
					t.Errorf("request escaped selected cloud: %s", origin)
					return nil, fmt.Errorf("unexpected origin %s", origin)
				}
				localRequest := r.Clone(r.Context())
				u := *r.URL
				u.Scheme, u.Host = localURL.Scheme, localURL.Host
				localRequest.URL = &u
				localRequest.Header.Set("X-Test-Origin", origin)
				return http.DefaultTransport.RoundTrip(localRequest)
			})

			require.NoError(t, c.VerifyCredential(context.Background()))
			devices, err := c.ListWindowsAutopilotDevices(context.Background())
			require.NoError(t, err)
			require.Len(t, devices, 2)
			assert.Equal(t, "2", devices[1].ID)

			rejectNext = true
			devices, err = c.ListWindowsAutopilotDevices(context.Background())
			require.ErrorContains(t, err, "unexpected origin")
			assert.Nil(t, devices)
		})
	}
}

func TestRejectUnknownCloud(t *testing.T) {
	c, err := NewClient(&fleet.MicrosoftGraphCredential{
		MicrosoftGraphCredentialMetadata: fleet.MicrosoftGraphCredentialMetadata{
			TenantID: testTenantID, ClientID: testClientID, Cloud: "https://example.com",
		},
		ClientSecret: testSecret,
	})
	require.ErrorContains(t, err, "unsupported microsoft graph cloud")
	assert.Nil(t, c)
}
