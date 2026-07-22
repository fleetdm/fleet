package maintained_apps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFMASignatureValidate(t *testing.T) {
	testCases := []struct {
		name     string
		sig      FMASignature
		platform string
		wantErr  string
	}{
		{
			name:     "valid darwin pin",
			sig:      FMASignature{AppleTeamID: "M683GB7CPW", Notarized: true},
			platform: "darwin",
		},
		{
			name:     "valid darwin pin without notarization",
			sig:      FMASignature{AppleTeamID: "XSYZ3E4B7D"},
			platform: "darwin",
		},
		{
			name:     "valid windows pin",
			sig:      FMASignature{SubjectCNs: []string{"Box, Inc."}},
			platform: "windows",
		},
		{
			name:     "valid windows pin with multiple CNs",
			sig:      FMASignature{SubjectCNs: []string{"Box, Inc.", "Box Inc"}},
			platform: "windows",
		},
		{
			name:     "valid unsigned pin",
			sig:      FMASignature{Unsigned: true, Justification: "vendor ships an unsigned MSI"},
			platform: "windows",
		},
		{
			name:     "unsigned without justification",
			sig:      FMASignature{Unsigned: true},
			platform: "windows",
			wantErr:  "requires a justification",
		},
		{
			name:     "unsigned with identity pin",
			sig:      FMASignature{Unsigned: true, Justification: "x", SubjectCNs: []string{"Foo"}},
			platform: "windows",
			wantErr:  "cannot also pin a signing identity",
		},
		{
			name:     "darwin pin missing team ID",
			sig:      FMASignature{Notarized: true},
			platform: "darwin",
			wantErr:  `must set "apple_team_id"`,
		},
		{
			name:     "darwin pin with invalid team ID",
			sig:      FMASignature{AppleTeamID: "not-a-team"},
			platform: "darwin",
			wantErr:  "invalid apple_team_id",
		},
		{
			name:     "darwin pin with windows CNs",
			sig:      FMASignature{AppleTeamID: "M683GB7CPW", SubjectCNs: []string{"Box, Inc."}},
			platform: "darwin",
			wantErr:  `"subject_cns" is a Windows pin`,
		},
		{
			name:     "windows pin with darwin fields",
			sig:      FMASignature{SubjectCNs: []string{"Box, Inc."}, AppleTeamID: "M683GB7CPW"},
			platform: "windows",
			wantErr:  "are darwin pins",
		},
		{
			name:     "windows pin missing CNs",
			sig:      FMASignature{},
			platform: "windows",
			wantErr:  `must set "subject_cns"`,
		},
		{
			name:     "windows pin with empty CN",
			sig:      FMASignature{SubjectCNs: []string{" "}},
			platform: "windows",
			wantErr:  "cannot be empty",
		},
		{
			name:     "unknown platform",
			sig:      FMASignature{AppleTeamID: "M683GB7CPW"},
			platform: "linux",
			wantErr:  "unknown platform",
		},
		{
			name:     "unknown platform with unsigned pin",
			sig:      FMASignature{Unsigned: true, Justification: "x"},
			platform: "linux",
			wantErr:  "unknown platform",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.sig.Validate(tc.platform)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestMatchesSubjectCN(t *testing.T) {
	sig := FMASignature{SubjectCNs: []string{"Box, Inc.", "Box Inc"}}
	require.True(t, sig.MatchesSubjectCN("Box, Inc."))
	require.True(t, sig.MatchesSubjectCN("Box Inc"))
	require.False(t, sig.MatchesSubjectCN("box, inc."))
	require.False(t, sig.MatchesSubjectCN("Evil Corp"))
}

func TestInputFilePathForSlug(t *testing.T) {
	path, err := InputFilePathForSlug("box-drive/darwin")
	require.NoError(t, err)
	require.Equal(t, filepath.Join("ee", "maintained-apps", "inputs", "homebrew", "box-drive.json"), path)

	path, err = InputFilePathForSlug("7-zip/windows")
	require.NoError(t, err)
	require.Equal(t, filepath.Join("ee", "maintained-apps", "inputs", "winget", "7-zip.json"), path)

	_, err = InputFilePathForSlug("bad-slug")
	require.ErrorContains(t, err, "invalid slug format")

	_, err = InputFilePathForSlug("app/linux")
	require.ErrorContains(t, err, "unknown platform")
}

func TestSignaturePinForSlug(t *testing.T) {
	repoRoot := t.TempDir()
	inputsDir := filepath.Join(repoRoot, "ee", "maintained-apps", "inputs", "winget")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	// input with a pin
	require.NoError(t, os.WriteFile(
		filepath.Join(inputsDir, "box-drive.json"),
		[]byte(`{"name": "Box Drive", "slug": "box-drive/windows", "signature": {"subject_cns": ["Box, Inc."]}}`),
		0o644,
	))
	pin, err := SignaturePinForSlug(repoRoot, "box-drive/windows")
	require.NoError(t, err)
	require.NotNil(t, pin)
	require.Equal(t, []string{"Box, Inc."}, pin.SubjectCNs)

	// input without a pin
	require.NoError(t, os.WriteFile(
		filepath.Join(inputsDir, "7-zip.json"),
		[]byte(`{"name": "7-Zip", "slug": "7-zip/windows"}`),
		0o644,
	))
	pin, err = SignaturePinForSlug(repoRoot, "7-zip/windows")
	require.NoError(t, err)
	require.Nil(t, pin)

	// missing input file
	_, err = SignaturePinForSlug(repoRoot, "missing/windows")
	require.ErrorContains(t, err, "reading app input file")
}
