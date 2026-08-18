package mysql

// Drop this file in server/datastore/mysql/ and run:
//   MYSQL_TEST=1 go test ./server/datastore/mysql/ -run TestZZHostVitalsLabelSoftwareScopeRepro -v -count=1

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestZZHostVitalsLabelSoftwareScopeRepro(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := context.Background()

	member := test.NewHost(t, ds, "hvmember", "", "hvmkey", "hvmuuid", time.Now(), test.WithPlatform("darwin"))
	nonMember := test.NewHost(t, ds, "hvnonmember", "", "hvnkey", "hvnuuid", time.Now(), test.WithPlatform("darwin"))
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// A host-vitals label, exactly as an IdP-group label is stored.
	hvLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "Okta - Engineering",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeHostVitals,
		HostVitalsCriteria:  new(json.RawMessage(`{"end_user_idp_group":"Engineering"}`)),
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, hvLabel.LabelMembershipType)

	manualLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name: "Manual control", LabelType: fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	dynLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name: "Dynamic control", LabelType: fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic, Query: "SELECT 1",
	})
	require.NoError(t, err)

	// Membership rows are written identically for all three types
	// (UpdateLabelMembershipByHostCriteria does exactly this INSERT).
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO label_membership (label_id, host_id) VALUES (?,?),(?,?),(?,?)`,
			hvLabel.ID, member.ID, manualLabel.ID, member.ID, dynLabel.ID, member.ID)
		return err
	})

	// Push both hosts' label_updated_at past the labels' created_at so the
	// dynamic-label freshness gate cannot explain any failure below.
	time.Sleep(1200 * time.Millisecond)
	for _, h := range []*fleet.Host{member, nonMember} {
		h.LabelUpdatedAt = time.Now().UTC()
		require.NoError(t, ds.UpdateHost(ctx, h))
	}

	newInstaller := func(name string) uint {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
		require.NoError(t, err)
		id, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "hello", UninstallScript: "bye", InstallerFile: tfr,
			StorageID: name, Filename: name + ".pkg", Title: name, Version: "1.0",
			Source: "apps", UserID: user.ID, BundleIdentifier: "bi-" + name,
			Platform: "darwin", SelfService: true,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		})
		require.NoError(t, err)
		return id
	}

	setScope := func(installerID uint, scope fleet.LabelScope, lbl *fleet.Label) {
		require.NoError(t, setOrUpdateSoftwareInstallerLabelsDB(ctx, ds.writer(ctx), installerID, fleet.LabelIdentsWithScope{
			LabelScope: scope,
			ByName:     map[string]fleet.LabelIdent{lbl.Name: {LabelName: lbl.Name, LabelID: lbl.ID}},
		}, softwareTypeInstaller))
	}

	cases := []struct {
		name  string
		scope fleet.LabelScope
		lbl   *fleet.Label
	}{
		{"exclude_any/host_vitals", fleet.LabelScopeExcludeAny, hvLabel},
		{"exclude_any/manual", fleet.LabelScopeExcludeAny, manualLabel},
		{"exclude_any/dynamic", fleet.LabelScopeExcludeAny, dynLabel},
		{"include_any/host_vitals", fleet.LabelScopeIncludeAny, hvLabel},
		{"include_all/host_vitals", fleet.LabelScopeIncludeAll, hvLabel},
	}

	opts := fleet.HostSoftwareTitleListOptions{
		ListOptions:                fleet.ListOptions{PerPage: 50, OrderKey: "name"},
		IncludeAvailableForInstall: true,
	}

	for i, c := range cases {
		title := c.name[:strings.Index(c.name, "/")] + "-" + c.lbl.Name + "-" + string(rune('a'+i))
		installerID := newInstaller(title)
		setScope(installerID, c.scope, c.lbl)

		scopedMember, err := ds.IsSoftwareInstallerLabelScoped(ctx, installerID, member.ID)
		require.NoError(t, err)
		scopedNon, err := ds.IsSoftwareInstallerLabelScoped(ctx, installerID, nonMember.ID)
		require.NoError(t, err)

		inScope, err := ds.GetIncludedHostIDMapForSoftwareInstaller(ctx, installerID)
		require.NoError(t, err)
		_, memberInMap := inScope[member.ID]
		_, nonInMap := inScope[nonMember.ID]

		listed := false
		sw, _, err := ds.ListHostSoftware(ctx, nonMember, opts)
		require.NoError(t, err)
		for _, s := range sw {
			if s.SoftwarePackage != nil && s.SoftwarePackage.Name == title+".pkg" {
				listed = true
			}
		}

		t.Logf("%-26s installer=%d | IsScoped(member)=%-5v IsScoped(nonMember)=%-5v | hostsInScope{member=%v nonMember=%v} | listedForNonMember=%v",
			c.name, installerID, scopedMember, scopedNon, memberInMap, nonInMap, listed)
	}
}
