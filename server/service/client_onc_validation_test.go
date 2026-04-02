package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateONCCertificateReferences(t *testing.T) {
	writeProfile := func(t *testing.T, dir, name, content string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		return path
	}

	t.Run("matching cert reference passes", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := writeProfile(t, dir, "wifi.json", `{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {"EAP": {"ClientCertKeyPairAlias": "my-cert"}}
				}]
			}
		}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: profilePath},
					}),
					Certificates: optjson.SetSlice([]fleet.CertificateTemplateSpec{
						{Name: "my-cert", CertificateAuthorityName: "test-ca", SubjectName: "CN=Test"},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, config.Controls.AndroidSettings.(fleet.AndroidSettings).Certificates.Value)
		require.NoError(t, err)
	})

	t.Run("missing cert reference fails", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := writeProfile(t, dir, "wifi.json", `{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {"EAP": {"ClientCertKeyPairAlias": "missing-cert"}}
				}]
			}
		}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: profilePath},
					}),
					Certificates: optjson.SetSlice([]fleet.CertificateTemplateSpec{
						{Name: "other-cert", CertificateAuthorityName: "test-ca", SubjectName: "CN=Test"},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, config.Controls.AndroidSettings.(fleet.AndroidSettings).Certificates.Value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `references certificate "missing-cert"`)
		assert.Contains(t, err.Error(), "ClientCertKeyPairAlias")
	})

	t.Run("profile without ONC passes", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := writeProfile(t, dir, "camera.json", `{"cameraDisabled": true}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: profilePath},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, nil)
		require.NoError(t, err)
	})

	t.Run("no profiles passes", func(t *testing.T) {
		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{},
			},
		}

		err := validateONCCertificateReferences(config, nil)
		require.NoError(t, err)
	})

	t.Run("nil android settings passes", func(t *testing.T) {
		config := &spec.GitOps{}
		err := validateONCCertificateReferences(config, nil)
		require.NoError(t, err)
	})

	t.Run("no certs but ONC references cert fails", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := writeProfile(t, dir, "wifi.json", `{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {"EAP": {"ClientCertKeyPairAlias": "my-cert"}}
				}]
			}
		}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: profilePath},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `references certificate "my-cert"`)
	})

	t.Run("ONC without cert alias passes", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := writeProfile(t, dir, "wifi.json", `{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {"SSID": "Guest", "Security": "WPA-PSK", "Passphrase": "pass123"}
				}]
			}
		}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: profilePath},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, nil)
		require.NoError(t, err)
	})

	t.Run("multiple profiles one bad reference fails", func(t *testing.T) {
		dir := t.TempDir()
		goodProfile := writeProfile(t, dir, "camera.json", `{"cameraDisabled": true}`)
		badProfile := writeProfile(t, dir, "wifi.json", `{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {"EAP": {"ClientCertKeyPairAlias": "bad-cert"}}
				}]
			}
		}`)

		config := &spec.GitOps{
			Controls: spec.GitOpsControls{
				AndroidSettings: fleet.AndroidSettings{
					CustomSettings: optjson.SetSlice([]fleet.MDMProfileSpec{
						{Path: goodProfile},
						{Path: badProfile},
					}),
					Certificates: optjson.SetSlice([]fleet.CertificateTemplateSpec{
						{Name: "good-cert", CertificateAuthorityName: "test-ca", SubjectName: "CN=Test"},
					}),
				},
			},
		}

		err := validateONCCertificateReferences(config, config.Controls.AndroidSettings.(fleet.AndroidSettings).Certificates.Value)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `references certificate "bad-cert"`)
	})
}
