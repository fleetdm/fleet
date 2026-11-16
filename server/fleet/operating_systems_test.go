package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperatingSystemIsWindows(t *testing.T) {
	testCases := []struct {
		platform  string
		isWindows bool
	}{
		{platform: "chrome"},
		{platform: "darwin"},
		{platform: "windows", isWindows: true},
	}

	for _, tc := range testCases {
		sut := OperatingSystem{Platform: tc.platform}
		require.Equal(t, tc.isWindows, sut.IsWindows())
	}
}

func TestHostAPIOS(t *testing.T) {
	testCases := []struct {
		inventoryName, inventoryVersion, display string
		wantName, wantVersion                    string
	}{
		{"macOS", "15.2", "macOS 15.2", "macOS", "15.2"},
		{"Microsoft Windows 11 Pro 24H2", "10.0.26100.1742", "Windows 11 Pro 24H2 10.0.26100.1742", "Windows 11 Pro 24H2", "10.0.26100.1742"},
		{"Arch Linux", "rolling", "Arch Linux rolling", "Arch Linux", "rolling"},
		{"Arch Linux", "rolling", "Arch Linux ARM rolling", "Arch Linux ARM", "rolling"},
		{"Arch Linux", "rolling", "CachyOS Linux rolling", "CachyOS Linux", "rolling"},
		{"Arch Linux", "rolling", "Omarchy 4.0.0", "Omarchy", "4.0.0"},
		{"", "", "Android", "", ""},
	}
	for _, tc := range testCases {
		name, version := HostAPIOS(tc.inventoryName, tc.inventoryVersion, tc.display)
		require.Equal(t, tc.wantName, name, tc.display)
		require.Equal(t, tc.wantVersion, version, tc.display)
	}
}

func TestInventoryOSNames(t *testing.T) {
	require.Equal(t, []string{"Windows 11 Pro 24H2", "Microsoft Windows 11 Pro 24H2"}, InventoryOSNames("Windows 11 Pro 24H2"))
	require.Equal(t, []string{"windows 11 pro 24h2", "Microsoft windows 11 pro 24h2"}, InventoryOSNames("windows 11 pro 24h2"))
	require.Equal(t, []string{"Microsoft Windows 11 Pro 24H2"}, InventoryOSNames("Microsoft Windows 11 Pro 24H2"))
	require.Equal(t, []string{"macOS"}, InventoryOSNames("macOS"))
}

func TestInventoryOSName(t *testing.T) {
	testCases := []struct {
		reported     string
		wantName     string
		forceRolling bool
	}{
		{"Arch Linux", "Arch Linux", false},
		{"Arch Linux ARM", "Arch Linux", false},
		{"CachyOS Linux", "Arch Linux", false},
		{"Omarchy", "Arch Linux", true},
		{"Manjaro Linux", "Manjaro Linux", false},
		{"Ubuntu", "Ubuntu", false},
	}
	for _, tc := range testCases {
		name, forceRolling := InventoryOSName(tc.reported)
		require.Equal(t, tc.wantName, name, tc.reported)
		require.Equal(t, tc.forceRolling, forceRolling, tc.reported)
	}
}
