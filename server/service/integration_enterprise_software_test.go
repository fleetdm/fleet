package service

// Software inventory and catalog tests for the enterprise (premium) suite.
//
// Belongs here: listing software versions and titles (globally, per fleet, per host), software
// authorization, software title icons, self-service categories and self-service label/vitals
// scoping, VPP apps without MDM, and software auto-update settings.
//
// Does not belong here: uploading or installing packages (integration_enterprise_software_installers_test.go),
// batch/GitOps software operations (integration_enterprise_software_batch_test.go), and
// Fleet-maintained apps (integration_enterprise_software_fma_test.go).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestListSoftware() {
	t := s.T()
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", ExtensionID: "fooextensionid"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		// A Go binary carries its toolchain version in release and its module path in
		// extension_id, which is suppressed for this source only.
		{Name: "air", Version: "v1.48.0", Source: "go_binaries", ExtensionID: "github.com/air-verse/air", Release: "go1.26.1"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	var bar fleet.HostSoftwareEntry
	for _, sw := range host.Software {
		if sw.Name == "bar" {
			bar = sw
			break
		}
	}
	require.NotZero(t, bar.ID)

	inserted, err := s.ds.InsertSoftwareVulnerability(
		ctx, fleet.SoftwareVulnerability{
			SoftwareID:        bar.ID,
			CVE:               "cve-123",
			ResolvedInVersion: new("1.2.3"),
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, s.ds.InsertCVEMeta(ctx, []fleet.CVEMeta{{
		CVE:              "cve-123",
		CVSSScore:        new(float64(5.4)),
		EPSSProbability:  new(float64(0.5)),
		CISAKnownExploit: new(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}))

	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now().UTC()))

	var resp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &resp)
	require.NotNil(t, resp)

	var fooPayload, barPayload, airPayload fleet.Software
	for _, s := range resp.Software {
		switch s.Name {
		case "foo":
			fooPayload = s
		case "bar":
			barPayload = s
		case "air":
			airPayload = s
		default:
			require.Failf(t, "unrecognized software %s", s.Name)

		}
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, "cve-123", barPayload.Vulnerabilities[0].CVE)
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, new(new(5.4)))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, new(new(0.5)))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, new(new(true)))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, new(new(now)))
	require.Equal(t, barPayload.Vulnerabilities[0].Description, new(new("a long description of the cve")))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, new(new("1.2.3")))

	require.Equal(t, "go1.26.1", airPayload.Release)
	require.Empty(t, airPayload.ExtensionID, "the Go module path must not reach the API")
	require.Equal(t, "fooextensionid", fooPayload.ExtensionID, "other sources keep their extension id")

	var respVersions listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &respVersions)
	require.NotNil(t, resp)

	// Reset so a payload missing from this response can't be satisfied by the value the
	// software-list loop above left behind.
	fooPayload, barPayload, airPayload = fleet.Software{}, fleet.Software{}, fleet.Software{}
	for _, s := range respVersions.Software {
		switch s.Name {
		case "foo":
			fooPayload = s
		case "bar":
			barPayload = s
		case "air":
			airPayload = s
		default:
			require.Failf(t, "unrecognized software %s", s.Name)

		}
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, "cve-123", barPayload.Vulnerabilities[0].CVE)
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, new(new(5.4)))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, new(new(0.5)))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, new(new(true)))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, new(new(now)))
	require.Equal(t, barPayload.Vulnerabilities[0].Description, new(new("a long description of the cve")))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, new(new("1.2.3")))

	require.Equal(t, "go1.26.1", airPayload.Release)
	require.Empty(t, airPayload.ExtensionID, "the Go module path must not reach the API")
	require.Equal(t, "fooextensionid", fooPayload.ExtensionID, "other sources keep their extension id")

	// vulnerable param required when using vulnerability filters
	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"without_vulnerability_details", "true",
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
	// without_vulnerability_details with vulnerability filter
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"exploit", "true",
		"vulnerable", "true",
		"without_vulnerability_details", "true",
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
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"exploit", "true",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"min_cvss_score", "1.1",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusUnprocessableEntity, &respVersions,
		"max_cvss_score", "10.0",
	)

	// vulnerability filters
	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Empty(t, respVersions.Software)
	require.Nil(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"max_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Empty(t, respVersions.Software)
	require.Nil(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)

	respVersions = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Len(t, respVersions.Software, 1)
	require.NotEmpty(t, respVersions.CountsUpdatedAt)
}

