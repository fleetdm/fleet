package wstransport

import (
	"strconv"
	"strings"
	"time"
)

// defaultExtensionsTimeout is the minimum --extensions_timeout (seconds)
// osquery is given to wait for orbit's extension to register. It gives orbit
// ample time to connect; if the extension never registers, osqueryd exits and
// the osquery runner brings the whole orbit process down for a clean restart.
const defaultExtensionsTimeout = 60

// OsqueryFlags returns the osquery command-line flags that route distributed
// queries, and the config when osqueryConfig is set, through orbit's extension
// plugins. These flags are passed after
// --flagfile and gflags is last-one-wins, so any --extensions_require or
// --extensions_timeout the user set in their flagfile would be silently
// replaced. To honor those, userFlags (the parsed flagfile, keys with the
// "--" prefix) is merged in: user-required extensions are kept alongside
// orbit's, and the larger of the two timeouts wins.
func OsqueryFlags(requiredExtension string, userFlags map[string]string, osqueryConfig bool) []string {
	required := []string{}
	for name := range strings.SplitSeq(flagValue(userFlags, "--extensions_require"), ",") {
		name = strings.TrimSpace(name)
		if name != "" && name != requiredExtension {
			required = append(required, name)
		}
	}
	required = append(required, requiredExtension)

	timeout := defaultExtensionsTimeout
	if userTimeout, err := strconv.Atoi(flagValue(userFlags, "--extensions_timeout")); err == nil && userTimeout > timeout {
		timeout = userTimeout
	}

	flags := []string{
		"--distributed_plugin", DistributedPluginName,
		"--extensions_require", strings.Join(required, ","),
		"--extensions_timeout", strconv.Itoa(timeout),
	}
	if osqueryConfig {
		// After the flagfile, so the user's --config_refresh (the server poll
		// interval while disconnected, see ServerConfigRefresh) doesn't apply
		// to the localhost poll.
		flags = append(flags,
			"--config_plugin", ConfigPluginName,
			"--config_refresh", strconv.Itoa(LocalConfigRefresh),
		)
	}
	return flags
}

// ServerConfigRefresh returns osquery's config refresh as set by userFlags
// (the parsed flagfile), or DefaultServerConfigRefresh.
func ServerConfigRefresh(userFlags map[string]string) time.Duration {
	for _, name := range []string{"--config_refresh", "--config_tls_refresh"} {
		if v, err := strconv.Atoi(flagValue(userFlags, name)); err == nil && v > 0 {
			return time.Duration(v) * time.Second
		}
	}
	return DefaultServerConfigRefresh
}

// flagValue returns the flag's value with any surrounding quotes stripped
// (hand-edited flagfiles sometimes quote values; Fleet-written ones don't).
func flagValue(flags map[string]string, name string) string {
	return strings.Trim(flags[name], `"'`)
}
