//go:build windows

package managedaccount

import (
	"fmt"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Windows account flags (lmaccess.h). UF_DONT_EXPIRE_PASSWD keeps the account usable as a
// break-glass login: Fleet owns the password lifecycle, so Windows must not expire it underneath us.
const (
	ufScript           = 0x0001
	ufNormalAccount    = 0x0200
	ufDontExpirePasswd = 0x10000

	usePrivUser = 1 // USER_PRIV_USER: a plain user; group membership grants admin rights.

	nerrUserNotFound     = 2221 // NERR_UserNotFound
	errorMemberInAlias   = 1378 // ERROR_MEMBER_IN_ALIAS: already a group member.
	userInfoPasswordOnly = 1003 // USER_INFO_1003: password-only update level.
)

// logonUIHiddenAccountsKey holds a DWORD per account name; 0 hides the account from the sign-in
// screen and from Settings > Accounts. There is no MDM CSP for this, which is why fleetd does it.
const logonUIHiddenAccountsKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts\UserList`

var (
	netapi32                 = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserAdd           = netapi32.NewProc("NetUserAdd")
	procNetUserGetInfo       = netapi32.NewProc("NetUserGetInfo")
	procNetUserSetInfo       = netapi32.NewProc("NetUserSetInfo")
	procNetLocalGroupAddMbrs = netapi32.NewProc("NetLocalGroupAddMembers")
	procNetAPIBufferFree     = netapi32.NewProc("NetApiBufferFree")
)

// userInfo1 mirrors USER_INFO_1 (lmaccess.h), the level-1 structure NetUserAdd takes.
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

// userInfo1003 mirrors USER_INFO_1003, which sets only the password.
type userInfo1003 struct {
	Password *uint16
}

// localGroupMembersInfo3 mirrors LOCALGROUP_MEMBERS_INFO_3, which identifies a member by name
// rather than SID.
type localGroupMembersInfo3 struct {
	DomainAndName *uint16
}

// provisionAccount creates the managed local admin account if it is missing, resets its password if
// it already exists, ensures it is a member of the local Administrators group, and hides it from the
// sign-in screen. Every step is idempotent so the whole function is safe to re-run, which is what
// makes retrying after a failed escrow safe.
func provisionAccount(username, password string) error {
	if err := ensureUser(username, password); err != nil {
		return err
	}
	if err := addToAdministrators(username); err != nil {
		return err
	}
	return hideFromSignInScreen(username)
}

// ensureUser creates the account, or resets its password when it already exists. The reset branch is
// what lets a retry recover an account whose password Fleet never successfully escrowed.
func ensureUser(username, password string) error {
	namePtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return fmt.Errorf("converting username: %w", err)
	}
	passwordPtr, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return fmt.Errorf("converting password: %w", err)
	}

	exists, err := userExists(namePtr)
	if err != nil {
		return err
	}

	if exists {
		info := userInfo1003{Password: passwordPtr}
		ret, _, _ := procNetUserSetInfo.Call(
			0, // servername: NULL means the local machine
			uintptr(unsafe.Pointer(namePtr)),
			userInfoPasswordOnly,
			uintptr(unsafe.Pointer(&info)),
			0, // parm_err
		)
		if ret != 0 {
			return fmt.Errorf("resetting password for %s: %w", username, windows.Errno(ret))
		}
		return nil
	}

	comment, err := windows.UTF16PtrFromString("Fleet-managed local administrator account.")
	if err != nil {
		return fmt.Errorf("converting comment: %w", err)
	}
	info := userInfo1{
		Name:     namePtr,
		Password: passwordPtr,
		Priv:     usePrivUser,
		Comment:  comment,
		Flags:    ufScript | ufNormalAccount | ufDontExpirePasswd,
	}
	ret, _, _ := procNetUserAdd.Call(
		0, // servername
		1, // level
		uintptr(unsafe.Pointer(&info)),
		0, // parm_err
	)
	if ret != 0 {
		return fmt.Errorf("creating %s: %w", username, windows.Errno(ret))
	}
	return nil
}

func userExists(namePtr *uint16) (bool, error) {
	var buf uintptr
	ret, _, _ := procNetUserGetInfo.Call(
		0, // servername
		uintptr(unsafe.Pointer(namePtr)),
		0, // level 0: name only, all we need is presence
		uintptr(unsafe.Pointer(&buf)),
	)
	switch ret {
	case 0:
		if buf != 0 {
			//nolint:errcheck // freeing the buffer cannot meaningfully fail here
			procNetAPIBufferFree.Call(buf)
		}
		return true, nil
	case nerrUserNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("looking up account: %w", windows.Errno(ret))
	}
}

// addToAdministrators adds the account to the local Administrators group. The group name is resolved
// from its well-known SID rather than hardcoded, because it is localized on non-English Windows.
func addToAdministrators(username string) error {
	groupName, err := administratorsGroupName()
	if err != nil {
		return err
	}
	groupPtr, err := windows.UTF16PtrFromString(groupName)
	if err != nil {
		return fmt.Errorf("converting group name: %w", err)
	}
	memberPtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return fmt.Errorf("converting member name: %w", err)
	}

	member := localGroupMembersInfo3{DomainAndName: memberPtr}
	ret, _, _ := procNetLocalGroupAddMbrs.Call(
		0, // servername
		uintptr(unsafe.Pointer(groupPtr)),
		3, // level
		uintptr(unsafe.Pointer(&member)),
		1, // totalentries
	)
	// Already a member is the expected outcome on every run after the first.
	if ret != 0 && ret != errorMemberInAlias {
		return fmt.Errorf("adding %s to %s: %w", username, groupName, windows.Errno(ret))
	}
	return nil
}

func administratorsGroupName() (string, error) {
	sid, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return "", fmt.Errorf("building Administrators SID: %w", err)
	}
	account, _, _, err := sid.LookupAccount("")
	if err != nil {
		return "", fmt.Errorf("resolving Administrators group name: %w", err)
	}
	return account, nil
}

// hideFromSignInScreen keeps the account off the sign-in screen and out of Settings > Accounts. It
// remains fully usable through "Other user" with an explicit username.
func hideFromSignInScreen(username string) error {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, logonUIHiddenAccountsKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening UserList registry key: %w", err)
	}
	defer key.Close()

	if err := key.SetDWordValue(username, 0); err != nil {
		return fmt.Errorf("hiding %s from the sign-in screen: %w", username, err)
	}
	log.Debug().Str("username", username).Msg("managed local account: hidden from sign-in screen")
	return nil
}
