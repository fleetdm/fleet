package toast

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
)

func TestNotificationXML(t *testing.T) {
	t.Parallel()

	n := Notification{
		Tag:         "tag",
		Group:       "group",
		Title:       `Set <your> "PIN"`,
		Body:        "Files & folders aren't safe",
		ButtonLabel: "Create PIN",
		URL:         `https://fleet.example.com/device/token?create_pin=1&x="<y>"`,
	}
	payload, err := n.xml()
	require.NoError(t, err)

	for _, raw := range []string{n.Title, n.Body, n.URL} {
		require.NotContains(t, payload, raw, "unescaped value in the toast XML")
	}

	var parsed toastXML
	require.NoError(t, xml.Unmarshal([]byte(payload), &parsed))
	require.Equal(t, "protocol", parsed.ActivationType)
	require.Equal(t, n.URL, parsed.Launch)
	require.Equal(t, "ToastGeneric", parsed.Binding.Template)
	require.Equal(t, []string{n.Title, n.Body}, parsed.Binding.Text)
	require.Equal(t, []toastAction{{Content: n.ButtonLabel, ActivationType: "protocol", Arguments: n.URL}}, parsed.Actions)
}

func TestShowEnv(t *testing.T) {
	t.Parallel()

	n := Notification{
		Tag:           "bitlocker-pin",
		Group:         "fleet",
		Title:         "Title",
		Body:          "Body",
		ButtonLabel:   "Create PIN",
		URL:           "https://fleet.example.com/device/token?create_pin=1",
		ExpiresIn:     time.Hour,
		SuppressPopup: true,
	}
	env, err := showEnv(n)
	require.NoError(t, err)
	vars := envMap(t, env)

	wantXML, err := n.xml()
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"FLEET_TOAST_XML":             wantXML,
		"FLEET_TOAST_TAG":             "bitlocker-pin",
		"FLEET_TOAST_GROUP":           "fleet",
		"FLEET_TOAST_EXPIRES_SECONDS": "3600",
		"FLEET_TOAST_SUPPRESS_POPUP":  "true",
		"FLEET_TOAST_APP_ID":          FleetDesktopAppID,
	}, vars)
	requireScriptReadsEnv(t, showScript, vars)
}

func TestRemoveEnv(t *testing.T) {
	t.Parallel()

	vars := envMap(t, removeEnv("bitlocker-pin", "fleet"))
	require.Equal(t, map[string]string{
		"FLEET_TOAST_TAG":    "bitlocker-pin",
		"FLEET_TOAST_GROUP":  "fleet",
		"FLEET_TOAST_APP_ID": FleetDesktopAppID,
	}, vars)
	requireScriptReadsEnv(t, removeScript, vars)
}

func TestEncodeCommand(t *testing.T) {
	t.Parallel()

	const script = "Write-Output 'héllo 🔒'\n"
	raw, err := base64.StdEncoding.DecodeString(encodeCommand(script))
	require.NoError(t, err)
	require.Zero(t, len(raw)%2)

	units := make([]uint16, len(raw)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(raw[2*i:])
	}
	require.Equal(t, script, string(utf16.Decode(units)))
}

func envMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	vars := make(map[string]string, len(env))
	for _, kv := range env {
		name, value, ok := strings.Cut(kv, "=")
		require.True(t, ok, kv)
		vars[name] = value
	}
	return vars
}

// requireScriptReadsEnv checks that the script uses every variable it is given, so a renamed variable fails here
// rather than silently on a Windows host.
func requireScriptReadsEnv(t *testing.T, script string, vars map[string]string) {
	t.Helper()
	for name := range vars {
		require.Contains(t, script, "$env:"+name)
	}
}