func (s *integrationEnterpriseTestSuite) TestAllSoftwareTitles() {
	t := s.T()
	ctx := context.Background()

	softwareTitleListResultsMatch := func(want, got []fleet.SoftwareTitleListResult) {
		// compare only the fields we care about
		for i := range got {
			require.NotZero(t, got[i].ID)
			got[i].ID = 0
			require.NotNil(t, got[i].AITool)
			require.False(t, *got[i].AITool)
			got[i].AITool = nil

			for j := range got[i].Versions {
				require.NotZero(t, got[i].Versions[j].ID)
				got[i].Versions[j].ID = 0
			}
			// Sort versions by version
			sort.Slice(
				got[i].Versions, func(a, b int) bool {
					return got[i].Versions[a].Version < got[i].Versions[b].Version
				},
			)
		}

		// sort and use EqualValues instead of ElementsMatch in order
		// to do a deep comparison of nested structures
		sort.Slice(got, func(i, j int) bool {
			return got[i].Name < got[j].Name
		})
		sort.Slice(want, func(i, j int) bool {
			return want[i].Name < want[j].Name
		})
		for _, v := range got {
			sort.Slice(v.Versions, func(i, j int) bool {
				return v.Versions[i].Version < v.Versions[j].Version
			})
		}
		for _, v := range want {
			sort.Slice(v.Versions, func(i, j int) bool {
				return v.Versions[i].Version < v.Versions[j].Version
			})
		}

		require.Equal(t, want, got)
	}

	softwareTitlesMatch := func(want, got []fleet.SoftwareTitle) {
		// compare only the fields we care about
		for i := range got {
			require.NotZero(t, got[i].ID)
			got[i].CountsUpdatedAt = nil
			got[i].ID = 0
			require.NotNil(t, got[i].AITool)
			require.False(t, *got[i].AITool)
			got[i].AITool = nil

			for j := range got[i].Versions {
				require.NotZero(t, got[i].Versions[j].ID)
				got[i].Versions[j].ID = 0
			}
			// Sort versions by version
			sort.Slice(
				got[i].Versions, func(a, b int) bool {
					return got[i].Versions[a].Version < got[i].Versions[b].Version
				},
			)
		}

		// sort and use EqualValues instead of ElementsMatch in order
		// to do a deep comparison of nested structures
		sort.Slice(got, func(i, j int) bool {
			return got[i].Name < got[j].Name
		})
		sort.Slice(want, func(i, j int) bool {
			return want[i].Name < want[j].Name
		})

		require.Equal(t, want, got)
	}

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	tmHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + "tm"),
		NodeKey:         new(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// create a couple of teams and add tmHost to one
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{tmHost.ID})))

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
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

	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "cve-123-123-132",
			CVSSScore:        new(float64(7.8)),
			CISAKnownExploit: new(true),
		},
	})
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	var resp listSoftwareTitlesResponse
	// self-service flag is ignored if no team specified see https://github.com/fleetdm/fleet/issues/26375
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "self_service", "1")
	require.Equal(t, 2, resp.Count)

	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	require.Equal(t, 2, resp.Count)

	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// per_page equals 1, so we get only one item, but the total count is
	// still 2
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// get the second item
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// asking for a non-existent page returns an empty list
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "4",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	// asking for vulnerable only software returns the expected values
	expectedVulnSoftware := []fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	// vulnerable param required when using vulnerability filters
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"exploit", "true",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"min_cvss_score", "1",
	)
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusUnprocessableEntity, &resp,
		"max_cvss_score", "10",
	)

	// vulnerability filters
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Zero(t, resp.Count)
	require.Nil(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"max_cvss_score", "1",
		"vulnerable", "true",
	)
	require.Zero(t, resp.Count)
	require.Nil(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"min_cvss_score", "1",
		"max_cvss_score", "10",
		"exploit", "true",
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch(expectedVulnSoftware, resp.SoftwareTitles)

	// request titles for team1, nothing there yet
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Equal(t, 0, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{}, resp.SoftwareTitles)

	// add new software for tmHost
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "baz", Version: "0.0.5", Source: "deb_packages"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// request software for the team, this time we get results
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", fmt.Sprintf("%d", team1.ID),
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			HostsCount:    1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil}, // NOTE: this only includes versions present in the team
			},
		},
	}, resp.SoftwareTitles)

	// request software for no-team, we get all results and 2 hosts for
	// `"foo"`
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 3, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2, // NOTE: this value is 2, important because no team filter was applied
			HostsCount:    2, // NOTE: this value is 2, important because no team filter was applied
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match cve by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "123",
	)
	require.Equal(t, 1, resp.Count)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ba",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with leading whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "  ba",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with trailing whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ba  ",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name with leading and trailing whitespace
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "  ba  ",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitleListResultsMatch([]fleet.SoftwareTitleListResult{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// find the ID of "foo"
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "foo",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	fooTitle := resp.SoftwareTitles[0]
	require.Equal(t, "foo", fooTitle.Name)

	// find the ID of "baz" (team 1)
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "baz",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	bazTitle := resp.SoftwareTitles[0]
	require.Equal(t, "baz", bazTitle.Name)

	// non-existent id is a 404
	var stResp getSoftwareTitleResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles/999", getSoftwareTitleRequest{}, http.StatusNotFound, &stResp)

	// valid title
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	s.NotZero(*stResp.SoftwareTitle.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    2,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil, HostsCount: new(uint(2))},
				{Version: "0.0.3", Vulnerabilities: nil, HostsCount: new(uint(1))},
			},
		},
	}, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// valid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "foo",
				Source:        "homebrew",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{Version: "0.0.1", Vulnerabilities: nil, HostsCount: new(uint(1))},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// find the ID of "bar"
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "bar",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	barTitle := resp.SoftwareTitles[0]
	require.Equal(t, "bar", barTitle.Name)

	// valid title with vulnerabilities
	expected := []fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{
					Version:         "0.0.4",
					Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
					HostsCount:      new(uint(1)),
				},
			},
		},
	}
	// Global
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	softwareTitlesMatch(expected, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// No Team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", "0")
	softwareTitlesMatch(expected, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// invalid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)

	// invalid title for no team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", bazTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", "0")

	// add bar tmHost
	software = []fleet.Software{
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// valid title with vulnerabilities
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "bar",
				Source:        "apps",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{
						Version:         "0.0.4",
						Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
						HostsCount:      new(uint(1)),
					},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// Team without hosts
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team2.ID),
	)

	// Non-existent team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", "99999",
	)

	// verify that software installers contain SoftwarePackage field
	payloadRubyTm1 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		SelfService:   false,
		TeamID:        &team1.ID,
		Platform:      "linux",
	}
	s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusOK, "")

	payloadEmacsMissingSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install $FLEET_SECRET_INVALID",
		Filename:          "emacs.deb",
		PostInstallScript: "d",
		SelfService:       true,
		Platform:          "linux",
	}
	s.uploadSoftwareInstallerWithErrorNameReason(t, payloadEmacsMissingSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID",
		"install script")

	payloadEmacsMissingPostSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install",
		Filename:          "emacs.deb",
		PostInstallScript: "d $FLEET_SECRET_INVALID",
		SelfService:       true,
		Platform:          "linux",
	}
	s.uploadSoftwareInstallerWithErrorNameReason(t, payloadEmacsMissingPostSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID",
		"post-install script")

	payloadEmacsMissingUnSecret := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install",
		Filename:          "emacs.deb",
		PostInstallScript: "d",
		UninstallScript:   "delet $FLEET_SECRET_INVALID",
		SelfService:       true,
		Platform:          "linux",
	}
	s.uploadSoftwareInstallerWithErrorNameReason(t, payloadEmacsMissingUnSecret, http.StatusUnprocessableEntity, "$FLEET_SECRET_INVALID",
		"uninstall script")

	// not specifiying a team_id translates to "no team" or team_id of 0
	payloadEmacs := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install",
		Filename:         "emacs.deb",
		SelfService:      true,
		AutomaticInstall: true,
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, payloadEmacs, http.StatusOK, "")

	payloadVim := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install",
		Filename:         "vim.deb",
		SelfService:      true,
		TeamID:           new(uint(0)),
		AutomaticInstall: true,
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, payloadVim, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)

	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	// Test that the software titles endpoint returns a SHA256 hash.
	require.Equal(t, "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628", *resp.SoftwareTitles[0].HashSHA256)

	// Upload an installer for the same software but different arch to a different team
	payloadRubyTm2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby_arm64.deb",
		TeamID:        &team2.ID,
		Platform:      "linux",
	}
	s.uploadSoftwareInstaller(t, payloadRubyTm2, http.StatusOK, "")

	// We should only see the one we uploaded to team 1
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)

	// software installer not returned with self-service only (not marked as such)
	resp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp,
		"self_service", "1", "query", "ruby", "team_id", fmt.Sprint(team1.ID))
	require.Empty(t, resp.SoftwareTitles)

	// update it to be self-service, check that it gets returned
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE software_installers SET self_service = 1 WHERE filename = ?", payloadRubyTm1.Filename)
		return err
	})
	resp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp,
		"self_service", "1", "query", "ruby", "team_id", fmt.Sprint(team1.ID))
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[0].SoftwarePackage.SelfService)

	// "All teams" returns all software regardless of self_service see https://github.com/fleetdm/fleet/issues/26375
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"self_service", "true",
	)

	require.Equal(t, 2, resp.Count)

	// "No team" returns the emacs software
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"self_service", "true",
		"team_id", "0",
	)

	require.Len(t, resp.SoftwareTitles, 2)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[0].SoftwarePackage.SelfService)
	require.NotNil(t, resp.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, resp.SoftwareTitles[1].SoftwarePackage.SelfService)
	require.True(t, *resp.SoftwareTitles[1].SoftwarePackage.SelfService)

	emacsPath := fmt.Sprintf("/api/latest/fleet/software/titles/%d", resp.SoftwareTitles[0].ID)
	respTitle := getSoftwareTitleResponse{}
	s.DoJSON("GET", emacsPath, listSoftwareTitlesRequest{}, http.StatusOK, &respTitle)

	require.NotNil(t, respTitle.SoftwareTitle)
	require.Equal(t, "emacs.deb", respTitle.SoftwareTitle.SoftwarePackage.Name)
	require.True(t, respTitle.SoftwareTitle.SoftwarePackage.SelfService)

	darwinSwPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		SelfService:   false,
		TeamID:        &team2.ID,
		Platform:      "darwin",
	}
	s.uploadSoftwareInstaller(t, darwinSwPayload, http.StatusOK, "")

	// Filtering by platform without team errors
	res := s.Do("GET", "/api/latest/fleet/software/titles?platform=darwin", nil, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.FilterTitlesByPlatformNeedsTeamIdErrMsg)

	// Filtering by platform with team ok
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"platform", "darwin",
		"team_id", fmt.Sprint(team2.ID))

	require.Len(t, resp.SoftwareTitles, 1)
	dummyTitleId := resp.SoftwareTitles[0].ID
	dummyPath := fmt.Sprintf("/api/latest/fleet/software/titles/%d", dummyTitleId)
	respTitle = getSoftwareTitleResponse{}
	s.DoJSON("GET", dummyPath, listSoftwareTitlesRequest{}, http.StatusOK, &respTitle)
	title := respTitle.SoftwareTitle
	require.NotNil(t, title)
	require.Equal(t, "DummyApp", title.Name)
}

