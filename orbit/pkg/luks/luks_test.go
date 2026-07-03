package luks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tj/assert"
)

// output from cryptsetup luksDump /dev/sda3 --debug-json command
// on cryptsetup 2.2.2
var output = `# cryptsetup 2.2.2 processing "cryptsetup luksDump /dev/sda3 --debug-json"
# Running command luksDump.
# Locking memory.
# Installing SIGINT/SIGTERM handler.
# Unblocking interruption on signal.
# Allocating context for crypt device /dev/sda3.
# Trying to open and read device /dev/sda3 with direct-io.
# Initialising device-mapper backend library.
# Trying to load any crypt type from device /dev/sda3.
# Crypto backend (OpenSSL 1.1.1f  31 Mar 2020) initialized in cryptsetup library version 2.2.2.
# Detected kernel Linux 5.4.0-200-generic aarch64.
# Loading LUKS2 header (repair disabled).
# Acquiring read lock for device /dev/sda3.
# Opening lock resource file /run/cryptsetup/L_8:3
# Verifying lock handle for /dev/sda3.
# Device /dev/sda3 READ lock taken.
# Trying to read primary LUKS2 header at offset 0x0.
# Opening locked device /dev/sda3
# Veryfing locked device handle (bdev)
# LUKS2 header version 2 of size 16384 bytes, checksum sha256.
# Checksum:f16cfd36bbc588eb178d9f40e7da030edc6ed04cfb012ee14be14e3af459439b (on-disk)
# Checksum:f16cfd36bbc588eb178d9f40e7da030edc6ed04cfb012ee14be14e3af459439b (in-memory)
# Trying to read secondary LUKS2 header at offset 0x4000.
# Reusing open ro fd on device /dev/sda3
# LUKS2 header version 2 of size 16384 bytes, checksum sha256.
# Checksum:0ce47e7c0460addb7a5ee2546e8404521ca0caf2e02830b1d4ef7172bf617d84 (on-disk)
# Checksum:0ce47e7c0460addb7a5ee2546e8404521ca0caf2e02830b1d4ef7172bf617d84 (in-memory)
# Device size 65442676736, offset 16777216.
# Device /dev/sda3 READ lock released.
# Only 2 active CPUs detected, PBKDF threads decreased from 4 to 2.
# Not enough physical memory detected, PBKDF max memory decreased from 1048576kB to 1010800kB.
# PBKDF argon2i, time_ms 2000 (iterations 0), max_memory_kb 1010800, parallel_threads 2.
# {
  "keyslots":{
    "0":{
      "type":"luks2",
      "key_size":64,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"258048",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2i",
        "time":5,
        "memory":1011024,
        "cpus":2,
        "salt":"b8+t4hTY/IFecqsKR20UZSUPFDZqyAtJ9lxYg5ye2Hg="
      }
    }
  },
  "tokens":{
  },
  "segments":{
    "0":{
      "type":"crypt",
      "offset":"16777216",
      "size":"dynamic",
      "iv_tweak":"0",
      "encryption":"aes-xts-plain64",
      "sector_size":512
    }
  },
  "digests":{
    "0":{
      "type":"pbkdf2",
      "keyslots":[
        "0"
      ],
      "segments":[
        "0"
      ],
      "hash":"sha256",
      "iterations":457493,
      "salt":"BbZbHfAY5e90aoKjTYSJoj8ZLiRueUofVJWWG/4trWw=",
      "digest":"f2sSMaJlh5qbdVUYT+RhabOmEit96KBIq0ltzAnfARc="
    }
  },
  "config":{
    "json_size":"12288",
    "keyslots_size":"16744448"
  }
}LUKS header information
Version:       	2
Epoch:         	5
Metadata area: 	16384 [bytes]
Keyslots area: 	16744448 [bytes]
UUID:          	3cf8a080-045d-4bc2-889c-994c9b4a18aa
Label:         	(no label)
Subsystem:     	(no subsystem)
Flags:       	(no flags)

Data segments:
  0: crypt
	offset: 16777216 [bytes]
	length: (whole device)
	cipher: aes-xts-plain64
	sector: 512 [bytes]

Keyslots:
  0: luks2
	Key:        512 bits
	Priority:   normal
	Cipher:     aes-xts-plain64
	Cipher key: 512 bits
	PBKDF:      argon2i
	Time cost:  5
	Memory:     1011024
	Threads:    2
	Salt:       6f cf ad e2 14 d8 fc 81 5e 72 ab 0a 47 6d 14 65
	            25 0f 14 36 6a c8 0b 49 f6 5c 58 83 9c 9e d8 78
	AF stripes: 4000
	AF hash:    sha256
	Area offset:32768 [bytes]
	Area length:258048 [bytes]
	Digest ID:  0
Tokens:
Digests:
  0: pbkdf2
	Hash:       sha256
	Iterations: 457493
	Salt:       05 b6 5b 1d f0 18 e5 ef 74 6a 82 a3 4d 84 89 a2
	            3f 19 2e 24 6e 79 4a 1f 54 95 96 1b fe 2d ad 6c
	Digest:     7f 6b 12 31 a2 65 87 9a 9b 75 55 18 4f e4 61 69
	            b3 a6 12 2b 7d e8 a0 48 ab 49 6d cc 09 df 01 17
# Releasing crypt device /dev/sda3 context.
# Releasing device-mapper backend.
# Closing read only fd for /dev/sda3.
# Unlocking memory.
Command successful.`

