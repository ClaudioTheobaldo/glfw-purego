//go:build windows

package glfw

import (
	"runtime"
	"syscall"
	"unsafe"
)

// ----------------------------------------------------------------------------
// Package-level global state
// ----------------------------------------------------------------------------

var (
	gHInstance   uintptr // HINSTANCE
	gClassAtom   uint16  // atom from RegisterClassExW
	gWndProcCB   uintptr // syscall.NewCallback(wndProc)
	gDefaultCursor uintptr // IDC_ARROW
)

// Init initialises the GLFW subsystem on Windows.
//
// Must be called from the main goroutine. Locks the calling goroutine to its
// OS thread permanently (matching the GLFW contract).
func Init() error {
	runtime.LockOSThread()

	var err error
	gHInstance, err = getModuleHandleW(nil)
	if err != nil {
		return &Error{Code: PlatformError, Desc: err.Error()}
	}

	// Request per-monitor DPI awareness (Windows 8.1+). Ignore errors on
	// older systems where shcore.dll is absent.
	setProcessDpiAwareness(_PROCESS_PER_MONITOR_DPI_AWARE)

	// Load opengl32.dll so WGL calls are available from the start.
	if err := loadOpenGL32(); err != nil {
		return err
	}

	// Pre-load the default arrow cursor once.
	gDefaultCursor, _ = loadCursorW(0, _IDC_ARROW)

	// Register window class.
	gWndProcCB = syscall.NewCallback(wndProc)

	className, _ := syscall.UTF16PtrFromString(_wndClassName)
	icon, _ := loadIconW(0, _IDI_APPLICATION)

	wc := _WNDCLASSEXW{
		Style:         _CS_HREDRAW | _CS_VREDRAW | _CS_OWNDC,
		LpfnWndProc:   gWndProcCB,
		HInstance:     gHInstance,
		HCursor:       gDefaultCursor,
		HIcon:         icon,
		LpszClassName: className,
	}
	wc.CbSize = uint32(unsafe.Sizeof(wc))

	atom, err := registerClassExW(&wc)
	if err != nil {
		return &Error{Code: PlatformError, Desc: err.Error()}
	}
	gClassAtom = atom
	resetHints()
	return nil
}

// Terminate destroys all windows and unregisters the window class.
func Terminate() {
	windowByHandle.Range(func(k, v any) bool {
		w := v.(*Window)
		destroyWindowPlatform(w)
		return true
	})

	if gClassAtom != 0 {
		className, _ := syscall.UTF16PtrFromString(_wndClassName)
		unregisterClassW(className, gHInstance)
		gClassAtom = 0
	}
}

// GetTime returns the elapsed time in seconds since Init was called, or
// since the last SetTime — whichever happened most recently.
//
// timeGetTicks returns the count delta since the base; we add timerBaseT
// (the value last passed to SetTime) so the seconds returned match the
// expectation that GetTime() right after SetTime(N) is approximately N.
func GetTime() float64 {
	return float64(timeGetTicks())/float64(timeFrequency()) + timeBaseT()
}

// SetTime resets the timer base so GetTime returns t immediately after.
func SetTime(t float64) {
	timeSetBase(t)
}

// _wndClassName is the Win32 class name used for all glfw-purego windows.
// Matches the string used by the reference GLFW C implementation.
const _wndClassName = "GLFW30"

// ── Version / init hints ──────────────────────────────────────────────────────

// InitHint sets a hint for the next Init call.
// Stub — hint storage is not needed in the purego implementation.
func InitHint(hint Hint, value int) {}

// GetVersion returns the compile-time version of the GLFW library.
func GetVersion() (major, minor, revision int) { return 3, 3, 0 }

// GetVersionString returns a human-readable version string.
func GetVersionString() string { return "3.3.0 purego" }

// RawMouseMotionSupported reports whether raw (unscaled, unaccelerated) mouse
// motion is supported on the current platform.
// On Windows, WM_INPUT-based raw mouse motion is always available.
func RawMouseMotionSupported() bool { return true }

// WindowHintString sets a string-valued window or context creation hint.
// Stub — no string hints are used in the purego implementation.
func WindowHintString(hint Hint, value string) {}
