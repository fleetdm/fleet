//go:build windows
// +build windows

package eefleetctl

import (
	"errors"

	"github.com/urfave/cli/v2"
)

func UpdatesCommand() *cli.Command {
	// Update management is disabled on Windows because file permissions need to be very particular
	// and the Windows permission model is vastly different from the Unix model. Instead, recommend
	// the user manages updates from Linux.
	return &cli.Command{
		Name:        "updates",
		Usage:       "Manage client updates",
		Description: "Update management is not supported on Windows. Please use a Linux environment to continue.",
		Before: func(*cli.Context) error {
			return errors.New("Update management is not supported on Windows. Please use a Linux environment to continue.")
		},
	}
}
