package service

// Software and vulnerability tests for the core (no-license) suite.
//
// Belongs here: software titles and versions listing and details, vulnerable
// software, the vulnerabilities listing endpoints, and osquery software ingest
// including how long or invalid fields are handled.
//
// Does not belong here: the software shown on a single host's detail
// (integration_core_hosts_test.go, integration_core_hosts_reports_test.go).

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	logtestutils "github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestVulnerableSoftware() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OSVersion:       "Mac OS X 10.14.6",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", ExtensionID: "abc", ExtensionFor: "edge"},
		{Name: "bar", Version: "0.0.3", Source: "apps", ExtensionID: "xyz", ExtensionFor: "chrome"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	var hostResponse getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResponse)

	assertSoftware := func(t *testing.T, software []fleet.HostSoftwareEntry, contains *fleet.Software) {
		t.Helper()
		var found bool
		for _, s := range software {
			if s.Name == contains.Name {
				found = true
				assert.Equal(t, s.Name, contains.Name)
				assert.Equal(t, s.Version, contains.Version)
				assert.Equal(t, s.Source, contains.Source)
				assert.Equal(t, s.ExtensionID, contains.ExtensionID)
				assert.Equal(t, s.ExtensionFor, contains.ExtensionFor)
				assert.Equal(t, s.GenerateCPE, contains.GenerateCPE)
				assert.Len(t, contains.Vulnerabilities, len(s.Vulnerabilities))
				for i, vuln := range s.Vulnerabilities {
					assert.Equal(t, vuln.CVE, contains.Vulnerabilities[i].CVE)
					assert.Equal(t, vuln.DetailsLink, contains.Vulnerabilities[i].DetailsLink)
				}
			}
		}
		if !found {
			t.Fatalf("software not found")
		}
	}

	expectedSoft2 := &fleet.Software{
		Name:         "bar",
		Version:      "0.0.3",
		Source:       "apps",
		ExtensionID:  "xyz",
		ExtensionFor: "chrome",
		GenerateCPE:  "somecpe",
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:         "cve-123-123-132",
				DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-123-123-132",
			},
		},
	}

	expectedSoft1 := &fleet.Software{
		Name:            "foo",
		Version:         "0.0.1",
		Source:          "chrome_extensions",
		ExtensionID:     "abc",
		ExtensionFor:    "edge",
		GenerateCPE:     "",
		Vulnerabilities: nil,
	}

	assertSoftware(t, hostResponse.Host.Software, expectedSoft1)
	assertSoftware(t, hostResponse.Host.Software, expectedSoft2)

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	resp := s.Do("GET", "/api/latest/fleet/software", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &lsResp))
	require.Empty(t, lsResp.Software)
	assert.Nil(t, lsResp.CountsUpdatedAt)

	var versionsResp listSoftwareVersionsResponse
	resp = s.Do("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &versionsResp))
	require.Empty(t, versionsResp.Software)
	require.Equal(t, 0, versionsResp.Count)
	assert.Nil(t, versionsResp.CountsUpdatedAt)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	countReq := countSoftwareRequest{}
	countResp := countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp)
	assert.Equal(t, 3, countResp.Count)

	// the software/count endpoint is different, it doesn't care about hosts counts
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// now the list software endpoint returns the software
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Equal(t, soft1.ExtensionID, lsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.ExtensionFor, lsResp.Software[0].ExtensionFor)
	// Browser field should be populated for browser extensions
	if soft1.Source == "chrome_extensions" || soft1.Source == "firefox_addons" || soft1.Source == "ie_extensions" || soft1.Source == "safari_extensions" {
		assert.Equal(t, soft1.ExtensionFor, lsResp.Software[0].Browser)
	} else {
		assert.Empty(t, lsResp.Software[0].Browser)
	}
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.Equal(t, 1, versionsResp.Count)
	assert.Equal(t, soft1.ID, versionsResp.Software[0].ID)
	assert.Equal(t, soft1.ExtensionID, versionsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.ExtensionFor, versionsResp.Software[0].ExtensionFor)
	// Browser field should be populated for browser extensions
	if soft1.Source == "chrome_extensions" || soft1.Source == "firefox_addons" || soft1.Source == "ie_extensions" || soft1.Source == "safari_extensions" {
		assert.Equal(t, soft1.ExtensionFor, versionsResp.Software[0].Browser)
	} else {
		assert.Empty(t, versionsResp.Software[0].Browser)
	}
	assert.Len(t, versionsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// the count endpoint still returns 1
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// an unsupported order_key returns 422
	s.Do("GET", "/api/latest/fleet/software", nil, http.StatusUnprocessableEntity, "order_key", "vendor_old")
	s.Do("GET", "/api/latest/fleet/software/versions", nil, http.StatusUnprocessableEntity, "order_key", "vendor_old")

	// default sort, not only vulnerable
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp)
	require.GreaterOrEqual(t, len(lsResp.Software), len(software))
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp)
	require.GreaterOrEqual(t, len(versionsResp.Software), len(software))
	require.GreaterOrEqual(t, versionsResp.Count, len(software))
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request with a per_page limit (see #4058)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 2)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 2)
	require.GreaterOrEqual(t, versionsResp.Count, 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request next page, with per_page limit
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.GreaterOrEqual(t, versionsResp.Count, 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request one past the last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Empty(t, lsResp.Software)
	require.Nil(t, lsResp.CountsUpdatedAt)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Empty(t, versionsResp.Software)
	require.GreaterOrEqual(t, versionsResp.Count, 2)
	require.Nil(t, versionsResp.CountsUpdatedAt) // CONFIRM: legacy counts updated at is calculated by the server based on the software entries in the paginated response so how should we handle now?

	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusBadRequest, &lsResp, "per_page", "2", "page", "-10")
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
	s.DoJSON("GET", "/api/latest/fleet/software/count", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
}

func (s *integrationTestSuite) TestListSoftwareAndSoftwareDetails() {
	t := s.T()

	// create a few hosts specific to this test
	hosts := make([]*fleet.Host, 20)
	for i := range hosts {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(t.Name() + strconv.Itoa(i)),
			OsqueryHostID:   new(t.Name() + strconv.Itoa(i)),
			UUID:            t.Name() + strconv.Itoa(i),
			Hostname:        t.Name() + "foo" + strconv.Itoa(i) + ".local",
			PrimaryIP:       "192.168.1." + strconv.Itoa(i),
			PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-%02d", i),
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		hosts[i] = host
	}

	// create a bunch of software
	sws := make([]fleet.Software, 20)
	for i := range sws {
		sw := fleet.Software{Name: fmt.Sprintf("sw%02d", i), Version: fmt.Sprintf("0.0.%02d", i), Source: "apps"}
		if i%2 == 0 {
			sw.Source = "chrome_extensions"
			sw.ExtensionFor = "chrome"
		}
		sws[i] = sw
	}

	sortByNameAlphanumeric := func(sw []fleet.Software, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
	}
	sortEntryByNameAlphanumeric := func(sw []fleet.HostSoftwareEntry, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
	}

	// mark them as installed on the hosts, with host at index 0 having all 20,
	// at index 1 having 19, index 2 = 18, etc. until index 19 = 1. So software
	// sws[0] is only used by 1 host, while sws[19] is used by all.
	for i, h := range hosts {
		_, err := s.ds.UpdateHostSoftware(context.Background(), h.ID, sws[i:])
		require.NoError(t, err)
		require.NoError(t, s.ds.LoadHostSoftware(context.Background(), h, false))

		if i == 0 {
			// this host has all software, refresh the list so we have the software.ID filled
			sws = make([]fleet.Software, 0, len(h.Software))
			for _, s := range h.Software {
				sws = append(sws, s.Software)
			}
			// Sort software by Name (alphanumeric)
			sort.Slice(
				sws, func(a, b int) bool {
					return sortByNameAlphanumeric(sws, a, b)
				},
			)
		}
	}

	var cpes []fleet.SoftwareCPE
	for i, sw := range sws {
		cpes = append(cpes, fleet.SoftwareCPE{SoftwareID: sw.ID, CPE: "somecpe" + strconv.Itoa(i)})
	}

	_, err := s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software to load GeneratedCPEID
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), hosts[0], false))

	// add CVEs for the first 10 software, which are the least used (lower hosts_count)
	// Sort software by Name (alphanumeric)
	sort.Slice(
		hosts[0].Software, func(a, b int) bool {
			return sortEntryByNameAlphanumeric(hosts[0].Software, a, b)
		},
	)
	testCvePrefix := "cve-123-123"
	for i, sw := range hosts[0].Software[:10] {
		inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: sw.ID,
			CVE:        fmt.Sprintf(testCvePrefix+"-%03d", i),
		}, fleet.NVDSource)
		require.NoError(t, err)
		require.True(t, inserted)
	}
	expectedVulnVersionsCount := 10

	// create a team and make the last 3 hosts part of it (meaning 3 that use
	// sws[19], 2 for sws[18], and 1 for sws[17])
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: t.Name(),
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[19].ID, hosts[18].ID, hosts[17].ID})))
	expectedTeamVersionsCount := 3

	assertSoftwareDetails := func(expectedSoftware []fleet.Software, team string) {
		t.Helper()
		// this is just a basic sanity check of the software details endpoints and doesn't test all of the
		// fields that may be present in the response (e.g., vulnerabilities)
		for _, sw := range expectedSoftware {
			var detailsResp getSoftwareResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.ExtensionFor, detailsResp.Software.ExtensionFor)
			// Browser field should be populated for browser extensions
			if sw.Source == "chrome_extensions" || sw.Source == "firefox_addons" || sw.Source == "ie_extensions" || sw.Source == "safari_extensions" {
				assert.Equal(t, sw.ExtensionFor, detailsResp.Software.Browser)
			} else {
				assert.Empty(t, detailsResp.Software.Browser)
			}

			detailsResp = getSoftwareResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.ExtensionFor, detailsResp.Software.ExtensionFor)
			// Browser field should be populated for browser extensions
			if sw.Source == "chrome_extensions" || sw.Source == "firefox_addons" || sw.Source == "ie_extensions" || sw.Source == "safari_extensions" {
				assert.Equal(t, sw.ExtensionFor, detailsResp.Software.Browser)
			} else {
				assert.Empty(t, detailsResp.Software.Browser)
			}
			if len(sw.Vulnerabilities) > 0 {
				assert.Len(t, detailsResp.Software.Vulnerabilities, len(sw.Vulnerabilities))
				assert.Greater(t, detailsResp.Software.Vulnerabilities[0].CreatedAt, time.Now().Add(-time.Hour)) // asserting a non-zero time
			}
		}
	}

	assertResp := func(resp listSoftwareResponse, want []fleet.Software, ts time.Time, team string, counts ...int) {
		t.Helper()
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID, "want.Name: %s got.Name: %s", want[i].Name, resp.Software[i].Name)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantExtensionFor, gotExtensionFor := want[i].ExtensionFor, resp.Software[i].ExtensionFor
			assert.Equal(t, wantExtensionFor, gotExtensionFor)
			// Browser field should be populated for browser extensions
			if want[i].Source == "chrome_extensions" || want[i].Source == "firefox_addons" || want[i].Source == "ie_extensions" || want[i].Source == "safari_extensions" {
				assert.Equal(t, want[i].ExtensionFor, resp.Software[i].Browser)
			} else {
				assert.Empty(t, resp.Software[i].Browser)
			}
			wantCount, gotCount := counts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	assertVersionsResp := func(
		resp listSoftwareVersionsResponse, want []fleet.Software, ts time.Time, team string, swCount int, hostCounts ...int,
	) {
		require.Equal(t, swCount, resp.Count)
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID)
			wantCount, gotCount := hostCounts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantExtensionFor, gotExtensionFor := want[i].ExtensionFor, resp.Software[i].ExtensionFor
			assert.Equal(t, wantExtensionFor, gotExtensionFor)
			// Browser field should be populated for browser extensions
			if want[i].Source == "chrome_extensions" || want[i].Source == "firefox_addons" || want[i].Source == "ie_extensions" || want[i].Source == "safari_extensions" {
				assert.Equal(t, want[i].ExtensionFor, resp.Software[i].Browser)
			} else {
				assert.Empty(t, resp.Software[i].Browser)
			}
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	var versResp listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", 0)

	// same with a team filter
	teamStr := fmt.Sprintf("%d", tm.ID)
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc", "team_id",
		teamStr,
	)
	assertResp(lsResp, nil, time.Time{}, teamStr)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc",
		"team_id", teamStr,
	)
	assertVersionsResp(versResp, nil, time.Time{}, teamStr, 0)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software, get the first page without vulns
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", 20, 19, 18, 17, 16)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", len(sws), 20, 19, 18, 17, 16,
	)
	require.False(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// second page (page=1)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", 15, 14, 13, 12, 11)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", len(sws), 15, 14, 13, 12, 11,
	)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// third page (page=2)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", len(sws), 10, 9, 8, 7, 6)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// last page (page=3)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", len(sws), 5, 4, 3, 2, 1)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.False(t, versResp.Meta.HasNextResults)

	// past the end
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", len(sws))

	// no explicit sort order, defaults to hosts_count DESC
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 20, 19)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0")
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", len(sws), 20, 19)

	// hosts_count ascending
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertResp(lsResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", 1, 2, 3)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertVersionsResp(versResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", len(sws), 1, 2, 3)

	// vulnerable software only
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", expectedVulnVersionsCount, 10, 9, 8, 7, 6,
	)

	// vulnerable software only, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", expectedVulnVersionsCount, 5, 4, 3, 2, 1,
	)

	// vulnerable software only, past last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", expectedVulnVersionsCount)

	// /software/versions  filtered by name, version, cve (`/software` is deprecated)
	versionsResp := listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Name)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", " "+sws[0].Name+"\n")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Version)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "\n"+sws[0].Version+"  ")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	// All 10 CVEs added to the first 10 software have the same cvePrefix, so should return all
	// 10 vulnerable software versions
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", testCvePrefix)
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "  "+testCvePrefix+"\n")
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)

	// filter by the team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, 3, 2)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 3, 2)

	// filter by the team, 2 by page, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "1", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 1)

	// filter by no team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "name",
		"order_direction", "desc", "team_id", "0",
	)
	fmt.Printf("lsResp: %+v\n", lsResp)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 17, 17)

	// Invalid software team -- admin gets a 404, team users get a 403
	detailsResp := getSoftwareResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", versResp.Software[0].ID), nil, http.StatusNotFound, &detailsResp,
		"team_id", "999999",
	)

	// a request with without_vulnerability_details set to false does not return extra details
	respVersions := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"without_vulnerability_details", "false",
	)
	for _, s := range respVersions.Software {
		for _, cve := range s.Vulnerabilities {
			require.Nil(t, cve.CVSSScore)
			require.Nil(t, cve.EPSSProbability)
			require.Nil(t, cve.CISAKnownExploit)
			require.Nil(t, cve.CVEPublished)
			require.Nil(t, cve.Description)
			require.Nil(t, cve.ResolvedInVersion)
		}
	}
}