func (s *integrationEnterpriseTestSuite) TestSoftwareAuth() {
	t := s.T()
	ctx := context.Background()
	// create two hosts, one belongs to team1 and one has no team
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	tmHost, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + "tm"),
		NodeKey:         new(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// Create two teams, team1 and team2.
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{tmHost.ID}))
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	allSoftware := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	// add all the software entries to the "no team host"
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, allSoftware)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	// add only one version of "foo" to the team host
	_, err = s.ds.UpdateHostSoftware(ctx, tmHost.ID, []fleet.Software{allSoftware[0]})
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, tmHost, false))

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// add variations of user roles to different teams
	extraTestUsers := make(map[string]fleet.User)
	for k, u := range map[string]struct {
		Email      string
		GlobalRole *string
		Teams      *[]fleet.UserTeam
	}{
		"team-1-admin": {
			Email: "team-1-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-1-maintainer": {
			Email: "team-1-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-1-observer": {
			Email: "team-1-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleObserver,
			}}),
		},
		"team-2-admin": {
			Email: "team-2-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-2-maintainer": {
			Email: "team-2-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-2-observer": {
			Email: "team-2-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleObserver,
			}}),
		},
	} {
		uu := u
		cur := createUserResponse{}
		s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
			Email:                    &uu.Email,
			Password:                 &test.GoodPassword,
			Name:                     &uu.Email,
			Teams:                    uu.Teams,
			AdminForcedPasswordReset: new(false),
		}, http.StatusOK, &cur)
		extraTestUsers[k] = *cur.User
	}

	// List all software titles with an admin
	var listSoftwareTitlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)

	var softwareFoo, softwareBar *fleet.SoftwareTitleListResult
	for _, s := range listSoftwareTitlesResp.SoftwareTitles {
		switch s.Name {
		case "foo":
			softwareFoo = &s
		case "bar":
			softwareBar = &s
		}
	}
	require.NotNil(t, softwareFoo)
	require.NotNil(t, softwareBar)

	var teamFooVersion *fleet.SoftwareVersion
	for _, sv := range softwareFoo.Versions {
		if sv.Version == "0.0.1" {
			teamFooVersion = &sv
		}
	}
	require.NotNil(t, teamFooVersion)

	// set the admin token again to avoid breaking other tests
	t.Cleanup(func() {
		s.token = s.getTestAdminToken()
	})

	for _, tc := range []struct {
		name                 string
		user                 fleet.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
	}{
		{
			name:                 "global-admin",
			user:                 s.users["admin1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-maintainer",
			user:                 s.users["user1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-observer",
			user:                 s.users["user2@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-belongs-to-team",
			user:                 extraTestUsers["team-1-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-maintainer-belongs-to-team",
			user:                 extraTestUsers["team-1-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-observer-belongs-to-team",
			user:                 extraTestUsers["team-1-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-maintainer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-observer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// to make the request as the user
			s.token = s.getTestToken(tc.user.Email, test.GoodPassword)

			if tc.shouldFailGlobalRead {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareTitlesRequest{}, http.StatusForbidden, &resp)

				// Get a global software title (only on the "no team" host, which
				// no team-scoped user can see): NotFound, not Forbidden, so its
				// existence can't be inferred from the response.
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 2, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusOK, &resp)
				require.Equal(t, 3, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a global software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 3, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusOK, &softwareListResp)
				require.Equal(t, 3, countSoftwareResp.Count)
			}

			if tc.shouldFailTeamRead {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{TeamID: &team1.ID}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{TeamID: &team1.ID}, http.StatusForbidden, &resp)

				// Get a team software title (on team1 and "no team", neither
				// visible to this team-2 user): NotFound, not Forbidden, so its
				// existence can't be inferred from the response.
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a count of team software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{TeamID: &team1.ID}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 1, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{TeamID: &team1.ID}, http.StatusOK, &resp)
				require.Equal(t, 1, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a team software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{TeamID: &team1.ID}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 1, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{TeamID: &team1.ID}, http.StatusOK, &softwareListResp)
				require.Equal(t, 1, countSoftwareResp.Count)
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestListHostSoftware() {
	t := s.T()
	ctx := context.Background()

	token := "good_token"
	host := createOrbitEnrolledHost(t, "ubuntu", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	otherToken := "other_token"
	otherHost := createOrbitEnrolledHost(t, "rhel", "host2", s.ds)
	createDeviceTokenForHost(t, s.ds, otherHost.ID, otherToken)

	// no software yet
	var getHostSw getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Empty(t, getHostSw.Software)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Empty(t, getHostSw.Software)

	var getDeviceSw getDeviceSoftwareResponse
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	err := json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Empty(t, getDeviceSw.Software)
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Empty(t, getDeviceSw.Software)

	// create some software for that host
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.1", Source: "deb_packages"},
	}
	us, err := s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)

	// Note: the ID returned by ListHostSoftware is the title ID, not the software ID. We need the
	// software ID to assign the vulnerabilities correctly below.
	var barSoftwareID uint
	for _, s := range us.Inserted {
		if s.Name == "bar" {
			barSoftwareID = s.ID
		}
	}

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 2) // foo and bar
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	// no package information as there is no installer
	require.Nil(t, getHostSw.Software[0].SoftwarePackage)
	require.Nil(t, getHostSw.Software[0].AppStoreApp)
	require.Nil(t, getHostSw.Software[1].SoftwarePackage)
	require.Nil(t, getHostSw.Software[1].AppStoreApp)

	// Add vulnerabilities to software to check query param filtering
	_, err = s.ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{SoftwareID: barSoftwareID, CVE: "CVE-bar-1234"}, fleet.NVDSource)
	require.NoError(t, err)
	_, err = s.ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{SoftwareID: barSoftwareID, CVE: "CVE-bar-5678"}, fleet.NVDSource)
	require.NoError(t, err)

	// Add CVSS scores and exploit metadata so the severity filters have data to match against.
	// "bar" has one critical, actively-exploited CVE and one low-severity CVE, so it should match
	// both a critical filter and a low filter (matches if ANY CVE is in range).
	require.NoError(t, s.ds.InsertCVEMeta(ctx, []fleet.CVEMeta{
		{CVE: "CVE-bar-1234", CVSSScore: new(float64(9.5)), CISAKnownExploit: new(true)},
		{CVE: "CVE-bar-5678", CVSSScore: new(float64(3.0)), CISAKnownExploit: new(false)},
	}))

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "vulnerable", "true")
	require.Len(t, getHostSw.Software, 1)
	require.NoError(t, err)
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Nil(t, getHostSw.Software[0].SoftwarePackage)
	require.Nil(t, getHostSw.Software[0].AppStoreApp)
	require.Len(t, getHostSw.Software[0].InstalledVersions, 1)
	require.Len(t, getHostSw.Software[0].InstalledVersions[0].Vulnerabilities, 2)
	require.Equal(t, []string{"CVE-bar-1234", "CVE-bar-5678"}, getHostSw.Software[0].InstalledVersions[0].Vulnerabilities)

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 2) // foo and bar
	require.Equal(t, "bar", getDeviceSw.Software[0].Name)
	require.Equal(t, "foo", getDeviceSw.Software[1].Name)
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)
	// no package information as there is no installer
	require.Nil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.Nil(t, getDeviceSw.Software[1].SoftwarePackage)
	require.Nil(t, getDeviceSw.Software[1].AppStoreApp)

	// create a software installer, not installed on the host
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "ruby.deb",
		Version:       "1:2.5.1",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

	// update it to be self-service
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE software_installers SET self_service = 1 WHERE filename = ?", payload.Filename)
		return err
	})

	// default (no include_available_for_install param) should only return installed software, not library items (#41631)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 2) // foo and bar only — ruby is not installed, just available
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)

	// user authenticated endpoint, explicitly request to include available for install software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 3) // foo, bar and ruby.deb
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)
	require.Equal(t, "ruby", getHostSw.Software[2].Name)
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	require.Nil(t, getHostSw.Software[2].AppStoreApp)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[2].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getHostSw.Software[2].SoftwarePackage.Version)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[2].SoftwarePackage.SelfService)
	require.Nil(t, getHostSw.Software[2].Status)

	// user authenticated endpoint, explicitly request to not include available for install software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=false", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 2) // foo and bar
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)

	// only the installer is returned for self-service only
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)

	// available installer is not returned by device-authenticated endpoint
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 2) // foo and bar
	require.Equal(t, "bar", getDeviceSw.Software[0].Name)
	require.Equal(t, "foo", getDeviceSw.Software[1].Name)
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)

	// but it gets returned for self-service only
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Equal(t, payload.Filename, getDeviceSw.Software[0].SoftwarePackage.Name)
	require.Equal(t, payload.Version, getDeviceSw.Software[0].SoftwarePackage.Version)

	// =========================================
	// test label scoping
	// =========================================

	// TODO(JVE): remove/update this once the API is in place
	updateInstallerLabel := func(siID, labelID uint, exclude bool) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err = q.ExecContext(
				ctx,
				`INSERT INTO software_installer_labels (software_installer_id, label_id, exclude) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE exclude = VALUES(exclude)`,
				siID, labelID, exclude,
			)
			return err
		})
	}

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, "SELECT id FROM software_installers WHERE title_id = ?", titleID)
	})
	require.NotEmpty(t, installerID)

	// create some labels
	var labelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  "label1",
		Hosts: []string{host.Hostname}}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)

	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  "label2",
		Hosts: []string{host.Hostname}}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)

	// Set to "exclude any". Installer should be missing from the response for both host details and
	// for self service
	updateInstallerLabel(installerID, labelResp.Label.ID, true)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Empty(t, getHostSw.Software)

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Empty(t, getDeviceSw.Software)

	// Set to "include any". Installer should be in response.
	updateInstallerLabel(installerID, labelResp.Label.ID, false)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)

	// request installation on the host
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		host.ID, titleID), nil, http.StatusAccepted, &installResp)

	// default (no param) should still only return installed software, even after install is requested (#41631)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 2) // foo and bar — ruby has a pending install but is not yet installed
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)

	// with include_available_for_install=true, the pending install is visible
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 3) // foo, bar and ruby.deb
	require.Equal(t, "bar", getHostSw.Software[0].Name)
	require.Equal(t, "foo", getHostSw.Software[1].Name)
	require.Equal(t, "ruby", getHostSw.Software[2].Name)
	require.Len(t, getHostSw.Software[1].InstalledVersions, 2)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage)
	require.Equal(t, "ruby.deb", getHostSw.Software[2].SoftwarePackage.Name)
	require.NotNil(t, getHostSw.Software[2].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[2].Status)
	require.NotNil(t, getHostSw.Software[2].SoftwarePackage.SelfService)
	require.True(t, *getHostSw.Software[2].SoftwarePackage.SelfService)

	rubyLastInstallInstallUUID := getHostSw.Software[2].SoftwarePackage.LastInstall.InstallUUID

	// still returned with self-service filter
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)

	// now returned by device-authenticated endpoint
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 2) // foo, bar
	require.Equal(t, "bar", getDeviceSw.Software[0].Name)
	require.Equal(t, "foo", getDeviceSw.Software[1].Name)
	require.Len(t, getDeviceSw.Software[0].InstalledVersions, 1)
	require.Len(t, getDeviceSw.Software[1].InstalledVersions, 2)

	// confirm device-auth'd install results are visible
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software/install/"+rubyLastInstallInstallUUID+"/results", nil, http.StatusOK)
	getDeviceInstallResult := getSoftwareInstallResultsResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceInstallResult)
	require.NoError(t, err)
	require.Equal(t, "ruby", getDeviceInstallResult.Results.SoftwareTitle)
	require.Equal(t, host.ID, getDeviceInstallResult.Results.HostID)
	require.Equal(t, fleet.SoftwareInstallPending, getDeviceInstallResult.Results.Status)
	s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software/install/nope/results", nil, http.StatusNotFound)
	s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+otherToken+"/software/install/"+rubyLastInstallInstallUUID+"/results", nil, http.StatusNotFound)

	// still returned for self-service only too
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?self_service=1", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1)
	require.Equal(t, "ruby", getDeviceSw.Software[0].Name)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage)
	require.NotNil(t, getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.True(t, *getDeviceSw.Software[0].SoftwarePackage.SelfService)
	require.Nil(t, getDeviceSw.Software[0].AppStoreApp)

	// test with a query
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "query", "foo")
	require.Len(t, getHostSw.Software, 1) // foo only
	require.Equal(t, "foo", getHostSw.Software[0].Name)
	require.Len(t, getHostSw.Software[0].InstalledVersions, 2)

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?query=bar", nil, http.StatusOK)
	getDeviceSw = getDeviceSoftwareResponse{}
	err = json.NewDecoder(res.Body).Decode(&getDeviceSw)
	require.NoError(t, err)
	require.Len(t, getDeviceSw.Software, 1) // bar only
	require.Equal(t, "bar", getDeviceSw.Software[0].Name)
	require.Len(t, getDeviceSw.Software[0].InstalledVersions, 1)

	// Get software available for install
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install",
		"true", "order_key", "name", "order_direction", "asc")
	require.Len(t, getHostSw.Software, 1) // ruby.app
	assert.Equal(t, "ruby", getHostSw.Software[0].Name)
	assert.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	assert.NotNil(t, getHostSw.Software[0].SoftwarePackage)
	assert.Equal(t, "1:2.5.1", getHostSw.Software[0].SoftwarePackage.Version)

	// =========================================
	// vulnerability severity filters on the device-authenticated "My device" software endpoint
	// =========================================
	//
	// "bar" has two CVEs: CVE-bar-1234 (CVSS 9.5, actively exploited) and CVE-bar-5678 (CVSS 3.0).
	// "foo" has no vulnerabilities. The host also has the (not installed) "ruby" installer, which
	// must never appear in the inventory-only device responses below.
	deviceSoftware := func(t *testing.T, query string) getDeviceSoftwareResponse {
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?"+query, nil, http.StatusOK)
		defer res.Body.Close()
		var resp getDeviceSoftwareResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp
	}

	// vulnerable=true returns only software with vulnerabilities.
	resp := deviceSoftware(t, "vulnerable=true")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)
	require.Equal(t, 1, resp.Count)

	// min_cvss_score is inclusive: 9.5 is returned for min_cvss_score=9.5 (the critical band edge).
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=9.5")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// a min above all scores returns nothing.
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=9.6")
	require.Empty(t, resp.Software)
	require.Equal(t, 0, resp.Count)

	// max_cvss_score is inclusive: a low max still matches the low-severity CVE (3.0).
	resp = deviceSoftware(t, "vulnerable=true&max_cvss_score=3.0")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// a max below all scores returns nothing.
	resp = deviceSoftware(t, "vulnerable=true&max_cvss_score=2.9")
	require.Empty(t, resp.Software)

	// min + max together apply both bounds (range filter).
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=1&max_cvss_score=10")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// min == max returns only CVEs scored exactly that value (3.0 matches, 5.0 does not).
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=3.0&max_cvss_score=3.0")
	require.Len(t, resp.Software, 1)
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=5.0&max_cvss_score=5.0")
	require.Empty(t, resp.Software)

	// min > max yields an empty result, not a server error.
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=8&max_cvss_score=2")
	require.Empty(t, resp.Software)

	// a title with multiple CVEs of differing scores appears under BOTH a low and a critical
	// filter (matches if ANY CVE is in range).
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=9.0&max_cvss_score=10")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)
	resp = deviceSoftware(t, "vulnerable=true&min_cvss_score=0.1&max_cvss_score=3.9")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// exploit=true returns only software with a CISA known-exploited CVE.
	resp = deviceSoftware(t, "vulnerable=true&exploit=true")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// exploit combined with a range that only covers the non-exploited CVE returns nothing,
	// since the exploited CVE (9.5) is outside the 0.1-3.9 band.
	resp = deviceSoftware(t, "vulnerable=true&exploit=true&min_cvss_score=0.1&max_cvss_score=3.9")
	require.Empty(t, resp.Software)

	// filters combine with the search query: "foo" has no vulnerabilities so it is filtered out.
	resp = deviceSoftware(t, "vulnerable=true&query=foo")
	require.Empty(t, resp.Software)
	resp = deviceSoftware(t, "vulnerable=true&query=bar")
	require.Len(t, resp.Software, 1)
	require.Equal(t, "bar", resp.Software[0].Name)

	// severity filters require vulnerable=true; otherwise the request is rejected with 422.
	for _, premiumFilter := range []string{"min_cvss_score=1", "max_cvss_score=10", "exploit=true"} {
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software?"+premiumFilter, nil, http.StatusUnprocessableEntity)
		require.NoError(t, res.Body.Close())
	}
}

