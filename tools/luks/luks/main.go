//go:build linux

package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/siderolabs/go-blockdevice/v2/encryption"
	"github.com/siderolabs/go-blockdevice/v2/encryption/luks"
	"golang.org/x/term"
)

func main() {
	devicepath, err := lvm.FindRootDisk()
	if err != nil {
		fmt.Println("devicepath err:", err)
		panic(err)
	}

	// Prompt existing passphrase from the user.
	fmt.Printf("Enter passphrase: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	fmt.Println()
	currentPassphrase := string(password)

	const escrowPassPhrase = "fleet123"

	userKey := encryption.NewKey(0, []byte(currentPassphrase))
	escrowKey := encryption.NewKey(1, []byte(escrowPassPhrase))

	device := luks.New(luks.AESXTSPlain64Cipher)
	err = device.AddKey(context.Background(), devicepath, userKey, escrowKey)
	if err != nil {
		fmt.Println("AddKey err:", err)
		panic(err)
	}

	fmt.Println("Key escrowed successfully.")
}
