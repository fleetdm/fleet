package msrc_input

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProdXML(t *testing.T) {
	t.Run("ProductBranchXML", func(t *testing.T) {
		t.Run("#WindowsProducts", func(t *testing.T) {
			windowsBranch := ProductBranchXML{
				Type: "Product Family", Name: "Windows",
				Products: []ProductXML{
					{ProductID: "11572", Name: "Windows Server 2019 (Server Core installation)"},
					{ProductID: "11712", Name: "Windows 10 Version 1909 for 32-bit Systems"},
				},
			}

			esuBranch := ProductBranchXML{
				Type: "Product Family", Name: "ESU",
				Products: []ProductXML{
					{ProductID: "10051", Name: "Windows Server 2008 R2 for x64-based Systems Service Pack 1"},
					{ProductID: "10049", Name: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)"},
				},
			}

			devToolsBranch := ProductBranchXML{
				Type: "Product Family", Name: "Developer Tools",
				Products: []ProductXML{
					{ProductID: "11676-11927", Name: "Microsoft .NET Framework 3.5 AND 4.8 on Windows 11 for ARM64-based Systems"},
					{ProductID: "9495-10047", Name: "Microsoft .NET Framework 3.5.1 on Windows 7 for 32-bit Systems Service Pack 1"},
					{ProductID: "9495-10048", Name: "Microsoft .NET Framework 3.5.1 on Windows 7 for x64-based Systems Service Pack 1"},
					{ProductID: "9495-10051", Name: "Microsoft .NET Framework 3.5.1 on Windows Server 2008 R2 for x64-based Systems Service Pack 1"},
				},
			}

			rootBranch := &ProductBranchXML{
				Type: "Vendor", Name: "Microsoft",
				Branches: []ProductBranchXML{
					windowsBranch,
					esuBranch,
					devToolsBranch,
				},
			}

			winProds := rootBranch.WindowsProducts()
			require.Subset(t, winProds, windowsBranch.Products)
			require.Subset(t, winProds, esuBranch.Products)
			require.NotSubset(t, winProds, devToolsBranch.Products)
		})
	})
}
