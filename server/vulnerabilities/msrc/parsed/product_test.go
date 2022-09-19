package parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestNewProductFromOS(t *testing.T) {
	os := fleet.OperatingSystem{
		Name:          "Microsoft Windows 11 Enterprise Evaluation",
		Version:       "21H2",
		Arch:          "64-bit",
		KernelVersion: "10.0.22000.795",
		Platform:      "windows",
	}

	pA := NewProductFromOS(os)
	pB := NewProductFromFullName("Windows 11 for x64-based Systems")

	require.Equal(t, "Windows 11", pA.Name())
	require.Equal(t, "64-bit", pA.Arch())

	require.True(t, pA.Matches(pB))
}

func TestMatches(t *testing.T) {
	t.Run("from differect products", func(t *testing.T) {
		pA := NewProductFromFullName("Windows 10 Version 1809 for ARM64-based Systems")
		pB := NewProductFromFullName("Windows 11 for x64-based Systems")

		require.False(t, pA.Matches(pB))
		require.False(t, pB.Matches(pA))
	})

	t.Run("from differect arch", func(t *testing.T) {
		pA := NewProductFromFullName("Windows 11 for ARM64-based Systems")
		pB := NewProductFromFullName("Windows 11 for x64-based Systems")

		require.False(t, pA.Matches(pB))
		require.False(t, pB.Matches(pA))
	})

	t.Run("same product but for different architecture", func(t *testing.T) {
		pA := NewProductFromFullName("Windows 10 Version 1809 for ARM64-based Systems")
		pB := NewProductFromFullName("Windows 10 Version 1809 for x64-based Systems")
		require.False(t, pA.Matches(pB))
		require.False(t, pB.Matches(pA))
	})

	t.Run("same product one with no architecture", func(t *testing.T) {
		pA := NewProductFromFullName("Windows 10 Version 1809")
		pB := NewProductFromFullName("Windows 10 Version 1809 for x64-based Systems")
		require.True(t, pA.Matches(pB))
		require.True(t, pB.Matches(pA))
	})

	t.Run("same product same arch", func(t *testing.T) {
		pA := NewProductFromFullName("Windows 10 Version 1809 for x64-based Systems")
		pB := NewProductFromFullName("Windows 10 Version 1809 for x64-based Systems")
		require.True(t, pA.Matches(pB))
		require.True(t, pB.Matches(pA))
	})
}

