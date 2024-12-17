//go:build linux

package luks

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os/exec"
	"regexp"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	luksdevice "github.com/siderolabs/go-blockdevice/v2/encryption/luks"
)

const (
	entryDialogTitle     = "Enter disk encryption passphrase"
	entryDialogText      = "Passphrase:"
	retryEntryDialogText = "Passphrase incorrect. Please try again."
	infoTitle            = "Disk encryption"
	infoFailedText       = "Failed to escrow key. Please try again later."
	infoSuccessText      = "Success!  Now, return to your browser window and follow the instructions to verify disk encryption."
	timeoutMessage       = "Please visit Fleet Desktop > My device and click Create key"
	maxKeySlots          = 8
	userKeySlot          = 0 // Key slot 0 is assumed to be the location of the user's passphrase
)

var ErrKeySlotFull = regexp.MustCompile(`Key slot \d+ is full`)

func (lr *LuksRunner) Run(oc *fleet.OrbitConfig) error {
	ctx := context.Background()

	if !oc.Notifications.RunDiskEncryptionEscrow {
		return nil
	}
	log.Debug().Msg("Finding root disk")
	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		return fmt.Errorf("Failed to find LUKS Root Partition: %w", err)
	}
	log.Debug().Msgf("LUKS: Found root disk: %s", devicePath)

	var response LuksResponse
	key, keyslot, err := lr.getEscrowKey(ctx, devicePath)
	if err != nil {
		log.Debug().Err(err).Msg("LUKS: Failed to get escrow key")
		response.Err = err.Error()
	}

	if len(key) == 0 && err == nil {
		log.Debug().Msg("LUKS: Escrow key is empty, no password supplied")
		// dialog was canceled or timed out
		return nil
	}

	response.Passphrase = string(key)
	response.KeySlot = keyslot

	if keyslot != nil {
		log.Debug().Msgf("LUKS: Getting salt for key slot %d", *keyslot)
		salt, err := getSaltforKeySlot(ctx, devicePath, *keyslot)
		if err != nil {
			log.Debug().Err(err).Msgf("Failed to get salt for key slot %d", *keyslot)
			if err := removeKeySlot(ctx, devicePath, *keyslot); err != nil {
				log.Error().Err(err).Msgf("failed to remove key slot %d", *keyslot)
			}
			return fmt.Errorf("Failed to get salt for key slot: %w", err)
		}
		response.Salt = salt
	}

	log.Debug().Msg("LUKS: Sending escrow key to Fleet")
	if err := lr.escrower.SendLinuxKeyEscrowResponse(response); err != nil {
		// If sending the response fails, remove the key slot
		if keyslot != nil {
			log.Debug().Msgf("LUKS: Removing key slot %d", *keyslot)
			if err := removeKeySlot(ctx, devicePath, *keyslot); err != nil {
				log.Error().Err(err).Msg("failed to remove key slot")
			}
		}

		// Show error in dialog
		log.Debug().Err(err).Msg("LUKS: Failed to escrow key, showing dialog")
		if err := lr.infoPrompt(ctx, infoTitle, infoFailedText); err != nil {
			log.Info().Err(err).Msg("failed to show failed escrow key dialog")
		}

		return fmt.Errorf("escrower escrowKey err: %w", err)
	}

	if response.Err != "" {
		log.Debug().Msg("LUKS: Showing response error dialog")
		if err := lr.infoPrompt(ctx, infoTitle, response.Err); err != nil {
			log.Info().Err(err).Msg("failed to show response error dialog")
		}
		return fmt.Errorf("error getting linux escrow key: %s", response.Err)
	}

	// Show success dialog
	log.Debug().Msg("LUKS: Showing success dialog")
	if err := lr.infoPrompt(ctx, infoTitle, infoSuccessText); err != nil {
		log.Info().Err(err).Msg("failed to show success escrow key dialog")
	}

	return nil
}

