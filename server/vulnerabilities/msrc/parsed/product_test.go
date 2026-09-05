package parsed

import (
	"context"
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
		fullName  string
		arch      string
		prodName  string
		finalName string
	}{
		{
			fullName:  "Windows 10 Version 1809 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1809 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 1809 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1809 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1809 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1809 for ARM64-based Systems",
		},
		{
			fullName:  "Windows Server 2019",
			arch:      "all",
			prodName:  "Windows Server 2019",
			finalName: "Windows Server 2019 Version 1809",
		},
		{
			fullName:  "Windows Server 2019  (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2019",
			finalName: "Windows Server 2019  (Server Core installation) Version 1809",
		},
		{
			fullName:  "Windows 10 Version 1909 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1909 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 1909 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1909 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1909 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1909 for ARM64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 21H1 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H1 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 21H1 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H1 for ARM64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 21H1 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H1 for 32-bit Systems",
		},
		{
			fullName:  "Windows Server 2022",
			arch:      "all",
			prodName:  "Windows Server 2022",
			finalName: "Windows Server 2022 Version 21H2",
		},
		{
			fullName:  "Windows Server 2022 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2022",
			finalName: "Windows Server 2022 (Server Core installation) Version 21H2",
		},
		{
			fullName:  "Windows Server 2025",
			arch:      "all",
			prodName:  "Windows Server 2025",
			finalName: "Windows Server 2025 Version 24H2",
		},
		{
			fullName:  "Windows Server 2025 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2025",
			finalName: "Windows Server 2025 (Server Core installation) Version 24H2",
		},
		{
			fullName:  "Windows 10 Version 20H2 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 20H2 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 20H2 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 20H2 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 20H2 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 20H2 for ARM64-based Systems",
		},
		{
			fullName:  "Windows Server, version 20H2 (Server Core Installation)",
			arch:      "all",
			prodName:  "Windows Server",
			finalName: "Windows Server, version 20H2 (Server Core Installation)",
		},
		{
			fullName:  "Windows 11 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 11",
			finalName: "Windows 11 for x64-based Systems",
		},
		{
			fullName:  "Windows 11 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 11",
			finalName: "Windows 11 for ARM64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 21H2 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H2 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 21H2 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H2 for ARM64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 21H2 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 21H2 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1607 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1607 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 1607 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1607 for x64-based Systems",
		},
		{
			fullName:  "Windows Server 2016",
			arch:      "all",
			prodName:  "Windows Server 2016",
			finalName: "Windows Server 2016 Version 1607",
		},
		{
			fullName:  "Windows Server 2016  (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2016",
			finalName: "Windows Server 2016  (Server Core installation) Version 1607",
		},
		{
			fullName:  "Windows 8.1 for 32-bit systems",
			arch:      "32-bit",
			prodName:  "Windows 8.1",
			finalName: "Windows 8.1 for 32-bit systems Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows 8.1 for x64-based systems",
			arch:      "64-bit",
			prodName:  "Windows 8.1",
			finalName: "Windows 8.1 for x64-based systems Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows RT 8.1",
			arch:      "all",
			prodName:  "Windows RT 8.1",
			finalName: "Windows RT 8.1 Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows Server 2012",
			arch:      "all",
			prodName:  "Windows Server 2012",
			finalName: "Windows Server 2012 Version 6.2 / NT 6.2",
		},
		{
			fullName:  "Windows Server 2012 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2012",
			finalName: "Windows Server 2012 (Server Core installation) Version 6.2 / NT 6.2",
		},
		{
			fullName:  "Windows Server 2012 R2",
			arch:      "all",
			prodName:  "Windows Server 2012 R2",
			finalName: "Windows Server 2012 R2 Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows Server 2012 R2 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2012 R2",
			finalName: "Windows Server 2012 R2 (Server Core installation) Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows 7 for 32-bit Systems Service Pack 1",
			arch:      "32-bit",
			prodName:  "Windows 7",
			finalName: "Windows 7 for 32-bit Systems Service Pack 1 Version 6.1 / NT 6.1",
		},
		{
			fullName:  "Windows 7 for x64-based Systems Service Pack 1",
			arch:      "64-bit",
			prodName:  "Windows 7",
			finalName: "Windows 7 for x64-based Systems Service Pack 1 Version 6.1 / NT 6.1",
		},
		{
			fullName:  "Windows Server 2008 for 32-bit Systems Service Pack 2",
			arch:      "32-bit",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for 32-bit Systems Service Pack 2 Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)",
			arch:      "32-bit",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation) Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 for x64-based Systems Service Pack 2",
			arch:      "64-bit",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for x64-based Systems Service Pack 2 Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation)",
			arch:      "64-bit",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation) Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 R2 for x64-based Systems Service Pack 1",
			arch:      "64-bit",
			prodName:  "Windows Server 2008 R2",
			finalName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 Version 6.1 / NT 6.1",
		},
		{
			fullName:  "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)",
			arch:      "64-bit",
			prodName:  "Windows Server 2008 R2",
			finalName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation) Version 6.1 / NT 6.1",
		},
		{
			fullName:  "Windows 10 Version 1803 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1803 for x64-based Systems",
		},
		{
			fullName:  "Windows Server, version 1803 (Server Core Installation)",
			arch:      "all",
			prodName:  "Windows Server",
			finalName: "Windows Server, version 1803 (Server Core Installation)",
		},
		{
			fullName:  "Windows 10 Version 1809 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1809 for x64-based Systems",
		},
		{
			fullName:  "Windows Server 2019",
			arch:      "all",
			prodName:  "Windows Server 2019",
			finalName: "Windows Server 2019 Version 1809",
		},
		{
			fullName:  "Windows Server 2019 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2019",
			finalName: "Windows Server 2019 (Server Core installation) Version 1809",
		},
		{
			fullName:  "Windows 10 Version 1709 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1709 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1903 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1903 for x64-based Systems",
		},
		{
			fullName:  "Windows Server, version 1903 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server",
			finalName: "Windows Server, version 1903 (Server Core installation)",
		},
		{
			fullName:  "Windows 10 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 for x64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1607 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1607 for x64-based Systems",
		},
		{
			fullName:  "Windows Server 2016",
			arch:      "all",
			prodName:  "Windows Server 2016",
			finalName: "Windows Server 2016 Version 1607",
		},
		{
			fullName:  "Windows Server 2016 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2016",
			finalName: "Windows Server 2016 (Server Core installation) Version 1607",
		},
		{
			fullName:  "Windows 8.1 for x64-based systems",
			arch:      "64-bit",
			prodName:  "Windows 8.1",
			finalName: "Windows 8.1 for x64-based systems Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows Server 2012",
			arch:      "all",
			prodName:  "Windows Server 2012",
			finalName: "Windows Server 2012 Version 6.2 / NT 6.2",
		},
		{
			fullName:  "Windows Server 2012 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2012",
			finalName: "Windows Server 2012 (Server Core installation) Version 6.2 / NT 6.2",
		},
		{
			fullName:  "Windows Server 2012 R2",
			arch:      "all",
			prodName:  "Windows Server 2012 R2",
			finalName: "Windows Server 2012 R2 Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows Server 2012 R2 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server 2012 R2",
			finalName: "Windows Server 2012 R2 (Server Core installation) Version 6.3 / NT 6.3",
		},
		{
			fullName:  "Windows 10 Version 1909 for x64-based Systems",
			arch:      "64-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1909 for x64-based Systems",
		},
		{
			fullName:  "Windows Server, version 1909 (Server Core installation)",
			arch:      "all",
			prodName:  "Windows Server",
			finalName: "Windows Server, version 1909 (Server Core installation)",
		},
		{
			fullName:  "Windows 10 Version 1803 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1803 for 32-bit Systems",
		},
		{
			fullName:  "Windows 10 Version 1803 for ARM64-based Systems",
			arch:      "arm64",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1803 for ARM64-based Systems",
		},
		{
			fullName:  "Windows 10 Version 1809 for 32-bit Systems",
			arch:      "32-bit",
			prodName:  "Windows 10",
			finalName: "Windows 10 Version 1809 for 32-bit Systems",
		},
		{
			fullName:  "None Available",
			arch:      "all",
			prodName:  "",
			finalName: "None Available",
		},
		{
			fullName:  "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)",
			arch:      "32-bit",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation) Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 for Itanium-Based Systems Service Pack 2",
			arch:      "itanium",
			prodName:  "Windows Server 2008",
			finalName: "Windows Server 2008 for Itanium-Based Systems Service Pack 2 Version 6.0 / NT 6.0",
		},
		{
			fullName:  "Windows Server 2008 R2 for Itanium-Based Systems Service Pack 1",
			arch:      "itanium",
			prodName:  "Windows Server 2008 R2",
			finalName: "Windows Server 2008 R2 for Itanium-Based Systems Service Pack 1 Version 6.1 / NT 6.1",
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
			require.Equal(t, tCase.finalName, string(sut), tCase)
		}
	})
}