func (s *integrationEnterpriseTestSuite) TestSoftwareTitleIcons() {
	t := s.T()
	ctx := context.Background()

	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)
	host1 := test.NewHost(t, s.ds, "host1", "", "host1key", "host1uuid", time.Now())
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create software installer + software title
	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.3", Source: "apps", BundleIdentifier: "foo.bundle.id"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	_, titleID, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "hello",
		InstallerFile:    tfr1,
		StorageID:        "storage1",
		Filename:         "foo.pkg",
		Title:            "foo",
		Version:          "0.0.3",
		Source:           "apps",
		TeamID:           &tm.ID,
		UserID:           user.ID,
		BundleIdentifier: "foo.bundle.id",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	parsePutResponse := func(resp *http.Response) putSoftwareTitleIconResponse {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		result := putSoftwareTitleIconResponse{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		return result
	}

	// get icon from file system
	file, err := os.Open("testdata/icons/valid-icon.png")
	require.NoError(t, err)
	defer file.Close()

	// get icon size for later
	fileInfo, err := file.Stat()
	require.NoError(t, err)
	iconSize := fileInfo.Size()

	// create proper form data to post
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	fileWriter, err := writer.CreateFormFile("icon", "icon.png")
	require.NoError(t, err)
	_, err = io.Copy(fileWriter, file)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	headers := map[string]string{
		"Content-Type":  writer.FormDataContentType(),
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}

	// get activities before the PUT request
	activitiesBefore := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesBefore)

	// PUT: create new software title icon
	resp := s.DoRawWithHeaders(
		"PUT",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		buf.Bytes(),
		http.StatusOK,
		headers,
	)
	result := parsePutResponse(resp)
	iconUrl := fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", titleID, tm.ID)
	require.NoError(t, result.Err)
	require.Contains(t, result.IconUrl, iconUrl)

	// get activities after the PUT request
	activitiesAfter := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesAfter)

	// verify exactly 1 new edited_software activity was created
	var editedSoftwareActivitiesBefore []fleet.Activity
	var editedSoftwareActivitiesAfter []fleet.Activity

	for _, activity := range activitiesBefore.Activities {
		if activity.Type == "edited_software" {
			editedSoftwareActivitiesBefore = append(editedSoftwareActivitiesBefore, *activity)
		}
	}

	for _, activity := range activitiesAfter.Activities {
		if activity.Type == "edited_software" {
			editedSoftwareActivitiesAfter = append(editedSoftwareActivitiesAfter, *activity)
		}
	}

	require.Len(t, editedSoftwareActivitiesAfter, len(editedSoftwareActivitiesBefore)+1, "Expected exactly 1 new edited_software activity")

	// find the new activity by comparing before and after lists
	var newActivity *fleet.Activity
	for _, afterActivity := range editedSoftwareActivitiesAfter {
		isNew := true
		for _, beforeActivity := range editedSoftwareActivitiesBefore {
			if afterActivity.ID == beforeActivity.ID {
				isNew = false
				break
			}
		}
		if isNew {
			newActivity = &afterActivity
			break
		}
	}
	require.NotNil(t, newActivity, "Should have found exactly one new activity")

	// verify the new activity has the correct details
	var details fleet.ActivityTypeEditedSoftware
	err = json.Unmarshal(*newActivity.Details, &details)
	require.NoError(t, err)
	require.Equal(t, "foo", details.SoftwareTitle)
	require.Equal(t, "foo.pkg", *details.SoftwarePackage)
	require.Equal(t, iconUrl, *details.SoftwareIconURL)
	require.Equal(t, titleID, details.SoftwareTitleID)

	// PUT: Icon too large
	// Create a fake large file (101KB of data)
	largeData := make([]byte, 101*1024)
	reader := bytes.NewReader(largeData)
	iconFile, err := fleet.NewTempFileReader(reader, func() string { return t.TempDir() })
	require.NoError(t, err)

	var bufInvalid bytes.Buffer
	writer = multipart.NewWriter(&bufInvalid)
	fileWriter, err = writer.CreateFormFile("icon", "icon.png")
	require.NoError(t, err)
	_, err = io.Copy(fileWriter, iconFile)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	headers = map[string]string{
		"Content-Type":  writer.FormDataContentType(),
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}
	resp = s.DoRawWithHeaders(
		"PUT",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		bufInvalid.Bytes(),
		http.StatusBadRequest,
		headers,
	)
	var errorResp struct {
		Message string `json:"message"`
		Errors  []struct {
			Name   string `json:"name"`
			Reason string `json:"reason"`
		} `json:"errors"`
	}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &errorResp)
	require.NoError(t, err)
	assert.Equal(t, "Bad request", errorResp.Message)
	require.Len(t, errorResp.Errors, 1)
	assert.Contains(t, errorResp.Errors[0].Reason, "icon must be less than 100KB")

	// PUT: gitops workflow, passing in sha256 & filename
	var storedIcons []fleet.SoftwareTitleIcon
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.SelectContext(ctx, q, &storedIcons, "SELECT storage_id, filename FROM software_title_icons")
		if err != nil {
			return err
		}
		return nil
	})
	require.Len(t, storedIcons, 1)

	storedIcon := storedIcons[0]
	var buf2 bytes.Buffer
	writer = multipart.NewWriter(&buf2)
	err = writer.WriteField("hash_sha256", storedIcon.StorageID)
	require.NoError(t, err)
	err = writer.WriteField("filename", storedIcon.Filename)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	headers = map[string]string{
		"Content-Type":  writer.FormDataContentType(),
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}
	resp = s.DoRawWithHeaders(
		"PUT",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		buf2.Bytes(),
		http.StatusOK,
		headers,
	)
	result = parsePutResponse(resp)
	require.NoError(t, result.Err)
	require.Contains(t, result.IconUrl, iconUrl)

	// verify no new software title icons were created
	storedIcons = make([]fleet.SoftwareTitleIcon, 0)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.SelectContext(ctx, q, &storedIcons, "SELECT storage_id, filename FROM software_title_icons")
		if err != nil {
			return err
		}
		return nil
	})
	require.Len(t, storedIcons, 1)
	require.Equal(t, storedIcon.StorageID, storedIcons[0].StorageID)
	require.Equal(t, storedIcon.Filename, storedIcons[0].Filename)

	// verify no new activity was created (should still be the same count as before)
	activitiesAfterGitops := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesAfterGitops)
	var editedSoftwareActivitiesAfterGitops []fleet.Activity
	for _, activity := range activitiesAfterGitops.Activities {
		if activity.Type == "edited_software" {
			editedSoftwareActivitiesAfterGitops = append(editedSoftwareActivitiesAfterGitops, *activity)
		}
	}
	require.Len(t, editedSoftwareActivitiesAfterGitops, len(editedSoftwareActivitiesAfter), "No new activity should be created for gitops upload")

	// GET software title icon
	headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}
	resp = s.DoRawWithHeaders(
		"GET",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil,
		http.StatusOK,
		headers,
	)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	assert.Equal(t, fmt.Sprintf(`inline; filename="%s"`, "icon.png"), resp.Header.Get("Content-Disposition"))
	assert.Equal(t, fmt.Sprintf("%d", iconSize), resp.Header.Get("Content-Length"))

	// DELETE software title icon
	s.DoRawWithHeaders(
		"DELETE",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", titleID, tm.ID),
		nil,
		http.StatusOK,
		headers,
	)
	require.NoError(t, err)

	// verify new activity was created (should be one more than before)
	activitiesAfterDelete := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesAfterDelete)
	var editedSoftwareActivitiesAfterDelete []fleet.Activity
	for _, activity := range activitiesAfterDelete.Activities {
		if activity.Type == "edited_software" {
			editedSoftwareActivitiesAfterDelete = append(editedSoftwareActivitiesAfterDelete, *activity)
		}
	}
	require.Len(t, editedSoftwareActivitiesAfterDelete, len(editedSoftwareActivitiesAfter)+1, "Expected exactly 1 new activity after delete")

	// ensure there is an activity where the icon url is set to empty
	// signifying it's deleted
	var deleteActivity *fleet.Activity
	for _, afterActivity := range editedSoftwareActivitiesAfterDelete {
		isNew := true
		for _, beforeActivity := range editedSoftwareActivitiesAfter {
			if afterActivity.ID == beforeActivity.ID {
				isNew = false
				break
			}
		}
		if isNew {
			deleteActivity = &afterActivity
			break
		}
	}
	require.NotNil(t, deleteActivity, "Should have found exactly one new activity from delete operation")

	// verify the delete activity has the correct details
	var deleteDetails fleet.ActivityTypeEditedSoftware
	err = json.Unmarshal(*deleteActivity.Details, &deleteDetails)
	require.NoError(t, err)
	require.Equal(t, titleID, deleteDetails.SoftwareTitleID)
	require.Equal(t, "foo", deleteDetails.SoftwareTitle)
	require.Equal(t, "foo.pkg", *deleteDetails.SoftwarePackage)
	require.Empty(t, *deleteDetails.SoftwareIconURL, "Icon URL should be empty after deletion")

	_, err = s.softwareInstallStore.Cleanup(ctx, nil, time.Now())
	require.NoError(t, err)
}

