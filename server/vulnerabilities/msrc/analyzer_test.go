package msrc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/stretchr/testify/require"
)

func TestPatched(t *testing.T) {
	op := fleet.OperatingSystem{
		Name:           "Microsoft Windows 11 Enterprise Evaluation",
		DisplayVersion: "21H2",
		Version:        "10.0.22000.795",
		Arch:           "64-bit",
		KernelVersion:  "10.0.22000.795",
		Platform:       "windows",
	}
	prod := parsed.NewProductFromOS(op)

	t.Run("#patched", func(t *testing.T) {
		// TODO: this tests a nil vulnerability, which should never happen
		// maybe return an error instead?
		t.Run("no updates", func(t *testing.T) {
			b := parsed.NewSecurityBulletin(prod.Name())
			b.Products["123"] = prod
			b.Vulnerabities["cve-123"] = parsed.NewVulnerability(nil)
			pIDs := map[string]bool{"123": true}
			isPatched, _ := isVulnPatched(op, b, b.Vulnerabities["cve-123"], pIDs)
			require.False(t, isPatched)
		})

		t.Run("remediated by build", func(t *testing.T) {
			b := parsed.NewSecurityBulletin(prod.Name())
			b.Products["123"] = prod
			pIDs := map[string]bool{"123": true}

			vuln := parsed.NewVulnerability(nil)
			vuln.RemediatedBy[456] = true
			b.Vulnerabities["cve-123"] = vuln

			vfA := parsed.NewVendorFix("10.0.22000.794")
			vfA.Supersedes = ptr.Uint(123)
			vfA.ProductIDs["123"] = true
			b.VendorFixes[456] = vfA

			isPatched, _ := isVulnPatched(op, b, b.Vulnerabities["cve-123"], pIDs)
			require.True(t, isPatched)
		})
	})

	t.Run("#loadBulletin", func(t *testing.T) {
		t.Run("dir does not exists", func(t *testing.T) {
			bulletin, err := loadBulletin(op, "over_the_rainbow")
			require.Error(t, err)
			require.Nil(t, bulletin)
		})

		t.Run("returns the latest bulletin", func(t *testing.T) {
			d := time.Now()
			dir := t.TempDir()

			b := parsed.NewSecurityBulletin(prod.Name())
			b.Products["1235"] = prod

			fileName := io.MSRCFileName(b.ProductName, d)
			filePath := filepath.Join(dir, fileName)

			payload, err := json.Marshal(b)
			require.NoError(t, err)

			err = os.WriteFile(filePath, payload, 0o644)
			require.NoError(t, err)

			actual, err := loadBulletin(op, dir)
			require.NoError(t, err)
			require.Equal(t, prod.Name(), actual.ProductName)
		})
	})
}

