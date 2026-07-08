//go:build windows

package terminal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows ConPTY (pseudo-console) implementation.
// Requires Windows 10 version 1809 (build 17763) or later.
//
// Architecture:
//
//	inW ──► [pipe] ──► hInR ──► ConPTY ──► hOutW ──► [pipe] ──► outR
//	(our write end)   (pty reads)         (pty writes)          (our read end)
//
// We write keyboard input to inW; we read PTY output from outR.

var (
	kernel32                         = windows.NewLazySystemDLL("kernel32.dll")
	procCreatePseudoConsole          = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole          = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole           = kernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttrList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttr         = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttrList     = kernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE attaches a ConPTY to a new process.
	procThreadAttrPseudoConsole = 0x00020016
	// EXTENDED_STARTUPINFO_PRESENT signals that lpStartupInfo is STARTUPINFOEXW.
	extendedStartupInfoPresent = 0x00080000
)

// winCoord mirrors the Win32 COORD struct (two int16s packed into 32 bits).
type winCoord struct{ X, Y int16 }

// startupInfoEx mirrors STARTUPINFOEXW.
type startupInfoEx struct {
	windows.StartupInfo
	ProcThreadAttributeList *byte
}

// shell wraps a PowerShell process connected to a Windows ConPTY.
type shell struct {
	hpc     windows.Handle // HPCON pseudo-console handle
	proc    windows.Handle
	thread  windows.Handle
	inW     *os.File // write end → PTY stdin
	outR    *os.File // read end ← PTY stdout
	attrBuf []byte   // attribute list buffer (must outlive the process)
}

