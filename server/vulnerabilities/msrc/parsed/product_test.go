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

var msrcWinProducts = Products{
	"10729": "Windows 10 for 32-bit Systems",
	"10735": "Windows 10 for x64-based Systems",
	"10852": "Windows 10 Version 1607 for 32-bit Systems",
	"10853": "Windows 10 Version 1607 for x64-based Systems",
	"11926": "Windows 11 for x64-based Systems",
	"11927": "Windows 11 for ARM64-based Systems",
	"12085": "Windows 11 Version 22H2 for ARM64-based Systems",
	"12086": "Windows 11 Version 22H2 for x64-based Systems",
	"12242": "Windows 11 Version 23H2 for ARM64-based Systems",
	"12243": "Windows 11 Version 23H2 for x64-based Systems",
	"11923": "Windows Server 2022",
	"11924": "Windows Server 2022 (Server Core installation)",
	"12244": "Windows Server 2022, 23H2 Edition (Server Core installation)",
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
			name: "unknown OS",
			os: fleet.OperatingSystem{
				Name: "Windows Foo Bar",
				Arch: "arm64",
			},
			want: "",
			err:  ErrNoMatch,
		},
	}

	for _, tt := range tc {
		match, err := msrcWinProducts.GetMatchForOS(ctx, tt.os)
		require.ErrorIs(t, err, tt.err, tt.name)
		require.Equal(t, tt.want, match, tt.name)
	}
}
