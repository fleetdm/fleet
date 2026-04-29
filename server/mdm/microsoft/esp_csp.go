package microsoft_mdm

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPSoftwareFailureErrorText is the message shown on the Windows ESP failure
// screen when a critical software install fails (with
// require_all_software_windows = true). Sent via the DMClient CSP
// CustomErrorText node.
const ESPSoftwareFailureErrorText = "Critical software failed to install. Please try again. If this keeps happening, please contact your IT admin."

// ESPTimeoutErrorText is the message shown on the Windows ESP failure screen
// when the 3-hour ESP timeout expires before setup completes. Distinct from
// the software-failure text because timeout is not necessarily caused by a
// failed install -- the device may have been offline, slow, or the team may
// have no software configured.
const ESPTimeoutErrorText = "Setup is taking longer than expected. Please try again. If this keeps happening, please contact your IT admin."
