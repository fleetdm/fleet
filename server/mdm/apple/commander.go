package apple_mdm

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	mdmcrypto "github.com/fleetdm/fleet/v4/server/mdm/crypto"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/micromdm/plist"
)

// commandPayload is the common structure all MDM commands use
type commandPayload struct {
	CommandUUID string
	Command     any
}

// MDMAppleCommander contains methods to enqueue commands managed by Fleet and
// send push notifications to hosts.
//
// It's intentionally decoupled from fleet.Service so it can be used internally
// in crons and other services, leaving authentication/permission handling to
// the caller.
type MDMAppleCommander struct {
	storage fleet.MDMAppleStore
	pusher  nanomdm_push.Pusher
}

// NewMDMAppleCommander creates a new commander instance.
func NewMDMAppleCommander(mdmStorage fleet.MDMAppleStore, mdmPushService nanomdm_push.Pusher) *MDMAppleCommander {
	return &MDMAppleCommander{
		storage: mdmStorage,
		pusher:  mdmPushService,
	}
}

// InstallProfile sends the homonymous MDM command to the given hosts, it also
// takes care of the base64 encoding of the provided profile bytes.
func (svc *MDMAppleCommander) InstallProfile(ctx context.Context, hostUUIDs []string, profile mobileconfig.Mobileconfig, uuid string, name string) error {
	raw, err := svc.SignAndEncodeInstallProfile(ctx, profile, uuid)
	if err != nil {
		return err
	}
	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding InstallProfile command")
	}
	err = svc.enqueueAndNotify(ctx, hostUUIDs, cmd, mdm.CommandSubtypeNone, name)
	return ctxerr.Wrap(ctx, err, "commander install profile")
}

func (svc *MDMAppleCommander) SignAndEncodeInstallProfile(ctx context.Context, profile []byte, commandUUID string) (string, error) {
	signedProfile, err := mdmcrypto.Sign(ctx, profile, svc.storage)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "signing profile")
	}

	base64Profile := base64.StdEncoding.EncodeToString(signedProfile)
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>%s</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>InstallProfile</string>
		<key>Payload</key>
		<data>%s</data>
	</dict>
</dict>
</plist>`, commandUUID, base64Profile)
	return raw, nil
}

// RemoveProfile sends the homonymous MDM command to the given hosts.
func (svc *MDMAppleCommander) RemoveProfile(ctx context.Context, hostUUIDs []string, profileIdentifier string, uuid string, name string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>%s</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>RemoveProfile</string>
		<key>Identifier</key>
		<string>%s</string>
	</dict>
</dict>
</plist>`, uuid, profileIdentifier)
	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding RemoveProfile command")
	}
	err = svc.enqueueAndNotify(ctx, hostUUIDs, cmd, mdm.CommandSubtypeNone, name)
	return ctxerr.Wrap(ctx, err, "commander remove profile")
}

func (svc *MDMAppleCommander) DeviceLock(ctx context.Context, host *fleet.Host, uuid string) (unlockPIN string, err error) {
	// Check for existing pending lock command first
	existingCmd, existingPIN, err := svc.storage.GetPendingLockCommand(ctx, host.UUID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "checking for pending lock command")
	}

	// If a pending lock command exists, just send a push notification and return the existing PIN
	if existingCmd != nil {
		if err := svc.SendNotifications(ctx, []string{host.UUID}); err != nil {
			return "", ctxerr.Wrap(ctx, err, "sending notifications for existing DeviceLock")
		}
		return existingPIN, nil
	}

	// No pending lock, create a new one
	unlockPIN, err = GenerateRandomPin(6)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "generating random PIN for DeviceLock")
	}
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>DeviceLock</string>
      <key>PIN</key>
      <string>%s</string>
    </dict>
  </dict>
