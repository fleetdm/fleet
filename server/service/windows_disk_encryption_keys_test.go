package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

func TestReencryptWSTEPDiskEncryptionKeys(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	appleCert, appleKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	appleCertPEM, appleKeyPEM := tokenpki.PEMCertificate(appleCert.Raw), tokenpki.PEMRSAPrivateKey(appleKey)
	// MicrosoftWSTEP caches the parsed pair on the config, so each scenario that needs a different pair starts fresh.
	newConfig := func(t *testing.T) config.FleetConfig {
		cfg := config.TestConfig()
		config.SetTestMDMConfig(t, &cfg, appleCertPEM, appleKeyPEM, "")
		return cfg
	}
	cfg := newConfig(t)
	wstep, _, _, err := cfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)

	encryptTo := func(t *testing.T, cert *x509.Certificate, plain string) string {
		enc, err := pkcs7.Encrypt([]byte(plain), []*x509.Certificate{cert})
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(enc)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	readable := encryptTo(t, wstep.Leaf, "key-1")
	// Same private key, new serial: what a renewed WSTEP certificate leaves behind.
	renewed := encryptTo(t, selfSignedCertForKey(t, wstep.PrivateKey.(*rsa.PrivateKey), 99), "key-2")
	dead := encryptTo(t, selfSignedCertForKey(t, otherKey, 100), "key-3")
	current, err := mdm.EncryptAndEncode("key-5", cfg.Server.PrivateKey)
	require.NoError(t, err)
	archivedReadable := encryptTo(t, wstep.Leaf, "archived-1")
	archivedDead := encryptTo(t, selfSignedCertForKey(t, otherKey, 101), "archived-2")

	// recorder captures what the job writes so each scenario can assert on it.
	type recorder struct {
		replaced         map[uint]string
		archivedReplaced map[uint]string
		marked           []fleet.HostDiskEncryptionKey
	}
	newStore := func(t *testing.T, keys []fleet.HostDiskEncryptionKey, archived []fleet.HostArchivedDiskEncryptionKey) (*mock.Store, *recorder) {
		ds := new(mock.Store)
		rec := &recorder{replaced: map[uint]string{}, archivedReplaced: map[uint]string{}}
		blobOf := map[uint]string{}
		for _, k := range keys {
			blobOf[k.HostID] = k.Base64Encrypted
		}
		ds.ListWSTEPEncryptedDiskEncryptionKeysFunc = func(ctx context.Context, afterHostID uint, limit int) ([]fleet.HostDiskEncryptionKey, error) {
			var page []fleet.HostDiskEncryptionKey
			for _, k := range keys {
				if k.HostID > afterHostID && len(page) < limit {
					page = append(page, k)
				}
			}
			return page, nil
		}
		ds.ListWSTEPEncryptedArchivedDiskEncryptionKeysFunc = func(ctx context.Context, afterID uint, limit int) ([]fleet.HostArchivedDiskEncryptionKey, error) {
			var page []fleet.HostArchivedDiskEncryptionKey
			for _, k := range archived {
				if k.ID > afterID && len(page) < limit {
					page = append(page, k)
				}
			}
			return page, nil
		}
		ds.ReplaceHostDiskEncryptionKeyBlobFunc = func(ctx context.Context, hostID uint, current, updated string) (bool, error) {
			require.Equal(t, blobOf[hostID], current, "compare-and-swap must use the blob that was read")
			rec.replaced[hostID] = updated
			return true, nil
		}
		ds.ReplaceWSTEPEncryptedArchivedDiskEncryptionKeyFunc = func(ctx context.Context, id uint, current, updated string) (bool, error) {
			rec.archivedReplaced[id] = updated
			return true, nil
		}
		ds.MarkWindowsDiskEncryptionKeysUndecryptableFunc = func(ctx context.Context, keys []fleet.HostDiskEncryptionKey) error {
			for _, k := range keys {
				require.Equal(t, blobOf[k.HostID], k.Base64Encrypted, "marking must name the blob that was read")
			}
			rec.marked = append(rec.marked, keys...)
			return nil
		}
		return ds, rec
	}
	markedHosts := func(rec *recorder) []uint {
		var ids []uint
		for _, k := range rec.marked {
			ids = append(ids, k.HostID)
		}
		return ids
	}
	decrypted := func(t *testing.T, blob string) string {
		plain, err := mdm.DecodeAndDecrypt(blob, cfg.Server.PrivateKey)
		require.NoError(t, err)
		return plain
	}

	t.Run("readable keys move to the server private key, unreadable ones are marked", func(t *testing.T) {
		ds, rec := newStore(t,
			[]fleet.HostDiskEncryptionKey{
				{HostID: 1, Base64Encrypted: readable, WSTEPEncrypted: true},
				{HostID: 2, Base64Encrypted: renewed, WSTEPEncrypted: true},
				{HostID: 3, Base64Encrypted: dead, WSTEPEncrypted: true},
				{HostID: 4, Base64Encrypted: "not even base64", WSTEPEncrypted: true},
				// Shorter than an AES-GCM nonce: a malformed blob must not stop the run.
				{HostID: 8, Base64Encrypted: "MIIA", WSTEPEncrypted: true},
				// Marked but already on the server key: only the marker goes.
				{HostID: 9, Base64Encrypted: current, WSTEPEncrypted: true},
			},
			[]fleet.HostArchivedDiskEncryptionKey{
				{ID: 10, HostID: 1, Base64Encrypted: archivedReadable, WSTEPEncrypted: true},
				{ID: 11, HostID: 3, Base64Encrypted: archivedDead, WSTEPEncrypted: true},
				{ID: 12, HostID: 9, Base64Encrypted: current, WSTEPEncrypted: true},
			})
		require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &cfg))

		require.Len(t, rec.replaced, 3)
		require.Equal(t, "key-1", decrypted(t, rec.replaced[1]))
		require.Equal(t, "key-2", decrypted(t, rec.replaced[2]))
		require.Equal(t, current, rec.replaced[9], "a current blob is written back unchanged to clear the marker")
		require.Equal(t, []uint{3, 4, 8}, markedHosts(rec))
		// Every marked archive row leaves the queue: re-encrypted, written back unchanged, or unreadable.
		require.Len(t, rec.archivedReplaced, 3)
		require.Equal(t, "archived-1", decrypted(t, rec.archivedReplaced[10]))
		require.Equal(t, archivedDead, rec.archivedReplaced[11], "an unreadable archived key keeps its blob")
		require.Equal(t, current, rec.archivedReplaced[12])
	})

	// A replaced certificate leaves every key unreadable; one run walks all batches and marks them all.
	t.Run("one run walks every batch and marks every unreadable key", func(t *testing.T) {
		var keys []fleet.HostDiskEncryptionKey
		for i := uint(1); i <= 3*wstepKeyBatchSize+1; i++ {
			blob := dead
			if i%2 == 0 {
				blob = readable
			}
			keys = append(keys, fleet.HostDiskEncryptionKey{HostID: i, Base64Encrypted: blob, WSTEPEncrypted: true})
		}
		ds, rec := newStore(t, keys, nil)
		require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &cfg))
		require.Len(t, rec.replaced, len(keys)/2)
		require.Len(t, rec.marked, len(keys)-len(keys)/2)
	})

	// Without a usable pair nothing can read a legacy key, and a host told to re-escrow could not: the escrow endpoint
	// needs Windows MDM, which needs the pair. The rows are left alone rather than marked.
	t.Run("without a WSTEP pair nothing runs", func(t *testing.T) {
		noWSTEP := config.TestConfig()
		ds, _ := newStore(t, []fleet.HostDiskEncryptionKey{{HostID: 1, Base64Encrypted: readable, WSTEPEncrypted: true}}, nil)
		require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &noWSTEP))
		require.False(t, ds.ListWSTEPEncryptedDiskEncryptionKeysFuncInvoked)
	})

	t.Run("with a WSTEP pair that does not load nothing runs", func(t *testing.T) {
		badWSTEP := config.TestConfig()
		badWSTEP.MDM.WindowsWSTEPIdentityCert = "testdata/does-not-exist.pem"
		badWSTEP.MDM.WindowsWSTEPIdentityKey = "testdata/does-not-exist.key"
		ds, _ := newStore(t, []fleet.HostDiskEncryptionKey{{HostID: 1, Base64Encrypted: readable, WSTEPEncrypted: true}}, nil)
		require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &badWSTEP))
		require.False(t, ds.ListWSTEPEncryptedDiskEncryptionKeysFuncInvoked)
	})

	t.Run("without a server private key nothing runs", func(t *testing.T) {
		noPrivateKey := newConfig(t)
		noPrivateKey.Server.PrivateKey = ""
		ds, _ := newStore(t, []fleet.HostDiskEncryptionKey{{HostID: 1, Base64Encrypted: readable, WSTEPEncrypted: true}}, nil)
		require.NoError(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &noPrivateKey))
		require.False(t, ds.ListWSTEPEncryptedDiskEncryptionKeysFuncInvoked)
	})

	t.Run("a server private key that cannot encrypt stops the run before any write", func(t *testing.T) {
		badPrivateKey := newConfig(t)
		badPrivateKey.Server.PrivateKey = "too short for AES"
		ds, rec := newStore(t, []fleet.HostDiskEncryptionKey{{HostID: 1, Base64Encrypted: readable, WSTEPEncrypted: true}}, nil)
		require.Error(t, ReencryptWSTEPDiskEncryptionKeys(ctx, logger, ds, &badPrivateKey))
		require.Empty(t, rec.replaced)
		require.Empty(t, rec.marked)
	})
}