func (lr *LuksRunner) getEscrowKey(ctx context.Context, devicePath string) ([]byte, *uint, error) {
	// AESXTSPlain64Cipher is the default cipher used by ubuntu/kubuntu/fedora
	device := luksdevice.New(luksdevice.AESXTSPlain64Cipher)

	log.Debug().Msg("LUKS: prompting for passphrase")
	// Prompt user for existing LUKS passphrase
	passphrase, err := lr.entryPrompt(ctx, entryDialogTitle, entryDialogText)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to show passphrase entry prompt")
		return nil, nil, fmt.Errorf("Failed to show passphrase entry prompt: %w", err)
	}

	if len(passphrase) == 0 {
		log.Debug().Msg("Passphrase is empty, no password supplied, dialog was canceled, or timed out")
		return nil, nil, nil
	}

	log.Debug().Msg("LUKS: showing progress dialog")
	err = lr.notifier.ShowProgress(ctx, dialog.ProgressOptions{
		Title: infoTitle,
		Text:  "Validating passphrase...",
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to show progress dialog")
	}

	// Validate the passphrase
	for {
		log.Debug().Msg("LUKS: validating passphrase")
		valid, err := lr.passphraseIsValid(ctx, device, devicePath, passphrase, userKeySlot)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed validating passphrase: %w", err)
		}

		if valid {
			log.Debug().Msg("LUKS: passphrase is valid")
			break
		}

		log.Debug().Msg("LUKS: passphrase is invalid, re-prompting")
		passphrase, err = lr.entryPrompt(ctx, entryDialogTitle, retryEntryDialogText)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to re-prompt for passphrase")
			return nil, nil, fmt.Errorf("Failed re-prompting for passphrase: %w", err)
		}

		if len(passphrase) == 0 {
			log.Debug().Msg("Passphrase is empty, no password supplied, dialog was canceled, or timed out")
			return nil, nil, nil
		}

		log.Debug().Msg("LUKS: showing progress dialog after retry")
		err = lr.notifier.ShowProgress(ctx, dialog.ProgressOptions{
			Title: infoTitle,
			Text:  "Validating passphrase...",
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to show progress dialog after retry")
		}
	}

	log.Debug().Msg("LUKS: showing key escrow in progress dialog")
	err = lr.notifier.ShowProgress(ctx, dialog.ProgressOptions{
		Title: infoTitle,
		Text:  "Key escrow in progress...",
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to show progress dialog")
	}

	log.Debug().Msg("LUKS: generating escrow passphrase")
	escrowPassphrase, err := generateRandomPassphrase()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate random passphrase")
		return nil, nil, fmt.Errorf("Failed to generate random passphrase: %w", err)
	}

	// Create a new key slot and error if all key slots are full
	// Start at slot 1 as keySlot 0 is assumed to be the location of
	// the user's passphrase
	var keySlot uint = userKeySlot + 1
	for {
		log.Debug().Msgf("LUKS: adding key to slot %d", keySlot)
		if keySlot == maxKeySlots {
			log.Debug().Msg("All LUKS key slots are full")
			return nil, nil, errors.New("all LUKS key slots are full")
		}

		log.Debug().Msg("LUKS: defining keys")
		userKey := encryption.NewKey(userKeySlot, passphrase)
		escrowKey := encryption.NewKey(int(keySlot), escrowPassphrase) // #nosec G115

		log.Debug().Msg("LUKS: adding key")
		if err := device.AddKey(ctx, devicePath, userKey, escrowKey); err != nil {
			if ErrKeySlotFull.MatchString(err.Error()) {
				log.Debug().Msg("Key slot is full, trying next slot")
				keySlot++
				continue
			}
			log.Debug().Err(err).Msg("Failed to add key")
			return nil, nil, fmt.Errorf("Failed to add key: %w", err)
		}

		log.Debug().Msg("LUKS: key added successfully")
		break
	}

	log.Debug().Msg("LUKS: validating escrow passphrase")
	valid, err := lr.passphraseIsValid(ctx, device, devicePath, escrowPassphrase, keySlot)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate escrow passphrase")
		return nil, nil, fmt.Errorf("Error while validating escrow passphrase: %w", err)
	}

	if !valid {
		log.Debug().Msg("Escrow passphrase is invalid")
		return nil, nil, errors.New("Failed to validate escrow passphrase")
	}

	return escrowPassphrase, &keySlot, nil
}