var extractedJSON = `{
  "keyslots":{
    "0":{
      "type":"luks2",
      "key_size":64,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"258048",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2i",
        "time":5,
        "memory":1011024,
        "cpus":2,
        "salt":"b8+t4hTY/IFecqsKR20UZSUPFDZqyAtJ9lxYg5ye2Hg="
      }
    }
  },
  "tokens":{
  },
  "segments":{
    "0":{
      "type":"crypt",
      "offset":"16777216",
      "size":"dynamic",
      "iv_tweak":"0",
      "encryption":"aes-xts-plain64",
      "sector_size":512
    }
  },
  "digests":{
    "0":{
      "type":"pbkdf2",
      "keyslots":[
        "0"
      ],
      "segments":[
        "0"
      ],
      "hash":"sha256",
      "iterations":457493,
      "salt":"BbZbHfAY5e90aoKjTYSJoj8ZLiRueUofVJWWG/4trWw=",
      "digest":"f2sSMaJlh5qbdVUYT+RhabOmEit96KBIq0ltzAnfARc="
    }
  },
  "config":{
    "json_size":"12288",
    "keyslots_size":"16744448"
  }
}`

func TestExtractJson(t *testing.T) {
	json, err := extractJSON([]byte(output))
	assert.NoError(t, err)
	assert.Equal(t, extractedJSON, string(json))

	_, err = extractJSON([]byte("no json"))
	assert.Error(t, err)
}

// tpm2AndRecoveryDumpJSON is a LUKS2 luksDump JSON fragment that includes a
// systemd-tpm2 entry alongside a recovery entry, matching what
// systemd-cryptenroll writes on a TPM-backed Ubuntu install. Used to assert
// that the Tokens field of LuksDump parses correctly.
const tpm2AndRecoveryDumpJSON = `{
  "keyslots":{
    "0":{
      "type":"luks2",
      "kdf":{"type":"argon2id","salt":"abc"}
    },
    "1":{
      "type":"luks2",
      "kdf":{"type":"argon2id","salt":"def"}
    }
  },
  "tokens":{
    "0":{
      "type":"systemd-tpm2",
      "keyslots":["1"],
      "tpm2-blob":"blob",
      "tpm2-pcrs":[7]
    },
    "1":{
      "type":"systemd-recovery",
      "keyslots":["0"]
    }
  },
  "segments":{},
  "digests":{},
  "config":{}
}`

func TestLuksDumpTokensUnmarshal(t *testing.T) {
	t.Run("cryptsetup <2.4 debug output has empty tokens", func(t *testing.T) {
		raw, err := extractJSON([]byte(output))
		require.NoError(t, err)

		var dump LuksDump
		require.NoError(t, json.Unmarshal(raw, &dump))
		assert.Empty(t, dump.Tokens)
		assert.NotEmpty(t, dump.Keyslots, "keyslots should still parse")
	})

	t.Run("tpm2 + recovery tokens parse with type field", func(t *testing.T) {
		var dump LuksDump
		require.NoError(t, json.Unmarshal([]byte(tpm2AndRecoveryDumpJSON), &dump))
		require.Len(t, dump.Tokens, 2)
		assert.Equal(t, systemdTPM2Type, dump.Tokens["0"].Type)
		assert.Equal(t, systemdRecoveryType, dump.Tokens["1"].Type)
	})
}

