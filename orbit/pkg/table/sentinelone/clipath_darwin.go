//go:build darwin

package sentinelone

// cliCandidatePaths lists where sentinelctl lives on macOS. Only the binary in
// the root-owned agent bundle is used: osquery runs it as root, and the
// installer's /usr/local/bin/sentinelctl symlink lives in a directory that can
// be user-writable (e.g. Homebrew on Intel Macs).
var cliCandidatePaths = []string{
	"/Library/Sentinel/sentinel-agent.bundle/Contents/MacOS/sentinelctl",
}

func resolveCLIPath() string {
	return firstExistingPath(cliCandidatePaths)
}
