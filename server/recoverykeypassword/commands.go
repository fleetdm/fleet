package recoverykeypassword

import "fmt"

// setRecoveryLockCommand returns the raw plist for the SetRecoveryLock MDM command.
// See https://developer.apple.com/documentation/devicemanagement/set_recovery_lock
func setRecoveryLockCommand(cmdUUID, password string) []byte {
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>SetRecoveryLock</string>
        <key>NewPassword</key>
        <string>%s</string>
    </dict>
</dict>
</plist>`, cmdUUID, password)
}

// verifyRecoveryLockCommand returns the raw plist for the VerifyRecoveryLock MDM command.
// See https://developer.apple.com/documentation/devicemanagement/verifyrecoverylockcommand
func verifyRecoveryLockCommand(cmdUUID, password string) []byte {
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>VerifyRecoveryLock</string>
        <key>Password</key>
        <string>%s</string>
    </dict>
</dict>
</plist>`, cmdUUID, password)
}