func TestDetectEncryptionType(t *testing.T) {
	cases := []struct {
		name string
		dump *LuksDump
		want string
	}{
		{
			name: "nil dump",
			dump: nil,
			want: EncryptionTypePassphrase,
		},
		{
			name: "no tokens map (nil)",
			dump: &LuksDump{},
			want: EncryptionTypePassphrase,
		},
		{
			name: "empty tokens map",
			dump: &LuksDump{Tokens: map[string]Token{}},
			want: EncryptionTypePassphrase,
		},
		{
			name: "only tpm2",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdTPM2Type},
			}},
			want: EncryptionTypeTPM2,
		},
		{
			name: "only fido2",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdFIDO2Type},
			}},
			want: EncryptionTypeFIDO2,
		},
		{
			name: "only recovery",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdRecoveryType},
			}},
			want: EncryptionTypeRecovery,
		},
		{
			name: "tpm2 + recovery -> tpm2",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdTPM2Type},
				"1": {Type: systemdRecoveryType},
			}},
			want: EncryptionTypeTPM2,
		},
		{
			name: "fido2 + recovery -> fido2",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdFIDO2Type},
				"1": {Type: systemdRecoveryType},
			}},
			want: EncryptionTypeFIDO2,
		},
		{
			name: "tpm2 + fido2 + recovery -> tpm2",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: systemdRecoveryType},
				"1": {Type: systemdFIDO2Type},
				"2": {Type: systemdTPM2Type},
			}},
			want: EncryptionTypeTPM2,
		},
		{
			name: "unknown token type -> passphrase",
			dump: &LuksDump{Tokens: map[string]Token{
				"0": {Type: "some-future-thing"},
			}},
			want: EncryptionTypePassphrase,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, DetectEncryptionType(tc.dump))
		})
	}
}

func TestIsSnapdManaged(t *testing.T) {
	cases := []struct {
		name string
		dump *LuksDump
		want bool
	}{
		{name: "nil dump", dump: nil, want: false},
		{name: "no tokens", dump: &LuksDump{}, want: false},
		{
			name: "systemd tpm2 only is not snapd-managed",
			dump: &LuksDump{Tokens: map[string]Token{"0": {Type: systemdTPM2Type}}},
			want: false,
		},
		{
			name: "snapd fde token",
			dump: &LuksDump{Tokens: map[string]Token{"0": {Type: "ubuntu-fde"}}},
			want: true,
		},
		{
			name: "snapd fde token mixed case",
			dump: &LuksDump{Tokens: map[string]Token{"0": {Type: "Ubuntu-FDE-hook"}}},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsSnapdManaged(tc.dump))
		})
	}
}

func TestParseRecoveryKey(t *testing.T) {
	key, err := parseRecoveryKey("Recovery key: 55055-39320-64491-48436-47667-15525-36879-32875\n")
	require.NoError(t, err)
	require.Equal(t, "55055-39320-64491-48436-47667-15525-36879-32875", key)

	_, err = parseRecoveryKey("no key here")
	require.Error(t, err)
}

type fakeEscrower struct {
	responses    []LuksResponse
	err          error
	capabilities fleet.CapabilityMap
}

func (f *fakeEscrower) SendLinuxKeyEscrowResponse(r LuksResponse) error {
	f.responses = append(f.responses, r)
	return f.err
}

func (f *fakeEscrower) GetServerCapabilities() fleet.CapabilityMap {
	if f.capabilities == nil {
		return fleet.CapabilityMap{}
	}
	return f.capabilities
}

// newRecoveryKeyEscrower returns a fake escrower whose server capabilities
// already include LUKSRecoveryKeyEscrow, so tests focused on the happy path
// don't have to repeat the setup.
func newRecoveryKeyEscrower() *fakeEscrower {
	return &fakeEscrower{
		capabilities: fleet.CapabilityMap{
			fleet.CapabilityLUKSRecoveryKeyEscrow: {},
		},
	}
}

// fakeNotifier records ShowInfo invocations so tests can assert the
// notification firing rules without a real desktop.
type fakeNotifier struct {
	infoCalls []dialog.InfoOptions
}

func (f *fakeNotifier) ShowEntry(dialog.EntryOptions) ([]byte, error) {
	return nil, errors.New("unused in these tests")
}

func (f *fakeNotifier) ShowInfo(opts dialog.InfoOptions) error {
	f.infoCalls = append(f.infoCalls, opts)
	return nil
}

type fakeSnapdFDE struct {
	recoveryKey string
	ensureErr   error
	ensureCalls int
}

func (f *fakeSnapdFDE) Detect(ctx context.Context) (bool, error) { return true, nil }
func (f *fakeSnapdFDE) EnsureFleetRecoveryKey(ctx context.Context) (string, error) {
	f.ensureCalls++
	return f.recoveryKey, f.ensureErr
}

