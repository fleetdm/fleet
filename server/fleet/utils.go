package fleet

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/fatih/color"
	"golang.org/x/text/unicode/norm"
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

func WriteAppleBMTermsExpiredBanner(w io.Writer) {
	warningColor := color.New(color.FgWhite, color.Bold, color.BgRed)
	warningColor.Fprintf(
		w,
		`Your organization can’t automatically enroll macOS hosts until you accept the new terms `+
			`and conditions for Apple Business Manager (ABM). An ABM administrator can accept these terms. `+
			`Go to ABM: https://business.apple.com/`,
	)
	// We need to disable color and print a new line to make it look somewhat neat, otherwise colors continue to the
	// next line
	warningColor.DisableColor()
	warningColor.Fprintln(w)
}

// JSONStrictDecode unmarshals the JSON value from the provided reader r into
// the destination value v. It returns an error if the unmarshaling fails.
// Compared to standard json.Unmarshal, this function will return an error if
// any unknown key is specified in the JSON value, and if there is any trailing
// byte after the JSON value.
func JSONStrictDecode(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}

	var extra json.RawMessage
	if dec.Decode(&extra) != io.EOF {
		return errors.New("json: extra bytes after end of object")
	}

	return nil
}

func Preprocess(input string) string {
	// Remove leading/trailing whitespace.
	input = strings.TrimSpace(input)
	// Normalize Unicode characters.
	return norm.NFC.String(input)
}

// CompareVersions returns an integer comparing two versions according to semantic version
// precedence. The result will be 0 if a == b, -1 if a < b, or +1 if a > b.
// An invalid semantic version string is considered less than a valid one. All invalid semantic
// version strings compare equal to each other.
func CompareVersions(a string, b string) int {
	verA, errA := semver.NewVersion(a)
	verB, errB := semver.NewVersion(b)
	switch {
	case errA != nil && errB != nil:
		return 0
	case errA != nil:
		return -1
	case errB != nil:
		return 1
	default:
		return verA.Compare(verB)
	}
}

// IsAtLeastVersion returns whether currentVersion is at least minimumVersion, using semantics
// of CompareVersions for version validity
func IsAtLeastVersion(currentVersion string, minimumVersion string) bool {
	return CompareVersions(currentVersion, minimumVersion) >= 0
}