func TestProductHasDisplayVersion(t *testing.T) {
	tc := []struct {
		name   Product
		result bool
	}{
		{
			name:   "Windows 11 for x64-based Systems",
			result: false,
		},
		{
			name:   "Windows 11 Version 22H2 for x64-based Systems",
			result: true,
		},
		{
			name:   "Windows Server 2022, 23H2 Edition (Server Core installation)",
			result: true,
		},
		{
			name:   "Windows Server 2022 (Server Core installation)",
			result: false,
		},
		{
			name:   "Windows Server 2022",
			result: false,
		},
		{
			name:   "Windows Server, version 1803  (Server Core Installation)",
			result: true,
		},
	}

	for _, tt := range tc {
		require.Equal(t, tt.result, tt.name.HasDisplayVersion(), tt.name)
	}
}

func TestExtractDisplayVersionFromName(t *testing.T) {
	tc := []struct {
		name string
		want string
	}{
		{"Microsoft Windows 10 Pro 22H2", "22H2"},
		{"Microsoft Windows 10 Enterprise 22H2", "22H2"},
		{"Microsoft Windows 10 Pro N 22H2", "22H2"},
		{"Microsoft Windows 11 Pro 25H2", "25H2"},
		{"Microsoft Windows 11 Enterprise 23H2", "23H2"},
		{"Microsoft Windows 11 Enterprise 24H2", "24H2"},
		{"Microsoft Windows 10 Version 1809", ""},
		{"Microsoft Windows 10 Version 1607", ""},
		{"Microsoft Windows Server 2022 Datacenter 21H2", "21H2"},
		{"Microsoft Windows 10 Pro", ""},
		{"Microsoft Windows 11 Enterprise", ""},
		{"Microsoft Windows Server 2022", ""},
		{"empty string", ""},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDisplayVersionFromName(tt.name)
			require.Equal(t, tt.want, got)
		})
	}
}