func (lr *LuksRunner) passphraseIsValid(ctx context.Context, device *luksdevice.LUKS, devicePath string, passphrase []byte, keyslot uint) (bool, error) {
	if len(passphrase) == 0 {
		log.Debug().Msg("Passphrase is empty, no password supplied")
		return false, nil
	}

	log.Debug().Msg("LUKS: Running CheckKey")
	valid, err := device.CheckKey(ctx, devicePath, encryption.NewKey(int(keyslot), passphrase)) // #nosec G115
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate passphrase")
		return false, fmt.Errorf("Error validating passphrase: %w", err)
	}

	return valid, nil
}

// generateRandomPassphrase generates a random passphrase with 32 characters
// in the format XXXX-XXXX-XXXX-XXXX where X is a random character from the
// set [0-9A-Za-z].
func generateRandomPassphrase() ([]byte, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const length = 35 // 32 characters + 3 dashes
	passphrase := make([]byte, length)

	for i := 0; i < length; i++ {
		// Insert dashes at positions 8, 17, and 26
		if i == 8 || i == 17 || i == 26 {
			passphrase[i] = '-'
			continue
		}

		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return nil, err
		}
		passphrase[i] = chars[num.Int64()]
	}

	return passphrase, nil
}

func (lr *LuksRunner) entryPrompt(ctx context.Context, title, text string) ([]byte, error) {
	passphrase, err := lr.notifier.ShowEntry(ctx, dialog.EntryOptions{
		Title:    title,
		Text:     text,
		HideText: true,
		TimeOut:  1 * time.Minute,
	})
	if err != nil {
		switch {
		case errors.Is(err, dialog.ErrCanceled):
			log.Debug().Msg("end user canceled key escrow dialog")
			return nil, nil
		case errors.Is(err, dialog.ErrTimeout):
			log.Debug().Msg("key escrow dialog timed out")
			err := lr.infoPrompt(ctx, infoTitle, timeoutMessage)
			if err != nil {
				log.Info().Err(err).Msg("failed to show timeout dialog")
			}
			return nil, nil
		case errors.Is(err, dialog.ErrUnknown):
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
		TimeOut: 1 * time.Minute,
	})
	if err != nil {
		switch {
		case errors.Is(err, dialog.ErrTimeout):
			log.Debug().Msg("successPrompt timed out")
			return nil
		default:
			return err
		}
	}

	return nil
}

type LuksDump struct {
	Keyslots map[string]Keyslot `json:"keyslots"`
}

type Keyslot struct {
	KDF KDF `json:"kdf"`
}

type KDF struct {
	Salt string `json:"salt"`
}

func getSaltforKeySlot(ctx context.Context, devicePath string, keySlot uint) (string, error) {
	cmd := exec.CommandContext(ctx, "cryptsetup", "luksDump", "--dump-json-metadata", devicePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to run cryptsetup luksDump: %w", err)
	}

	var dump LuksDump
	if err := json.Unmarshal(output, &dump); err != nil {
		return "", fmt.Errorf("Failed to unmarshal luksDump output: %w", err)
	}

	slot, ok := dump.Keyslots[fmt.Sprintf("%d", keySlot)]
	if !ok {
		return "", errors.New("key slot not found")
	}

	return slot.KDF.Salt, nil
}

func removeKeySlot(ctx context.Context, devicePath string, keySlot uint) error {
	cmd := exec.CommandContext(ctx, "cryptsetup", "luksKillSlot", devicePath, fmt.Sprintf("%d", keySlot)) // #nosec G204
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to run cryptsetup luksKillSlot: %w", err)
	}

	return nil
}
