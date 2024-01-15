package msrc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/io"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer(t *testing.T) {
	op := fleet.OperatingSystem{
		Name:          "Microsoft Windows 11 Enterprise Evaluation",
		Version:       "21H2",
		Arch:          "64-bit",
		KernelVersion: "10.0.22000.795",
		Platform:      "windows",
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
			require.False(t, patched(op, b, b.Vulnerabities["cve-123"], pIDs, nil))
		})

		t.Run("directly remediated", func(t *testing.T) {
			b := parsed.NewSecurityBulletin(prod.Name())
			b.Products["123"] = prod

			vuln := parsed.NewVulnerability(nil)
			vuln.RemediatedBy[123] = true
			b.Vulnerabities["cve-123"] = vuln

			pIDs := map[string]bool{"123": true}

			updates := []fleet.WindowsUpdate{
				{KBID: 123},
				{KBID: 456},
			}

			require.True(t, patched(op, b, b.Vulnerabities["cve-123"], pIDs, updates))
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

			updates := []fleet.WindowsUpdate{
				{KBID: 789},
			}

			require.True(t, patched(op, b, b.Vulnerabities["cve-123"], pIDs, updates))
		})

		t.Run("remediated by a cumulative update", func(t *testing.T) {
			b := parsed.NewSecurityBulletin(prod.Name())
			b.Products["123"] = prod
			pIDs := map[string]bool{"123": true}

			vuln := parsed.NewVulnerability(nil)
			vuln.RemediatedBy[456] = true
			b.Vulnerabities["cve-123"] = vuln

			vfA := parsed.NewVendorFix("10.0.22000.796")
			vfA.Supersedes = ptr.Uint(123)
			vfA.ProductIDs["123"] = true
			b.VendorFixes[456] = vfA

			vfB := parsed.NewVendorFix("10.0.22000.796")
			vfB.Supersedes = ptr.Uint(456)
			vfB.ProductIDs["123"] = true
			b.VendorFixes[789] = vfA

			updates := []fleet.WindowsUpdate{
				{KBID: 789},
			}

			require.True(t, patched(op, b, b.Vulnerabities["cve-123"], pIDs, updates))
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
			errMessage: "comparing different product versions",
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
