//go:build linux

package luks_runner

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

	key, err := lr.getEscrowKey(ctx)
	if len(key) == 0 {
		// Dialog was canceled or timed out
		return nil
	}

	response := LuksResponse{
		Err: err.Error(),
		Key: key,
	}

	if err := lr.escrower.SendLinuxKeyEscrowResponse(response); err != nil {
		if err := lr.infoPrompt(ctx, infoFailedTitle, infoFailedText); err != nil {
			log.Debug().Err(err).Msg("failed to show failed escrow key dialog")
		}
		return fmt.Errorf("escrower escrowKey err: %w", err)
	}

	if response.Err != "" {
		if err := lr.infoPrompt(ctx, infoFailedTitle, response.Err); err != nil {
			log.Debug().Err(err).Msg("failed to show failed escrow key dialog")
		}
		return fmt.Errorf("error getting linux escrow key: %s", response.Err)
	}

	// Show success dialog
	if err := lr.infoPrompt(ctx, infoSuccessTitle, infoSuccessText); err != nil {
		log.Debug().Err(err).Msg("failed to show success escrow key dialog")
	}

	return nil
}

func (lr *LuksRunner) getEscrowKey(ctx context.Context) ([]byte, error) {
	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		return nil, fmt.Errorf("Failed to find LUKS Root Partition: %w", err)
	}

	device := luks.New(luks.AESXTSPlain64Cipher)

	// Prompt user for existing LUKS passphrase
	passphrase, err := lr.entryPrompt(ctx, entryDialogTitle, entryDialogText)
	if err != nil {
		return nil, fmt.Errorf("Failed to show passphrase entry prompt: %w", err)
	}

	// Validate the passphrase
	for {
		valid, err := lr.passphraseIsValid(ctx, device, devicePath, passphrase)
		if err != nil {
			return nil, fmt.Errorf("Failed validating passphrase: %w", err)
		}

		if !valid {
			passphrase, err = lr.entryPrompt(ctx, entryDialogTitle, retryEntryDialogText)
			if err != nil {
				return nil, fmt.Errorf("Failed re-prompting for passphrase: %w", err)
			}
			continue
		}

		break
	}

	if passphrase == nil {
		log.Debug().Msg("Passphrase is nil, dialog was canceled or timed out")
		return nil, nil
	}

	escrowPassphrase, err := generateRandomPassphrase(randPassphraseLength)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate random passphrase: %w", err)
	}

	// Create a new key slot, error if all key slots are full
	// keySlot 0 is assumed to be the user's passphrase
	// so we start at 1
	keySlot := 1
	for {
		if keySlot == maxKeySlots {
			return nil, errors.New("all LUKS key slots are full")
		}

		userKey := encryption.NewKey(0, passphrase)
		escrowKey := encryption.NewKey(keySlot, escrowPassphrase)

		if err := device.AddKey(ctx, devicePath, userKey, escrowKey); err != nil {
			if errors.Is(err, encryption.ErrEncryptionKeyRejected) {
				keySlot++
				continue
			} else {
				return nil, fmt.Errorf("Failed to add key: %w", err)
			}
		}

		break
	}

	return escrowPassphrase, nil
}

func (lr *LuksRunner) passphraseIsValid(ctx context.Context, device *luks.LUKS, devicePath string, passphrase []byte) (bool, error) {
	valid, err := device.CheckKey(ctx, devicePath, encryption.NewKey(0, passphrase))
	if err != nil {
		return false, fmt.Errorf("Error validating passphrase: %w", err)
	}

	return valid, nil
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
