package parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSecurityBulletin(t *testing.T) {
	t.Run("#Merge", func(t *testing.T) {
		t.Run("fails if product names don't match", func(t *testing.T) {
			a := NewSecurityBulletin("Windows 10")
			b := NewSecurityBulletin("Windows 11")
			require.Error(t, a.Merge(b))
		})

		t.Run("with empty bulletins", func(t *testing.T) {
			a := NewSecurityBulletin("Windows 10")
			b := NewSecurityBulletin("Windows 10")
			require.NoError(t, a.Merge(b))
		})

		t.Run(".Products", func(t *testing.T) {
			a := NewSecurityBulletin("Windows 10")
			a.Products["123"] = "Windows 10 A"
			a.Products["456"] = "Windows 10 B"

			b := NewSecurityBulletin("Windows 10")
			a.Products["780"] = "Windows 10 C"
			a.Products["980"] = "Windows 10 D"

			require.NoError(t, a.Merge(b))

			require.Equal(t, a.Products["123"], NewProductFromFullName("Windows 10 A"))
			require.Equal(t, a.Products["456"], NewProductFromFullName("Windows 10 B"))
			require.Equal(t, a.Products["780"], NewProductFromFullName("Windows 10 C"))
			require.Equal(t, a.Products["980"], NewProductFromFullName("Windows 10 D"))
		})

		t.Run(".Vulnerabities", func(t *testing.T) {
			cve1 := NewVulnerability(ptr.Int64(123))
			cve1.ProductIDs = map[string]bool{"111": true, "222": true}
			cve1.RemediatedBy = map[uint]bool{1: true}

			cve2 := NewVulnerability(ptr.Int64(456))
			cve2.ProductIDs = map[string]bool{"333": true, "444": true}
			cve2.RemediatedBy = map[uint]bool{2: true}

			cve3 := NewVulnerability(ptr.Int64(555))
			cve3.ProductIDs = map[string]bool{"aaa": true, "bbb": true}
			cve3.RemediatedBy = map[uint]bool{3: true}

			cve4 := NewVulnerability(ptr.Int64(777))
			cve4.ProductIDs = map[string]bool{"ccc": true, "ddd": true}
			cve3.RemediatedBy = map[uint]bool{4: true}

			a := NewSecurityBulletin("Windows 10")
			a.Vulnerabities["cve-1"] = cve1
			a.Vulnerabities["cve-2"] = cve2

			b := NewSecurityBulletin("Windows 10")
			b.Vulnerabities["cve-3"] = cve3
			b.Vulnerabities["cve-4"] = cve4

			require.NoError(t, a.Merge(b))

			require.Equal(t, *a.Vulnerabities["cve-1"].PublishedEpoch, int64(123))
			require.Equal(t, *a.Vulnerabities["cve-2"].PublishedEpoch, int64(456))
			require.Equal(t, *a.Vulnerabities["cve-3"].PublishedEpoch, int64(555))
			require.Equal(t, *a.Vulnerabities["cve-4"].PublishedEpoch, int64(777))

			require.Equal(t, a.Vulnerabities["cve-1"].ProductIDs, cve1.ProductIDs)
			require.Equal(t, a.Vulnerabities["cve-2"].ProductIDs, cve2.ProductIDs)
			require.Equal(t, a.Vulnerabities["cve-3"].ProductIDs, cve3.ProductIDs)
			require.Equal(t, a.Vulnerabities["cve-4"].ProductIDs, cve4.ProductIDs)

			require.Equal(t, a.Vulnerabities["cve-1"].RemediatedBy, cve1.RemediatedBy)
			require.Equal(t, a.Vulnerabities["cve-2"].RemediatedBy, cve2.RemediatedBy)
			require.Equal(t, a.Vulnerabities["cve-3"].RemediatedBy, cve3.RemediatedBy)
			require.Equal(t, a.Vulnerabities["cve-4"].RemediatedBy, cve4.RemediatedBy)
		})

		t.Run(".VendorFixes", func(t *testing.T) {
			vf1 := NewVendorFix("")
			vf1.ProductIDs = map[string]bool{"111": true, "222": true}
			vf1.Supersedes = ptr.Uint(1)

			vf2 := NewVendorFix("")
			vf2.ProductIDs = map[string]bool{"333": true, "444": true}
			vf2.Supersedes = ptr.Uint(2)

			a := NewSecurityBulletin("Windows 10")
			a.VendorFixes[1] = vf1

			b := NewSecurityBulletin("Windows 10")
			b.VendorFixes[2] = vf2

			require.NoError(t, a.Merge(b))

			require.Equal(t, *a.VendorFixes[1].Supersedes, uint(1))
			require.Equal(t, *a.VendorFixes[2].Supersedes, uint(2))

			require.Equal(t, a.VendorFixes[1].ProductIDs, vf1.ProductIDs)
			require.Equal(t, a.VendorFixes[2].ProductIDs, vf2.ProductIDs)
		})
	})
}