// msrcWinProducts mirrors what production code produces: raw MSRC product names
// are always processed through NewProductFromFullName before reaching GetMatchForOS
var msrcWinProducts = Products{
	"10729": NewProductFromFullName("Windows 10 for 32-bit Systems"),
	"10735": NewProductFromFullName("Windows 10 for x64-based Systems"),
	"10852": NewProductFromFullName("Windows 10 Version 1607 for 32-bit Systems"),
	"10853": NewProductFromFullName("Windows 10 Version 1607 for x64-based Systems"),
	"11926": NewProductFromFullName("Windows 11 for x64-based Systems"),
	"11927": NewProductFromFullName("Windows 11 for ARM64-based Systems"),
	"12085": NewProductFromFullName("Windows 11 Version 22H2 for ARM64-based Systems"),
	"12086": NewProductFromFullName("Windows 11 Version 22H2 for x64-based Systems"),
	"12242": NewProductFromFullName("Windows 11 Version 23H2 for ARM64-based Systems"),
	"12243": NewProductFromFullName("Windows 11 Version 23H2 for x64-based Systems"),
	"11923": NewProductFromFullName("Windows Server 2022"),
	"11924": NewProductFromFullName("Windows Server 2022 (Server Core installation)"),
	"12244": NewProductFromFullName("Windows Server 2022, 23H2 Edition (Server Core installation)"),
	"12436": NewProductFromFullName("Windows Server 2025 Version 24H2"),
	"12437": NewProductFromFullName("Windows Server 2025 (Server Core installation)"),
	"12097": NewProductFromFullName("Windows 10 Version 22H2 for x64-based Systems"),
	"10378": NewProductFromFullName("Windows Server 2012"),
	"10379": NewProductFromFullName("Windows Server 2012 (Server Core installation)"),
	"10483": NewProductFromFullName("Windows Server 2012 R2"),
	"10543": NewProductFromFullName("Windows Server 2012 R2 (Server Core installation)"),
	"10816": NewProductFromFullName("Windows Server 2016"),
	"10855": NewProductFromFullName("Windows Server 2016  (Server Core installation)"),
	"11571": NewProductFromFullName("Windows Server 2019"),
	"11572": NewProductFromFullName("Windows Server 2019  (Server Core installation)"),
	// MSRC files the Hardware Lab Kit under the operating system bulletins. These
	// products carry no OS CVEs but reduce to the same product name, and most carry
	// no architecture, so they match any host unless excluded.
	"16751": NewProductFromFullName("Windows 11 HLK 22H2"),
	"16752": NewProductFromFullName("Windows HLK for Windows Server 2025 Version 24H2"),
	"16755": NewProductFromFullName("Windows HLK for Windows Server 2019 Version 1809"),
	"16757": NewProductFromFullName("Windows 10 HLK Version 22H2"),
}

