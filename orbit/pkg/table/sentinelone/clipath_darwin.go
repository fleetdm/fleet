//go:build darwin

package sentinelone

// cliCandidatePaths lists where sentinelctl lives on macOS, most canonical
// first. /usr/local/bin/sentinelctl is a symlink the installer creates; the
// bundle paths cover installs where it is missing.
var cliCandidatePaths = []string{
	"/usr/local/bin/sentinelctl",
	"/Library/Sentinel/sentinel-agent.bundle/Contents/MacOS/sentinelctl",
	"/Applications/SentinelOne.app/Contents/MacOS/sentinelctl",
}

func resolveCLIPath() string {
	return firstExistingPath(cliCandidatePaths)
}