func TestWinBuildVersionGreaterOrEqual(t *testing.T) {
	tc := []struct {
		name       string
		feed       string
		os         string
		result     bool
		errMessage string
	}{
		{
			name:       "equal",
			feed:       "10.0.22000.795",
			os:         "10.0.22000.795",
			result:     true,
			errMessage: "",
		},
		{
			name:       "greater",
			feed:       "10.0.22000.795",
			os:         "10.0.22000.796",
			result:     true,
			errMessage: "",
		},
		{
			name:       "less",
			feed:       "10.0.22000.795",
			os:         "10.0.22000.794",
			result:     false,
			errMessage: "",
		},
		{
			name:       "too many parts in feed version",
			feed:       "10.0.22000.795.9999",
			os:         "10.0.22000.794",
			result:     false,
			errMessage: "invalid feed version",
		},
		{
			name:       "too many parts in os version",
			feed:       "10.0.22000.795",
			os:         "10.0.22000.794.9999",
			result:     false,
			errMessage: "invalid os version",
		},
		{
			name:       "too few parts in feed version",
			feed:       "10.0.22000",
			os:         "10.0.22000.794",
			result:     false,
			errMessage: "invalid feed version",
		},
		{
			name:       "empty feed version",
			feed:       "",
			os:         "10.0.22000.794",
			result:     false,
			errMessage: "empty feed version",
		},
		{
			name:       "comparing different product versions",
			feed:       "10.0.22000.795",
			os:         "10.0.22621.795",
			result:     false,
			errMessage: "",
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			result, err := winBuildVersionGreaterOrEqual(c.feed, c.os)
			require.Equal(t, c.result, result)
			if c.errMessage != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAnalyze(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	vulnPath := "./testdata"

	tc := []struct {
		name           string
		osName         string
		osVersion      string
		displayVersion string
		vulns          []fleet.OSVulnerability
	}{
		{
			name:           "OS With No Display Version",
			osName:         "Microsoft Windows 11 Enterprise",
			osVersion:      "10.0.22000.2652",
			displayVersion: "",
			vulns: []fleet.OSVulnerability{
				{CVE: "CVE-2024-21320", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21307", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21306", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21305", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20692", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20687", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20683", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21316", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20660", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20658", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20653", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20652", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20674", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20666", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21314", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21313", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21311", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21310", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21309", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20700", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20699", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20698", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20696", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20694", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20691", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20690", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20682", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20681", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20680", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20664", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20663", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20661", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20657", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20654", ResolvedInVersion: ptr.String("10.0.22000.2713"), OSID: 1, Source: fleet.MSRCSource},
			},
		},
		{
			name:           "OS With Display Version",
			osName:         "Microsoft Windows 11 Enterprise 22H2",
			osVersion:      "10.0.22621.2861",
			displayVersion: "22H2",
			vulns: []fleet.OSVulnerability{
				{CVE: "CVE-2024-21320", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21307", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21306", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21305", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20697", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20692", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20687", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20683", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21316", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20660", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20658", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20653", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20652", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20674", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20666", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21314", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21313", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21311", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21310", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21309", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20700", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20699", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20698", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20696", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20694", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20691", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20690", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20682", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20681", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20680", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20664", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20663", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20661", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20657", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20654", ResolvedInVersion: ptr.String("10.0.22621.3007"), OSID: 1, Source: fleet.MSRCSource},
			},
		},
		{
			name:           "OS With Display Version and No Vulnerabilities",
			osName:         "Microsoft Windows 11 Enterprise 22H2",
			osVersion:      "10.0.22621.3007",
			displayVersion: "22H2",
			vulns:          []fleet.OSVulnerability{},
		},
		{
			name:           "Vulnerable 23H2 Version",
			osName:         "Microsoft Windows 11 Enterprise 23H2",
			osVersion:      "10.0.22631.2861",
			displayVersion: "23H2",
			vulns: []fleet.OSVulnerability{
				{CVE: "CVE-2024-20697", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21320", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21307", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21306", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21305", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20692", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20687", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20683", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21316", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20660", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20658", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20653", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20652", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20674", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20666", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21314", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21313", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21311", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21310", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21309", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20700", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20699", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20698", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20696", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20694", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20691", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20690", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20682", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20681", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20680", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20664", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20663", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20661", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20657", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20654", ResolvedInVersion: ptr.String("10.0.22631.3007"), OSID: 1, Source: fleet.MSRCSource},
			},
		},
		{
			name:           "Windows 10 22H2",
			osName:         "Microsoft Windows 10 Enterprise 22H2",
			osVersion:      "10.0.19045.3803",
			displayVersion: "22H2",
			vulns: []fleet.OSVulnerability{
				{CVE: "CVE-2024-21320", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21307", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21306", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21305", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2022-35737", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20692", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20687", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20683", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21316", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20660", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20658", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20653", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20652", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20674", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20666", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21314", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21313", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21311", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-21310", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20700", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20699", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20698", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20696", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20694", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20691", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20690", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20682", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20681", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20680", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20664", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20663", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20661", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20657", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
				{CVE: "CVE-2024-20654", ResolvedInVersion: ptr.String("10.0.19045.3930"), OSID: 1, Source: fleet.MSRCSource},
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			os := fleet.OperatingSystem{
				ID:             uint(1),
				Name:           c.osName,
				DisplayVersion: c.displayVersion,
				Version:        c.osVersion,
				Arch:           "64-bit",
				KernelVersion:  c.osVersion,
				Platform:       "windows",
			}
			ds.ListOSVulnerabilitiesByOSFunc = func(ctx context.Context, osID uint) ([]fleet.OSVulnerability, error) {
				return nil, nil
			}

			ds.DeleteOSVulnerabilitiesFunc = func(ctx context.Context, vulnerabilities []fleet.OSVulnerability) error {
				return nil
			}

			ds.InsertOSVulnerabilitiesFunc = func(ctx context.Context, vulnerabilities []fleet.OSVulnerability, source fleet.VulnerabilitySource) (int64, error) {
				return int64(len(c.vulns)), nil
			}

			results, err := Analyze(ctx, ds, os, vulnPath, true)
			require.NoError(t, err)
			require.ElementsMatch(t, c.vulns, results)
		})
	}
}
