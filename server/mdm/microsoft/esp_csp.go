package microsoft_mdm

import (
	"fmt"
	"slices"
	"strings"
)

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPSoftwareFailureErrorText is the message shown on the Windows ESP failure screen when a critical software install fails.
const ESPSoftwareFailureErrorText = "Critical software failed to install. Reset your device to try again. If failures keep happening, please contact your IT admin."

// ESPTimeoutErrorText is the message shown on the Windows ESP failure screen when the 3-hour ESP timeout expires before setup completes.
const ESPTimeoutErrorText = "Setup is taking longer than expected. Please try again. If this keeps happening, please contact your IT admin."

// espContinuableErrorSuffix follows the failed-software list when the end user is allowed to continue past the failure.
const espContinuableErrorSuffix = "Reset your device to try again, or proceed and install missing software via self-service. " +
	"If unavailable, contact your IT admin."

// espMaxFailedNamesShown caps how many software names the ESP failure message lists before summarizing the rest as
// "N more". CustomErrorText renders in a small ESP area and the DMClient CSP documents no maximum length.
const espMaxFailedNamesShown = 3

// ESPSoftwareFailureContinuableErrorText returns the message shown on the Windows ESP failure screen when one or more
// software installs failed but "Cancel setup if software fails" (require_all_software_windows) is off, so the end user
// is allowed to continue to the desktop.
func ESPSoftwareFailureContinuableErrorText(failedNames []string) string {
	list := joinFailedNames(failedNames)
	if list == "" {
		// Defensive: a failure was detected but no names were collected (e.g. rows with empty names).
		return "Some software failed to install. " + espContinuableErrorSuffix
	}
	return fmt.Sprintf("%s failed to install. %s", list, espContinuableErrorSuffix)
}

// joinFailedNames renders names as a human-readable list: "A", "A and B", "A, B, and C". Empty names are dropped. When
// more than espMaxFailedNamesShown names remain, only the first that many are listed and the rest is summarized, e.g.
// "A, B, C, and 2 more". Returns "" when there are no non-empty names.
func joinFailedNames(names []string) string {
	items := slices.DeleteFunc(slices.Clone(names), func(s string) bool { return s == "" })
	if len(items) == 0 {
		return ""
	}
	if extra := len(items) - espMaxFailedNamesShown; extra > 0 {
		items = append(items[:espMaxFailedNamesShown:espMaxFailedNamesShown], fmt.Sprintf("%d more", extra))
	}

	switch len(items) {
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}