func TestMatchesOperatingSystem(t *testing.T) {
	ctx := context.Background()
	tc := []struct {
		name string
		os   fleet.OperatingSystem
		want string
		err  error
	}{
		{
			name: "OS with known Display Version Match x64",
			os: fleet.OperatingSystem{
				Name:           "Windows 11 Pro Version 22H2",
				Arch:           "x86_64",
				DisplayVersion: "22H2",
			},
			want: "12086",
			err:  nil,
		},
		{
			name: "OS with known Display Version Match ARM64",
			os: fleet.OperatingSystem{
				Name:           "Windows 11 Pro Version 22H2",
				Arch:           "ARM 64-bit Processor",
				DisplayVersion: "22H2",
			},
			want: "12085",
			err:  nil,
		},
		{
			name: "Win 11 with no Display Version and matching build number",
			os: fleet.OperatingSystem{
				Name:          "Windows 11 Pro",
				Arch:          "64-bit",
				KernelVersion: "10.0.22000.795", // matches on build number for 22000 only
			},
			want: "11926",
			err:  nil,
		},
		{
			name: "Win 11 with no Display Version with wrong build number",
			os: fleet.OperatingSystem{
				Name:          "Windows 11 Pro",
				Arch:          "64-bit",
				KernelVersion: "10.0.22631.795", // matches on build number for 22000 only
			},
			err: ErrNoMatch,
		},
		{
			name: "Win 10 with no Display Version and matching build number",
			os: fleet.OperatingSystem{
				Name:          "Windows 10 Pro",
				Arch:          "64-bit",
				KernelVersion: "10.0.10240.795", // matches on build number for 10240 only
			},
			want: "10735",
			err:  nil,
		},
		{
			name: "Win10 with no Display Version with wrong build number",
			os: fleet.OperatingSystem{
				Name:          "Windows 10 Pro",
				Arch:          "64-bit",
				KernelVersion: "10.0.19045.795", // matches on build number for 10240 only
			},
			want: "",
			err:  ErrNoMatch,
		},
		{
			name: "Product contains 'Edition' keyword",
			os: fleet.OperatingSystem{
				Name:           "Windows Server 2022 Edition 23H2",
				Arch:           "64-bit",
				DisplayVersion: "23H2",
			},
			want: "12244",
			err:  nil,
		},
		{
			name: "Windows Server 2025 with display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2025 Datacenter 24H2",
				Arch:             "64-bit",
				DisplayVersion:   "24H2",
				InstallationType: "Server",
			},
			want: "12436",
			err:  nil,
		},
		{
			name: "unknown OS",
			os: fleet.OperatingSystem{
				Name: "Windows Foo Bar",
				Arch: "arm64",
			},
			want: "",
			err:  ErrNoMatch,
		},
		{
			name: "Windows Server 2022 full desktop with installation type",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2022 Datacenter 21H2",
				Arch:             "64-bit",
				DisplayVersion:   "21H2",
				InstallationType: "Server",
			},
			want: "11923",
			err:  nil,
		},
		{
			name: "Windows Server 2022 Server Core with installation type",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2022 Datacenter 21H2",
				Arch:             "64-bit",
				DisplayVersion:   "21H2",
				InstallationType: "Server Core",
			},
			want: "11924",
			err:  nil,
		},
		{
			name: "Windows Server 2025 full desktop with installation type",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2025 Datacenter 24H2",
				Arch:             "64-bit",
				DisplayVersion:   "24H2",
				InstallationType: "Server",
			},
			want: "12436",
			err:  nil,
		},
		{
			name: "Windows Server 2025 Server Core with installation type",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2025 Datacenter 24H2",
				Arch:             "64-bit",
				DisplayVersion:   "24H2",
				InstallationType: "Server Core",
			},
			want: "12437",
			err:  nil,
		},
		{
			name: "Windows Server 2022 without installation type deterministically picks full desktop",
			os: fleet.OperatingSystem{
				Name:           "Microsoft Windows Server 2022 Datacenter 21H2",
				Arch:           "64-bit",
				DisplayVersion: "21H2",
			},
			// When InstallationType is empty, the full desktop product is preferred
			// as a deterministic fallback (desktop CVEs are a superset of Server Core).
			want: "11923",
			err:  nil,
		},
		{
			name: "Windows Server 2025 without installation type deterministically picks full desktop",
			os: fleet.OperatingSystem{
				Name:           "Microsoft Windows Server 2025 Datacenter 24H2",
				Arch:           "64-bit",
				DisplayVersion: "24H2",
			},
			want: "12436",
			err:  nil,
		},
		// Windows Server releases before 2022 have no DisplayVersion registry value,
		// so they are matched on the kernel build number instead.
		{
			name: "Windows Server 2019 Datacenter with no display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2019 Datacenter",
				Arch:             "64-bit",
				KernelVersion:    "10.0.17763.5122",
				InstallationType: "Server",
			},
			want: "11571",
			err:  nil,
		},
		{
			name: "Windows Server 2019 Standard with no display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2019 Standard",
				Arch:             "64-bit",
				KernelVersion:    "10.0.17763.5122",
				InstallationType: "Server",
			},
			want: "11571",
			err:  nil,
		},
		{
			name: "Windows Server 2019 without an edition suffix",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2019",
				Arch:             "64-bit",
				KernelVersion:    "10.0.17763.5122",
				InstallationType: "Server",
			},
			want: "11571",
			err:  nil,
		},
		{
			name: "Windows Server 2019 Server Core",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2019 Datacenter (Server Core)",
				Arch:             "64-bit",
				KernelVersion:    "10.0.17763.5122",
				InstallationType: "Server Core",
			},
			want: "11572",
			err:  nil,
		},
		{
			name: "Windows Server 2019 without installation type picks full desktop",
			os: fleet.OperatingSystem{
				Name:          "Microsoft Windows Server 2019 Datacenter",
				Arch:          "64-bit",
				KernelVersion: "10.0.17763.5122",
			},
			want: "11571",
			err:  nil,
		},
		{
			name: "Windows Server 2019 with a build from another release",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2019 Datacenter",
				Arch:             "64-bit",
				KernelVersion:    "10.0.19045.5122",
				InstallationType: "Server",
			},
			want: "",
			err:  ErrNoMatch,
		},
		{
			name: "Windows Server 2016 Datacenter with no display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2016 Datacenter",
				Arch:             "64-bit",
				KernelVersion:    "10.0.14393.6351",
				InstallationType: "Server",
			},
			want: "10816",
			err:  nil,
		},
		{
			name: "Windows Server 2016 Server Core",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2016 Standard (Server Core)",
				Arch:             "64-bit",
				KernelVersion:    "10.0.14393.6351",
				InstallationType: "Server Core",
			},
			want: "10855",
			err:  nil,
		},
		{
			name: "Windows Server 2012 R2 Standard with no display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2012 R2 Standard",
				Arch:             "64-bit",
				KernelVersion:    "6.3.9600.22371",
				InstallationType: "Server",
			},
			want: "10483",
			err:  nil,
		},
		{
			name: "Windows Server 2012 R2 Server Core",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2012 R2 Standard (Server Core)",
				Arch:             "64-bit",
				KernelVersion:    "6.3.9600.22371",
				InstallationType: "Server Core",
			},
			want: "10543",
			err:  nil,
		},
		{
			name: "Windows Server 2012 Datacenter with no display version",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2012 Datacenter",
				Arch:             "64-bit",
				KernelVersion:    "6.2.9200.24519",
				InstallationType: "Server",
			},
			want: "10378",
			err:  nil,
		},
		// Windows 10 1607 and 1809 share build numbers with Windows Server 2016 and
		// 2019, and must keep taking the display version path.
		{
			name: "Windows 10 sharing the Server 2016 build number",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows 10 Enterprise",
				Arch:             "64-bit",
				KernelVersion:    "10.0.14393.6351",
				InstallationType: "Client",
			},
			want: "",
			err:  ErrNoMatch,
		},
		{
			name: "Windows 10 sharing the Server 2019 build number",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows 10 Enterprise",
				Arch:             "64-bit",
				KernelVersion:    "10.0.17763.5122",
				InstallationType: "Client",
			},
			want: "",
			err:  ErrNoMatch,
		},
		// A Hardware Lab Kit product must never be preferred over the operating system.
		{
			name: "Windows 10 22H2 alongside a Hardware Lab Kit product",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows 10 Enterprise LTSC 22H2",
				Arch:             "64-bit",
				DisplayVersion:   "22H2",
				KernelVersion:    "10.0.19045.3803",
				InstallationType: "Client",
			},
			want: "12097",
			err:  nil,
		},
		{
			name: "Windows Server 2025 alongside a Hardware Lab Kit product",
			os: fleet.OperatingSystem{
				Name:             "Microsoft Windows Server 2025 Datacenter 24H2",
				Arch:             "64-bit",
				DisplayVersion:   "24H2",
				KernelVersion:    "10.0.26100.2033",
				InstallationType: "Server",
			},
			want: "12436",
			err:  nil,
		},
	}

	for _, tt := range tc {
		match, err := msrcWinProducts.GetMatchForOS(ctx, tt.os)
		require.ErrorIs(t, err, tt.err, tt.name)
		require.Equal(t, tt.want, match, tt.name)
	}
}