func (s *integrationEnterpriseTestSuite) TestVPPAppsWithoutMDM() {
	t := s.T()
	ctx := context.Background()

	// Create host
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// Create team and add host to team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1")}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{orbitHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add an app so that we don't get a not found error
	app, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		AdamID:           "adam_test_vpp1", // max 16 chars
		Platform:         fleet.MacOSPlatform,
	}, &team.ID, nil)
	require.NoError(t, err)

	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some pkg install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	// We don't see VPP, but we do still see the installers (need include_available_for_install=true since it's not in osquery inventory)
	resp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", orbitHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp)
	assert.Len(t, resp.Software, 1)
	assert.NotNil(t, resp.Software[0].SoftwarePackage)
	assert.Nil(t, resp.Software[0].AppStoreApp)

	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", orbitHost.ID, app.TitleID), &installSoftwareRequest{},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't install. MDM is turned off. Please make sure that MDM is turned on to install App Store apps.")
}

func (s *integrationEnterpriseTestSuite) TestListHostSoftwareWithLabelScoping() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)

	// Create software installers and corresponding host install requests.
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.deb",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")

	// create install request for the software and record a successful result
	resp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/software/%d/install", host.ID, titleID), nil, http.StatusAccepted, &resp)
	installUUID := getLatestSoftwareInstallExecID(t, s.ds, host.ID)

	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "",
			"install_script_exit_code": 0,
			"install_script_output": "success"
		}`, *host.OrbitNodeKey, installUUID)),
		http.StatusNoContent)

	// Software is now installed on the host. We should see it in the host software list
	// (need include_available_for_install=true since the install was tracked via an installer, not osquery inventory)
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, "ruby", getHostSw.Software[0].Name)

	// De-scope the software by adding an exclude any label that the host has.
	// TODO(JVE): remove/update this once the API is in place
	updateInstallerLabel := func(siID, labelID uint, exclude bool) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(
				ctx,
				`INSERT INTO software_installer_labels (software_installer_id, label_id, exclude) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE exclude = VALUES(exclude)`,
				siID, labelID, exclude,
			)
			return err
		})
	}

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, "SELECT id FROM software_installers WHERE title_id = ?", titleID)
	})
	require.NotEmpty(t, installerID)

	// create some labels and assign them to the host
	var labelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  "label1",
		Hosts: []string{host.Hostname}}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)
	lbl1 := labelResp.Label

	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name:  "label2",
		Query: "SELECT 1"}, http.StatusOK, &labelResp)
	require.NotZero(t, labelResp.Label.ID)
	lbl2 := labelResp.Label
	err := s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{lbl2.ID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	updateInstallerLabel(installerID, lbl1.ID, true)
	updateInstallerLabel(installerID, lbl2.ID, true)

	// We should still see the software at this point, because we haven't uninstalled it yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 1)

	// installer should be out of scope since the label is "exclude any"
	scoped, err := s.ds.IsSoftwareInstallerLabelScoped(ctx, installerID, host.ID)
	require.NoError(t, err)
	require.False(t, scoped)

	// host should be excluded from the software installer
	hostsNotInScope, err := s.ds.GetExcludedHostIDMapForSoftwareInstaller(ctx, installerID)
	require.NoError(t, err)
	require.Contains(t, hostsNotInScope, host.ID)

	// uninstall the software
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", host.ID, titleID), nil, http.StatusAccepted, &resp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?include_available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Len(t, getHostSw.Software, 1)
	// TODO this is a corner case where we have visibility on a descoped uninstall, but this will disappear
	// once we complete the uninstall because at that point we'll be relying solely on inventory to determine software
	// status, because we should be treating descoped installers as if they don't exist.
	assert.Equal(t, fleet.SoftwareUninstallPending, *getHostSw.Software[0].Status)
	require.Nil(t, getHostSw.Software[0].SoftwarePackage)

	var uninstallExecutionID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &uninstallExecutionID, `
			SELECT execution_id
			FROM host_software_installs
			WHERE host_id = ? AND status='pending_uninstall'
		`, host.ID)
	})

	// Host sends failed uninstall result
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "uninstall"}`, *host.OrbitNodeKey,
			uninstallExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// Now that the software is uninstalled, we should no longer see it in the host software list,
	// because it is de-scoped via labels.
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software?available_for_install=true", host.ID), nil, http.StatusOK, &getHostSw)
	require.Empty(t, getHostSw.Software)
}

