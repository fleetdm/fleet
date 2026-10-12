package netskope

// columnOrder lists every column in the netskope table, in schema order.
// The nsdiag-derived columns keep the field names `nsdiag -f` reports, lowercased
// with spaces replaced by underscores.
var columnOrder = []string{
	"client_installed",
	"error",
	"install_path",
	"orgname",
	"tenant_url",
	"addonhost",
	"addoncheckerhost",
	"gateway",
	"gateway_ip",
	"config",
	"steering_config",
	"email",
	"peruser_config",
	"tunnel_status",
	"client_status",
	"dynamic_steering",
	"onpremdetection",
	"explicit_proxy",
	"tunnel_protocol",
	"sni_enable",
	"traffic_mode",
	"client_version",
}

// integerColumns are the columns reported as INTEGER rather than TEXT.
var integerColumns = map[string]struct{}{
	"client_installed": {},
}

// nsdiagKeyToColumn maps the lowercased key of each `nsdiag -f` line to its
// column. Keys nsdiag reports that aren't listed here are ignored, so a Netskope
// release that adds fields doesn't break the table.
var nsdiagKeyToColumn = map[string]string{
	"orgname":          "orgname",
	"tenant url":       "tenant_url",
	"addonhost":        "addonhost",
	"addoncheckerhost": "addoncheckerhost",
	"gateway":          "gateway",
	"gateway ip":       "gateway_ip",
	"config":           "config",
	"steering config":  "steering_config",
	"email":            "email",
	"peruser config":   "peruser_config",
	"tunnel status":    "tunnel_status",
	"client status":    "client_status",
	"dynamic steering": "dynamic_steering",
	"onpremdetection":  "onpremdetection",
	"explicit proxy":   "explicit_proxy",
	"tunnel protocol":  "tunnel_protocol",
	"sni enable":       "sni_enable",
	"traffic mode":     "traffic_mode",
	"client version":   "client_version",
}