func TestIsServerCore(t *testing.T) {
	tc := []struct {
		product  Product
		expected bool
	}{
		{"Windows Server 2022", false},
		{"Windows Server 2022 (Server Core installation)", true},
		{"Windows Server 2022, 23H2 Edition (Server Core installation)", true},
		{"Windows Server, version 1803 (Server Core Installation)", true},
		{"Windows 11 Version 22H2 for x64-based Systems", false},
		{"Windows Server 2025 Version 24H2", false},
		{"Windows Server 2025 (Server Core installation) Version 24H2", true},
	}

	for _, tt := range tc {
		require.Equal(t, tt.expected, tt.product.IsServerCore(), string(tt.product))
	}
}

func TestIsOperatingSystem(t *testing.T) {
	tc := []struct {
		product  Product
		expected bool
	}{
		{"Windows 10 Version 22H2 for x64-based Systems", true},
		{"Windows 11 Version 23H2 for ARM64-based Systems", true},
		{"Windows Server 2016  (Server Core installation) Version 1607", true},
		{"Windows Server 2022, 23H2 Edition (Server Core installation)", true},
		{"Windows Server, version 1803  (Server Core Installation)", true},
		{"Windows 8.1 for x64-based systems Version 6.3 / NT 6.3", true},
		{"Windows Server 2008 R2 for x64-based Systems Service Pack 1 Version 6.1 / NT 6.1", true},

		{"Windows 10 HLK Version 22H2", false},
		{"Windows 10 HLK version 21H2", false},
		{"Windows 11 HLK 22H2", false},
		{"Windows 11 HLK 23H2", false},
		{"Windows 11 HLK 24H2", false},
		{"Windows HLK for Windows 10 version 2004", false},
		{"Windows HLK for Windows Server 2019 Version 1809", false},
		{"Windows HLK for Windows Server 2025 Version 24H2", false},
	}

	for _, tt := range tc {
		require.Equal(t, tt.expected, tt.product.IsOperatingSystem(), string(tt.product))
	}
}

