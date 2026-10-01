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
		Tag:          "tag",
		Group:        "group",
		Title:        `Set <your> "PIN"`,
		Body:         "Files & folders aren't safe",
		ButtonLabel:  "Create PIN",
		URL:          `https://fleet.example.com/device/token?create_pin=1&x="<y>"`,
		StayOnScreen: true,
	}
	payload, err := n.xml()
	require.NoError(t, err)

	for _, raw := range []string{n.Title, n.Body, n.URL} {
		require.NotContains(t, payload, raw, "unescaped value in the toast XML")
	}

	var parsed toastXML
	require.NoError(t, xml.Unmarshal([]byte(payload), &parsed))
	require.Equal(t, "protocol", parsed.ActivationType)
	require.Equal(t, "reminder", parsed.Scenario)
	require.Equal(t, n.URL, parsed.Launch)
	require.Equal(t, "ToastGeneric", parsed.Binding.Template)
	require.Equal(t, []string{n.Title, n.Body}, parsed.Binding.Text)
	require.Equal(t, []toastAction{{Content: n.ButtonLabel, ActivationType: "protocol", Arguments: n.URL}}, parsed.Actions)
}

// validNotification is a notification Windows accepts, for tests that change one thing about it.
var validNotification = Notification{
	Tag:         "bitlocker-pin",
	Group:       "fleet",
	Title:       "Title",
	Body:        "Body",
	ButtonLabel: "Create PIN",
	URL:         "https://fleet.example.com/device/token?create_pin=1",
}

// with returns a copy of the notification with set applied.
func with(n Notification, set func(*Notification)) Notification {
	set(&n)
	return n
}

func TestShowScriptEnv(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                        string
		n                           Notification
		wantExpires, wantSuppressed string
	}{
		{
			name:           "expires in an hour, posted silently",
			n:              with(validNotification, func(n *Notification) { n.ExpiresIn = time.Hour; n.SuppressPopup = true }),
			wantExpires:    "3600",
			wantSuppressed: "true",
		},
		{
			// The script leaves the expiration alone for zero, so an unset one does not make the toast arrive expired.
			name:           "no expiration, pops up",
			n:              validNotification,
			wantExpires:    "0",
			wantSuppressed: "false",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env, err := showScriptEnv(tc.n)
			require.NoError(t, err)
			vars := envMap(t, env)

			wantXML, err := tc.n.xml()
			require.NoError(t, err)
			require.Equal(t, map[string]string{
				"FLEET_TOAST_XML":             wantXML,
				"FLEET_TOAST_TAG":             tc.n.Tag,
				"FLEET_TOAST_GROUP":           tc.n.Group,
				"FLEET_TOAST_EXPIRES_SECONDS": tc.wantExpires,
				"FLEET_TOAST_SUPPRESS_POPUP":  tc.wantSuppressed,
				"FLEET_TOAST_APP_ID":          FleetDesktopAppID,
			}, vars)
			requireScriptReadsEnv(t, showScript, vars)
		})
	}
}

func TestRemoveScriptEnv(t *testing.T) {
	t.Parallel()

	vars := envMap(t, removeScriptEnv("bitlocker-pin", "fleet"))
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

func TestNotificationValidate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		n       Notification
		wantErr string
	}{
		{name: "valid", n: validNotification},
		{name: "no title", n: with(validNotification, func(n *Notification) { n.Title = "" }), wantErr: "toast title is empty"},
		{name: "no body", n: with(validNotification, func(n *Notification) { n.Body = "" }), wantErr: "toast body is empty"},
		{name: "no button label", n: with(validNotification, func(n *Notification) { n.ButtonLabel = "" }), wantErr: "toast button label is empty"},
		{name: "no URL", n: with(validNotification, func(n *Notification) { n.URL = "" }), wantErr: "toast URL is empty"},
		{name: "no tag", n: with(validNotification, func(n *Notification) { n.Tag = "" }), wantErr: "toast tag is empty"},
		// Windows throws on a longer tag or group, which would surface as an opaque PowerShell failure.
		{
			name:    "group Windows would reject",
			n:       with(validNotification, func(n *Notification) { n.Group = strings.Repeat("g", maxTagLength+1) }),
			wantErr: "toast group is longer than the 64 characters Windows accepts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.n.validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestChildEnv(t *testing.T) {
	t.Parallel()

	// Fleet Desktop's own variables carry the Fleet client TLS key and the device URL, and PowerShell has no use for them.
	env := childEnv([]string{
		"PATH=C:\\Windows",
		"FLEET_DESKTOP_FLEET_TLS_CLIENT_KEY=-----BEGIN PRIVATE KEY-----",
		"fleet_desktop_device_url=https://fleet.example.com/device/token",
	}, []string{"FLEET_TOAST_TAG=bitlocker-pin"})

	require.Equal(t, []string{"PATH=C:\\Windows", "FLEET_TOAST_TAG=bitlocker-pin"}, env)
}
