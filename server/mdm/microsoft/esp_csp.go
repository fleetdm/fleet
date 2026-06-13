package microsoft_mdm

import (
	"fmt"
	"strings"
)

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPSoftwareFailureErrorText is the message shown on the Windows ESP failure screen when a critical software install fails.
const ESPSoftwareFailureErrorText = "Critical software failed to install. Please try again. If this keeps happening, please contact your IT admin."

// ESPTimeoutErrorText is the message shown on the Windows ESP failure screen when the 3-hour ESP timeout expires before setup completes.
const ESPTimeoutErrorText = "Setup is taking longer than expected. Please try again. If this keeps happening, please contact your IT admin."

// espContinuableErrorSuffix follows the failed-software list when the end user is allowed to continue past the failure.
const espContinuableErrorSuffix = "You can reset your device to start over or proceed and install missing software via self-service."

// espMaxFailedNamesLen caps the rendered failed-software name list. The DMClient CSP does not document a maximum
// length for CustomErrorText and the ESP renders it in a small area, so overly long lists are truncated to
// "..., and N more".
const espMaxFailedNamesLen = 400

// ESPSoftwareFailureContinuableErrorText returns the message shown on the Windows ESP failure screen when one or more
// software installs failed but "Cancel setup if software fails" (require_all_software_windows) is off, so the end user
// is allowed to continue to the desktop. The failed software is listed by name, e.g. "Slack, Zoom, and Docker failed
// to install. You can reset your device to start over or proceed and install missing software via self-service."
func ESPSoftwareFailureContinuableErrorText(failedNames []string) string {
	list := joinFailedNames(failedNames, espMaxFailedNamesLen)
	if list == "" {
		// Defensive: a failure was detected but no names were collected (e.g. rows with empty names).
		return "Some software failed to install. " + espContinuableErrorSuffix
	}
	return fmt.Sprintf("%s failed to install. %s", list, espContinuableErrorSuffix)
}

// joinFailedNames joins names into a human-readable list: "A", "A and B", "A, B, and C". If the joined list would
// exceed maxLen, only the names that fit are listed and the remainder is summarized as "N more"; at least one name is
// always included.
func joinFailedNames(names []string, maxLen int) string {
	nonEmpty := make([]string, 0, len(names))
	for _, n := range names {
		if n != "" {
			nonEmpty = append(nonEmpty, n)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}

	// included counts how many names fit in maxLen. The running length approximates the final rendering (", " between
	// items); exact "and"/Oxford-comma accounting isn't needed for a display cap.
	included := 1
	lenSoFar := len(nonEmpty[0])
	for ; included < len(nonEmpty); included++ {
		next := lenSoFar + 2 + len(nonEmpty[included])
		if next > maxLen {
			break
		}
		lenSoFar = next
	}

	items := nonEmpty[:included:included]
	if rest := len(nonEmpty) - included; rest > 0 {
		items = append(items, fmt.Sprintf("%d more", rest))
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
