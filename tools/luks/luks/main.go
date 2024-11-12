//go:build linux

package main

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dialog"
	"github.com/fleetdm/fleet/v4/orbit/pkg/lvm"
	"github.com/fleetdm/fleet/v4/orbit/pkg/zenity"
	"github.com/martinjungblut/go-cryptsetup"
)

type ErrCode int

var (
	ErrKeySlotFull   ErrCode = -22
	ErrBadPassphrase ErrCode = -1
)

func main() {
	// /dev/sda3 is the block device for my root partition "/".
	devicePath, err := lvm.FindRootDisk()
	if err != nil {
		fmt.Println("devicepath err")
		panic(err)
	}
	fmt.Println(devicePath)
	device, err := cryptsetup.Init(devicePath)
	if err != nil {
		fmt.Println("Init err")
		panic(err)
	}
	defer device.Free()

	// Load block device knowing it's encrypted with LUKS2.
	luks2 := cryptsetup.LUKS2{}
	if err := device.Load(luks2); err != nil {
		fmt.Println("load err")
		panic(err)
	}

	// Prompt existing passphrase from the user.
	prompt := zenity.New()
	passphrase, err := prompt.ShowEntry(context.Background(), dialog.EntryOptions{
		Title:    "Enter Passphrase",
		Text:     "Enter the passphrase for the encrypted device",
		HideText: true,
		TimeOut:  10,
	})
	if err != nil {
		switch err {
		case dialog.ErrCanceled:
			fmt.Println("canceled")
		case dialog.ErrTimeout:
			fmt.Println("timeout")
		case dialog.ErrUnknown:
			fmt.Println("unknown error", err)
		}
		panic(err)
	}

	// Create a new slot with an "escrow key".
	keySlot := 1
	for {
		const escrowPassphrase = "fleet123"
		if err := device.KeyslotAddByPassphrase(keySlot, string(passphrase), escrowPassphrase); err != nil {
			code := err.(*cryptsetup.Error).Code()
			fmt.Println("KeyslotAddByPassphrase err", "code", code)
			if code == int(ErrBadPassphrase) {
				fmt.Println("bad passphrase")
				panic(err)
			}
			if keySlot == 8 {
				panic("all key slots full")
			}
			keySlot++
			continue
		}
		fmt.Println("Key slot added in slot", keySlot)
		break
	}
}