func (s *integrationEnterpriseTestSuite) TestUpdateSoftwareAutoUpdateConfig() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestUpdateSoftwareAutoUpdateConfig Team"})
	require.NoError(t, err)
	teamID := team.ID

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// create two VPP apps
	vppApp, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name: "vpp1", BundleIdentifier: "com.app.vpp1",
		AdamID: "adam_vpp_app_1", Platform: fleet.IPadOSPlatform,
	}, &teamID, nil)
	require.NoError(t, err)

	// Get the software title.
	var titlesResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppApp.TitleID), nil, http.StatusOK, &titlesResp)
	require.Nil(t, titlesResp.SoftwareTitle.AutoUpdateEnabled)
	require.Nil(t, titlesResp.SoftwareTitle.AutoUpdateStartTime)
	require.Nil(t, titlesResp.SoftwareTitle.AutoUpdateEndTime)

	// Update the auto-update config
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", vppApp.TitleID), updateAppStoreAppRequest{
		TeamID:              &teamID,
		AutoUpdateEnabled:   new(true),
		AutoUpdateStartTime: new("02:00"),
		AutoUpdateEndTime:   new("04:00"),
	}, http.StatusOK, &titlesResp)

	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":"adam_vpp_app_1", "auto_update_enabled":true, "auto_update_window_end":"04:00", "auto_update_window_start":"02:00", "platform":"ipados", "self_service":false, "software_display_name":"", "software_icon_url":null, "software_title":"vpp1", "software_title_id":%d, "team_id":%d, "team_name":"%s", "fleet_id":%d, "fleet_name":"%s"}`, vppApp.TitleID, team.ID, team.Name, team.ID, team.Name), 0)

	// Get the software title again.
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppApp.TitleID), nil, http.StatusOK, &titlesResp, "team_id", fmt.Sprintf("%d", teamID))
	require.NotNil(t, titlesResp.SoftwareTitle.AutoUpdateEnabled)
	require.True(t, *titlesResp.SoftwareTitle.AutoUpdateEnabled)
	require.NotNil(t, titlesResp.SoftwareTitle.AutoUpdateStartTime)
	require.Equal(t, "02:00", *titlesResp.SoftwareTitle.AutoUpdateStartTime)
	require.NotNil(t, titlesResp.SoftwareTitle.AutoUpdateEndTime)
	require.Equal(t, "04:00", *titlesResp.SoftwareTitle.AutoUpdateEndTime)

	// Do an update without auto-update fields to check that it still includes the auto-update values.
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", vppApp.TitleID), updateAppStoreAppRequest{
		TeamID:      &teamID,
		DisplayName: new("New Display Name"),
	}, http.StatusOK, &titlesResp)

	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":"adam_vpp_app_1",  "auto_update_enabled":true, "auto_update_window_end":"04:00", "auto_update_window_start":"02:00", "platform":"ipados", "self_service":false, "software_display_name":"New Display Name", "software_icon_url":null, "software_title":"vpp1", "software_title_id":%d, "team_id":%d, "team_name":"%s", "fleet_id":%d, "fleet_name":"%s"}`, vppApp.TitleID, team.ID, team.Name, team.ID, team.Name), 0)

	// Disable the auto-update config
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", vppApp.TitleID), updateAppStoreAppRequest{
		TeamID:            &teamID,
		AutoUpdateEnabled: new(false),
	}, http.StatusOK, &titlesResp)

	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":"adam_vpp_app_1", "auto_update_enabled":false, "platform":"ipados", "self_service":false, "software_display_name":"New Display Name", "software_icon_url":null, "software_title":"vpp1", "software_title_id":%d, "team_id":%d, "team_name":"%s", "fleet_id":%d, "fleet_name":"%s"}`, vppApp.TitleID, team.ID, team.Name, team.ID, team.Name), 0)

	// Do an update without auto-update fields to check that it still includes the auto-update values.
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", vppApp.TitleID), updateAppStoreAppRequest{
		TeamID:      &teamID,
		DisplayName: new("Updated Display Name"),
	}, http.StatusOK, &titlesResp)

	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":"adam_vpp_app_1", "auto_update_enabled":false, "platform":"ipados", "self_service":false, "software_display_name":"Updated Display Name", "software_icon_url":null, "software_title":"vpp1", "software_title_id":%d, "team_id":%d, "team_name":"%s", "fleet_id":%d, "fleet_name":"%s"}`, vppApp.TitleID, team.ID, team.Name, team.ID, team.Name), 0)
}

func (s *integrationEnterpriseTestSuite) TestSelfServiceCategoriesCRUD() {
	t := s.T()
	ctx := t.Context()
	const path = "/api/latest/fleet/software/self_service_categories"

	// create — leading/trailing whitespace in the name is trimmed before storage
	var addResp addSelfServiceCategoriesResponse
	s.DoJSON("POST", path, map[string]any{"fleet_id": 0, "name": "  💼 Engineering  "}, http.StatusOK, &addResp)
	require.NotZero(t, addResp.SelfServiceCategory.ID)
	require.Equal(t, "💼 Engineering", addResp.SelfServiceCategory.Name)
	require.Equal(t, uint(0), addResp.SelfServiceCategory.TeamID)
	catID := addResp.SelfServiceCategory.ID
	s.lastActivityMatches(fleet.ActivityTypeAddedSelfServiceCategory{}.ActivityName(),
		`{"self_service_category_name":"💼 Engineering","team_id":0,"team_name":null,"fleet_id":0,"fleet_name":null}`, 0)

	// list includes new category
	var listResp getSelfServiceCategoriesResponse
	s.DoJSON("GET", path, nil, http.StatusOK, &listResp, "fleet_id", "0")
	names := make([]string, 0, len(listResp.SelfServiceCategories))
	for _, c := range listResp.SelfServiceCategories {
		names = append(names, c.Name)
	}
	require.Contains(t, names, "💼 Engineering")

	// rename
	var patchResp patchSelfServiceCategoriesResponse
	s.DoJSON("PATCH", fmt.Sprintf("%s/%d", path, catID), map[string]any{"name": "💼 Engineering tools"}, http.StatusOK, &patchResp)
	require.Equal(t, catID, patchResp.SelfServiceCategory.ID)
	require.Equal(t, "💼 Engineering tools", patchResp.SelfServiceCategory.Name)
	editActID := s.lastActivityMatches(fleet.ActivityTypeEditedSelfServiceCategory{}.ActivityName(),
		`{"self_service_category_name":"💼 Engineering tools","team_id":0,"team_name":null,"fleet_id":0,"fleet_name":null}`, 0)

	// patch with the same name is a no-op — no new activity emitted
	s.DoJSON("PATCH", fmt.Sprintf("%s/%d", path, catID), map[string]any{"name": "💼 Engineering tools"}, http.StatusOK, &patchResp)
	require.Equal(t, editActID, s.lastActivityMatches(fleet.ActivityTypeEditedSelfServiceCategory{}.ActivityName(), "", 0),
		"no-op PATCH should not emit a new activity")

	// delete + activity
	s.Do("DELETE", fmt.Sprintf("%s/%d", path, catID), nil, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityTypeDeletedSelfServiceCategory{}.ActivityName(),
		`{"self_service_category_name":"💼 Engineering tools","team_id":0,"team_name":null,"fleet_id":0,"fleet_name":null}`, 0)

	// post without name returns 422
	s.Do("POST", path, map[string]any{"fleet_id": 0}, http.StatusUnprocessableEntity)

	// whitespace-only name returns 422
	s.Do("POST", path, map[string]any{"fleet_id": 0, "name": "   "}, http.StatusUnprocessableEntity)

	// post without fleet_id returns 422
	s.Do("POST", path, map[string]any{"name": "no-fleet"}, http.StatusUnprocessableEntity)

	// get without fleet_id returns 422 — same error as POST for consistency
	s.Do("GET", path, nil, http.StatusUnprocessableEntity)

	// get unknown fleet_id returns 404
	s.Do("GET", path, nil, http.StatusNotFound, "fleet_id", "9999999")

	// patch without name returns 422 — use a fixture for this since the no-op
	// PATCH above leaves catID deleted
	var fixture addSelfServiceCategoriesResponse
	s.DoJSON("POST", path, map[string]any{"fleet_id": 0, "name": "patch-target"}, http.StatusOK, &fixture)
	s.Do("PATCH", fmt.Sprintf("%s/%d", path, fixture.SelfServiceCategory.ID), map[string]any{}, http.StatusUnprocessableEntity)

	// 255-rune name succeeds; 256-rune name returns 422
	s.Do("POST", path, map[string]any{"fleet_id": 0, "name": strings.Repeat("a", 255)}, http.StatusOK)
	s.Do("POST", path, map[string]any{"fleet_id": 0, "name": strings.Repeat("a", 256)}, http.StatusUnprocessableEntity)

	// duplicate across fleets allowed
	var teamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: "SelfServiceCategoryCrossFleet"}, http.StatusOK, &teamResp)
	t.Cleanup(func() { require.NoError(t, s.ds.DeleteTeam(context.Background(), teamResp.Team.ID)) })

	var cf0, cf1 addSelfServiceCategoriesResponse
	s.DoJSON("POST", path, map[string]any{"fleet_id": 0, "name": "CrossFleet"}, http.StatusOK, &cf0)
	s.DoJSON("POST", path, map[string]any{"fleet_id": teamResp.Team.ID, "name": "CrossFleet"}, http.StatusOK, &cf1)
	require.NotEqual(t, cf0.SelfServiceCategory.ID, cf1.SelfServiceCategory.ID)
	s.lastActivityMatches(fleet.ActivityTypeAddedSelfServiceCategory{}.ActivityName(),
		fmt.Sprintf(`{"self_service_category_name":"CrossFleet","team_id":%d,"team_name":%q,"fleet_id":%d,"fleet_name":%q}`, teamResp.Team.ID, teamResp.Team.Name, teamResp.Team.ID, teamResp.Team.Name), 0)

	// list scoped to the new fleet returns its seeded defaults plus the added category
	s.DoJSON("GET", path, nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(teamResp.Team.ID))
	require.Len(t, listResp.SelfServiceCategories, len(fleet.DefaultSelfServiceCategoryNames)+1)
	newFleetNames := make([]string, 0, len(listResp.SelfServiceCategories))
	for _, c := range listResp.SelfServiceCategories {
		newFleetNames = append(newFleetNames, c.Name)
	}
	require.ElementsMatch(t, append(fleet.DefaultSelfServiceCategoryNames, "CrossFleet"), newFleetNames)

	// unknown fleet_id returns 404
	s.Do("POST", path, map[string]any{"fleet_id": 9999999, "name": "ghost"}, http.StatusNotFound)

	// categories still surface as strings on existing endpoints
	cat1, err := s.ds.NewSoftwareCategory(ctx, 0, "smoke-test-cat-1")
	require.NoError(t, err)
	cat2, err := s.ds.NewSoftwareCategory(ctx, 0, "smoke-test-cat-2")
	require.NoError(t, err)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		TeamID:        new(uint(0)),
		Filename:      "dummy_installer.pkg",
		InstallScript: "echo install",
	}, http.StatusOK, "")

	titleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	titlePath := fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID)

	// linking categories happens via PATCH, not upload
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TeamID:     new(uint(0)),
		TitleID:    titleID,
		Categories: []string{cat1.Name, cat2.Name},
	}, http.StatusOK, "")

	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", titlePath, nil, http.StatusOK, &titleResp, "team_id", "0")
	require.ElementsMatch(t, []string{cat1.Name, cat2.Name}, titleResp.SoftwareTitle.SoftwarePackage.Categories)

	// deleting a category cascades the link
	s.Do("DELETE", fmt.Sprintf("%s/%d", path, cat1.ID), nil, http.StatusNoContent)
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", titlePath, nil, http.StatusOK, &titleResp, "team_id", "0")
	require.ElementsMatch(t, []string{cat2.Name}, titleResp.SoftwareTitle.SoftwarePackage.Categories)

	// deleting a team cascades to its categories
	var deletableResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: "SelfServiceCategoryDeleteScope"}, http.StatusOK, &deletableResp)
	seeded, err := s.ds.ListSoftwareCategories(ctx, deletableResp.Team.ID)
	require.NoError(t, err)
	require.Len(t, seeded, len(fleet.DefaultSelfServiceCategoryNames))
	require.NoError(t, s.ds.DeleteTeam(ctx, deletableResp.Team.ID))
	leftover, err := s.ds.ListSoftwareCategories(ctx, deletableResp.Team.ID)
	require.NoError(t, err)
	require.Empty(t, leftover, "categories should be cleaned up when a team is deleted")
}

