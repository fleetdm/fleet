package main

import (
	"encoding/json"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

func TestInsertSignatureBlock(t *testing.T) {
	input := []byte(`{
  "name": "Box Drive",
  "slug": "box-drive/windows",
  "package_identifier": "Box.Box",
  "unique_identifier": "Box",
  "installer_arch": "x64",
  "installer_type": "msi",
  "installer_scope": "machine",
  "default_categories": ["Productivity"]
}
`)

	sig := &maintained_apps.FMASignature{SubjectCNs: []string{"Box, Inc."}}
	updated, err := insertSignatureBlock(input, sig)
	require.NoError(t, err)

	want := `{
  "name": "Box Drive",
  "slug": "box-drive/windows",
  "package_identifier": "Box.Box",
  "unique_identifier": "Box",
  "installer_arch": "x64",
  "installer_type": "msi",
  "installer_scope": "machine",
  "default_categories": ["Productivity"],
  "signature": {
    "subject_cns": [
      "Box, Inc."
    ]
  }
}
`
	require.Equal(t, want, string(updated))

	// The result parses back to the same pin.
	var parsed struct {
		Name      string                        `json:"name"`
		Signature *maintained_apps.FMASignature `json:"signature"`
	}
	require.NoError(t, json.Unmarshal(updated, &parsed))
	require.Equal(t, "Box Drive", parsed.Name)
	require.NotNil(t, parsed.Signature)
	require.Equal(t, []string{"Box, Inc."}, parsed.Signature.SubjectCNs)
}

func TestInsertSignatureBlockDarwin(t *testing.T) {
	input := []byte(`{
  "name": "Rectangle",
  "slug": "rectangle/darwin",
  "token": "rectangle",
  "default_categories": ["Productivity"]
}
`)
	sig := &maintained_apps.FMASignature{AppleTeamID: "XSYZ3E4B7D", Notarized: true}
	updated, err := insertSignatureBlock(input, sig)
	require.NoError(t, err)

	var parsed struct {
		Signature *maintained_apps.FMASignature `json:"signature"`
	}
	require.NoError(t, json.Unmarshal(updated, &parsed))
	require.Equal(t, "XSYZ3E4B7D", parsed.Signature.AppleTeamID)
	require.True(t, parsed.Signature.Notarized)
}

func TestInsertSignatureBlockRejectsExisting(t *testing.T) {
	input := []byte(`{
  "name": "X",
  "signature": { "subject_cns": ["X Corp"] }
}
`)
	_, err := insertSignatureBlock(input, &maintained_apps.FMASignature{SubjectCNs: []string{"Y"}})
	require.ErrorContains(t, err, "already has a signature block")
}

func TestInsertSignatureBlockRejectsInvalidJSON(t *testing.T) {
	_, err := insertSignatureBlock([]byte("not json"), &maintained_apps.FMASignature{SubjectCNs: []string{"Y"}})
	require.ErrorContains(t, err, "not valid JSON")
}

func TestRecordPinSkipsCases(t *testing.T) {
	repoRoot := t.TempDir()

	// Pin already present.
	recorded, err := recordPin(repoRoot, &appVerification{PinPresent: true, SignatureObservable: true})
	require.NoError(t, err)
	require.False(t, recorded)

	// Signature not observable in this environment.
	recorded, err = recordPin(repoRoot, &appVerification{SignatureObservable: false})
	require.NoError(t, err)
	require.False(t, recorded)

	// Unsigned installers need a human-written justification.
	recorded, err = recordPin(repoRoot, &appVerification{SignatureObservable: true, ObservedUnsigned: true})
	require.NoError(t, err)
	require.False(t, recorded)
}
