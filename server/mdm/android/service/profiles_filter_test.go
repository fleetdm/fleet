package service

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterProfilesWithPendingCerts(t *testing.T) {
	t.Parallel()
	oncProfile := func(uuid, name, certAlias string) (*fleet.MDMAndroidProfilePayload, json.RawMessage) {
		payload := &fleet.MDMAndroidProfilePayload{
			ProfileUUID: uuid,
			ProfileName: name,
		}
		content := json.RawMessage([]byte(`{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [{
					"WiFi": {
						"EAP": {
							"ClientCertType": "KeyPairAlias",
							"ClientCertKeyPairAlias": "` + certAlias + `"
						}
					}
				}]
			}
		}`))
		return payload, content
	}

	nonONCProfile := func(uuid, name string) (*fleet.MDMAndroidProfilePayload, json.RawMessage) {
		payload := &fleet.MDMAndroidProfilePayload{
			ProfileUUID: uuid,
			ProfileName: name,
		}
		content := json.RawMessage([]byte(`{"cameraDisabled": true}`))
		return payload, content
	}

	filter := func(profiles []*fleet.MDMAndroidProfilePayload, contents map[string]json.RawMessage, certStatuses map[string]fleet.CertificateTemplateStatus) (ready, withheld []*fleet.MDMAndroidProfilePayload) {
		return filterProfilesWithPendingCerts(profiles, extractProfileCertAliases(t.Context(), testutils.TestLogger(t), contents), certStatuses)
	}

	t.Run("pending cert withholds ONC profile", func(t *testing.T) {
		prof, content := oncProfile("p1", "wifi-profile", "my-cert")
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{"p1": content}
		certStatuses := map[string]fleet.CertificateTemplateStatus{
			"my-cert": fleet.CertificateTemplatePending,
		}

		ready, withheld := filter(profiles, contents, certStatuses)
		require.Empty(t, ready)
		require.Len(t, withheld, 1)
		assert.Contains(t, withheld[0].Detail, `Waiting for certificate "my-cert"`)
	})

	t.Run("verified cert releases ONC profile", func(t *testing.T) {
		prof, content := oncProfile("p1", "wifi-profile", "my-cert")
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{"p1": content}
		certStatuses := map[string]fleet.CertificateTemplateStatus{
			"my-cert": fleet.CertificateTemplateVerified,
		}

		ready, withheld := filter(profiles, contents, certStatuses)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})

	t.Run("failed cert releases ONC profile", func(t *testing.T) {
		prof, content := oncProfile("p1", "wifi-profile", "my-cert")
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{"p1": content}
		certStatuses := map[string]fleet.CertificateTemplateStatus{
			"my-cert": fleet.CertificateTemplateFailed,
		}

		ready, withheld := filter(profiles, contents, certStatuses)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})

	t.Run("unknown alias releases ONC profile", func(t *testing.T) {
		prof, content := oncProfile("p1", "wifi-profile", "unknown-cert")
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{"p1": content}
		certStatuses := map[string]fleet.CertificateTemplateStatus{}

		ready, withheld := filter(profiles, contents, certStatuses)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})

	t.Run("non-ONC profile is never withheld", func(t *testing.T) {
		prof, content := nonONCProfile("p1", "camera-policy")
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{"p1": content}
		certStatuses := map[string]fleet.CertificateTemplateStatus{
			"my-cert": fleet.CertificateTemplatePending,
		}

		ready, withheld := filter(profiles, contents, certStatuses)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})

	t.Run("multiple cert refs all must be terminal", func(t *testing.T) {
		payload := &fleet.MDMAndroidProfilePayload{
			ProfileUUID: "p1",
			ProfileName: "multi-net",
		}
		content := json.RawMessage([]byte(`{
			"openNetworkConfiguration": {
				"NetworkConfigurations": [
					{"WiFi": {"EAP": {"ClientCertType": "KeyPairAlias", "ClientCertKeyPairAlias": "cert-a"}}},
					{"VPN": {"ClientCertType": "KeyPairAlias", "ClientCertKeyPairAlias": "cert-b"}}
				]
			}
		}`))
		profiles := []*fleet.MDMAndroidProfilePayload{payload}
		contents := map[string]json.RawMessage{"p1": content}

		// cert-a verified but cert-b pending: withhold
		certStatuses := map[string]fleet.CertificateTemplateStatus{
			"cert-a": fleet.CertificateTemplateVerified,
			"cert-b": fleet.CertificateTemplatePending,
		}
		ready, withheld := filter(profiles, contents, certStatuses)
		require.Empty(t, ready)
		require.Len(t, withheld, 1)
		assert.Contains(t, withheld[0].Detail, `"cert-b"`)

		// both verified: release
		certStatuses["cert-b"] = fleet.CertificateTemplateVerified
		payload.Detail = ""
		ready, withheld = filter(profiles, contents, certStatuses)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})

	t.Run("profile with missing content is not withheld", func(t *testing.T) {
		prof := &fleet.MDMAndroidProfilePayload{
			ProfileUUID: "p1",
			ProfileName: "missing-content",
		}
		profiles := []*fleet.MDMAndroidProfilePayload{prof}
		contents := map[string]json.RawMessage{} // no content for p1

		ready, withheld := filter(profiles, contents, nil)
		require.Len(t, ready, 1)
		require.Empty(t, withheld)
	})
}
