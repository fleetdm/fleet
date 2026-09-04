package wstransport

import (
	"strconv"
	"strings"
)

// defaultExtensionsTimeout is the minimum --extensions_timeout (seconds)
// osquery is given to wait for orbit's extension to register. It gives orbit
// ample time to connect; if the extension never registers, osqueryd exits and
// the osquery runner brings the whole orbit process down for a clean restart.
const defaultExtensionsTimeout = 60

// OsqueryFlags returns the osquery command-line flags that route distributed
// queries through orbit's extension plugin. These flags are passed after
// --flagfile and gflags is last-one-wins, so any --extensions_require or
// --extensions_timeout the user set in their flagfile would be silently
// replaced. To honor those, userFlags (the parsed flagfile, keys with the
// "--" prefix) is merged in: user-required extensions are kept alongside
// orbit's, and the larger of the two timeouts wins.
func OsqueryFlags(requiredExtension string, userFlags map[string]string) []string {
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

	return []string{
		"--distributed_plugin", DistributedPluginName,
		"--extensions_require", strings.Join(required, ","),
		"--extensions_timeout", strconv.Itoa(timeout),
	}
}

// flagValue returns the flag's value with any surrounding quotes stripped
// (hand-edited flagfiles sometimes quote values; Fleet-written ones don't).
func flagValue(flags map[string]string, name string) string {
	return strings.Trim(flags[name], `"'`)
}
