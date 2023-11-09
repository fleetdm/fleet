package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterByHostPlatform(t *testing.T) {
	var extensions Extensions
	extensions.FilterByHostPlatform("darwin")
	require.Len(t, extensions, 0)

	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "macos",
			Channel:  "stable",
		},
	}

	extensions.FilterByHostPlatform("darwin")
	require.Contains(t, extensions, "hello_world")

	extensions.FilterByHostPlatform("macos")
	require.Contains(t, extensions, "hello_world")

	extensions.FilterByHostPlatform("ubuntu")
	require.Len(t, extensions, 0)

	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "linux",
			Channel:  "stable",
		},
	}

	extensions.FilterByHostPlatform("ubuntu")
	require.Contains(t, extensions, "hello_world")

	extensions = Extensions{
		"hello_world": ExtensionInfo{
			Platform: "windows",
			Channel:  "stable",
		},
		"hello_world_2": ExtensionInfo{
			Platform: "windows",
			Channel:  "edge",
		},
	}

	extensions.FilterByHostPlatform("darwin")
	require.Len(t, extensions, 0)

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
	}

	extensions.FilterByHostPlatform("linux")
	require.Len(t, extensions, 1)
	require.Contains(t, extensions, "hello_world_2")
}
