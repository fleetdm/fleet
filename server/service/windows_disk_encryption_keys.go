package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// wstepKeyBatchSize is how many legacy rows one query returns. Re-encryption happens on the server and costs one RSA
// decryption per row, so a run walks every row in batches of this size.
const wstepKeyBatchSize = 500

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
			return "", fmt.Errorf("encrypt with the server private key: %w", err)
		}
		// Decrypted once before it replaces the only copy of the key.
		back, err := mdm.DecodeAndDecrypt(enc, cfg.Server.PrivateKey)
		if err != nil {
			return "", fmt.Errorf("verify the re-encrypted key: %w", err)
		}
		if back != string(plain) {
			return "", errors.New("verify the re-encrypted key: mismatch")
		}
		return enc, nil
	}

	var reencrypted, marked, raced, alreadyCurrent, unreadableUnmarked int
	var archivedReencrypted, archivedAlreadyCurrent, archivedUnreadable, archivedUnreadableUnmarked int
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
			// An unmarked row is listed by its shape alone; one that the server key reads is current and left alone.
			if !key.WSTEPEncrypted {
				if _, err := mdm.DecodeAndDecrypt(key.Base64Encrypted, cfg.Server.PrivateKey); err == nil {
					alreadyCurrent++
					continue
				}
			}
			plain, err := decrypt(key.Base64Encrypted)
			if err != nil {
				// Only a row the migration marked is known to be a legacy key. An unmarked one that neither key reads is
				// left as it is: it was not written by any Fleet release this code knows, and marking it would let the
				// host learn whether the WSTEP key could read a blob of its choosing.
				if key.WSTEPEncrypted {
					unreadable = append(unreadable, key)
				} else {
					unreadableUnmarked++
				}
				continue
			}
			enc, err := encrypt(plain)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "re-encrypt disk encryption key")
			}
			replaced, err := ds.ReplaceWSTEPEncryptedDiskEncryptionKey(ctx, key.HostID, key.Base64Encrypted, enc)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "replace WSTEP encrypted disk encryption key")
			}
			if replaced {
				reencrypted++
			} else {
				// The host escrowed a new key between the read and the write; that key is already on the server key.
				raced++
			}
		}
		if err := ds.MarkWindowsDiskEncryptionKeysUndecryptable(ctx, unreadable); err != nil {
			return ctxerr.Wrap(ctx, err, "mark Windows disk encryption keys undecryptable")
		}
		marked += len(unreadable)
		if len(keys) < wstepKeyBatchSize {
			break
		}
	}

	// Archived keys never trigger a host, so there is nothing to mark. A marked one that cannot be read is taken out of
	// the queue instead of being retried every hour; an unmarked one is left alone, like its live counterpart.
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
			if !key.WSTEPEncrypted {
				if _, err := mdm.DecodeAndDecrypt(key.Base64Encrypted, cfg.Server.PrivateKey); err == nil {
					archivedAlreadyCurrent++
					continue
				}
			}
			plain, err := decrypt(key.Base64Encrypted)
			if err != nil {
				if !key.WSTEPEncrypted {
					archivedUnreadableUnmarked++
					continue
				}
				if err := ds.ClearWSTEPEncryptedArchivedDiskEncryptionKeyMarker(ctx, key.ID, key.Base64Encrypted); err != nil {
					return ctxerr.Wrap(ctx, err, "clear WSTEP encrypted archived disk encryption key marker")
				}
				archivedUnreadable++
				continue
			}
			enc, err := encrypt(plain)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "re-encrypt archived disk encryption key")
			}
			replaced, err := ds.ReplaceWSTEPEncryptedArchivedDiskEncryptionKey(ctx, key.ID, key.Base64Encrypted, enc)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "replace WSTEP encrypted archived disk encryption key")
			}
			if replaced {
				archivedReencrypted++
			}
		}
		if len(keys) < wstepKeyBatchSize {
			break
		}
	}

	if reencrypted+marked+raced+unreadableUnmarked+archivedReencrypted+archivedUnreadable+archivedUnreadableUnmarked == 0 {
		return nil
	}
	logger.InfoContext(ctx, "re-encrypted legacy Windows disk encryption keys with the server private key",
		"reencrypted", reencrypted,
		"marked_undecryptable", marked,
		"replaced_by_host_meanwhile", raced,
		"already_current", alreadyCurrent,
		"unreadable_unmarked_left_alone", unreadableUnmarked,
		"archived_reencrypted", archivedReencrypted,
		"archived_already_current", archivedAlreadyCurrent,
		"archived_unreadable", archivedUnreadable,
		"archived_unreadable_unmarked_left_alone", archivedUnreadableUnmarked,
	)
	return nil
}
