//go:build linux

package luks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/martinjungblut/go-cryptsetup"
	"github.com/rs/zerolog/log"
)

type ErrCode int

const (
	ErrKeySlotFull       ErrCode = -22
	ErrBadPassphrase     ErrCode = -1
	entryDialogTitle             = "Enter disk encryption passphrase"
	entryDialogText              = "Passphrase:"
	retryEntryDialogText         = "Passphrase incorrect. Please try again."
	infoFailedTitle              = "Encryption key escrow"
	infoFailedText               = "Failed to escrow key. Please try again later."
	infoSuccessTitle             = "Encryption key escrow"
	infoSuccessText              = "Key escrowed successfully."
	maxKeySlots                  = 8
	randPassphraseLength         = 32
)

type KeyEscrower interface {
	EscrowKey() error
}

type LuksRunner struct {
	escrower KeyEscrower
	notifier dialog.Dialog
}

func (lr *LuksRunner) Run(oc *fleet.OrbitConfig) error {
	if !oc.Notifications.EscrowLinuxKey {
		log.Debug().Msg("EscrowLinuxKey is false, skipping")
		return nil
	}

	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		return fmt.Errorf("devicepath err: %w", err)
	}

	device, err := cryptsetup.Init(devicePath)
	if err != nil {
		return fmt.Errorf("cryptsetup init err: %w", err)
	}
	defer device.Free()

	luks2 := cryptsetup.LUKS2{}
	if err := device.Load(luks2); err != nil {
		return fmt.Errorf("cryptsetup load err: %w", err)
	}

	var passphrase string

	// Prompt user for existing LUKS passphrase
	passphrase, err = lr.entryPrompt(context.Background(), EntryDialogTitle, EntryDialogText)
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

		if err := device.KeyslotAddByPassphrase(keySlot, string(passphrase), escrowPassphrase); err != nil {
			code := err.(*cryptsetup.Error).Code()
			if code == int(ErrBadPassphrase) {
				passphrase, err = lr.entryPrompt(context.Background(), EntryDialogTitle, RetryEntryDialogText)
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
	if err := lr.escrower.EscrowKey(); err != nil {
		// Not supported in go-cryptsetup
		// err := device.KeyslotRemove(keySlot)
		// if err != nil {
		// 	log.Debug().Err(err).Msg("failed to remove key slot")
		// }

		if err := lr.infoPrompt(context.Background(), InfoFailedTitle, InfoFailedText); err != nil {
			log.Debug().Err(err).Msg("failed to show failed escrow key dialog")
		}

		return fmt.Errorf("escrower escrowKey err: %w", err)
	}

	// Show success dialog
	if err := lr.infoPrompt(context.Background(), InfoSuccessTitle, InfoSuccessText); err != nil {
		log.Debug().Err(err).Msg("failed to show success escrow key dialog")
	}

	return nil
}

func generateRandomPassphrase(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	passphrase := base64.URLEncoding.EncodeToString(bytes)

	// Truncate to desired length if necessary
	return passphrase[:length], nil
}

func (lr *LuksRunner) entryPrompt(ctx context.Context, title, text string) (string, error) {
	passphrase, err := lr.notifier.ShowEntry(context.Background(), dialog.EntryOptions{
		Title:    title,
		Text:     text,
		HideText: true,
		TimeOut:  1 * time.Minute,
	})
	if err != nil {
		switch err {
		case dialog.ErrCanceled:
			log.Info().Msg("end user canceled key escrow dialog")
			return "", nil
		case dialog.ErrTimeout:
			log.Info().Msg("key escrow dialog timed out")
			return "", nil
		case dialog.ErrUnknown:
			return "", err
		default:
			return "", err
		}
	}

	return string(passphrase), nil
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