func (s *integrationTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Order Key is an invalid order key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "cvss_score", "order_direction", "asc")

	// Exploit is an EE only filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusPaymentRequired, &resp, "exploit", "true")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	s.Require().Empty(resp.Vulnerabilities)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo1.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
		Name:     "Windows 11 Enterprise 22H2",
		Version:  "10.0.19042.1234",
		Platform: "windows",
	})
	require.NoError(t, err)
	allos, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	var os fleet.OperatingSystem
	for _, o := range allos {
		if o.ID > os.ID {
			os = o
		}
	}

	err = s.ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID:              os.ID,
		CVE:               "CVE-2021-12345",
		ResolvedInVersion: new("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "Google Chrome", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), []fleet.SoftwareCPE{
		{
			SoftwareID: sw.ID,
			CPE:        "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*",
		},
	})
	require.NoError(t, err)

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)

	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	res2, err := s.ds.UpdateHostSoftware(context.Background(), host2.ID, []fleet.Software{
		{Name: "Firefox", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw2 := res2.Inserted[0]

	// insert software vuln outside of host scope
	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw2.ID,
		CVE:        "CVE-2021-1246",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	knownCVE := "cve-2021-12999"
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-12345",
			CVSSScore:        new(7.5),
			EPSSProbability:  new(0.5),
			CISAKnownExploit: new(true),
			Published:        &mockTime,
			Description:      "Test CVE 2021-12345",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        new(5.4),
			EPSSProbability:  new(0.6),
			CISAKnownExploit: new(false),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-1235",
		},
		{
			CVE:              "CVE-2021-1246",
			CVSSScore:        new(5.4),
			EPSSProbability:  new(0.6),
			CISAKnownExploit: new(false),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-1246",
		},
		{
			CVE:              knownCVE,
			CVSSScore:        new(6.4),
			EPSSProbability:  new(0.61),
			CISAKnownExploit: new(true),
			Published:        new(mockTime),
			Description:      fmt.Sprintf("Test %s", knownCVE),
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	// test list
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	s.Require().Len(resp.Vulnerabilities, 3)
	require.Equal(t, uint(3), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected := map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		"CVE-2021-1246": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1246",
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok, vuln.CVE.CVE)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with matching query containing leading/trailing whitespace
	// TODO(jacob) - this may be another parsing bug
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "  123	")
	require.NoError(t, resp.Err)
	s.Require().Len(resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected = map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		// ...1246 should not match the query
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with non-matching query
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVB")
	require.NoError(t, resp.Err)
	s.Require().Empty(resp.Vulnerabilities)
	require.Equal(t, uint(0), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	// test with a known CVE that does not match on software/OS
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", knownCVE)
	require.NoError(t, resp.Err)
	s.Empty(resp.Vulnerabilities)
	assert.Equal(t, uint(0), resp.Count)
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)

	// test with a substring of a known CVE -- results are returned
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVE-2021-1234")
	require.NoError(t, resp.Err)
	s.Len(resp.Vulnerabilities, 1)
	assert.Equal(t, uint(1), resp.Count)
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)
	_ = s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusNotFound)

	// Team 1 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "1")
	s.Require().Empty(resp.Vulnerabilities)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.NoError(t, resp.Err)

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// No filter (global)
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(t, resp.Vulnerabilities, 3)
	require.Equal(t, uint(3), resp.Count)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[1].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[2].HostsCount)

	// Team 0 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, uint(1), resp.Count)
	require.Equal(t, "CVE-2021-1246", resp.Vulnerabilities[0].CVE.CVE)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)

	var gResp getVulnerabilityResponse
	// invalid cve
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/foobar", nil, http.StatusBadRequest, &gResp)

	// Valid CVE but not in team scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusNoContent, "team_id",
		fmt.Sprintf("%d", team.ID))

	// Valid CVE in "no team" scope
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusOK, &gResp, "team_id", "0")

	// Valid CVE not in "no team" scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusNoContent, "team_id", "0")

	// Invalid TeamID
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusForbidden, "team_id", "100")

	// Valid Global Request
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusOK, &gResp)
	require.NoError(t, gResp.Err)
	require.Equal(t, "CVE-2021-12345", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-12345", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1235", nil, http.StatusOK, &gResp)
	require.NoError(t, gResp.Err)
	require.Equal(t, "CVE-2021-1235", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1235", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.Software, 1)
	require.Equal(t, "Google Chrome", gResp.Software[0].Name)
	require.Equal(t, "0.0.1", gResp.Software[0].Version)
	require.Equal(t, "programs", gResp.Software[0].Source)
	require.Equal(t, "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*", gResp.Software[0].GenerateCPE)
	require.Equal(t, 1, gResp.Software[0].HostsCount)
}