// TestGetMatchForOSIsDeterministic guards against the selected product depending on
// map iteration order. Matching a product that carries no CVEs makes the analyzer
// treat every CVE already found for the host as remediated, so an unstable choice
// deletes and reinserts the host's whole vulnerability list between scans.
func TestGetMatchForOSIsDeterministic(t *testing.T) {
	const iterations = 500

	tc := []struct {
		name string
		os   fleet.OperatingSystem
		want string
	}{
		{
			name: "Windows 10 22H2",
			os: fleet.OperatingSystem{
				Name: "Microsoft Windows 10 Enterprise LTSC 22H2", Arch: "64-bit",
				DisplayVersion: "22H2", KernelVersion: "10.0.19045.3803", InstallationType: "Client",
			},
			want: "12097",
		},
		{
			name: "Windows 11 with no display version on the initial build",
			os: fleet.OperatingSystem{
				Name: "Windows 11 Pro", Arch: "64-bit",
				KernelVersion: "10.0.22000.795", InstallationType: "Client",
			},
			want: "11926",
		},
		{
			name: "Windows Server 2019",
			os: fleet.OperatingSystem{
				Name: "Microsoft Windows Server 2019 Datacenter", Arch: "64-bit",
				KernelVersion: "10.0.17763.5122", InstallationType: "Server",
			},
			want: "11571",
		},
		{
			name: "Windows Server 2025",
			os: fleet.OperatingSystem{
				Name: "Microsoft Windows Server 2025 Datacenter 24H2", Arch: "64-bit",
				DisplayVersion: "24H2", KernelVersion: "10.0.26100.2033", InstallationType: "Server",
			},
			want: "12436",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			for range iterations {
				match, err := msrcWinProducts.GetMatchForOS(t.Context(), tt.os)
				require.NoError(t, err)
				require.Equal(t, tt.want, match)
			}
		})
	}
}

