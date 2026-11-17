package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/entratoken"
	"github.com/urfave/cli/v2"
)

// entraDeviceTokenCommand is run by orbit as the logged-in user (see entratoken.Runner),
// because the SSO extension only serves that user's session.
var entraDeviceTokenCommand = &cli.Command{
	Name:   "entra-device-token",
	Usage:  "Print an Entra ID token with the device ID claim (macOS, must run as the user logged in to Entra)",
	Hidden: true,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "tenant-id", Required: true},
		&cli.StringFlag{Name: "client-id", Required: true},
		&cli.StringFlag{Name: "nonce", Required: true},
		&cli.StringFlag{Name: "redirect-uri", Value: entratoken.DefaultRedirectURI},
	},
	Action: func(c *cli.Context) error {
		ctx, cancel := context.WithTimeout(c.Context, 30*time.Second)
		defer cancel()
		token, err := entratoken.GetIDToken(ctx, entratoken.Request{
			TenantID:    c.String("tenant-id"),
			ClientID:    c.String("client-id"),
			RedirectURI: c.String("redirect-uri"),
			Nonce:       c.String("nonce"),
		})
		if err != nil {
			// main exits 0 on errors; the caller needs a non-zero exit to report stderr.
			return cli.Exit(err, 1)
		}
		fmt.Println(token)
		return nil
	},
}