func startShell(_ context.Context) (*shell, error) {
	// Prefer pwsh (PowerShell 7+), fall back to legacy powershell.exe.
	bin, err := exec.LookPath("pwsh.exe")
	if err != nil {
		if bin, err = exec.LookPath("powershell.exe"); err != nil {
			return nil, fmt.Errorf("terminal: PowerShell not found: %w", err)
		}
	}

	// ── Pipe pair for PTY stdin ──────────────────────────────────────────────
	// ConPTY reads from hInR; we write to hInW.
	var hInR, hInW windows.Handle
	if err := windows.CreatePipe(&hInR, &hInW, nil, 0); err != nil {
		return nil, fmt.Errorf("terminal: create input pipe: %w", err)
	}

	// ── Pipe pair for PTY stdout ─────────────────────────────────────────────
	// ConPTY writes to hOutW; we read from hOutR.
	var hOutR, hOutW windows.Handle
	if err := windows.CreatePipe(&hOutR, &hOutW, nil, 0); err != nil {
		windows.CloseHandle(hInR) //nolint:errcheck
		windows.CloseHandle(hInW) //nolint:errcheck
		return nil, fmt.Errorf("terminal: create output pipe: %w", err)
	}

	// ── CreatePseudoConsole ──────────────────────────────────────────────────
	// COORD size is two int16s packed; pass as the low 32 bits of a uintptr.
	initSize := winCoord{X: 80, Y: 24}
	var hpc windows.Handle
	hr, _, _ := procCreatePseudoConsole.Call(
		uintptr(*(*uint32)(unsafe.Pointer(&initSize))),
		uintptr(hInR),
		uintptr(hOutW),
		0,
		uintptr(unsafe.Pointer(&hpc)),
	)
	if hr != 0 {
		windows.CloseHandle(hInR)  //nolint:errcheck
		windows.CloseHandle(hInW)  //nolint:errcheck
		windows.CloseHandle(hOutR) //nolint:errcheck
		windows.CloseHandle(hOutW) //nolint:errcheck
		return nil, fmt.Errorf("terminal: CreatePseudoConsole HRESULT 0x%x", hr)
	}
	// ConPTY now owns hInR and hOutW; close our copies.
	windows.CloseHandle(hInR)  //nolint:errcheck
	windows.CloseHandle(hOutW) //nolint:errcheck

	// ── Process thread attribute list ────────────────────────────────────────
	// First call: get the required buffer size.
	var attrListSize uintptr
	procInitializeProcThreadAttrList.Call(0, 1, 0, uintptr(unsafe.Pointer(&attrListSize))) //nolint:errcheck

	attrBuf := make([]byte, attrListSize)
	if ret, _, e := procInitializeProcThreadAttrList.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])), 1, 0, uintptr(unsafe.Pointer(&attrListSize)),
	); ret == 0 {
		windows.CloseHandle(hInW) //nolint:errcheck
		windows.CloseHandle(hOutR) //nolint:errcheck
		procClosePseudoConsole.Call(uintptr(hpc)) //nolint:errcheck
		return nil, fmt.Errorf("terminal: InitializeProcThreadAttributeList: %w", e)
	}

	if ret, _, e := procUpdateProcThreadAttr.Call(
		uintptr(unsafe.Pointer(&attrBuf[0])),
		0,
		procThreadAttrPseudoConsole,
		uintptr(unsafe.Pointer(&hpc)),
		unsafe.Sizeof(hpc),
		0, 0,
	); ret == 0 {
		procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0]))) //nolint:errcheck
		windows.CloseHandle(hInW) //nolint:errcheck
		windows.CloseHandle(hOutR) //nolint:errcheck
		procClosePseudoConsole.Call(uintptr(hpc)) //nolint:errcheck
		return nil, fmt.Errorf("terminal: UpdateProcThreadAttribute: %w", e)
	}

	// ── Build STARTUPINFOEXW ─────────────────────────────────────────────────
	siEx := startupInfoEx{}
	siEx.Cb = uint32(unsafe.Sizeof(siEx))
	siEx.ProcThreadAttributeList = &attrBuf[0]

	// Quote the binary path so CreateProcess correctly identifies the module
	// even when the path contains spaces (e.g. C:\Program Files\PowerShell\…).
	// lpApplicationName is set explicitly as a belt-and-suspenders measure.
	appUTF16, err := windows.UTF16PtrFromString(bin)
	if err != nil {
		procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0]))) //nolint:errcheck
		windows.CloseHandle(hInW)                                               //nolint:errcheck
		windows.CloseHandle(hOutR)                                              //nolint:errcheck
		procClosePseudoConsole.Call(uintptr(hpc))                               //nolint:errcheck
		return nil, fmt.Errorf("terminal: UTF16PtrFromString application: %w", err)
	}
	// lpCommandLine provides argv[0]; quote path to handle embedded spaces.
	cmdUTF16, err := windows.UTF16PtrFromString(`"` + bin + `"`)
	if err != nil {
		procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0]))) //nolint:errcheck
		windows.CloseHandle(hInW)                                               //nolint:errcheck
		windows.CloseHandle(hOutR)                                              //nolint:errcheck
		procClosePseudoConsole.Call(uintptr(hpc))                               //nolint:errcheck
		return nil, fmt.Errorf("terminal: UTF16PtrFromString command: %w", err)
	}

	// ── CreateProcessW ───────────────────────────────────────────────────────
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(
		appUTF16,
		cmdUTF16,
		nil, nil,
		false,
		extendedStartupInfoPresent,
		nil, nil,
		// Cast *startupInfoEx → *windows.StartupInfo; Windows reads cb to know
		// the struct is actually STARTUPINFOEXW and accesses ProcThreadAttributeList.
		(*windows.StartupInfo)(unsafe.Pointer(&siEx)),
		&pi,
	); err != nil {
		procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&attrBuf[0]))) //nolint:errcheck
		windows.CloseHandle(hInW) //nolint:errcheck
		windows.CloseHandle(hOutR) //nolint:errcheck
		procClosePseudoConsole.Call(uintptr(hpc)) //nolint:errcheck
		return nil, fmt.Errorf("terminal: CreateProcess: %w", err)
	}

	return &shell{
		hpc:     hpc,
		proc:    pi.Process,
		thread:  pi.Thread,
		inW:     os.NewFile(uintptr(hInW), "conpty-in"),
		outR:    os.NewFile(uintptr(hOutR), "conpty-out"),
		attrBuf: attrBuf,
	}, nil
}

func (s *shell) read(p []byte) (int, error)  { return s.outR.Read(p) }
func (s *shell) write(p []byte) (int, error) { return s.inW.Write(p) }

func (s *shell) resize(cols, rows uint16) error {
	size := winCoord{X: int16(cols), Y: int16(rows)}
	hr, _, _ := procResizePseudoConsole.Call(
		uintptr(s.hpc),
		uintptr(*(*uint32)(unsafe.Pointer(&size))),
	)
	if hr != 0 {
		return fmt.Errorf("terminal: ResizePseudoConsole HRESULT 0x%x", hr)
	}
	return nil
}

func (s *shell) close() {
	s.inW.Close()
	s.outR.Close()
	procClosePseudoConsole.Call(uintptr(s.hpc))                               //nolint:errcheck
	procDeleteProcThreadAttrList.Call(uintptr(unsafe.Pointer(&s.attrBuf[0]))) //nolint:errcheck
	windows.TerminateProcess(s.proc, 1)                                        //nolint:errcheck
	windows.WaitForSingleObject(s.proc, windows.INFINITE)                      //nolint:errcheck
	windows.CloseHandle(s.proc)                                                //nolint:errcheck
	windows.CloseHandle(s.thread)                                              //nolint:errcheck
}