</plist>`, uuid, unlockPIN,
	)

	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decoding command")
	}

	if err := svc.storage.EnqueueDeviceLockCommand(ctx, host, cmd, unlockPIN); err != nil {
		// Check if another request just created a lock
		type conflictInterface interface {
			IsConflict() bool
		}
		if c, ok := err.(conflictInterface); ok && c.IsConflict() {
			// Another goroutine won the race, fetch the command that was created
			existingCmd, existingPIN, err := svc.storage.GetPendingLockCommand(ctx, host.UUID)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "getting existing lock after race condition")
			}
			if existingCmd != nil {
				// Send push notification for the existing command and return its PIN
				if pushErr := svc.SendNotifications(ctx, []string{host.UUID}); pushErr != nil {
					// Log the push error but still return the PIN since the command exists
					// The push can be retried on subsequent requests
					ctxerr.Handle(ctx, ctxerr.Wrap(ctx, pushErr, "failed to send push notification after lock race"))
					return existingPIN, nil
				}
				return existingPIN, nil
			}
			// This shouldn't happen, but if we can't find the command, return the original error
			return "", ctxerr.Wrap(ctx, err, "lock command conflict but no existing command found")
		}
		return "", ctxerr.Wrap(ctx, err, "enqueuing for DeviceLock")
	}

	if err := svc.SendNotifications(ctx, []string{host.UUID}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "sending notifications for DeviceLock")
	}

	return unlockPIN, nil
}

func (svc *MDMAppleCommander) EnableLostMode(ctx context.Context, host *fleet.Host, commandUUID string, orgName string) error {
	msg := fmt.Sprintf("This device is locked. It belongs to %s.", orgName)
	cmdPayload := commandPayload{
		CommandUUID: commandUUID,
		Command: map[string]any{
			"RequestType": "EnableLostMode",
			"Message":     msg,
		},
	}
	rawBytes, err := plist.MarshalIndent(cmdPayload, "    ")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshalling EnableLostMode payload")
	}
	raw := string(rawBytes)

	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding EnableLostMode command")
	}

	if err := svc.storage.EnqueueDeviceLockCommand(ctx, host, cmd, ""); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing for EnableLostMode")
	}

	if err := svc.SendNotifications(ctx, []string{host.UUID}); err != nil {
		return ctxerr.Wrap(ctx, err, "sending notifications for EnableLostMode")
	}

	return nil
}

func (svc *MDMAppleCommander) DisableLostMode(ctx context.Context, host *fleet.Host, commandUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
	<key>CommandUUID</key>
	<string>%s</string>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>DisableLostMode</string>
	</dict>
</dict>
</plist>`, commandUUID)

	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding command for DisableLostMode")
	}

	if err := svc.storage.EnqueueDeviceUnlockCommand(ctx, host, cmd); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing device unlock command for DisableLostMode")
	}

	if err := svc.SendNotifications(ctx, []string{host.UUID}); err != nil {
		return ctxerr.Wrap(ctx, err, "sending notifications for DisableLostMode")
	}

	return nil
}

func (svc *MDMAppleCommander) EraseDevice(ctx context.Context, host *fleet.Host, uuid string) error {
	pin, err := GenerateRandomPin(6)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating random PIN for EraseDevice")
	}
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>CommandUUID</key>
    <string>%s</string>
    <key>Command</key>
    <dict>
      <key>RequestType</key>
      <string>EraseDevice</string>
      <key>PIN</key>
      <string>%s</string>
      <key>ObliterationBehavior</key>
      <string>Default</string>
    </dict>
  </dict>
</plist>`, uuid, pin)

	cmd, err := mdm.DecodeCommand([]byte(raw))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding DeviceWipe command")
	}

	if err := svc.storage.EnqueueDeviceWipeCommand(ctx, host, cmd); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing for DeviceWipe")
	}

	if err := svc.SendNotifications(ctx, []string{host.UUID}); err != nil {
		return ctxerr.Wrap(ctx, err, "sending notifications for DeviceWipe")
	}

	return nil
}

func (svc *MDMAppleCommander) InstallEnterpriseApplication(ctx context.Context, hostUUIDs []string, uuid string, manifestURL string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Command</key>
    <dict>
      <key>ManifestURL</key>
      <string>%s</string>
      <key>RequestType</key>
      <string>InstallEnterpriseApplication</string>
    </dict>

    <key>CommandUUID</key>
    <string>%s</string>
  </dict>
</plist>`, manifestURL, uuid)
	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

type installEnterpriseApplicationPayload struct {
	Manifest    *appmanifest.Manifest
	RequestType string
}

func (svc *MDMAppleCommander) InstallEnterpriseApplicationWithEmbeddedManifest(
	ctx context.Context,
	hostUUIDs []string,
	uuid string,
	manifest *appmanifest.Manifest,
) error {
	cmd := commandPayload{
		CommandUUID: uuid,
		Command: installEnterpriseApplicationPayload{
			RequestType: "InstallEnterpriseApplication",
			Manifest:    manifest,
		},
	}

	raw, err := plist.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command payload plist: %w", err)
	}

	return svc.EnqueueCommand(ctx, hostUUIDs, string(raw))
}

