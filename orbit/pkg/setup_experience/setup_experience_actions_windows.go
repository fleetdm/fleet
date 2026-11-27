//go:build windows

//nolint:gosec // G103: unsafe required for Windows API calls.
package setupexperience

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	procWTSLogoffSession = windows.NewLazySystemDLL("wtsapi32.dll").NewProc("WTSLogoffSession")
)

// Windows account flags and NetAPI levels (lmaccess.h).
const (
	ufScript             = 0x0001
	ufAccountDisable     = 0x0002
	ufNormalAccount      = 0x0200
	usePrivUser          = 1
	userInfoFullName     = 1011
	nerrUserExists       = 2224
	nerrPasswordTooShort = 2245
)

var (
	netapi32                 = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserAdd           = netapi32.NewProc("NetUserAdd")
	procNetUserDel           = netapi32.NewProc("NetUserDel")
	procNetUserGetInfo       = netapi32.NewProc("NetUserGetInfo")
	procNetUserModalsGet     = netapi32.NewProc("NetUserModalsGet")
	procNetUserSetInfo       = netapi32.NewProc("NetUserSetInfo")
	procNetLocalGroupAddMbrs = netapi32.NewProc("NetLocalGroupAddMembers")
	procNetAPIBufferFree     = netapi32.NewProc("NetApiBufferFree")
)

// userInfo1 mirrors USER_INFO_1, the structure NetUserAdd level 1 uses.
type userInfo1 struct {
	Name        *uint16
	Password    *uint16
	PasswordAge uint32
	Priv        uint32
	HomeDir     *uint16
	Comment     *uint16
	Flags       uint32
	ScriptPath  *uint16
}

// createLocalAccount creates a standard local Windows account with the password
// the end user chose. NetUserAdd enforces the local password policy.
func createLocalAccount(username, fullname, password string) error {
	namePtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return fmt.Errorf("converting username: %w", err)
	}
	passwordPtr, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return errPasswordPolicy
	}
	info := userInfo1{
		Name:     namePtr,
		Password: passwordPtr,
		Priv:     usePrivUser,
		Flags:    ufScript | ufNormalAccount,
	}
	// Fails with NERR_UserExists rather than taking over an existing account.
	if ret, _, _ := procNetUserAdd.Call(0, 1, uintptr(unsafe.Pointer(&info)), 0); ret != 0 {
		switch ret {
		case nerrUserExists:
			return fmt.Errorf("an account named %s already exists, refusing to take it over", username)
		case nerrPasswordTooShort:
			return errPasswordPolicy
		}
		return fmt.Errorf("NetUserAdd (%s): %w", username, windows.Errno(ret))
	}

	if err := finishLocalAccount(username, namePtr, fullname); err != nil {
		if ret, _, _ := procNetUserDel.Call(0, uintptr(unsafe.Pointer(namePtr))); ret != 0 {
			log.Error().Err(windows.Errno(ret)).Str("username", username).Msg("rolling back partially created account")
		}
		return err
	}

	log.Info().Str("username", username).Str("fullname", fullname).Msg("created local account from IdP identity")
	return nil
}

// minPasswordLength returns the local minimum password length policy, or 0 if
// it can't be read.
func minPasswordLength() int {
	// USER_MODALS_INFO_0 starts with usrmod0_min_passwd_len.
	var buf *uint32
	if ret, _, _ := procNetUserModalsGet.Call(0, 0, uintptr(unsafe.Pointer(&buf))); ret != 0 {
		log.Warn().Err(windows.Errno(ret)).Msg("reading minimum password length")
		return 0
	}
	if buf == nil {
		return 0
	}
	n := int(*buf)
	procNetAPIBufferFree.Call(uintptr(unsafe.Pointer(buf))) //nolint:errcheck
	return n
}

func finishLocalAccount(username string, namePtr *uint16, fullname string) error {
	fullnamePtr, err := windows.UTF16PtrFromString(fullname)
	if err != nil {
		return fmt.Errorf("converting full name: %w", err)
	}
	if ret, _, _ := procNetUserSetInfo.Call(0, uintptr(unsafe.Pointer(namePtr)), userInfoFullName, uintptr(unsafe.Pointer(&fullnamePtr)), 0); ret != 0 {
		return fmt.Errorf("setting full name for %s: %w", username, windows.Errno(ret))
	}

	// NetUserAdd, unlike net user /add, doesn't add the account to Users, which
	// grants interactive logon. The group name is localized, so resolve it by SID.
	sid, err := windows.CreateWellKnownSid(windows.WinBuiltinUsersSid)
	if err != nil {
		return fmt.Errorf("building Users SID: %w", err)
	}
	group, _, _, err := sid.LookupAccount("")
	if err != nil {
		return fmt.Errorf("resolving Users group name: %w", err)
	}
	groupPtr, err := windows.UTF16PtrFromString(group)
	if err != nil {
		return fmt.Errorf("converting group name: %w", err)
	}
	member := namePtr // LOCALGROUP_MEMBERS_INFO_3 is a single DOMAIN\name pointer.
	if ret, _, _ := procNetLocalGroupAddMbrs.Call(0, uintptr(unsafe.Pointer(groupPtr)), 3, uintptr(unsafe.Pointer(&member)), 1); ret != 0 {
		return fmt.Errorf("adding %s to %s: %w", username, group, windows.Errno(ret))
	}
	return nil
}

