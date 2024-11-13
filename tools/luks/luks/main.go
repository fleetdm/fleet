//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	"github.com/siderolabs/go-blockdevice/v2/encryption/luks"
	"golang.org/x/term"
)

func main() {
	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		fmt.Println("devicepath err:", err)
		panic(err)
	}

	// Prompt existing passphrase from the user.
	fmt.Printf("Enter passphrase: ")
	currentPassphrase, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	fmt.Println()

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
				currentPassphrase, err = term.ReadPassword(int(syscall.Stdin))
				if err != nil {
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
