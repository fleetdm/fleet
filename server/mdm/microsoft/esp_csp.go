package microsoft_mdm

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPSoftwareFailureErrorText is the message shown on the Windows ESP failure screen when a critical software install fails.
const ESPSoftwareFailureErrorText = "Critical software failed to install. Please try again. If this keeps happening, please contact your IT admin."

// ESPTimeoutErrorText is the message shown on the Windows ESP failure screen when the 3-hour ESP timeout expires before setup completes.
const ESPTimeoutErrorText = "Setup is taking longer than expected. Please try again. If this keeps happening, please contact your IT admin."
