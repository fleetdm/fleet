package xml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProdXML(t *testing.T) {
	t.Run("ProductBranchXML", func(t *testing.T) {
		t.Run("#WindowsProducts", func(t *testing.T) {
			windowsBranch := ProductBranch{
				Type: "Product Family", Name: "Windows",
				Products: []Product{
					{ProductID: "11572", FullName: "Windows Server 2019 (Server Core installation)"},
					{ProductID: "11712", FullName: "Windows 10 Version 1909 for 32-bit Systems"},
				},
			}

			esuBranch := ProductBranch{
				Type: "Product Family", Name: "ESU",
				Products: []Product{
					{ProductID: "10051", FullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1"},
					{ProductID: "10049", FullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)"},
				},
			}

			devToolsBranch := ProductBranch{
				Type: "Product Family", Name: "Developer Tools",
				Products: []Product{
					{ProductID: "11676-11927", FullName: "Microsoft .NET Framework 3.5 AND 4.8 on Windows 11 for ARM64-based Systems"},
					{ProductID: "9495-10047", FullName: "Microsoft .NET Framework 3.5.1 on Windows 7 for 32-bit Systems Service Pack 1"},
					{ProductID: "9495-10048", FullName: "Microsoft .NET Framework 3.5.1 on Windows 7 for x64-based Systems Service Pack 1"},
					{ProductID: "9495-10051", FullName: "Microsoft .NET Framework 3.5.1 on Windows Server 2008 R2 for x64-based Systems Service Pack 1"},
				},
			}

			rootBranch := &ProductBranch{
				Type: "Vendor", Name: "Microsoft",
				Branches: []ProductBranch{
					windowsBranch,
					esuBranch,
					devToolsBranch,
				},
			}

			winProds := rootBranch.WinProducts()
			require.Subset(t, winProds, windowsBranch.Products)
			require.Subset(t, winProds, esuBranch.Products)
			require.NotSubset(t, winProds, devToolsBranch.Products)
		})
	})
}
