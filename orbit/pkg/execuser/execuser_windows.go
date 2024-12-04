package execuser

// NOTE: The following was copied from
// https://gist.github.com/LiamHaworth/1ac37f7fb6018293fc43f86993db24fc
//
// To view what was modified/added, you can use the execuser_windows_diff.sh script.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"unsafe"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"golang.org/x/sys/windows"
)

var (
	modwtsapi32 *windows.LazyDLL = windows.NewLazySystemDLL("wtsapi32.dll")
	modkernel32 *windows.LazyDLL = windows.NewLazySystemDLL("kernel32.dll")
	modadvapi32 *windows.LazyDLL = windows.NewLazySystemDLL("advapi32.dll")
	moduserenv  *windows.LazyDLL = windows.NewLazySystemDLL("userenv.dll")

	procWTSEnumerateSessionsW        *windows.LazyProc = modwtsapi32.NewProc("WTSEnumerateSessionsW")
	procWTSGetActiveConsoleSessionId *windows.LazyProc = modkernel32.NewProc("WTSGetActiveConsoleSessionId")
	procWTSQueryUserToken            *windows.LazyProc = modwtsapi32.NewProc("WTSQueryUserToken")
	procDuplicateTokenEx             *windows.LazyProc = modadvapi32.NewProc("DuplicateTokenEx")
	procCreateEnvironmentBlock       *windows.LazyProc = moduserenv.NewProc("CreateEnvironmentBlock")
	procCreateProcessAsUser          *windows.LazyProc = modadvapi32.NewProc("CreateProcessAsUserW")
)

const (
	WTS_CURRENT_SERVER_HANDLE uintptr = 0
)

type WTS_CONNECTSTATE_CLASS int

const (
	WTSActive WTS_CONNECTSTATE_CLASS = iota
	WTSConnected
	WTSConnectQuery
	WTSShadow
	WTSDisconnected
	WTSIdle
	WTSListen
	WTSReset
	WTSDown
	WTSInit
)

type SECURITY_IMPERSONATION_LEVEL int

const (
	SecurityAnonymous SECURITY_IMPERSONATION_LEVEL = iota
	SecurityIdentification
	SecurityImpersonation
	SecurityDelegation
)

type TOKEN_TYPE int

const (
	TokenPrimary TOKEN_TYPE = iota + 1
	TokenImpersonazion
)

type SW int

const (
	SW_HIDE            SW = 0
	SW_SHOWNORMAL         = 1
	SW_NORMAL             = 1
	SW_SHOWMINIMIZED      = 2
	SW_SHOWMAXIMIZED      = 3
	SW_MAXIMIZE           = 3
	SW_SHOWNOACTIVATE     = 4
	SW_SHOW               = 5
	SW_MINIMIZE           = 6
	SW_SHOWMINNOACTIVE    = 7
	SW_SHOWNA             = 8
	SW_RESTORE            = 9
	SW_SHOWDEFAULT        = 10
	SW_MAX                = 1
)

type WTS_SESSION_INFO struct {
	SessionID      windows.Handle
	WinStationName *uint16
	State          WTS_CONNECTSTATE_CLASS
}

const (
	CREATE_UNICODE_ENVIRONMENT uint16 = 0x00000400
	CREATE_NO_WINDOW                  = 0x08000000
	CREATE_NEW_CONSOLE                = 0x00000010
)

// run uses the Windows API to run a child process as the current login user.
// It assumes the caller is running as a SYSTEM Windows service.
//
// It sets the environment of the current process so that it gets inherited by
// the child process (see call to CreateEnvironmentBlock).
// From https://docs.microsoft.com/en-us/windows/win32/procthread/changing-environment-variables:
//
//	"If you want the child process to inherit most of the parent's environment with
//	only a few changes, retrieve the current values using GetEnvironmentVariable, save these values,
//	create an updated block for the child process to inherit, create the child process, and then
//	restore the saved values using SetEnvironmentVariable, as shown in the following example."
func run(path string, opts eopts) (lastLogs string, err error) {
	if err := setupSyncChannel(constant.DesktopAppExecName); err != nil {
		return "", fmt.Errorf("sync channel creation failed: %w", err)
	}

	for _, nv := range opts.env {
		os.Setenv(nv[0], nv[1])
	}
	return "", startProcessAsCurrentUser(path, "", "")
}