// openPINEnrollment opens Sign-in options, where PIN (Windows Hello) is the
// first entry, on the logged-in user's desktop. A Hello PIN on a local account is
// built into Windows and needs no Entra ID or Windows Hello for Business.
// Windows 11 ignores ms-settings:signinoptions-launchpinenrollment and opens
// Settings home instead.
func openPINEnrollment() error {
	if _, err := execuser.Run("cmd.exe", execuser.WithArg("/c start ms-settings:signinoptions", "")); err != nil {
		return fmt.Errorf("opening Windows Hello PIN enrollment: %w", err)
	}
	return nil
}

// HideDesktop stops Explorer in the provisioning session, removing the desktop
// icons and taskbar behind the setup experience kiosk, and turns off Task
// Manager for the provisioning account, which is an administrator. Winlogon
// doesn't restart a shell that was terminated this way.
func HideDesktop(sessionID uint32) error {
	if err := disableTaskManager(ProvisioningAccountName); err != nil {
		log.Warn().Err(err).Msg("turning off Task Manager for the provisioning account")
	}
	filter := fmt.Sprintf("SESSION eq %d", sessionID)
	if out, err := exec.Command(systemTool("taskkill.exe"), "/F", "/FI", filter, "/IM", "explorer.exe").CombinedOutput(); err != nil { //nolint:gosec
		return fmt.Errorf("taskkill explorer.exe (%s): %w: %s", filter, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// disableTaskManager sets the DisableTaskMgr user policy in username's loaded
// registry hive. Teardown deletes the profile, so it's never cleared.
func disableTaskManager(username string) error {
	sid, _, _, err := windows.LookupSID("", username)
	if err != nil {
		return fmt.Errorf("looking up SID for %s: %w", username, err)
	}
	k, _, err := registry.CreateKey(registry.USERS, sid.String()+`\Software\Microsoft\Windows\CurrentVersion\Policies\System`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open user policies key: %w", err)
	}
	defer k.Close()
	return k.SetDWordValue("DisableTaskMgr", 1)
}

// localAccountExists reports whether a local account named username exists.
func localAccountExists(username string) bool {
	_, _, _, err := windows.LookupSID("", username)
	return err == nil
}

// localAccountEnabled reports whether a local account named username exists
// and isn't disabled.
func localAccountEnabled(username string) bool {
	namePtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return false
	}
	var info *userInfo1
	if ret, _, _ := procNetUserGetInfo.Call(0, uintptr(unsafe.Pointer(namePtr)), 1, uintptr(unsafe.Pointer(&info))); ret != 0 || info == nil {
		return false
	}
	defer procNetAPIBufferFree.Call(uintptr(unsafe.Pointer(info))) //nolint:errcheck
	return info.Flags&ufAccountDisable == 0
}

// systemTool returns the System32 path of a Windows tool, so a SYSTEM process
// doesn't resolve it through PATH.
func systemTool(name string) string {
	dir, err := windows.GetSystemDirectory()
	if err != nil {
		return name
	}
	return filepath.Join(dir, name)
}

// teardownProvisioningAccount logs off the provisioning session and disables
// the account, so the end user's first login is to their own account (newUser).
// The account and its profile are kept: Windows ties the MDM enrollment made
// during setup to the signed-in account and removes it when that account's
// profile is deleted.
func teardownProvisioningAccount(username string, newUser string) error {
	// LogonUI reads these when it starts at logoff, so set them first: select
	// the new user, keep the provisioning account off the sign-in screen, and
	// stop Winlogon from auto-logging in an account that's about to be disabled.
	if newUser != "" {
		if err := selectLogonUser(newUser); err != nil {
			log.Warn().Err(err).Msg("selecting new user on the sign-in screen")
		}
	}
	if err := hideFromSignInScreen(username, true); err != nil {
		log.Warn().Err(err).Msg("hiding provisioning account from the sign-in screen")
	}
	// Leave the account enabled if this fails, so a retry clears its stored password first.
	if err := clearAutoLogon(username); err != nil {
		return fmt.Errorf("clearing provisioning auto-logon: %w", err)
	}

	// Log off the provisioning session before disabling the account.
	// Re-check the session: the account may have signed in again since setup began.
	if user, sessionID, present, err := execuser.InteractiveSession(); err == nil && present && strings.EqualFold(user, username) {
		if rc, _, err := procWTSLogoffSession.Call(uintptr(windows.Handle(0)), uintptr(sessionID), 1); rc == 0 {
			log.Warn().Err(err).Uint32("session", sessionID).Msg("WTSLogoffSession")
		}
	}

	out, err := exec.Command(systemTool("net.exe"), "user", username, "/active:no").CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("net user /active:no (%s): %w: %s", username, err, strings.TrimSpace(string(out)))
	}
	log.Info().Str("username", username).Msg("disabled provisioning account")
	return nil
}

// blankShell makes username's next sign-in start without Explorer, so its
// desktop never shows. The account's registry hive must be loaded.
func blankShell(username string) error {
	sid, _, _, err := windows.LookupSID("", username)
	if err != nil {
		return fmt.Errorf("looking up SID for %s: %w", username, err)
	}
	k, _, err := registry.CreateKey(registry.USERS, sid.String()+`\Software\Microsoft\Windows NT\CurrentVersion\Winlogon`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open user Winlogon key: %w", err)
	}
	defer k.Close()
	// Winlogon needs a shell command; this one exits at once and shows nothing.
	return k.SetStringValue("Shell", systemTool("rundll32.exe"))
}

// applyProvisioningPackage installs a provisioning package (.ppkg) as SYSTEM.
func applyProvisioningPackage(path string) error {
	ps := filepath.Join(systemTool("WindowsPowerShell"), "v1.0", "powershell.exe")
	script := "$ErrorActionPreference = 'Stop'; Install-ProvisioningPackage -PackagePath '" + strings.ReplaceAll(path, "'", "''") + "' -QuietInstall -ForceInstall | Out-Null"
	out, err := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("Install-ProvisioningPackage: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// entraJoined reports whether the device is joined to an Entra tenant.
func entraJoined() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CloudDomainJoin\JoinInfo`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return false
	}
	defer k.Close()
	names, err := k.ReadSubKeyNames(1)
	return err == nil && len(names) > 0
}

func restartWindows() error {
	out, err := exec.Command(systemTool("shutdown.exe"), "/r", "/t", "0", "/f").CombinedOutput() //nolint:gosec
	if err != nil {
		return fmt.Errorf("shutdown /r: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// hideFromSignInScreen adds (or removes) username in Winlogon's SpecialAccounts
// list, which keeps an account's tile off the sign-in screen.
func hideFromSignInScreen(username string, hide bool) error {
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts\UserList`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open SpecialAccounts UserList key: %w", err)
	}
	defer k.Close()
	if hide {
		return k.SetDWordValue(username, 0)
	}
	if err := k.DeleteValue(username); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

// selectLogonUser makes the sign-in screen default to username instead of the
// provisioning account, the last user to sign in.
func selectLogonUser(username string) error {
	sid, _, _, err := windows.LookupSID("", username)
	if err != nil {
		return fmt.Errorf("looking up SID for %s: %w", username, err)
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Authentication\LogonUI`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open LogonUI key: %w", err)
	}
	defer k.Close()
	for name, v := range map[string]string{
		"LastLoggedOnUser":    `.\` + username,
		"LastLoggedOnSAMUser": `.\` + username,
		"LastLoggedOnUserSID": sid.String(),
		"SelectedUserSID":     sid.String(),
	} {
		if err := k.SetStringValue(name, v); err != nil {
			return fmt.Errorf("set LogonUI %s: %w", name, err)
		}
	}
	if err := k.DeleteValue("LastLoggedOnDisplayName"); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete LogonUI LastLoggedOnDisplayName: %w", err)
	}
	return nil
}

// clearAutoLogon removes the Winlogon auto-logon the provisioning unattend.xml
// configured, which otherwise keeps the account's plaintext password in the
// registry and fails on the next boot by trying to sign in a disabled account.
func clearAutoLogon(username string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Winlogon key: %w", err)
	}
	defer k.Close()

	if name, _, err := k.GetStringValue("DefaultUserName"); err != nil || !strings.EqualFold(name, username) {
		return nil
	}
	if err := k.SetStringValue("AutoAdminLogon", "0"); err != nil {
		return fmt.Errorf("disable AutoAdminLogon: %w", err)
	}
	for _, v := range []string{"DefaultPassword", "AutoLogonCount", "DefaultUserName"} {
		if err := k.DeleteValue(v); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("delete Winlogon %s: %w", v, err)
		}
	}
	return nil
}