// SSOAccountConfig holds the SSO (end-user authentication) parameters for an
// AccountConfiguration MDM command.
type SSOAccountConfig struct {
	FullName               string
	UserName               string
	LockPrimaryAccountInfo bool
}

// AdminAccountConfig holds the parameters for an AutoSetupAdminAccounts entry
// in an AccountConfiguration MDM command.
type AdminAccountConfig struct {
	ShortName    string // e.g. "_fleetadmin"
	FullName     string // e.g. "Fleet Admin"
	PasswordHash []byte // SALTED-SHA512-PBKDF2 plist from GenerateSaltedSHA512PBKDF2Hash
	Hidden       bool   // true → hidden from login window
}

func (svc *MDMAppleCommander) AccountConfiguration(ctx context.Context, hostUUIDs []string,
	cmdUUID string,
	ssoAccount *SSOAccountConfig,
	adminAccount *AdminAccountConfig,
) error {
	var payload string

	if ssoAccount != nil {
		payload += fmt.Sprintf(`
      <key>PrimaryAccountFullName</key>
      <string>%s</string>
      <key>PrimaryAccountUserName</key>
      <string>%s</string>
      <key>LockPrimaryAccountInfo</key>
      <%t />
`, ssoAccount.FullName, ssoAccount.UserName, ssoAccount.LockPrimaryAccountInfo)
	}

	if adminAccount != nil {
		passwordHashEncoded := base64.StdEncoding.EncodeToString(adminAccount.PasswordHash)
		payload += fmt.Sprintf(`
      <key>AutoSetupAdminAccounts</key>
      <array>
        <dict>
          <key>hidden</key>
          <%t />
          <key>passwordHash</key>
          <data>%s</data>
          <key>shortName</key>
          <string>%s</string>
          <key>fullName</key>
          <string>%s</string>
        </dict>
      </array>
`, adminAccount.Hidden, passwordHashEncoded, adminAccount.ShortName, adminAccount.FullName)
	}

	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Command</key>
    <dict>%s
      <key>RequestType</key>
      <string>AccountConfiguration</string>
    </dict>

    <key>CommandUUID</key>
    <string>%s</string>
  </dict>
</plist>`, payload, cmdUUID)
	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

// DeclarativeManagement sends the homonym [command][1] to the device to enable DDM or start a new DDM session.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/declarativemanagementcommand
func (svc *MDMAppleCommander) DeclarativeManagement(ctx context.Context, hostUUIDs []string, uuid string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
 <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
 <plist version="1.0">
   <dict>
     <key>Command</key>
     <dict>
       <key>RequestType</key>
       <string>DeclarativeManagement</string>
     </dict>

     <key>CommandUUID</key>
     <string>%s</string>
   </dict>
 </plist>`, uuid)

	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

func (svc *MDMAppleCommander) DeviceConfigured(ctx context.Context, hostUUID, cmdUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>DeviceConfigured</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)

	return svc.EnqueueCommand(ctx, []string{hostUUID}, raw)
}

func (svc *MDMAppleCommander) DeviceInformation(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>Queries</key>
        <array>
            <string>DeviceName</string>
            <string>DeviceCapacity</string>
            <string>AvailableDeviceCapacity</string>
            <string>OSVersion</string>
            <string>SupplementalOSVersionExtra</string>
            <string>WiFiMAC</string>
            <string>ProductName</string>
			<string>IsMDMLostModeEnabled</string>
			<string>TimeZone</string>
        </array>
        <key>RequestType</key>
        <string>DeviceInformation</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)

	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

func (svc *MDMAppleCommander) InstalledApplicationList(ctx context.Context, hostUUIDs []string, cmdUUID string, managedOnly bool) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Command</key>
        <dict>
            <key>ManagedAppsOnly</key>
            <%t/>
            <key>RequestType</key>
            <string>InstalledApplicationList</string>
            <key>Items</key>
            <array>
                <string>Name</string>
                <string>ShortVersion</string>
                <string>Identifier</string>
                <string>Installing</string>
            </array>
        </dict>
        <key>CommandUUID</key>
        <string>%s</string>
    </dict>