func runWithOutput(path string, opts eopts) (output []byte, exitCode int, err error) {
	return nil, 0, errors.New("not implemented")
}

func runWithStdin(path string, opts eopts) (io.WriteCloser, error) {
	return nil, errors.New("not implemented")
}

func runWithContext(ctx context.Context, path string, opts eopts) error {
	return errors.New("not implemented")
}

// getCurrentUserSessionId will attempt to resolve
// the session ID of the user currently active on
// the system.
func getCurrentUserSessionId() (windows.Handle, error) {
	sessionList, err := wtsEnumerateSessions()
	if err != nil {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: %s", err)
	}

	for i := range sessionList {
		if sessionList[i].State == WTSActive {
			return sessionList[i].SessionID, nil
		}
	}

	// TODO(lucas): Check which sessions is assigned to current log in user.
	if sessionId, _, err := procWTSGetActiveConsoleSessionId.Call(); sessionId == 0xFFFFFFFF {
		return 0xFFFFFFFF, fmt.Errorf("get current user session token: call native WTSGetActiveConsoleSessionId: %s", err)
	} else {
		return windows.Handle(sessionId), nil
	}
}

// wtsEnumerateSession will call the native
// version for Windows and parse the result
// to a Golang friendly version
func wtsEnumerateSessions() ([]*WTS_SESSION_INFO, error) {
	var (
		sessionInformation windows.Handle      = windows.Handle(0)
		sessionCount       int                 = 0
		sessionList        []*WTS_SESSION_INFO = make([]*WTS_SESSION_INFO, 0)
	)

	if returnCode, _, err := procWTSEnumerateSessionsW.Call(WTS_CURRENT_SERVER_HANDLE, 0, 1, uintptr(unsafe.Pointer(&sessionInformation)), uintptr(unsafe.Pointer(&sessionCount))); returnCode == 0 {
		return nil, fmt.Errorf("call native WTSEnumerateSessionsW: %s", err)
	}

	structSize := unsafe.Sizeof(WTS_SESSION_INFO{})
	current := uintptr(sessionInformation)
	for i := 0; i < sessionCount; i++ {
		sessionList = append(sessionList, (*WTS_SESSION_INFO)(unsafe.Pointer(current)))
		current += structSize
	}

	return sessionList, nil
}

