package osv

import (
	"bytes"
	"compress/gzip"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeTestArtifact(t *testing.T, dir, name, body string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o600))
}

const testArtifactDate = "2026-11-26"

func readTestArtifact(t *testing.T, prefix, version, body string, dst any) {
	t.Helper()
	dir := t.TempDir()
	writeTestArtifact(t, dir, prefix+version+"-"+testArtifactDate+".json.gz", body)
	date, err := time.Parse(dateLayout, testArtifactDate)
	require.NoError(t, err)
	_, err = readArtifact(dir, prefix, version, date, dst)
	require.NoError(t, err)
}

func TestReadArtifactSharesRepeatedVersionLists(t *testing.T) {
	body := `{"schema_version":"1.0","ubuntu_version":"24.04","vulnerabilities":{
		"openssh":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1:9.6p1-3ubuntu13.5","versions":["1:9.6p1-3ubuntu13","1:9.6p1-3ubuntu13.4"]}],
		"openssh-client":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1:9.6p1-3ubuntu13.5","versions":["1:9.6p1-3ubuntu13","1:9.6p1-3ubuntu13.4"]}],
		"openssh-server":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1:9.6p1-3ubuntu13.5","versions":["1:9.6p1-3ubuntu13","1:9.6p1-3ubuntu13.4"]},
			{"cve":"CVE-2026-0002","introduced":"0","versions":["1:9.6p1-3ubuntu13"]}]
	}}`
	var artifact OSVArtifact
	readTestArtifact(t, OSVFilePrefix, "2404", body, &artifact)

	shared := OSVVulnerability{CVE: "CVE-2026-0001", Introduced: "0", Fixed: "1:9.6p1-3ubuntu13.5", Versions: []string{"1:9.6p1-3ubuntu13", "1:9.6p1-3ubuntu13.4"}}
	require.Equal(t, map[string][]OSVVulnerability{
		"openssh":        {shared},
		"openssh-client": {shared},
		"openssh-server": {shared, {CVE: "CVE-2026-0002", Introduced: "0", Versions: []string{"1:9.6p1-3ubuntu13"}}},
	}, artifact.Vulnerabilities)

	backing := map[uintptr]int{}
	for _, rows := range artifact.Vulnerabilities {
		for _, row := range rows {
			if row.CVE == "CVE-2026-0001" {
				backing[reflect.ValueOf(row.Versions).Pointer()]++
			}
		}
	}
	require.Len(t, backing, 1, "identical version lists must share one backing array")
}

func TestReadArtifactMatchesPlainDecode(t *testing.T) {
	cases := []struct {
		name    string
		prefix  string
		version string
		body    string
		decode  func(t *testing.T, prefix, version, body string) (got, want any)
	}{
		{
			name:    "ubuntu",
			prefix:  OSVFilePrefix,
			version: "2404",
			body: `{"schema_version":"1.0","ubuntu_version":"24.04","generated":"2026-11-26T00:00:00Z","total_cves":2,"total_packages":3,
				"vulnerabilities":{
					"a":[{"cve":"CVE-2026-0001","published":"2026-01-01T00:00:00Z","modified":"2026-01-02T00:00:00Z","details":"unused","introduced":"0","fixed":"1.0-1","versions":["0.9-1","0.9-2"]},
						{"cve":"CVE-2026-0002","introduced":null,"fixed":"2.0+1","versions":[]}],
					"b":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1.0-1","versions":["0.9-1","0.9-2"],"unknown":{"nested":[1,2]}}],
					"c\"quoted":[{"cve":"CVE-2026-0003","introduced":"0","versions":null},{"cve":"CVE-2026-0003","introduced":"0","versions":["0.9-2","0.9-1"]}]
				}}`,
			decode: func(t *testing.T, prefix, version, body string) (any, any) {
				var got, want OSVArtifact
				readTestArtifact(t, prefix, version, body, &got)
				require.NoError(t, json.Unmarshal([]byte(body), &want))
				return got, want
			},
		},
		{
			name:    "rhel",
			prefix:  OSVRHELFilePrefix,
			version: "9",
			body: `{"schema_version":"1.0","rhel_version":"9","generated":"2026-11-26T00:00:00Z","total_cves":1,"total_packages":2,
				"vulnerabilities":{
					"openssl":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1:3.0.7-27.el9"}],
					"openssl-libs":[{"cve":"CVE-2026-0001","introduced":"0","fixed":"1:3.0.7-27.el9"}]
				}}`,
			decode: func(t *testing.T, prefix, version, body string) (any, any) {
				var got, want RHELOSVArtifact
				readTestArtifact(t, prefix, version, body, &got)
				require.NoError(t, json.Unmarshal([]byte(body), &want))
				return got, want
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, want := tc.decode(t, tc.prefix, tc.version, tc.body)
			require.Equal(t, want, got)
		})
	}
}

func FuzzDecodeArtifact(f *testing.F) {
	for _, seed := range []string{
		`{"vulnerabilities":{"a":[{"cve":"CVE-1","introduced":"0","fixed":"1.0","versions":["0.9","0.8"]}],"b":[{"cve":"CVE-1","introduced":"0","fixed":"1.0","versions":["0.9","0.8"]}]}}`,
		`{"vulnerabilities":{"a":[{"cve":"CVE-1","fixed":"a\"b\\u00e9","versions":[]}],"b":[{"cve":null,"versions":null}]}}`,
		`{"vulnerabilities":{"a":[{"cve":1}]}}`,
		`{"vulnerabilities":{"a":[{"versions":"0.9"}]}}`,
		`{"vulnerabilities":{"a":[{"versions":[1]}]}}`,
		`{"generated":"2026-11-26T00:00:00Z","vulnerabilities":{}}`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var want, got OSVArtifact
		wantErr := json.Unmarshal(data, &want)
		gotErr := decodeArtifact(bytes.NewReader(data), &got)
		require.Equal(t, wantErr == nil, gotErr == nil, "plain error: %v, interned error: %v", wantErr, gotErr)
		if wantErr == nil {
			require.Equal(t, want, got)
		}
	})
}