</plist>`, managedOnly, cmdUUID)

	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

// CertificateList sends the homonym [command][1] to the device to get a list of installed
// certificates on the device.
//
// Note that user-enrolled devices ignore the [ManagedOnly][2] value set below and will always
// include only managed certificates. This is a limitation imposed by Apple.
//
// [1]: https://developer.apple.com/documentation/devicemanagement/certificatelistcommand
// [2]: https://developer.apple.com/documentation/devicemanagement/certificatelistcommand/command-data.dictionary
func (svc *MDMAppleCommander) CertificateList(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>CommandUUID</key>
		<string>%s</string>
		<key>Command</key>
		<dict>
			<key>ManagedOnly</key>
			<false/>
			<key>RequestType</key>
			<string>CertificateList</string>
		</dict>
	</dict>
</plist>`, cmdUUID)

	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

func (svc *MDMAppleCommander) DeviceLocation(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>DeviceLocation</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>
`, cmdUUID)

	return svc.EnqueueCommand(ctx, hostUUIDs, raw)
}

func (svc *MDMAppleCommander) ClearPasscode(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	raw := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>ClearPasscode</string>
		<key>UnlockToken</key>
		<data>%s</data>
	</dict>
	<key>CommandUUID</key>
	<string>%s</string>
</dict>
</plist>`, "$"+fleet.HostSecretPrefix+fleet.HostSecretMDMUnlockToken, cmdUUID)

	// We skip EnqueueCommand here, to avoid decoding the command as <data> is binary, which fails to decode with placeholder.
	cmd := &mdm.Command{
		CommandUUID: cmdUUID,
		Raw:         []byte(raw),
	}
	cmd.Command.RequestType = fleet.AppleMDMCommandTypeClearPasscode

	return svc.enqueueAndNotify(ctx, hostUUIDs, cmd, mdm.CommandSubtypeNone, "")
}

// EnqueueCommand takes care of enqueuing the commands and sending push
// notifications to the devices.
//
// Always sending the push notification when a command is enqueued was decided
// internally, leaving making pushes optional as an optimization to be tackled
// later.
func (svc *MDMAppleCommander) EnqueueCommand(ctx context.Context, hostUUIDs []string, rawCommand string) error {
	cmd, err := mdm.DecodeCommand([]byte(rawCommand))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "decoding command")
	}

	return svc.enqueueAndNotify(ctx, hostUUIDs, cmd, mdm.CommandSubtypeNone, "")
}

func (svc *MDMAppleCommander) enqueueAndNotify(ctx context.Context, hostUUIDs []string, cmd *mdm.Command,
	subtype mdm.CommandSubtype, name string,
) error {
	if _, err := svc.storage.EnqueueCommand(ctx, hostUUIDs,
		&mdm.CommandWithSubtype{Command: *cmd, Subtype: subtype, Name: name}); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing command")
	}

	if err := svc.SendNotifications(ctx, hostUUIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "sending notifications")
	}
	return nil
}

// EnqueueCommandInstallProfileWithSecrets is a special case of EnqueueCommand that does not expand secret variables.
// Secret variables are expanded when the command is sent to the device, and secrets are never stored in the database unencrypted.
func (svc *MDMAppleCommander) EnqueueCommandInstallProfileWithSecrets(ctx context.Context, hostUUIDs []string,
	rawCommand mobileconfig.Mobileconfig, commandUUID string, name string,
) error {
	cmd := &mdm.Command{
		CommandUUID: commandUUID,
		Raw:         []byte(rawCommand),
	}
	cmd.Command.RequestType = "InstallProfile"

	return svc.enqueueAndNotify(ctx, hostUUIDs, cmd, mdm.CommandSubtypeProfileWithSecrets, name)
}

func (svc *MDMAppleCommander) SendNotifications(ctx context.Context, hostUUIDs []string) error {
	apnsResponses, err := svc.pusher.Push(ctx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "commander push")
	}

	// Even if we didn't get an error, some of the APNs
	// responses might have failed, signal that to the caller.
	failed := map[string]error{}
	for uuid, response := range apnsResponses {
		if response.Err != nil {
			failed[uuid] = response.Err
		}
	}

	if len(failed) > 0 {
		return &APNSDeliveryError{errorsByUUID: failed}
	}

	return nil
}

// BulkDeleteHostUserCommandsWithoutResults calls the storage method with the same name.
func (svc *MDMAppleCommander) BulkDeleteHostUserCommandsWithoutResults(ctx context.Context, commandToIDs map[string][]string) error {
	return svc.storage.BulkDeleteHostUserCommandsWithoutResults(ctx, commandToIDs)
}

