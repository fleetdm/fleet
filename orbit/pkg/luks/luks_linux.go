//go:build linux

package luks

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	"github.com/siderolabs/go-blockdevice/v2/encryption/luks"
)

func (lr *LuksRunner) Run(oc *fleet.OrbitConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !oc.Notifications.EscrowLinuxKey {
		log.Debug().Msg("EscrowLinuxKey is false, skipping")
		return nil
	}

	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		return fmt.Errorf("devicepath err: %w", err)
	}

	device := luks.New(luks.AESXTSPlain64Cipher)

	// Prompt user for existing LUKS passphrase
	passphrase, err := lr.entryPrompt(ctx, entryDialogTitle, entryDialogText)
	if err != nil {
		return fmt.Errorf("entryPrompt err: %w", err)
	}

	escrowPassphrase, err := generateRandomPassphrase(randPassphraseLength)
	if err != nil {
		return fmt.Errorf("generateRandomPassphrase err: %w", err)
	}

	// Create a new key slot, error if all key slots are full
	// keySlot 0 is assumed to be the user's passphrase
	// so we start at 1
	keySlot := 1
	for {
		if keySlot == maxKeySlots {
			return errors.New("all LUKS key slots are full")
		}

		userKey := encryption.NewKey(0, passphrase)
		escrowKey := encryption.NewKey(keySlot, escrowPassphrase)

		if err := device.AddKey(ctx, devicePath, userKey, escrowKey); err != nil {
			if errors.Is(err, encryption.ErrEncryptionKeyRejected) {
				passphrase, err = lr.entryPrompt(ctx, entryDialogTitle, retryEntryDialogText)
				if err != nil {
					return fmt.Errorf("reEntryPrompt err: %w", err)
				}
				continue
			}

			keySlot++
			continue
		}

		break
	}

	// Escrow the escrow passphrase
	// TODO(tim): add retry or key removal
	if err := lr.escrower.EscrowLinuxKey(passphrase); err != nil {
		if err := lr.infoPrompt(ctx, infoFailedTitle, infoFailedText); err != nil {
			log.Debug().Err(err).Msg("failed to show failed escrow key dialog")
		}
		return fmt.Errorf("escrower escrowKey err: %w", err)
	}

	// Show success dialog
	if err := lr.infoPrompt(ctx, infoSuccessTitle, infoSuccessText); err != nil {
		log.Debug().Err(err).Msg("failed to show success escrow key dialog")
	}

	return nil
}

func generateRandomPassphrase(length int) ([]byte, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}

	return bytes, nil
}

func (lr *LuksRunner) entryPrompt(ctx context.Context, title, text string) ([]byte, error) {
	passphrase, err := lr.notifier.ShowEntry(ctx, dialog.EntryOptions{
		Title:    title,
		Text:     text,
		HideText: true,
		TimeOut:  1 * time.Minute,
	})
	if err != nil {
		switch err {
		case dialog.ErrCanceled:
			log.Info().Msg("end user canceled key escrow dialog")
			return nil, nil
		case dialog.ErrTimeout:
			log.Info().Msg("key escrow dialog timed out")
			return nil, nil
		case dialog.ErrUnknown:
			return nil, err
		default:
			return nil, err
		}
	}

	return passphrase, nil
}

func (lr *LuksRunner) infoPrompt(ctx context.Context, title, text string) error {
	err := lr.notifier.ShowInfo(ctx, dialog.InfoOptions{
		Title:   title,
		Text:    text,
		TimeOut: 30 * time.Second,
	})
	if err != nil {
		switch err {
		case dialog.ErrTimeout:
			log.Debug().Msg("successPrompt timed out")
			return nil
		default:
			return err
		}
	}

	return nil
}