// duplicateUserTokenFromSessionID will attempt
// to duplicate the user token for the user logged
// into the provided session ID
func duplicateUserTokenFromSessionID(sessionId windows.Handle) (windows.Token, error) {
	var (
		impersonationToken windows.Handle = 0
		userToken          windows.Token  = 0
	)

	if returnCode, _, err := procWTSQueryUserToken.Call(uintptr(sessionId), uintptr(unsafe.Pointer(&impersonationToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native WTSQueryUserToken: %s", err)
	}

	if returnCode, _, err := procDuplicateTokenEx.Call(uintptr(impersonationToken), 0, 0, uintptr(SecurityImpersonation), uintptr(TokenPrimary), uintptr(unsafe.Pointer(&userToken))); returnCode == 0 {
		return 0xFFFFFFFF, fmt.Errorf("call native DuplicateTokenEx: %s", err)
	}

	if err := windows.CloseHandle(impersonationToken); err != nil {
		return 0xFFFFFFFF, fmt.Errorf("close windows handle used for token duplication: %s", err)
	}

	return userToken, nil
}

func startProcessAsCurrentUser(appPath, cmdLine, workDir string) error {
	var (
		sessionId windows.Handle
		userToken windows.Token
		envInfo   windows.Handle

		startupInfo windows.StartupInfo
		processInfo windows.ProcessInformation

		commandLine uintptr = 0
		workingDir  uintptr = 0

		err error
	)

	if sessionId, err = getCurrentUserSessionId(); err != nil {
		return err
	}

	if userToken, err = duplicateUserTokenFromSessionID(sessionId); err != nil {
		return fmt.Errorf("get duplicate user token for current user session: %s", err)
	}

	if returnCode, _, err := procCreateEnvironmentBlock.Call(uintptr(unsafe.Pointer(&envInfo)), uintptr(userToken), 1); returnCode == 0 {
		return fmt.Errorf("create environment details for process: %s", err)
	}

	// TODO(lucas): Test out creation flags and startup info values.
	creationFlags := CREATE_UNICODE_ENVIRONMENT | CREATE_NEW_CONSOLE
	startupInfo.ShowWindow = SW_SHOW
	startupInfo.Desktop = windows.StringToUTF16Ptr("winsta0\\default")

	if len(cmdLine) > 0 {
		commandLine = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(cmdLine)))
	}
	if len(workDir) > 0 {
		workingDir = uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(workDir)))
	}

	if returnCode, _, err := procCreateProcessAsUser.Call(
		uintptr(userToken), uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(appPath))), commandLine, 0, 0, 0,
		uintptr(creationFlags), uintptr(envInfo), workingDir, uintptr(unsafe.Pointer(&startupInfo)), uintptr(unsafe.Pointer(&processInfo)),
	); returnCode == 0 {
		return fmt.Errorf("create process as user: %s", err)
	}

	return nil
}

// setupSyncChannel creates the synchronization channel through which child process will be
// signalled for termination. Code is using kernel named object as the sync primitive.
// https://learn.microsoft.com/en-us/windows/win32/sync/using-event-objects
func setupSyncChannel(channelId string) error {
	if channelId == "" {
		return errors.New("communication channel name should not be empty")
	}

	// converting go string to UTF16 windows compatible string
	targetChannel := "Global\\comm-" + channelId
	ev, err := windows.UTF16PtrFromString(targetChannel)
	if (err != nil) && (err != windows.ERROR_SUCCESS) {
		return fmt.Errorf("there was a problem generating UTF16 string: %w", err)
	}

	// checking first if channel is already present through the OpenEvent API
	// OpenEvent API opens a named event object from the kernel object manager
	// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-openeventw
	ha, err := windows.OpenEvent(windows.EVENT_ALL_ACCESS, false, ev)
	if (ha != windows.InvalidHandle) && (err == nil) {
		// Closing the handle to avoid handle leaks.
		windows.CloseHandle(ha) //nolint:errcheck
		return nil              // channel is already present - nothing to do here
	}

	// Security descriptor to be used with event object
	// Security descriptor is set to
	//  - Allow SYSTEM user to have full control
	//    (SY=SYSTEM, 0x001F0003=EVENT_ALL_ACCESS)
	//  - Allow Authenticated user to have only SYNCHRONIZE privilege
	//    (AU=Authenticated Users, 0x00100000=SYNCHRONIZE)
	// https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-security_descriptor
	sd, err := windows.SecurityDescriptorFromString("D:(A;;0x001F0003;;;SY)(A;;0x00100000;;;AU)")
	if err != nil {
		return fmt.Errorf("SecurityDescriptorFromString failed: %v", err)
	}

	// Security attributes passed to the event object
	// https://learn.microsoft.com/en-us/previous-versions/windows/desktop/legacy/aa379560(v=vs.85)
	sa := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: sd,
	}

	// CreateEvent Api creates the named event object on the kernel object manager
	// The obtained handle is not closed on purpose, the lifetime of the event object
	// is going to be bound to the lifetime of the service
	// https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-createeventw
	h, err := windows.CreateEvent(&sa, 0, 0, ev)
	if (err != nil) && (err != windows.ERROR_SUCCESS) {
		return fmt.Errorf("there was a problem calling CreateEvent: %w", err)
	}

	if h == windows.InvalidHandle {
		return errors.New("event handle is invalid")
	}

	return nil
}
