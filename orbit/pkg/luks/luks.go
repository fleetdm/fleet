package luks_runner

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
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
	SendLinuxKeyEscrowResponse(LuksResponse) error
}

type LuksRunner struct {
	escrower KeyEscrower
	notifier dialog.Dialog
}

type LuksResponse struct {
	Key []byte `json:"key"`
	Err string `json:"err"`
}

func New(escrower KeyEscrower, notifier dialog.Dialog) *LuksRunner {
	return &LuksRunner{
		escrower: escrower,
		notifier: notifier,
	}
}
