//go:build !darwin && !linux && !windows

package sentinelone

// resolveCLIPath reports no sentinelctl on platforms SentinelOne doesn't ship
// an agent for, which makes Generate report no rows there.
func resolveCLIPath() string {
	return ""
}
