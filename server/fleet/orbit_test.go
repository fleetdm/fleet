package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterByHostPlatform(t *testing.T) {
	// Test with no extensions.
	var extensions Extensions
	extensions.FilterByHostPlatform("darwin", "x86_64")
	require.Len(t, extensions, 0)

	// Test with a macOS extension.
	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "macos",
			Channel:  "stable",
		},
	}
	extensions.FilterByHostPlatform("darwin", "arm64e")
	require.Contains(t, extensions, "hello_world")
	extensions.FilterByHostPlatform("macos", "arm64")
	require.Contains(t, extensions, "hello_world")
	extensions.FilterByHostPlatform("ubuntu", "x86_64")
	require.Len(t, extensions, 0)

	//
	// Test with Linux amd64 and arm64 extension on Ubuntu hosts.
	//

	// Linux amd64 extension on ubuntu.
	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "linux",
			Channel:  "stable",
		},
		"hello_world_arm64": ExtensionInfo{
			Platform: "linux-arm64",
			Channel:  "stable",
		},
	}
	extensions.FilterByHostPlatform("ubuntu", "x86_64")
	require.Contains(t, extensions, "hello_world")
	extensions.FilterByHostPlatform("ubuntu", "amd64")
	require.Contains(t, extensions, "hello_world")
	extensions.FilterByHostPlatform("ubuntu", "arm64")
	require.Len(t, extensions, 0)

	// Linux arm64 extension on ubuntu.
	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "linux",
			Channel:  "stable",
		},
		"hello_world_arm64": ExtensionInfo{
			Platform: "linux-arm64",
			Channel:  "stable",
		},
	}
	extensions.FilterByHostPlatform("ubuntu", "arm64")
	require.Contains(t, extensions, "hello_world_arm64")
	extensions.FilterByHostPlatform("ubuntu", "aarch64")
	require.Contains(t, extensions, "hello_world_arm64")
	extensions.FilterByHostPlatform("ubuntu", "amd64")
	require.Len(t, extensions, 0)

	//
	// Test with a Linux amd64 and arm64 extension on Linux hosts.
	//

	extensions = Extensions{
		"hello_world_0": ExtensionInfo{
			Platform: "macos",
			Channel:  "stable",
		},
		"hello_world_1": ExtensionInfo{
			Platform: "windows",
			Channel:  "stable",
		},
		"hello_world_2": ExtensionInfo{
			Platform: "linux",
			Channel:  "stable",
		},
		"hello_world_3": ExtensionInfo{
			Platform: "linux-arm64",
			Channel:  "stable",
		},
	}

	// Linux amd64
	extensions.FilterByHostPlatform("linux", "x86_64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_2")
	extensions.FilterByHostPlatform("linux", "amd64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_2")

	extensions = Extensions{
		"hello_world_0": ExtensionInfo{
			Platform: "macos",
			Channel:  "stable",
		},
		"hello_world_1": ExtensionInfo{
			Platform: "windows",
			Channel:  "stable",
		},
		"hello_world_2": ExtensionInfo{
			Platform: "linux",
			Channel:  "stable",
		},
		"hello_world_3": ExtensionInfo{
			Platform: "linux-arm64",
			Channel:  "stable",
		},
	}
	// Linux arm64
	extensions.FilterByHostPlatform("linux", "aarch64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_3")
	extensions.FilterByHostPlatform("linux", "arm64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_3")

	//
	// Test with a Windows amd64 and arm64 extension.
	//

	extensions = Extensions{
		"hello_world_0": ExtensionInfo{
			Platform: "windows",
			Channel:  "stable",
		},
		"hello_world_1": ExtensionInfo{
			Platform: "windows-arm64",
			Channel:  "stable",
		},
	}

	// Windows arm64
	extensions.FilterByHostPlatform("windows", "ARM")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_1")
	extensions.FilterByHostPlatform("windows", "arm64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_1")

	extensions = Extensions{
		"hello_world_0": ExtensionInfo{
			Platform: "windows",
			Channel:  "stable",
		},
		"hello_world_1": ExtensionInfo{
			Platform: "windows-arm64",
			Channel:  "stable",
		},
	}

	// Windows amd64
	extensions.FilterByHostPlatform("windows", "x86_64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_0")
	extensions.FilterByHostPlatform("windows", "amd64")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_0")
}