func TestFullProductName(t *testing.T) {
	testCases := []struct {
		fullName string
		arch     string
		prodName string
	}{
		{
			fullName: "Windows 10 Version 1809 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1809 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1809 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server 2019",
			arch:     "all",
			prodName: "Windows Server 2019",
		},
		{
			fullName: "Windows Server 2019  (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2019",
		},
		{
			fullName: "Windows 10 Version 1909 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1909 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1909 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 21H1 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 21H1 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 21H1 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server 2022",
			arch:     "all",
			prodName: "Windows Server 2022",
		},
		{
			fullName: "Windows Server 2022 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2022",
		},
		{
			fullName: "Windows 10 Version 20H2 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 20H2 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 20H2 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server, version 20H2 (Server Core Installation)",
			arch:     "all",
			prodName: "Windows Server",
		},
		{
			fullName: "Windows 11 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 11",
		},
		{
			fullName: "Windows 11 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 11",
		},
		{
			fullName: "Windows 10 Version 21H2 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 21H2 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 21H2 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1607 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1607 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server 2016",
			arch:     "all",
			prodName: "Windows Server 2016",
		},
		{
			fullName: "Windows Server 2016  (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2016",
		},
		{
			fullName: "Windows 8.1 for 32-bit systems",
			arch:     "32-bit",
			prodName: "Windows 8.1",
		},
		{
			fullName: "Windows 8.1 for x64-based systems",
			arch:     "64-bit",
			prodName: "Windows 8.1",
		},
		{
			fullName: "Windows RT 8.1",
			arch:     "all",
			prodName: "Windows RT 8.1",
		},
		{
			fullName: "Windows Server 2012",
			arch:     "all",
			prodName: "Windows Server 2012",
		},
		{
			fullName: "Windows Server 2012 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2012",
		},
		{
			fullName: "Windows Server 2012 R2",
			arch:     "all",
			prodName: "Windows Server 2012 R2",
		},
		{
			fullName: "Windows Server 2012 R2 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2012 R2",
		},
		{
			fullName: "Windows 7 for 32-bit Systems Service Pack 1",
			arch:     "32-bit",
			prodName: "Windows 7",
		},
		{
			fullName: "Windows 7 for x64-based Systems Service Pack 1",
			arch:     "64-bit",
			prodName: "Windows 7",
		},
		{
			fullName: "Windows Server 2008 for 32-bit Systems Service Pack 2",
			arch:     "32-bit",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)",
			arch:     "32-bit",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 for x64-based Systems Service Pack 2",
			arch:     "64-bit",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation)",
			arch:     "64-bit",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1",
			arch:     "64-bit",
			prodName: "Windows Server 2008 R2",
		},
		{
			fullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)",
			arch:     "64-bit",
			prodName: "Windows Server 2008 R2",
		},
		{
			fullName: "Windows 10 Version 1803 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server, version 1803 (Server Core Installation)",
			arch:     "all",
			prodName: "Windows Server",
		},
		{
			fullName: "Windows 10 Version 1809 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server 2019",
			arch:     "all",
			prodName: "Windows Server 2019",
		},
		{
			fullName: "Windows Server 2019 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2019",
		},
		{
			fullName: "Windows 10 Version 1709 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1903 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server, version 1903 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server",
		},
		{
			fullName: "Windows 10 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1607 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server 2016",
			arch:     "all",
			prodName: "Windows Server 2016",
		},
		{
			fullName: "Windows Server 2016 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2016",
		},
		{
			fullName: "Windows 8.1 for x64-based systems",
			arch:     "64-bit",
			prodName: "Windows 8.1",
		},
		{
			fullName: "Windows Server 2012",
			arch:     "all",
			prodName: "Windows Server 2012",
		},
		{
			fullName: "Windows Server 2012 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2012",
		},
		{
			fullName: "Windows Server 2012 R2",
			arch:     "all",
			prodName: "Windows Server 2012 R2",
		},
		{
			fullName: "Windows Server 2012 R2 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server 2012 R2",
		},
		{
			fullName: "Windows 10 Version 1909 for x64-based Systems",
			arch:     "64-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows Server, version 1909 (Server Core installation)",
			arch:     "all",
			prodName: "Windows Server",
		},
		{
			fullName: "Windows 10 Version 1803 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1803 for ARM64-based Systems",
			arch:     "arm64",
			prodName: "Windows 10",
		},
		{
			fullName: "Windows 10 Version 1809 for 32-bit Systems",
			arch:     "32-bit",
			prodName: "Windows 10",
		},
		{
			fullName: "None Available",
			arch:     "all",
			prodName: "",
		},
		{
			fullName: "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)",
			arch:     "32-bit",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 for Itanium-Based Systems Service Pack 2",
			arch:     "itanium",
			prodName: "Windows Server 2008",
		},
		{
			fullName: "Windows Server 2008 R2 for Itanium-Based Systems Service Pack 1",
			arch:     "itanium",
			prodName: "Windows Server 2008 R2",
		},
	}

	t.Run("#ArchFromProdName", func(t *testing.T) {
		for _, tCase := range testCases {
			sut := NewProductFromFullName(tCase.fullName)
			require.Equal(t, tCase.arch, sut.Arch(), tCase)
		}
	})

	t.Run("#NameFromFullProdName", func(t *testing.T) {
		for _, tCase := range testCases {
			sut := NewProductFromFullName(tCase.fullName)
			require.Equal(t, tCase.prodName, sut.Name(), tCase)
		}
	})
}