func (s *integrationEnterpriseTestSuite) TestDeviceSelfServiceCategories() {
	t := s.T()
	ctx := context.Background()

	getCats := func(token string, expectedStatus int) []fleet.SoftwareCategory {
		resp := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/software/self_service_categories", token), nil, expectedStatus)
		defer resp.Body.Close()
		if expectedStatus != http.StatusOK {
			return nil
		}
		var out getDeviceSelfServiceCategoriesResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
		return out.SelfServiceCategories
	}
	names := func(cats []fleet.SoftwareCategory) []string {
		out := make([]string, 0, len(cats))
		for _, c := range cats {
			out = append(out, c.Name)
		}
		return out
	}

	// bad device token -> 401
	getCats("not-a-real-token", http.StatusUnauthorized)

	// a team host sees its own fleet's categories, scoped away from another fleet's.
	// (every fleet also has the default seeded categories, so assert on unique markers.)
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-team"})
	require.NoError(t, err)
	teamCatName := t.Name() + "-team-cat"
	_, err = s.ds.NewSoftwareCategory(ctx, team.ID, teamCatName)
	require.NoError(t, err)
	noTeamCatName := t.Name() + "-noteam-cat"
	_, err = s.ds.NewSoftwareCategory(ctx, 0, noTeamCatName)
	require.NoError(t, err)

	teamHost := createOrbitEnrolledHost(t, "ubuntu", "ssc-team", s.ds)
	teamToken := "ssc-team-tok" //nolint:gosec // G101: test value only
	createDeviceTokenForHost(t, s.ds, teamHost.ID, teamToken)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID})))

	teamCats := names(getCats(teamToken, http.StatusOK))
	require.Contains(t, teamCats, teamCatName)
	require.NotContains(t, teamCats, noTeamCatName)

	// a no-team host sees the no-team category, not the team's
	noTeamHost := createOrbitEnrolledHost(t, "ubuntu", "ssc-noteam", s.ds)
	noTeamToken := "ssc-noteam-tok" //nolint:gosec // G101: test value only
	createDeviceTokenForHost(t, s.ds, noTeamHost.ID, noTeamToken)
	noTeamCats := names(getCats(noTeamToken, http.StatusOK))
	require.Contains(t, noTeamCats, noTeamCatName)
	require.NotContains(t, noTeamCats, teamCatName)
}

func (s *integrationEnterpriseTestSuite) TestSelfServiceHostVitalsExcludeAnyLabel() {
	t := s.T()
	ctx := context.Background()

	excludedHost := createOrbitEnrolledHost(t, "ubuntu", "hv-excluded", s.ds)
	excludedToken := "hv_excluded_token" // #nosec G101 -- device auth token for a test host, not a credential
	createDeviceTokenForHost(t, s.ds, excludedHost.ID, excludedToken)

	allowedHost := createOrbitEnrolledHost(t, "ubuntu", "hv-allowed", s.ds)
	allowedToken := "hv_allowed_token" // #nosec G101 -- device auth token for a test host, not a credential
	createDeviceTokenForHost(t, s.ds, allowedHost.ID, allowedToken)

	// Seed IdP data: both hosts have an end user, but only excludedHost's user is in the group the
	// software excludes. Ids are assigned by the database and names are scoped to this test so the
	// fixture can't collide with the other SCIM fixtures in this suite.
	groupName := "hv-excluded-group-" + t.Name()
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		excludedUserRes, err := db.ExecContext(ctx,
			"INSERT INTO scim_users (user_name, department) VALUES (?, ?)",
			"hv-excluded-user-"+t.Name(), "dept")
		if err != nil {
			return err
		}
		excludedUserID, err := excludedUserRes.LastInsertId()
		if err != nil {
			return err
		}
		allowedUserRes, err := db.ExecContext(ctx,
			"INSERT INTO scim_users (user_name, department) VALUES (?, ?)",
			"hv-allowed-user-"+t.Name(), "dept")
		if err != nil {
			return err
		}
		allowedUserID, err := allowedUserRes.LastInsertId()
		if err != nil {
			return err
		}
		groupRes, err := db.ExecContext(ctx,
			"INSERT INTO scim_groups (display_name) VALUES (?)", groupName)
		if err != nil {
			return err
		}
		groupID, err := groupRes.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			"INSERT INTO scim_user_group (scim_user_id, group_id) VALUES (?, ?)",
			excludedUserID, groupID,
		); err != nil {
			return err
		}
		_, err = db.ExecContext(ctx,
			"INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?), (?, ?)",
			excludedHost.ID, excludedUserID, allowedHost.ID, allowedUserID,
		)
		return err
	})

	// Create the host vitals label through the API, exactly as the UI/GitOps would.
	var labelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
		Name: "hv-idp-group-" + t.Name(),
		Criteria: &fleet.HostVitalCriteria{
			Vital: new("end_user_idp_group"),
			Value: new(groupName),
		}}, http.StatusOK, &labelResp)
	require.NotNil(t, labelResp.Label)
	require.Equal(t, fleet.LabelMembershipTypeHostVitals, labelResp.Label.LabelMembershipType)

	label, _, err := s.ds.Label(ctx, labelResp.Label.Label.ID, fleet.TeamFilter{User: test.UserAdmin})
	require.NoError(t, err)
	_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(ctx, label)
	require.NoError(t, err)

	hostsInLabel, err := s.ds.ListHostsInLabel(ctx, fleet.TeamFilter{User: test.UserAdmin}, label.ID, fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hostsInLabel, 1)
	require.Equal(t, excludedHost.ID, hostsInLabel[0].ID)

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install",
		Filename:         "ruby.deb",
		Title:            "ruby",
		SelfService:      true,
		LabelsExcludeAny: []string{label.Name},
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payload.Title, "deb_packages")

	selfServiceTitles := func(token string) []string {
		var resp getDeviceSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/device/%s/software", token), nil, http.StatusOK,
			&resp, "self_service", "true")
		names := make([]string, 0, len(resp.Software))
		for _, sw := range resp.Software {
			names = append(names, sw.Name)
		}
		return names
	}

	// The host that is not in the IdP group sees the software in Self-service and can install it.
	require.Contains(t, selfServiceTitles(allowedToken), payload.Title)
	s.DoRawNoAuth("POST",
		fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", allowedToken, titleID),
		nil, http.StatusAccepted)

	// The host that is in the group sees nothing and is refused the install.
	require.NotContains(t, selfServiceTitles(excludedToken), payload.Title)
	res := s.DoRawNoAuth("POST",
		fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", excludedToken, titleID),
		nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body),
		"Couldn't install. Host isn't member of the labels defined for this software title.")
}

