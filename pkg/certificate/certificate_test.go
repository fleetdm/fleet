package certificate

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPEM(t *testing.T) {
	t.Parallel()

	pool, err := LoadPEM(filepath.Join("testdata", "test.crt"))
	require.NoError(t, err)
	assert.True(t, len(pool.Subjects()) > 0)
}

func TestLoadErrorNoCertificates(t *testing.T) {
	t.Parallel()

	_, err := LoadPEM(filepath.Join("testdata", "empty.crt"))
	require.Error(t, err)
}

func TestLoadErrorMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadPEM(filepath.Join("testdata", "invalid_path"))
	require.Error(t, err)
}

func TestLoadClientCertificate(t *testing.T) {
	t.Parallel()

	validCrt, err := os.ReadFile(filepath.Join("testdata", "test.crt"))
	require.NoError(t, err)
	validKey, err := os.ReadFile(filepath.Join("testdata", "test.key"))
	require.NoError(t, err)

	for _, tc := range []struct {
		name          string
		crt           string
		key           string
		checkReturnFn func(*tls.Certificate, error) bool
	}{
		{
			name: "both values not set",
			crt:  "",
			key:  "",
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return crt == nil && err == nil
			},
		},
		{
			name: "key not set",
			crt:  "foo",
			key:  "",
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name: "crt not set",
			crt:  "",
			key:  "bar",
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name: "crt and key both set and invalid",
			crt:  "foo",
			key:  "bar",
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name: "crt and key both set but crt invalid",
			crt:  "foo",
			key:  string(validKey),
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name: "crt and key both set but key invalid",
			crt:  string(validCrt),
			key:  "bar",
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name: "crt and key set and valid",
			crt:  string(validCrt),
			key:  string(validKey),
			checkReturnFn: func(crt *tls.Certificate, err error) bool {
				return crt != nil && err == nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			crt, err := LoadClientCertificate(tc.crt, tc.key)
			require.True(t, tc.checkReturnFn(crt, err))
		})
	}
}

func TestLoadClientCertificateFromFiles(t *testing.T) {
	t.Parallel()

	validCrtPath := filepath.Join("testdata", "test.crt")
	validKeyPath := filepath.Join("testdata", "test.key")
	nonExistentPath := filepath.Join("testdata", "not-existent-file")
	invalidFilePath := filepath.Join("testdata", "empty.crt")

	for _, tc := range []struct {
		name          string
		crtPath       string
		keyPath       string
		checkReturnFn func(*Certificate, error) bool
	}{
		{
			name:    "both values not set",
			crtPath: "",
			keyPath: "",
			checkReturnFn: func(crt *Certificate, err error) bool {
				return crt == nil && err == nil
			},
		},
		{
			name:    "key not set",
			crtPath: "foo",
			keyPath: "",
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt not set",
			crtPath: "",
			keyPath: "bar",
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt and key both set and not existent",
			crtPath: nonExistentPath,
			keyPath: nonExistentPath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return crt == nil && err == nil
			},
		},
		{
			name:    "crt and key both set but crt file does not exist",
			crtPath: nonExistentPath,
			keyPath: validKeyPath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt and key both set but key file does not exist",
			crtPath: validCrtPath,
			keyPath: nonExistentPath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt and key both set but crt file contents are invalid",
			crtPath: invalidFilePath,
			keyPath: validKeyPath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt and key both set but key file contents are invalid",
			crtPath: validCrtPath,
			keyPath: invalidFilePath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return err != nil
			},
		},
		{
			name:    "crt and key set and valid",
			crtPath: validCrtPath,
			keyPath: validKeyPath,
			checkReturnFn: func(crt *Certificate, err error) bool {
				return crt != nil && len(crt.Crt.Certificate) > 0 && crt.RawCrt != nil && crt.RawKey != nil && crt.Crt.Leaf != nil && err == nil
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			crt, err := LoadClientCertificateFromFiles(tc.crtPath, tc.keyPath)
			require.True(t, tc.checkReturnFn(crt, err))
		})
	}
}