// TestCandidateBetterThan pins the ordering used to choose between products that
// all match the host. Some of these preferences do not arise with the current
// MSRC data and exist so that a future bulletin cannot reintroduce a match that
// varies between scans.
func TestCandidateBetterThan(t *testing.T) {
	tc := []struct {
		name     string
		a, b     candidate
		expected bool
	}{
		{
			name:     "a display version match beats a build number match",
			a:        candidate{pID: "2", byDisplayVersion: true},
			b:        candidate{pID: "1"},
			expected: true,
		},
		{
			name:     "a build number match loses to a display version match",
			a:        candidate{pID: "1"},
			b:        candidate{pID: "2", byDisplayVersion: true},
			expected: false,
		},
		{
			name:     "full desktop beats Server Core",
			a:        candidate{pID: "2"},
			b:        candidate{pID: "1", isCore: true},
			expected: true,
		},
		{
			name:     "Server Core loses to full desktop",
			a:        candidate{pID: "1", isCore: true},
			b:        candidate{pID: "2"},
			expected: false,
		},
		{
			name:     "a named architecture beats one that matches any",
			a:        candidate{pID: "2", explicitArch: true},
			b:        candidate{pID: "1"},
			expected: true,
		},
		{
			name:     "one that matches any architecture loses to a named one",
			a:        candidate{pID: "1"},
			b:        candidate{pID: "2", explicitArch: true},
			expected: false,
		},
		{
			name:     "the lowest product ID breaks a full tie",
			a:        candidate{pID: "1"},
			b:        candidate{pID: "2"},
			expected: true,
		},
		{
			name:     "the highest product ID loses a full tie",
			a:        candidate{pID: "2"},
			b:        candidate{pID: "1"},
			expected: false,
		},
		{
			name:     "a candidate does not beat itself",
			a:        candidate{pID: "1", byDisplayVersion: true, explicitArch: true},
			b:        candidate{pID: "1", byDisplayVersion: true, explicitArch: true},
			expected: false,
		},
		{
			name:     "a display version match wins even when the other is more specific otherwise",
			a:        candidate{pID: "2", byDisplayVersion: true, isCore: true},
			b:        candidate{pID: "1", explicitArch: true},
			expected: true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.a.betterThan(tt.b))
		})
	}

	// The ordering must be antisymmetric, or the winner would depend on the order
	// candidates happen to be compared in.
	all := []candidate{
		{pID: "1"},
		{pID: "2", explicitArch: true},
		{pID: "3", isCore: true},
		{pID: "4", byDisplayVersion: true},
		{pID: "5", byDisplayVersion: true, isCore: true, explicitArch: true},
	}
	for _, a := range all {
		for _, b := range all {
			if a.pID == b.pID {
				continue
			}
			require.NotEqual(t, a.betterThan(b), b.betterThan(a),
				"betterThan must be antisymmetric for %s and %s", a.pID, b.pID)
		}
	}
}