func (s *integrationEnterpriseTestSuite) TestAIToolFilter() {
	t := s.T()
	ctx := t.Context()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	newHost := func(name string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        t.Name() + name + ".local",
			Platform:        "darwin",
		})
		require.NoError(t, err)
		return h
	}
	teamHost := newHost("team")
	noTeamHost := newHost("noteam")
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{teamHost.ID})))

	_, err = s.ds.UpdateHostSoftware(ctx, teamHost.ID, []fleet.Software{
		{Name: "AIToolClaude", Version: "1.0", Source: "apps", BundleIdentifier: "com.example.aitoolclaude", AITool: new(true)},
		{Name: "AIToolSlack", Version: "1.0", Source: "apps", BundleIdentifier: "com.example.aitoolslack"},
		{Name: "aitool-codex", Version: "1.0", Source: "ai_clis", AITool: new(true)},
	})
	require.NoError(t, err)
	_, err = s.ds.UpdateHostSoftware(ctx, noTeamHost.ID, []fleet.Software{
		{Name: "AIToolSlack", Version: "1.0", Source: "apps", BundleIdentifier: "com.example.aitoolslack"},
		{Name: "aitool-github", Version: "0.1", Source: "mcp_servers", AITool: new(true)},
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	token := "ai_tool_premium_token"
	createDeviceTokenForHost(t, s.ds, teamHost.ID, token)

	// flag returns an item's ai_tool, which Premium sets on every item.
	flag := func(name string, aiTool *bool) bool {
		require.NotNil(t, aiTool, "ai_tool missing on %q", name)
		return *aiTool
	}

	// Restrict the cross-host lists to this test's software, which all match "aitool".
	listTitles := func(params ...string) (map[string]bool, int) {
		var resp listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &resp, append(params, "query", "aitool")...)
		got := make(map[string]bool, len(resp.SoftwareTitles))
		for _, title := range resp.SoftwareTitles {
			got[title.Name] = flag(title.Name, title.AITool)
		}
		return got, resp.Count
	}
	listVersions := func(params ...string) (map[string]bool, int) {
		var resp listSoftwareVersionsResponse
		s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &resp, append(params, "query", "aitool")...)
		var countResp countSoftwareResponse
		s.DoJSON("GET", "/api/latest/fleet/software/count", nil, http.StatusOK, &countResp, append(params, "query", "aitool")...)
		require.Equal(t, resp.Count, countResp.Count)
		got := make(map[string]bool, len(resp.Software))
		for _, sw := range resp.Software {
			got[sw.Name] = flag(sw.Name, sw.AITool)
		}
		return got, resp.Count
	}
	hostSoftware := func(sw []*fleet.HostSoftwareWithInstaller) map[string]bool {
		got := make(map[string]bool, len(sw))
		for _, s := range sw {
			got[s.Name] = flag(s.Name, s.AITool)
		}
		return got
	}
	listHostSoftware := func(params ...string) (map[string]bool, int) {
		var resp getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", teamHost.ID), nil, http.StatusOK, &resp, params...)
		return hostSoftware(resp.Software), resp.Count
	}
	listDeviceSoftware := func(params ...string) (map[string]bool, int) {
		var resp getDeviceSoftwareResponse
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/software", nil, http.StatusOK, params...)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		require.NoError(t, res.Body.Close())
		return hostSoftware(resp.Software), resp.Count
	}

	allFlags := map[string]bool{"AIToolClaude": true, "AIToolSlack": false, "aitool-codex": true, "aitool-github": true}
	teamHostFlags := map[string]bool{"AIToolClaude": true, "AIToolSlack": false, "aitool-codex": true}
	for _, c := range []struct {
		name string
		list func(params ...string) (map[string]bool, int)
		all  map[string]bool
	}{
		{"titles", listTitles, allFlags},
		{"versions", listVersions, allFlags},
		{"host", listHostSoftware, teamHostFlags},
		{"device", listDeviceSoftware, teamHostFlags},
	} {
		got, count := c.list()
		require.Equal(t, c.all, got, c.name)
		require.Equal(t, len(c.all), count, c.name)

		want := make(map[string]bool)
		for name, aiTool := range c.all {
			if aiTool {
				want[name] = true
			}
		}
		got, count = c.list("ai_tool", "true")
		require.Equal(t, want, got, c.name)
		require.Equal(t, len(want), count, c.name)

		got, count = c.list("ai_tool", "true", "source", "apps")
		require.Equal(t, map[string]bool{"AIToolClaude": true}, got, c.name)
		require.Equal(t, 1, count, c.name)

		// Paging through the filtered list returns every match once.
		paged := make(map[string]bool)
		for page := 0; page < len(want); page++ {
			got, count = c.list("ai_tool", "true", "per_page", "1", "page", strconv.Itoa(page), "order_key", "name")
			require.Len(t, got, 1, c.name)
			require.Equal(t, len(want), count, c.name)
			maps.Copy(paged, got)
		}
		require.Equal(t, want, paged, c.name)

		for _, source := range []string{"mcp_servers", "ai_clis", "ai_skills", "apps,mcp_servers"} {
			c.list("source", source)
		}
	}

	got, count := listTitles("ai_tool", "true", "team_id", fmt.Sprint(tm.ID))
	require.Equal(t, map[string]bool{"AIToolClaude": true, "aitool-codex": true}, got)
	require.Equal(t, 2, count)
	got, count = listVersions("ai_tool", "true", "team_id", fmt.Sprint(tm.ID))
	require.Equal(t, map[string]bool{"AIToolClaude": true, "aitool-codex": true}, got)
	require.Equal(t, 2, count)

	res := s.Do("GET", "/api/latest/fleet/software/titles", nil, http.StatusUnprocessableEntity, "source", "ai_tool")
	require.Contains(t, extractServerErrorText(res.Body), fmt.Sprintf(fleet.InvalidSoftwareSourceErrMsg, "ai_tool"))

	// By-id endpoints and host details carry the flag too.
	var titlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &titlesResp, "query", "aitool")
	for _, listed := range titlesResp.SoftwareTitles {
		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", listed.ID), nil, http.StatusOK, &titleResp)
		require.Equal(t, flag(listed.Name, listed.AITool), flag(listed.Name, titleResp.SoftwareTitle.AITool), listed.Name)
	}
	var versionsResp listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "aitool")
	for _, listed := range versionsResp.Software {
		var versionResp getSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", listed.ID), nil, http.StatusOK, &versionResp)
		require.Equal(t, flag(listed.Name, listed.AITool), flag(listed.Name, versionResp.Software.AITool), listed.Name)
	}
	teamHostAITool := make(map[string]*bool, len(teamHostFlags))
	for name, aiTool := range teamHostFlags {
		teamHostAITool[name] = &aiTool
	}
	get := func(path string, params ...string) *http.Response {
		return s.Do("GET", path, nil, http.StatusOK, params...)
	}
	require.Equal(t, teamHostAITool, aiToolByName(t, get(fmt.Sprintf("/api/latest/fleet/hosts/%d", teamHost.ID)), "host", "software"))
	require.Equal(t, teamHostAITool, aiToolByName(t, get("/api/latest/fleet/hosts/identifier/"+teamHost.UUID), "host", "software"))
	require.Equal(t, teamHostAITool, aiToolByName(t,
		s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK), "host", "software"))
	var hostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "populate_software", "true", "query", teamHost.Hostname)
	require.Len(t, hostsResp.Hosts, 1)
	populated := make(map[string]bool, len(hostsResp.Hosts[0].Software))
	for _, sw := range hostsResp.Hosts[0].Software {
		populated[sw.Name] = flag(sw.Name, sw.AITool)
	}
	require.Equal(t, teamHostFlags, populated)
}
