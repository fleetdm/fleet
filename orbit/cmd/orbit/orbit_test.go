package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeKeystore is a test double for enrollSecretKeystore.
type fakeKeystore struct {
	supported bool
	secret    string
	getErr    error
	addErr    error
	updateErr error

	getCalls    int
	addCalls    int
	updateCalls int
}

func (f *fakeKeystore) Supported() bool { return f.supported }
func (f *fakeKeystore) Name() string    { return "fake keystore" }

func (f *fakeKeystore) GetSecret() (string, error) {
	f.getCalls++
	if f.getErr != nil {
		return "", f.getErr
	}
	return f.secret, nil
}

func (f *fakeKeystore) AddSecret(secret string) error {
	f.addCalls++
	if f.addErr != nil {
		return f.addErr
	}
	f.secret = secret
	return nil
}

func (f *fakeKeystore) UpdateSecret(secret string) error {
	f.updateCalls++
	if f.updateErr != nil {
		return f.updateErr
	}
	f.secret = secret
	return nil
}

func TestReadEnrollSecretFromFile(t *testing.T) {
	t.Run("empty file does not touch keystore or set the secret", func(t *testing.T) {
		// Reproduces the --use-system-configuration ABM scenario: an empty
		// secret.txt must not trigger keystore.AddSecret (which rejects empty
		// secrets with "secret cannot be empty").
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte("   \n"), 0o600))

		ks := &fakeKeystore{supported: true}
		var setCalled bool
		err := readEnrollSecretFromFile(path, ks, false, func(string) error {
			setCalled = true
			return nil
		})
		require.NoError(t, err)
		require.False(t, setCalled, "enroll secret should not be set from an empty file")
		require.Zero(t, ks.addCalls, "AddSecret must not be attempted with an empty secret")
		require.Zero(t, ks.getCalls)
		require.FileExists(t, path, "empty file should be left untouched")
	})

	t.Run("empty file does not touch an existing keystore secret", func(t *testing.T) {
		// With a populated keystore, the old code reached the update branch and
		// logged a spurious "failed to update enroll secret" warning (UpdateSecret
		// rejects the empty secret). The early return must skip the keystore
		// entirely and leave the stored secret intact.
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte("\n  \n"), 0o600))

		ks := &fakeKeystore{supported: true, secret: "existing"}
		err := readEnrollSecretFromFile(path, ks, false, func(string) error {
			t.Fatal("setSecret must not be called for an empty file")
			return nil
		})
		require.NoError(t, err)
		require.Zero(t, ks.getCalls, "keystore must not be queried for an empty file")
		require.Zero(t, ks.addCalls)
		require.Zero(t, ks.updateCalls)
		require.Equal(t, "existing", ks.secret, "existing keystore secret must be preserved")
	})

	t.Run("missing file is a no-op when keystore is supported", func(t *testing.T) {
		ks := &fakeKeystore{supported: true}
		err := readEnrollSecretFromFile(filepath.Join(t.TempDir(), "missing.txt"), ks, false, func(string) error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("missing file errors when keystore is unsupported", func(t *testing.T) {
		ks := &fakeKeystore{supported: false}
		err := readEnrollSecretFromFile(filepath.Join(t.TempDir(), "missing.txt"), ks, false, func(string) error {
			return nil
		})
		require.Error(t, err)
	})

	t.Run("missing file errors as ErrNotExist when keystore is disabled", func(t *testing.T) {
		// The --use-system-configuration loop relies on errors.Is(err, os.ErrNotExist)
		// matching through the wrap to keep polling when the local secret file is
		// absent during bootstrap.
		ks := &fakeKeystore{supported: true}
		err := readEnrollSecretFromFile(filepath.Join(t.TempDir(), "missing.txt"), ks, true, func(string) error {
			return nil
		})
		require.Error(t, err)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("adds secret to empty keystore and deletes file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte(" mysecret \n"), 0o600))

		ks := &fakeKeystore{supported: true}
		var got string
		err := readEnrollSecretFromFile(path, ks, false, func(s string) error {
			got = s
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, "mysecret", got)
		require.Equal(t, 1, ks.addCalls)
		require.Equal(t, "mysecret", ks.secret)
		require.NoFileExists(t, path, "file should be deleted once stored in keystore")
	})

	t.Run("disabled keystore sets secret but does not store or delete", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte("mysecret"), 0o600))

		ks := &fakeKeystore{supported: true}
		var got string
		err := readEnrollSecretFromFile(path, ks, true, func(s string) error {
			got = s
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, "mysecret", got)
		require.Zero(t, ks.addCalls)
		require.FileExists(t, path)
	})

	t.Run("matching keystore secret deletes file without writing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte("mysecret"), 0o600))

		ks := &fakeKeystore{supported: true, secret: "mysecret"}
		err := readEnrollSecretFromFile(path, ks, false, func(string) error { return nil })
		require.NoError(t, err)
		require.Zero(t, ks.addCalls)
		require.Zero(t, ks.updateCalls)
		require.NoFileExists(t, path)
	})

	t.Run("different keystore secret is updated", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "secret.txt")
		require.NoError(t, os.WriteFile(path, []byte("newsecret"), 0o600))

		ks := &fakeKeystore{supported: true, secret: "oldsecret"}
		err := readEnrollSecretFromFile(path, ks, false, func(string) error { return nil })
		require.NoError(t, err)
		require.Equal(t, 1, ks.updateCalls)
		require.Equal(t, "newsecret", ks.secret)
		require.NoFileExists(t, path)
	})
}

