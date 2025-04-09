// based on github.com/kolide/launcher/pkg/osquery/tables
package cryptoinfo

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentify(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		in               []string
		password         string
		expectedCount    int
		expectedError    bool
		expectedSubjects []string
	}{

		{
			in:               []string{filepath.Join("testdata", "test_crt.pem")},
			expectedCount:    1,
			expectedSubjects: []string{"www.example.com"},
		},
		{
			in:               []string{filepath.Join("testdata", "test_crt.pem"), filepath.Join("testdata", "test_crt.pem")},
			expectedCount:    2,
			expectedSubjects: []string{"www.example.com", "www.example.com"},
		},
		{
			in:               []string{filepath.Join("testdata", "test_crt.der")},
			expectedCount:    1,
			expectedSubjects: []string{"www.example.com"},
		},
		{
			in:            []string{filepath.Join("testdata", "empty")},
			expectedCount: 0,
		},
		{
			in:            []string{filepath.Join("testdata", "sslcerts.pem")},
			expectedCount: 129,
			expectedSubjects: []string{
				"Autoridad de Certificacion Firmaprofesional CIF A62634068",
				"Chambers of Commerce Root - 2008",
				"Global Chambersign Root - 2008",
				"ACCVRAIZ1",
				"Actalis Authentication Root CA",
			},
		},
		{
			in:               []string{filepath.Join("testdata", "test-unenc.p12")},
			expectedCount:    2,
			expectedSubjects: []string{"www.example.com"},
		},
		{
			in:               []string{filepath.Join("testdata", "test-enc.p12")}, // password is test123
			password:         "test123",
			expectedCount:    2,
			expectedSubjects: []string{"www.example.com"},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(strings.Join(tt.in, ","), func(t *testing.T) {
			t.Parallel()

			in := []byte{}
			for _, file := range tt.in {
				fileBytes, err := os.ReadFile(file)
				require.NoError(t, err, "reading input %s for setup", file)
				in = bytes.Join([][]byte{in, fileBytes}, nil)
			}

			results, err := Identify(in, tt.password)
			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)

			// If we have expected subjects, do they match?
			count := 0
			for _, returnedCert := range results {
				// Some things aren't certs, just skep them for the expectedSubject test
				cert, ok := returnedCert.Data.(*certExtract)
				if !ok {
					continue
				}

				count++

				// If we don't have any more expected subjects, just break
				if count > len(tt.expectedSubjects) {
					break
				}

				assert.Equal(t, tt.expectedSubjects[count-1], cert.Subject.CommonName)
			}
		})
	}
}