// TestDirectIngestSoftwareWithLongFields tests that software with reported long fields
// are inserted properly and subsequent reports of the same software do not generate new
// entries in the `software` table. (It mainly tests the comparison between the currenly
// inserted software and the incoming software from a host.)
func (s *integrationTestSuite) TestDirectIngestSoftwareWithLongFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)

	// Simulate a osquery agent on Windows reporting a software row for Wireshark.
	rows := []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
		},
	}
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// Check that the software was properly ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	var wiresharkSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
	require.Equal(t, "Wireshark 4.0.8 64-bit", wiresharkSoftware.Name)
	require.Equal(t, "4.0.8", wiresharkSoftware.Version)
	require.Equal(t, "programs", wiresharkSoftware.Source)
	require.Empty(t, wiresharkSoftware.BundleIdentifier)
	require.Empty(t, wiresharkSoftware.Release)
	require.Empty(t, wiresharkSoftware.Arch)
	require.Equal(t, "The Wireshark developer community, https://www.wireshark.org", wiresharkSoftware.Vendor)
	hostSoftwareInstalledPathsQuery := `SELECT installed_path FROM host_software_installed_paths WHERE software_id = ?;`
	var wiresharkSoftwareInstalledPath string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftwareInstalledPath, hostSoftwareInstalledPathsQuery, wiresharkSoftware.ID)
	})
	require.Equal(t, "C:\\Program Files\\Wireshark", wiresharkSoftwareInstalledPath)

	// We now check that the same software is not created again as a new row when it is received again during software ingestion.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	var wiresharkSoftware2 fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware2, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware2.ID)
	require.Equal(t, wiresharkSoftware.ID, wiresharkSoftware2.ID)

	// Simulate a osquery agent on Windows reporting a software row with a longer than 114 chars vendor field.
	rows = []map[string]string{
		{
			"name":           "Foobar" + strings.Repeat("A", fleet.SoftwareNameMaxLength),
			"version":        "4.0.8" + strings.Repeat("B", fleet.SoftwareVersionMaxLength),
			"type":           "Program (Windows)",
			"source":         "programs" + strings.Repeat("C", fleet.SoftwareSourceMaxLength),
			"vendor":         strings.Repeat("D", fleet.SoftwareVendorMaxLength+1),
			"installed_path": "C:\\Program Files\\Foobar",
			// Test UTF-8 encoded strings.
			"bundle_identifier": strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength+1),
			"release":           strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1) + "⌘⌘",
			"arch":              strings.Repeat("G", fleet.SoftwareArchMaxLength+1),
		},
	}

	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.NotZero(t, foobarSoftware.ID)
	require.Equal(t, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6), foobarSoftware.Name)
	require.Equal(t, "4.0.8"+strings.Repeat("B", fleet.SoftwareNameMaxLength-5), foobarSoftware.Version)
	require.Equal(t, "programs"+strings.Repeat("C", fleet.SoftwareSourceMaxLength-8), foobarSoftware.Source)
	// Vendor field is currenty trimmed with a different method (... appended at the end)
	require.Equal(t, strings.Repeat("D", fleet.SoftwareVendorMaxLength-3)+"...", foobarSoftware.Vendor)
	require.Equal(t, strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength), foobarSoftware.BundleIdentifier)
	require.Equal(t, strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1)+"⌘", foobarSoftware.Release)
	require.Equal(t, strings.Repeat("G", fleet.SoftwareArchMaxLength), foobarSoftware.Arch)

	// We now check that the same software with long (to be trimmed) fields is not created again as a new row.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware2 fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware2, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.Equal(t, foobarSoftware.ID, foobarSoftware2.ID)
}