func TestTryReadEnrollSecretFromKeystore(t *testing.T) {
	t.Run("empty keystore is not an error", func(t *testing.T) {
		// Regression for the malformed `%!w(<nil>)` log: an empty keystore must
		// return nil (nothing to load), not a wrapped nil error.
		ks := &fakeKeystore{supported: true, secret: ""}
		var setCalled bool
		err := tryReadEnrollSecretFromKeystore("", ks, false, func(string) error {
			setCalled = true
			return nil
		})
		require.NoError(t, err)
		require.False(t, setCalled)
	})

	t.Run("propagates keystore read error", func(t *testing.T) {
		ks := &fakeKeystore{supported: true, getErr: errors.New("boom")}
		err := tryReadEnrollSecretFromKeystore("", ks, false, func(string) error { return nil })
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	})

	t.Run("loads secret from keystore", func(t *testing.T) {
		ks := &fakeKeystore{supported: true, secret: "fromkeystore"}
		var got string
		err := tryReadEnrollSecretFromKeystore("", ks, false, func(s string) error {
			got = s
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, "fromkeystore", got)
	})

	t.Run("no-op when secret already set", func(t *testing.T) {
		ks := &fakeKeystore{supported: true, secret: "fromkeystore"}
		err := tryReadEnrollSecretFromKeystore("already-set", ks, false, func(string) error { return nil })
		require.NoError(t, err)
		require.Zero(t, ks.getCalls)
	})

	t.Run("no-op when keystore unsupported", func(t *testing.T) {
		ks := &fakeKeystore{supported: false}
		err := tryReadEnrollSecretFromKeystore("", ks, false, func(string) error { return nil })
		require.NoError(t, err)
		require.Zero(t, ks.getCalls)
	})

	t.Run("no-op when keystore disabled", func(t *testing.T) {
		ks := &fakeKeystore{supported: true, secret: "fromkeystore"}
		err := tryReadEnrollSecretFromKeystore("", ks, true, func(string) error { return nil })
		require.NoError(t, err)
		require.Zero(t, ks.getCalls)
	})
}

func TestCfgsDiffer(t *testing.T) {
	for _, tc := range []struct {
		name           string
		overrideCfg    *serverOverridesConfig
		orbitConfig    *fleet.OrbitConfig
		desktopEnabled bool
		expected       bool
	}{
		{
			name:        "initial set of remote configuration",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Orbit:    "stable",
					Osqueryd: "stable",
					Desktop:  "stable",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, omit some channels",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Orbit: "stable",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, change orbit and omit some channels",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Orbit: "edge",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name:        "initial set of remote configuration, set desktop when Fleet Desktop disabled",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: false,
			expected:       false,
		},
		{
			name:        "initial set of remote configuration, set desktop with Fleet Desktop enabled",
			overrideCfg: &serverOverridesConfig{},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: true,
			expected:       true,
		},
		{
			name: "overrides update, set desktop with Fleet Desktop enabled",
			overrideCfg: &serverOverridesConfig{
				DesktopChannel: "other",
			},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Desktop: "foobar",
				},
			},
			desktopEnabled: true,
			expected:       true,
		},
		{
			name: "overrides update, change orbit",
			overrideCfg: &serverOverridesConfig{
				OrbitChannel: "first",
			},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Orbit: "second",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name: "overrides update, change osqueryd",
			overrideCfg: &serverOverridesConfig{
				OsquerydChannel: "first",
			},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{
					Osqueryd: "second",
				},
			},
			desktopEnabled: false,
			expected:       true,
		},
		{
			name: "overrides update, empty means stable",
			overrideCfg: &serverOverridesConfig{
				OrbitChannel:    "stable",
				OsquerydChannel: "stable",
				DesktopChannel:  "stable",
			},
			orbitConfig: &fleet.OrbitConfig{
				UpdateChannels: &fleet.OrbitUpdateChannels{},
			},
			desktopEnabled: true,
			expected:       false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := cfgsDiffer(tc.overrideCfg, tc.orbitConfig, tc.desktopEnabled)
			require.Equal(t, tc.expected, v)
		})
	}
}

func TestProcessLog(t *testing.T) {
	runner := desktopRunner{}
	runner.errorNotifyCh = make(chan string, 1)

	// Nothing to report
	runner.processLog("")
	assert.Empty(t, runner.errorNotifyCh)
	assert.Nil(t, runner.errorsReported)

	// No errors found
	runner.processLog("line 1\nline 2")
	assert.Empty(t, runner.errorNotifyCh)
	assert.Nil(t, runner.errorsReported)

	// Process log with known error
	runner.processLog("line 1\n" + string(logErrorLaunchServicesSubstr) + "bozo")
	require.Len(t, runner.errorNotifyCh, 1)
	msg := <-runner.errorNotifyCh
	assert.Equal(t, string(logErrorLaunchServicesMsg), msg)

	// Process known error again
	runner.processLog(string(logErrorLaunchServicesSubstr))
	assert.Empty(t, runner.errorNotifyCh)

	// Process another error
	runner.processLog("line 1" + string(logErrorMissingExecSubstr) + "\nbozo")
	require.Len(t, runner.errorNotifyCh, 1)
	msg = <-runner.errorNotifyCh
	assert.Equal(t, string(logErrorMissingExecMsg), msg)
}
