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
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestIsVulnPatched(t *testing.T) {
	op := fleet.OperatingSystem{
		Name:           "Microsoft Windows 11 Enterprise Evaluation",
		DisplayVersion: "21H2",
		Version:        "10.0.22000.795",
		Arch:           "64-bit",
		KernelVersion:  "10.0.22000.795",
		Platform:       "windows",
	}
	prod := parsed.NewProductFromOS(op)

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

func TestIsOSVulnerable(t *testing.T) {
	b := parsed.SecurityBulletin{
		Vulnerabities: map[string]parsed.Vulnerability{
			"CVE-Win11": {
				RemediatedBy: map[uint]bool{
					123: true,
				},
			},
			"CVE-too-many-parts": {
				RemediatedBy: map[uint]bool{
					124: true,
				},
			},
			"CVE-too-few-parts": {
				RemediatedBy: map[uint]bool{
					125: true,
				},
			},
			"CVE-empty-feed-version": {
				RemediatedBy: map[uint]bool{
					126: true,
				},
			},
			"CVE-wrong-build-version": {
				RemediatedBy: map[uint]bool{
					127: true,
				},
			},
			"CVE-multiple-fixed-builds": {
				RemediatedBy: map[uint]bool{
					128: true,
				},
			},
		},
		VendorFixes: map[uint]parsed.VendorFix{
			123: {
				FixedBuilds: []string{"10.0.22000.794"},
				ProductIDs:  map[string]bool{"123": true},
			},
			124: {
				FixedBuilds: []string{"10.0.22000.794.9999"},
				ProductIDs:  map[string]bool{"123": true},
			},
			125: {
				FixedBuilds: []string{"10.0.22000"},
				ProductIDs:  map[string]bool{"123": true},
			},
			126: {
				FixedBuilds: []string{""},
				ProductIDs:  map[string]bool{"123": true},
			},
			127: {
				FixedBuilds: []string{"10.0.22621.795"}, // bug in the feed
				ProductIDs:  map[string]bool{"123": true},
			},
			128: {
				FixedBuilds: []string{"10.0.22000.794", "10.0.22631.795"},
				ProductIDs:  map[string]bool{"123": true, "124": true},
			},
		},
	}

	tc := []struct {
		name         string
		feed         string
		os           string
		isVulnerable bool
		resolvedIn   string
	}{
		{
			name:         "os version equals fixed build",
			feed:         "CVE-Win11",
			os:           "10.0.22000.795",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "os version greater than fixed build",
			feed:         "CVE-Win11",
			os:           "10.0.22000.796",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "os version less than fixed build",
			feed:         "CVE-Win11",
			os:           "10.0.22000.793",
			isVulnerable: true,
			resolvedIn:   "10.0.22000.794",
		},
		{
			name:         "too many parts in feed version",
			feed:         "CVE-too-many-parts",
			os:           "10.0.22000.794",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "too many parts in os version",
			feed:         "CVE-Win11",
			os:           "10.0.22000.794.9999",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "too few parts in feed version",
			feed:         "CVE-too-few-parts",
			os:           "10.0.22000.794",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "empty feed version",
			feed:         "CVE-empty-feed-version",
			os:           "10.0.22000.794",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "comparing different product versions",
			feed:         "CVE-wrong-build-version",
			os:           "10.0.22000.794",
			isVulnerable: false,
			resolvedIn:   "",
		},
		{
			name:         "vulnerable with multiple fixed builds",
			feed:         "CVE-multiple-fixed-builds",
			os:           "10.0.22000.793",
			isVulnerable: true,
			resolvedIn:   "10.0.22000.794",
		},
		{
			name:         "not vulnerable with multiple fixed builds",
			feed:         "CVE-multiple-fixed-builds",
			os:           "10.0.22000.794",
			isVulnerable: false,
			resolvedIn:   "",
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			isVuln, resolvedIn := isOSVulnerable(c.os, &b, b.Vulnerabities[c.feed], map[string]bool{"123": true}, c.feed, log.NewNopLogger())
			require.Equal(t, c.isVulnerable, isVuln)
			require.Equal(t, c.resolvedIn, resolvedIn)
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
			fos := fleet.OperatingSystem{
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

			results, err := Analyze(ctx, ds, fos, vulnPath, true, log.NewNopLogger())
			require.NoError(t, err)
			require.ElementsMatch(t, c.vulns, results)
		})
	}
}
