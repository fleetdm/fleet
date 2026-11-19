package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// wstepKeyBatchSize is how many legacy rows one query returns. Re-encryption happens on the server and costs one RSA
// decryption per row, so a run walks every row in batches of this size.
const wstepKeyBatchSize = 500

// wstepKeyBatchPause keeps the run from hammering the database: each batch is a read plus up to one write per row. A
// variable so tests can skip it.
var wstepKeyBatchPause = time.Second

// ReencryptWSTEPDiskEncryptionKeys re-encrypts legacy Windows recovery keys, which older releases encrypted to the WSTEP
// identity certificate, so they are protected by the server private key like every other key. Keys the certificate
// cannot read are marked undecryptable, which makes the host escrow a new one; a replaced certificate makes every key
// unreadable at once and they are all marked in that run, since an unreadable key is already lost. It finishes in one
// run on a healthy server, so later runs find nothing to do.
func ReencryptWSTEPDiskEncryptionKeys(ctx context.Context, logger *slog.Logger, ds fleet.Datastore, cfg *config.FleetConfig) error {
	if cfg.Server.PrivateKey == "" {
		return nil
	}
	// Without the pair nothing can read a legacy key, and a host asked to re-escrow could not do it either: the escrow
	// endpoint requires Windows MDM, which cannot be on without the pair. The rows are left as they are.
	if !cfg.MDM.IsMicrosoftWSTEPSet() {
		return nil
	}
	cert, _, _, err := cfg.MDM.MicrosoftWSTEP()
	if err != nil {
		logger.WarnContext(ctx, "skipping re-encryption of legacy Windows disk encryption keys, cannot load the WSTEP certificate", "err", err)
		return nil
	}

	serverKeyReads := func(b64 string) bool {
		_, err := mdm.DecodeAndDecrypt(b64, cfg.Server.PrivateKey)
		return err == nil
	}
	decrypt := func(b64 string) ([]byte, error) {
		plain, err := mdm.DecryptBase64CMSWithKey(b64, cert.PrivateKey)
		if err != nil {
			return nil, err
		}
		if len(plain) == 0 {
			return nil, errors.New("decrypted key is empty")
		}
		return plain, nil
	}
	encrypt := func(plain []byte) (string, error) {
		enc, err := mdm.EncryptAndEncode(string(plain), cfg.Server.PrivateKey)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "encrypt with the server private key")
		}
		// Decrypted once before it replaces the only copy of the key.
		back, err := mdm.DecodeAndDecrypt(enc, cfg.Server.PrivateKey)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "verify the re-encrypted key")
		}
		if back != string(plain) {
			return "", ctxerr.New(ctx, "verify the re-encrypted key: mismatch")
		}
		return enc, nil
	}
	pause := func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wstepKeyBatchPause):
			return nil
		}
	}

	var reencrypted, marked, raced, alreadyCurrent, archivedReencrypted, archivedAlreadyCurrent, archivedUnreadable int
	var afterHostID uint
	for {
		keys, err := ds.ListWSTEPEncryptedDiskEncryptionKeys(ctx, afterHostID, wstepKeyBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list WSTEP encrypted disk encryption keys")
		}
		if len(keys) == 0 {
			break
		}
		var unreadable []fleet.HostDiskEncryptionKey
		for _, key := range keys {
			afterHostID = key.HostID
			// A marked row the server key already reads only needs its marker cleared, which the same compare-and-swap
			// does when given the stored blob back.
			if serverKeyReads(key.Base64Encrypted) {
				alreadyCurrent++
				if _, err := ds.ReplaceHostDiskEncryptionKeyBlob(ctx, key.HostID, key.Base64Encrypted, key.Base64Encrypted); err != nil {
					return ctxerr.Wrap(ctx, err, "clear marker on a current disk encryption key")
				}
				continue
			}
			plain, err := decrypt(key.Base64Encrypted)
			if err != nil {
				unreadable = append(unreadable, key)
				continue
			}
			enc, err := encrypt(plain)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "re-encrypt disk encryption key")
			}
			replaced, err := ds.ReplaceHostDiskEncryptionKeyBlob(ctx, key.HostID, key.Base64Encrypted, enc)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "replace WSTEP encrypted disk encryption key")
			}
			if !replaced {
				// The host escrowed a new key between the read and the write; that key is already on the server key.
				raced++
				continue
			}
			reencrypted++
		}
		if err := ds.MarkWindowsDiskEncryptionKeysUndecryptable(ctx, unreadable); err != nil {
			return ctxerr.Wrap(ctx, err, "mark Windows disk encryption keys undecryptable")
		}
		marked += len(unreadable)
		if len(keys) < wstepKeyBatchSize {
			break
		}
		if err := pause(); err != nil {
			return err
		}
	}

	// Archived keys never trigger a host, so there is nothing to mark. One that cannot be read has its marker cleared
	// instead of being retried every hour; it stays unreadable, which is the truth for it.
	var afterID uint
	for {
		keys, err := ds.ListWSTEPEncryptedArchivedDiskEncryptionKeys(ctx, afterID, wstepKeyBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list WSTEP encrypted archived disk encryption keys")
		}
		if len(keys) == 0 {
			break
		}
		for _, key := range keys {
			afterID = key.ID
			newBlob := key.Base64Encrypted
			if serverKeyReads(key.Base64Encrypted) {
				archivedAlreadyCurrent++
			} else if plain, err := decrypt(key.Base64Encrypted); err != nil {
				archivedUnreadable++
			} else if newBlob, err = encrypt(plain); err != nil {
				return ctxerr.Wrap(ctx, err, "re-encrypt archived disk encryption key")
			}
			replaced, err := ds.ReplaceWSTEPEncryptedArchivedDiskEncryptionKey(ctx, key.ID, key.Base64Encrypted, newBlob)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "replace WSTEP encrypted archived disk encryption key")
			}
			if replaced && newBlob != key.Base64Encrypted {
				archivedReencrypted++
			}
		}
		if len(keys) < wstepKeyBatchSize {
			break
		}
		if err := pause(); err != nil {
			return err
		}
	}

	if reencrypted+marked+raced+alreadyCurrent+archivedReencrypted+archivedAlreadyCurrent+archivedUnreadable == 0 {
		return nil
	}
	logger.InfoContext(ctx, "re-encrypted legacy Windows disk encryption keys with the server private key",
		"reencrypted", reencrypted,
		"marked_undecryptable", marked,
		"replaced_by_host_meanwhile", raced,
		"already_current", alreadyCurrent,
		"archived_reencrypted", archivedReencrypted,
		"archived_already_current", archivedAlreadyCurrent,
		"archived_unreadable", archivedUnreadable,
	)
	return nil
}
