package microsoft_mdm

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 3 * 60 * 60

// ESPSoftwareFailureErrorText is the message shown on the Windows ESP failure
// screen when a critical software install fails or the ESP times out. It is
// sent to the device via the DMClient CSP CustomErrorText node.
const ESPSoftwareFailureErrorText = "Critical software failed to install. Please try again. If this keeps happening, please contact your IT admin."
