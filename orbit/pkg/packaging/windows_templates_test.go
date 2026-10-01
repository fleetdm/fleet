package packaging

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wixNode is a generic XML node used to walk the rendered .wxs without modeling the whole WiX schema.
type wixNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []wixNode  `xml:",any"`
}

func (n wixNode) attr(name string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func (n wixNode) find(local string, pred func(wixNode) bool) *wixNode {
	if n.XMLName.Local == local && pred(n) {
		return &n
	}
	for _, c := range n.Nodes {
		if found := c.find(local, pred); found != nil {
			return found
		}
	}
	return nil
}

// windowsServiceEnvironment renders the MSI template and returns the NAME=value entries of the "Fleet osquery"
// service's per-service Environment registry value, failing if the output isn't well-formed XML.
func windowsServiceEnvironment(t *testing.T, opt Options) []string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, windowsWixTemplate.Execute(&buf, opt))
	output := buf.String()
	assert.NotContains(t, output, "Arguments=", "orbit must be configured via environment, not ServiceInstall Arguments")

	var root wixNode
	require.NoError(t, xml.Unmarshal(buf.Bytes(), &root), "rendered .wxs is not well-formed XML")

	env := root.find("RegistryValue", func(n wixNode) bool {
		return n.attr("Name") == "Environment" && strings.HasSuffix(n.attr("Key"), `\Services\Fleet osquery`)
	})
	require.NotNil(t, env, "service Environment RegistryValue not found")
	assert.Equal(t, "multiString", env.attr("Type"))

	var entries []string
	for _, c := range env.Nodes {
		require.Equal(t, "MultiStringValue", c.XMLName.Local)
		entries = append(entries, c.Content)
	}
	return entries
}

func TestWindowsWixTemplateServiceEnvironment(t *testing.T) {
	baseOpt := Options{
		FleetURL:            "https://fleet.example.com",
		EnrollSecret:        "secret",
		OrbitChannel:        "stable",
		OsquerydChannel:     "stable",
		DesktopChannel:      "stable",
		OrbitUpdateInterval: 900000000000,
		NativePlatform:      "windows",
		Architecture:        ArchAmd64,
	}

	t.Run("minimal options", func(t *testing.T) {
		entries := windowsServiceEnvironment(t, baseOpt)
		assert.Equal(t, []string{
			"ORBIT_ROOT_DIR=[ORBITROOT].",
			`ORBIT_LOG_FILE=[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log`,
			"ORBIT_FLEET_URL=[FLEET_URL]",
			"ORBIT_ENROLL_SECRET_PATH=[ORBITROOT]secret.txt",
			"ORBIT_UPDATE_INTERVAL=15m0s",
			"ORBIT_FLEET_DESKTOP=[FLEET_DESKTOP]",
			"ORBIT_DESKTOP_CHANNEL=stable",
			"ORBIT_ORBIT_CHANNEL=stable",
			"ORBIT_OSQUERYD_CHANNEL=stable",
			"ORBIT_ENABLE_SCRIPTS=[ENABLE_SCRIPTS]",
		}, entries)
	})

	t.Run("all options", func(t *testing.T) {
		opt := baseOpt
		opt.FleetCertificate = "fleet.pem"
		opt.Insecure = true
		opt.Debug = true
		opt.UpdateURL = "https://tuf.example.com"
		opt.UpdateTLSServerCertificate = "update.pem"
		opt.DisableUpdates = true
		opt.FleetDesktopAlternativeBrowserHost = "localhost:8080"
		opt.HostIdentifier = "instance"
		opt.EnableEndUserEmailProperty = true
		opt.EnableEUATokenProperty = true
		opt.OsqueryDB = `C:\osquery.db`
		opt.DisableSetupExperience = true
		opt.BypassEndUserAuth = true

		entries := windowsServiceEnvironment(t, opt)
		for _, want := range []string{
			"ORBIT_FLEET_CERTIFICATE=[ORBITROOT]fleet.pem",
			"ORBIT_INSECURE=true",
			"ORBIT_DEBUG=true",
			"ORBIT_UPDATE_URL=https://tuf.example.com",
			"ORBIT_UPDATE_TLS_CERTIFICATE=[ORBITROOT]update.pem",
			"ORBIT_DISABLE_UPDATES=true",
			"ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST=localhost:8080",
			"ORBIT_HOST_IDENTIFIER=instance",
			"ORBIT_END_USER_EMAIL=[END_USER_EMAIL]",
			"ORBIT_EUA_TOKEN=[EUA_TOKEN]",
			`ORBIT_OSQUERY_DB=C:\osquery.db`,
			"ORBIT_DISABLE_SETUP_EXPERIENCE=true",
			"ORBIT_BYPASS_END_USER_AUTH=true",
		} {
			assert.Contains(t, entries, want)
		}
		for _, e := range entries {
			assert.Regexp(t, `^ORBIT_[A-Z_]+=`, e)
		}
	})

	t.Run("uuid host identifier is orbit's default and omitted", func(t *testing.T) {
		opt := baseOpt
		opt.HostIdentifier = "uuid"
		for _, e := range windowsServiceEnvironment(t, opt) {
			assert.NotContains(t, e, "ORBIT_HOST_IDENTIFIER")
		}
	})

	t.Run("literal end user email without the MSI property", func(t *testing.T) {
		opt := baseOpt
		opt.EndUserEmail = "user@example.com"
		assert.Contains(t, windowsServiceEnvironment(t, opt), "ORBIT_END_USER_EMAIL=user@example.com")
	})
}
