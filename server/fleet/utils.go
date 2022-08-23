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
			"https://fleetdm.com/docs/using-fleet/faq#how-do-i-downgrade-from-fleet-premium-to-fleet-free and "+
			"contact us for help.",
	)
	// We need to disable color and print a new line to make it look somewhat neat, otherwise colors continue to the
	// next line
	warningColor.DisableColor()
	warningColor.Fprintln(w)
}

// NoopUnmarshaler is a type that implements the `json.Unmarshaler` interface but does
// nothing.
//
// This is useful when you have an embeded field that implements
// `json.Unmarshaler` but want to prevent the method from being promoted to the
// top-level struct.
type NoopUnmarshaler struct{}

// UnmarshalJSON implments the `json.Unmarshaler` interface.
func (*NoopUnmarshaler) UnmarshalJSON([]byte) error { return nil }