func TestRunRecoveryKeyEscrow(t *testing.T) {
	ctx := t.Context()
	const recoveryKey = "55055-39320-64491-48436-47667-15525-36879-32875"

	t.Run("success escrows the recovery key with no salt or key slot", func(t *testing.T) {
		escrower := newRecoveryKeyEscrower()
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		lr := New(escrower)

		require.NoError(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		require.Len(t, escrower.responses, 1)
		resp := escrower.responses[0]
		assert.Equal(t, fleet.LUKSKeyTypeRecoveryKey, resp.KeyType)
		assert.Equal(t, recoveryKey, resp.Passphrase)
		assert.Empty(t, resp.Salt)
		assert.Nil(t, resp.KeySlot)
		assert.Empty(t, resp.Err)
	})

	t.Run("reports a client error when key creation fails", func(t *testing.T) {
		escrower := newRecoveryKeyEscrower()
		snapd := &fakeSnapdFDE{ensureErr: errors.New("snap-tpmctl boom")}
		lr := New(escrower)

		require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		require.Len(t, escrower.responses, 1)
		resp := escrower.responses[0]
		assert.Equal(t, fleet.LUKSKeyTypeRecoveryKey, resp.KeyType)
		assert.Empty(t, resp.Passphrase)
		assert.NotEmpty(t, resp.Err)
	})

	t.Run("returns an error when escrow to the server fails", func(t *testing.T) {
		// snapd has no recovery-key delete operation, so there is no rollback;
		// the enrolled key is replaced on the next escrow attempt.
		escrower := newRecoveryKeyEscrower()
		escrower.err = errors.New("network down")
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		lr := New(escrower)

		require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
	})

	t.Run("server without capability skips snapd work entirely", func(t *testing.T) {
		escrower := &fakeEscrower{} // no capabilities advertised
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		notifier := &fakeNotifier{}
		lr := New(escrower)
		lr.notifier = notifier

		// First call: should notify.
		require.NoError(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		assert.Empty(t, escrower.responses, "must not POST to old servers")
		assert.Zero(t, snapd.ensureCalls, "must not touch snapd against old servers")
		require.Len(t, notifier.infoCalls, 1)
		assert.Equal(t, recoveryKeyServerTooOldText, notifier.infoCalls[0].Text)

		// Second call: notification is one-shot.
		require.NoError(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		assert.Len(t, notifier.infoCalls, 1, "must not spam the user")
	})

	t.Run("consecutive failures trigger one notification then stay quiet", func(t *testing.T) {
		escrower := newRecoveryKeyEscrower()
		escrower.err = errors.New("network down")
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		notifier := &fakeNotifier{}
		lr := New(escrower)
		lr.notifier = notifier

		// One below the threshold — no notification yet.
		for i := 0; i < recoveryKeyFailureNotifyThreshold-1; i++ {
			require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		}
		assert.Empty(t, notifier.infoCalls, "must stay silent below the threshold")

		// Crossing the threshold fires exactly one notification.
		require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		require.Len(t, notifier.infoCalls, 1)
		assert.Equal(t, recoveryKeyEscrowFailedText, notifier.infoCalls[0].Text)

		// Further failures do not re-fire.
		require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		assert.Len(t, notifier.infoCalls, 1, "notification is one-shot per streak")
	})

	t.Run("a successful escrow arms the notification for the next failure streak", func(t *testing.T) {
		escrower := newRecoveryKeyEscrower()
		escrower.err = errors.New("network down")
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		notifier := &fakeNotifier{}
		lr := New(escrower)
		lr.notifier = notifier

		// Fail past the threshold to fire the notification.
		for i := 0; i < recoveryKeyFailureNotifyThreshold; i++ {
			require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		}
		require.Len(t, notifier.infoCalls, 1)

		// Now a success resets the streak and the one-shot flag.
		escrower.err = nil
		require.NoError(t, lr.runRecoveryKeyEscrow(ctx, snapd))

		// A fresh streak eventually re-notifies.
		escrower.err = errors.New("network down")
		for i := 0; i < recoveryKeyFailureNotifyThreshold; i++ {
			require.Error(t, lr.runRecoveryKeyEscrow(ctx, snapd))
		}
		assert.Len(t, notifier.infoCalls, 2, "new streak should trigger a new notification")
	})

	t.Run("notifier stays optional (no dialog tool installed)", func(t *testing.T) {
		escrower := &fakeEscrower{}
		snapd := &fakeSnapdFDE{recoveryKey: recoveryKey}
		lr := New(escrower) // no notifier assigned

		// Must not panic despite no notifier.
		require.NoError(t, lr.runRecoveryKeyEscrow(ctx, snapd))
	})
}
