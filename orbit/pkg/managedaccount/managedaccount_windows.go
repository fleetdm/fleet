//go:build windows

package managedaccount

import (
	"errors"
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
	ufAccountDisable   = 0x0002 // UF_ACCOUNTDISABLE: the account exists but cannot log in.
	ufLockout          = 0x0010 // UF_LOCKOUT: locked out by failed logons. Can be cleared, not set.

	usePrivUser = 1 // USER_PRIV_USER: a plain user; group membership grants admin rights.

	nerrUserNotFound     = 2221 // NERR_UserNotFound
	errorMemberInAlias   = 1378 // ERROR_MEMBER_IN_ALIAS: already a group member.
	userInfoPasswordOnly = 1003 // USER_INFO_1003: password-only update level.
	userInfoFlagsOnly    = 1008 // USER_INFO_1008: flags-only update level.

	// NERR_PasswordTooShort is the catch-all Windows returns for any password-policy rejection, not just length: MSDN
	// lists it for "too long, too recent in its change history, not enough unique characters, or does not meet another
	// password policy requirement", which includes a custom password filter DLL.
	nerrPasswordTooShort = 2245
	// ERROR_PASSWORD_RESTRICTION, the equivalent from the system error range.
	errorPasswordRestriction = 1325
)

// logonUIHiddenAccountsKey holds a DWORD per account name; 0 hides the account from the sign-in
// screen and from Settings > Accounts. There is no MDM CSP for this, which is why fleetd does it.
const logonUIHiddenAccountsKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts\UserList`

// fleetAccountComment is written as the account's description at creation: it is how a later run tells an account Fleet
// created from an unrelated account that happens to share the name. Do not change this string.
const fleetAccountComment = "Fleet-managed local administrator account."

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

// userInfo1008 mirrors USER_INFO_1008, which sets only the account flags.
type userInfo1008 struct {
	Flags uint32
}

// localGroupMembersInfo3 mirrors LOCALGROUP_MEMBERS_INFO_3, which identifies a member by name rather than SID.
type localGroupMembersInfo3 struct {
	DomainAndName *uint16
}

// provisionAccount creates the managed local admin account if it is missing, resets its password if it already exists,
// ensures it is a member of the local Administrators group, and hides it from the sign-in screen. Every step is
// idempotent so the whole function is safe to re-run, which is what makes retrying after a failed escrow safe.
func provisionAccount(username, password string) error {
	if err := ensureUser(username, password); err != nil {
		return err
	}
	if err := addToAdministrators(username); err != nil {
		return err
	}
	return hideFromSignInScreen(username)
}

// ensureUser creates the account, or resets its password when it already exists. The reset branch is what lets a retry
// recover an account whose password Fleet never successfully escrowed.
func ensureUser(username, password string) error {
	namePtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return fmt.Errorf("converting username: %w", err)
	}
	passwordPtr, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return fmt.Errorf("converting password: %w", err)
	}

	existing, err := lookupUser(namePtr)
	if err != nil {
		return err
	}

	if existing != nil {
		// Only adopt an account Fleet created. Anything else with this name belongs to someone else, and resetting its
		// password and elevating it would be destructive; report it instead so it surfaces on the host rather than
		// silently changing an account we do not own.
		if existing.comment != fleetAccountComment {
			return fmt.Errorf(
				"an account named %s already exists and was not created by Fleet, refusing to take it over", username)
		}

		info := userInfo1003{Password: passwordPtr}
		ret, _, _ := procNetUserSetInfo.Call(
			0, // servername: NULL means the local machine
			uintptr(unsafe.Pointer(namePtr)),
			userInfoPasswordOnly,
			uintptr(unsafe.Pointer(&info)),
			0, // parm_err
		)
		if ret != 0 {
			return accountError(fmt.Sprintf("Resetting password for %s", username), ret, len(password))
		}
		// Resetting the password is not enough to make the account usable again. If it was disabled, locked out, or had
		// its never-expire flag removed after we created it, Fleet would escrow a password that cannot actually log in.
		// Only the flags we own are touched, so anything else set on the account is preserved.
		return normalizeUserFlags(namePtr, username, existing.flags)
	}

	comment, err := windows.UTF16PtrFromString(fleetAccountComment)
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
		return accountError(fmt.Sprintf("Creating %s", username), ret, len(password))
	}
	return nil
}

// accountError turns a netapi32 return code into an error, spelling out password-policy rejections.
// Windows reports every one of those as NERR_PasswordTooShort, whose text lives in netmsg.dll rather
// than the system message table, so Go cannot format it and the admin would otherwise see the reason
// their break-glass account never appeared as a bare "winapi error #2245" in Fleet.
func accountError(op string, ret uintptr, passwordLen int) error {
	if ret == nerrPasswordTooShort || ret == errorPasswordRestriction {
		return fmt.Errorf(
			"%s. This device's password policy rejected the generated %d-character password. "+
				"Check any custom password filter on the host.",
			op, passwordLen)
	}
	return fmt.Errorf("%s: %w", op, windows.Errno(ret))
}

// existingAccount is the subset of USER_INFO_1 the caller needs: the flags say whether a present
// account is actually usable, and the comment says whether it is ours to manage.
type existingAccount struct {
	flags   uint32
	comment string
}

// lookupUser returns the account, or nil when no account with that name exists.
func lookupUser(namePtr *uint16) (*existingAccount, error) {
	// buf is a real pointer rather than a uintptr so the garbage collector tracks the buffer netapi32
	// allocates for us; converting a uintptr back into a pointer is not safe.
	var buf *byte
	ret, _, _ := procNetUserGetInfo.Call(
		0, // servername
		uintptr(unsafe.Pointer(namePtr)),
		1, // level 1: USER_INFO_1, which carries Flags
		uintptr(unsafe.Pointer(&buf)),
	)
	switch ret {
	case 0:
		if buf == nil {
			return nil, errors.New("looking up account: NetUserGetInfo returned no data")
		}
		//nolint:errcheck // freeing the buffer cannot meaningfully fail here
		defer procNetAPIBufferFree.Call(uintptr(unsafe.Pointer(buf)))
		info := (*userInfo1)(unsafe.Pointer(buf))
		return &existingAccount{
			flags:   info.Flags,
			comment: windows.UTF16PtrToString(info.Comment),
		}, nil
	case nerrUserNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("looking up account: %w", windows.Errno(ret))
	}
}

// normalizeUserFlags re-applies the flags Fleet depends on to an account that already existed:
// enabled, not locked out, and password never expires. Other flags are left untouched.
func normalizeUserFlags(namePtr *uint16, username string, current uint32) error {
	desired := (current &^ (ufAccountDisable | ufLockout)) | ufDontExpirePasswd
	if desired == current {
		return nil
	}

	info := userInfo1008{Flags: desired}
	ret, _, _ := procNetUserSetInfo.Call(
		0, // servername
		uintptr(unsafe.Pointer(namePtr)),
		userInfoFlagsOnly,
		uintptr(unsafe.Pointer(&info)),
		0, // parm_err
	)
	if ret != 0 {
		return fmt.Errorf("restoring account flags for %s: %w", username, windows.Errno(ret))
	}
	log.Debug().Str("username", username).Uint32("from", current).Uint32("to", desired).
		Msg("managed local account: restored account flags")
	return nil
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
