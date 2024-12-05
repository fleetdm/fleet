//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/zenity"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	"github.com/siderolabs/go-blockdevice/v2/encryption/luks"
)

func main() {
	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		fmt.Println("devicepath err:", err)
		panic(err)
	}

	prompt := zenity.New()

	// Prompt existing passphrase from the user.
	currentPassphrase, err := prompt.ShowEntry(dialog.EntryOptions{
		Title:    "Enter Existing LUKS Passphrase",
		Text:     "Enter your existing LUKS passphrase:",
		HideText: true,
	})
	if err != nil {
		fmt.Println("Err ShowEntry")
		panic(err)
	}

	const escrowPassPhrase = "fleet123"

	device := luks.New(luks.AESXTSPlain64Cipher)

	keySlot := 1
	for {
		if keySlot == 8 {
			panic(errors.New("all LUKS key slots are full"))
		}

		userKey := encryption.NewKey(0, currentPassphrase)
		escrowKey := encryption.NewKey(keySlot, []byte(escrowPassPhrase))

		if err := device.AddKey(context.Background(), devicePath, userKey, escrowKey); err != nil {
			if errors.Is(err, encryption.ErrEncryptionKeyRejected) {
				currentPassphrase, err = prompt.ShowEntry(dialog.EntryOptions{
					Title:    "Enter Existing LUKS Passphrase",
					Text:     "Bad password. Enter your existing LUKS passphrase:",
					HideText: true,
				})
				if err != nil {
					fmt.Println("Err Retry ShowEntry")
					panic(err)
				}
				continue
			}

			keySlot++
			continue
		}

		break
	}

	fmt.Println("Key escrowed successfully.")
}
