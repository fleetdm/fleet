package fleet

import (
	"io"

	"github.com/fatih/color"
)

func WriteExpiredLicenseBanner(w io.Writer) {
	warningColor := color.New(color.FgWhite, color.Bold, color.BgRed)
	warningColor.Fprintf(
		w,
		"Your license for Fleet Premium is about to expire. If you’d like to renew or have questions about "+
			"downgrading, please navigate to "+
			"https://fleetdm.com/docs/using-fleet/teams#expired_license and "+
			"contact us for help.",
	)
	// We need to disable color and print a new line to make it look somewhat neat, otherwise colors continue to the
	// next line
	warningColor.DisableColor()
	warningColor.Fprintln(w)
}