func (s *integrationTestSuite) TestDirectIngestSoftwareWithInvalidFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Ingesting software without name should not fail, but the software won't be inserted.
	rows := []map[string]string{
		{
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	handler1 := logtestutils.NewTestHandler()
	logger1 := slog.New(handler1)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger1,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records1 := handler1.Records()
	require.NotEmpty(t, records1)
	// The "empty name" error is returned by SoftwareFromOsqueryRow and logged as an attribute
	// of the "failed to parse software row" debug message.
	var foundEmptyName bool
	for _, r := range records1 {
		if r.Level == slog.LevelDebug && r.Message == "failed to parse software row" {
			attrs := logtestutils.RecordAttrs(&r)
			if err, ok := attrs["err"]; ok {
				if errStr := fmt.Sprint(err); strings.Contains(errStr, "host reported software with empty name") {
					foundEmptyName = true
					break
				}
			}
		}
	}
	require.True(t, foundEmptyName, "expected a debug log about software with empty name")

	// Check that the software was not ingested.
	softwareQueryByVendor := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE vendor = ?;"
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByVendor, "The Wireshark developer community, https://www.wireshark.org") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software without source should not fail, but the software won't be inserted.
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	handler2 := logtestutils.NewTestHandler()
	logger2 := slog.New(handler2)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger2,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records2 := handler2.Records()
	require.NotEmpty(t, records2)
	var foundEmptySource bool
	for _, r := range records2 {
		if r.Level == slog.LevelDebug && r.Message == "failed to parse software row" {
			attrs := logtestutils.RecordAttrs(&r)
			if err, ok := attrs["err"]; ok {
				if errStr := fmt.Sprint(err); strings.Contains(errStr, "host reported software with empty source") {
					foundEmptySource = true
					break
				}
			}
		}
	}
	require.True(t, foundEmptySource, "expected a debug log about software with empty source")

	// Check that the software was not ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software with invalid last_opened_at should not fail (only log a debug error)
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	handler3 := logtestutils.NewTestHandler()
	logger3 := slog.New(handler3)
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger3,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records3 := handler3.Records()
	require.NotEmpty(t, records3)
	var foundInvalidTimestamp bool
	for _, r := range records3 {
		if r.Level == slog.LevelDebug && r.Message == "host reported software with invalid last opened timestamp" {
			foundInvalidTimestamp = true
			break
		}
	}
	require.True(t, foundInvalidTimestamp, "expected a debug log about software with invalid last opened timestamp")

	// Check that the software was properly ingested.
	var wiresharkSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
}
