package netskope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nsdiagFixture is representative `nsdiag -f` output, including the whitespace
// and casing inconsistencies the real binary emits.
const nsdiagFixture = `Orgname:: CompanyName.
Tenant URL :: CompanyName.goskope.com.
AddonHost:: addon-companyname.goskope.com.
AddonCheckerHost:: achecker-companyname.goskope.com.
Gateway:: gateway-xyz.goskope.com.
Gateway IP:: 203.0.113.10.
Config:: Pop Pinning Client Configuration.
Steering Config:: Default tenant config.
Email:: alice@example.com.
Peruser config:: FALSE.
Tunnel status:: NSTUNNEL_CONNECTED.
Client status:: enable.
Dynamic Steering:: FALSE.
OnPremDetection:: Not Configured.
Explicit Proxy:: false.
Tunnel Protocol:: TLS.
SNI Enable:: FALSE.
Traffic Mode:: All Traffic.
Client version:: 117.1.0.1234.
`

func TestParseNsdiagText(t *testing.T) {
	t.Parallel()

	got := parseNsdiagText([]byte(nsdiagFixture))

	want := map[string]string{
		"orgname":          "CompanyName",
		"tenant_url":       "CompanyName.goskope.com",
		"addonhost":        "addon-companyname.goskope.com",
		"addoncheckerhost": "achecker-companyname.goskope.com",
		"gateway":          "gateway-xyz.goskope.com",
		"gateway_ip":       "203.0.113.10",
		"config":           "Pop Pinning Client Configuration",
		"steering_config":  "Default tenant config",
		"email":            "alice@example.com",
		"peruser_config":   "FALSE",
		"tunnel_status":    "NSTUNNEL_CONNECTED",
		"client_status":    "enable",
		"dynamic_steering": "FALSE",
		"onpremdetection":  "Not Configured",
		// nsdiag reports this one lowercased; the parser normalizes it so a single
		// comparison works across every boolean field.
		"explicit_proxy":  "FALSE",
		"tunnel_protocol": "TLS",
		"sni_enable":      "FALSE",
		"traffic_mode":    "All Traffic",
		"client_version":  "117.1.0.1234",
	}
	assert.Equal(t, want, got)
}

func TestParseNsdiagTextEdgeCases(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "empty input",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "lines without a separator are skipped",
			input: "Netskope Client Diagnostics\n\nClient version:: 117.1.0.1234.\n",
			want:  map[string]string{"client_version": "117.1.0.1234"},
		},
		{
			name:  "unrecognized keys are ignored",
			input: "Client version:: 117.1.0.1234.\nSome New Field:: noise.\n",
			want:  map[string]string{"client_version": "117.1.0.1234"},
		},
		{
			name:  "empty values are preserved",
			input: "Email::.\n",
			want:  map[string]string{"email": ""},
		},
		{
			name:  "values containing a period keep it",
			input: "Tenant URL:: company.goskope.com.\n",
			want:  map[string]string{"tenant_url": "company.goskope.com"},
		},
		{
			// The parser splits on the first "::" so a value that contains one is
			// not truncated.
			name:  "separator inside the value",
			input: "Gateway:: host::8443.\n",
			want:  map[string]string{"gateway": "host::8443"},
		},
		{
			name:  "later lines win",
			input: "Client status:: enable.\nClient status:: disable.\n",
			want:  map[string]string{"client_status": "disable"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, parseNsdiagText([]byte(tc.input)))
		})
	}
}

func TestParseNsdiagTextNilInput(t *testing.T) {
	t.Parallel()

	got := parseNsdiagText(nil)
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestNormalizeNsdiagValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in, want string
	}{
		{"TRUE", "TRUE"},
		{"true", "TRUE"},
		{"True", "TRUE"},
		{"FALSE", "FALSE"},
		{"false", "FALSE"},
		// Only the literal words are normalized. Numeric values must pass through
		// untouched: strconv.ParseBool would turn "1" into TRUE and corrupt any
		// field that legitimately reports a number.
		{"1", "1"},
		{"0", "0"},
		{"t", "t"},
		{"203.0.113.10", "203.0.113.10"},
		{"Not Configured", "Not Configured"},
		{"", ""},
	} {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, normalizeNsdiagValue(tc.in))
		})
	}
}
