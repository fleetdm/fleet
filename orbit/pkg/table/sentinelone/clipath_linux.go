//go:build linux

package sentinelone

// cliCandidatePaths lists where sentinelctl lives on Linux, most canonical
// first. The agent installs under /opt via .deb/.rpm.
var cliCandidatePaths = []string{
	"/opt/sentinelone/bin/sentinelctl",
	"/usr/local/bin/sentinelctl",
	"/usr/bin/sentinelctl",
}

func resolveCLIPath() string {
	return firstExistingPath(cliCandidatePaths)
}