// SetRecoveryLock sends the SetRecoveryLock MDM command to set the recovery lock password.
// The password is not included in the command - instead, a placeholder is used that will be
// expanded at delivery time by looking up the password from host_recovery_key_passwords.
// The password must be stored (via SetHostsRecoveryLockPasswords) BEFORE calling this method.
// See https://developer.apple.com/documentation/devicemanagement/set_recovery_lock
func (svc *MDMAppleCommander) SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	// Use the host secret placeholder - the actual password will be injected at delivery time
	// by ExpandHostSecrets, which looks up the password from host_recovery_key_passwords.
	cmdPayload := commandPayload{
		CommandUUID: cmdUUID,
		Command: map[string]any{
			"RequestType": "SetRecoveryLock",
			"NewPassword": "$" + fleet.HostSecretPrefix + fleet.HostSecretRecoveryLockPassword,
		},
	}
	rawBytes, err := plist.MarshalIndent(cmdPayload, "    ")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshalling SetRecoveryLock payload")
	}

	if err := svc.EnqueueCommand(ctx, hostUUIDs, string(rawBytes)); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing SetRecoveryLock command")
	}

	return nil
}

// ClearRecoveryLock sends the SetRecoveryLock MDM command to clear the recovery lock password.
// The CurrentPassword is a placeholder that will be expanded at delivery time by looking up
// the existing password from host_recovery_key_passwords. NewPassword is empty to clear the lock.
// See https://developer.apple.com/documentation/devicemanagement/set_recovery_lock
func (svc *MDMAppleCommander) ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error {
	cmdPayload := commandPayload{
		CommandUUID: cmdUUID,
		Command: map[string]any{
			"RequestType":     "SetRecoveryLock",
			"CurrentPassword": "$" + fleet.HostSecretPrefix + fleet.HostSecretRecoveryLockPassword,
			"NewPassword":     "",
		},
	}
	rawBytes, err := plist.MarshalIndent(cmdPayload, "    ")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshalling ClearRecoveryLock payload")
	}

	if err := svc.EnqueueCommand(ctx, hostUUIDs, string(rawBytes)); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing ClearRecoveryLock command")
	}

	return nil
}

// RotateRecoveryLock sends the SetRecoveryLock MDM command to rotate the recovery lock password.
// Both CurrentPassword and NewPassword are placeholders that will be expanded at delivery time.
// CurrentPassword is the existing password from encrypted_password column.
// NewPassword is the new password from pending_encrypted_password column.
// See https://developer.apple.com/documentation/devicemanagement/set_recovery_lock
func (svc *MDMAppleCommander) RotateRecoveryLock(ctx context.Context, hostUUID string, cmdUUID string) error {
	cmdPayload := commandPayload{
		CommandUUID: cmdUUID,
		Command: map[string]any{
			"RequestType":     "SetRecoveryLock",
			"CurrentPassword": "$" + fleet.HostSecretPrefix + fleet.HostSecretRecoveryLockPassword,
			"NewPassword":     "$" + fleet.HostSecretPrefix + fleet.HostSecretRecoveryLockPendingPassword,
		},
	}
	rawBytes, err := plist.MarshalIndent(cmdPayload, "    ")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshalling RotateRecoveryLock payload")
	}

	if err := svc.EnqueueCommand(ctx, []string{hostUUID}, string(rawBytes)); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueuing RotateRecoveryLock command")
	}

	return nil
}

// APNSDeliveryError records an error and the associated host UUIDs in which it
// occurred.
type APNSDeliveryError struct {
	errorsByUUID map[string]error
}

func (e *APNSDeliveryError) Error() string {
	var uuids []string
	for uuid := range e.errorsByUUID {
		uuids = append(uuids, uuid)
	}

	// sort UUIDs alphabetically for deterministic output
	sort.Strings(uuids)

	var errStrings []string
	for _, uuid := range uuids {
		errStrings = append(errStrings, fmt.Sprintf("UUID: %s, Error: %v", uuid, e.errorsByUUID[uuid]))
	}

	return fmt.Sprintf(
		"APNS delivery failed with the following errors:\n%s",
		strings.Join(errStrings, "\n"),
	)
}

func (e *APNSDeliveryError) FailedUUIDs() []string {
	var uuids []string
	for uuid := range e.errorsByUUID {
		uuids = append(uuids, uuid)
	}

	// sort UUIDs alphabetically for deterministic output
	sort.Strings(uuids)
	return uuids
}

func (e *APNSDeliveryError) StatusCode() int { return http.StatusBadGateway }
