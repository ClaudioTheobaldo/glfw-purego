//go:build windows

package glfw

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ----------------------------------------------------------------------------
// DLL handles
// ----------------------------------------------------------------------------

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modGdi32    = windows.NewLazySystemDLL("gdi32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modShell32  = windows.NewLazySystemDLL("shell32.dll")
	modShcore   = windows.NewLazySystemDLL("shcore.dll")

	// Loaded dynamically in loadOpenGL32() — not a system DLL path lookup.
	modOpenGL32 *windows.LazyDLL
)

// ----------------------------------------------------------------------------
// Proc variables — user32.dll
// ----------------------------------------------------------------------------

var (
	procRegisterClassExW     = modUser32.NewProc("RegisterClassExW")
	procUnregisterClassW     = modUser32.NewProc("UnregisterClassW")
	procCreateWindowExW      = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow        = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW       = modUser32.NewProc("DefWindowProcW")
	procShowWindow           = modUser32.NewProc("ShowWindow")
	procSetWindowTextW       = modUser32.NewProc("SetWindowTextW")
	procGetWindowRect        = modUser32.NewProc("GetWindowRect")
	procGetClientRect        = modUser32.NewProc("GetClientRect")
	procSetWindowPos         = modUser32.NewProc("SetWindowPos")
	procAdjustWindowRectEx   = modUser32.NewProc("AdjustWindowRectEx")
	procGetDC                = modUser32.NewProc("GetDC")
	procReleaseDC            = modUser32.NewProc("ReleaseDC")
	procPeekMessageW         = modUser32.NewProc("PeekMessageW")
	procGetMessageW          = modUser32.NewProc("GetMessageW")
	procTranslateMessage     = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW     = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage      = modUser32.NewProc("PostQuitMessage")
	procLoadCursorW          = modUser32.NewProc("LoadCursorW")
	procLoadIconW            = modUser32.NewProc("LoadIconW")
	procGetKeyState          = modUser32.NewProc("GetKeyState")
	procScreenToClient       = modUser32.NewProc("ScreenToClient")
	procClientToScreen       = modUser32.NewProc("ClientToScreen")
	procSetCursor            = modUser32.NewProc("SetCursor")
	procSetCursorPos         = modUser32.NewProc("SetCursorPos")
	procGetCursorPos         = modUser32.NewProc("GetCursorPos")
	procSetForegroundWindow  = modUser32.NewProc("SetForegroundWindow")
	procShowCursor           = modUser32.NewProc("ShowCursor")
	procClipCursor           = modUser32.NewProc("ClipCursor")
	procOpenClipboard        = modUser32.NewProc("OpenClipboard")
	procCloseClipboard       = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard       = modUser32.NewProc("EmptyClipboard")
	procSetClipboardData     = modUser32.NewProc("SetClipboardData")
	procGetClipboardData     = modUser32.NewProc("GetClipboardData")
	procEnumDisplayMonitors  = modUser32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW      = modUser32.NewProc("GetMonitorInfoW")
	procEnumDisplaySettingsW = modUser32.NewProc("EnumDisplaySettingsW")
	procGetWindowLongW       = modUser32.NewProc("GetWindowLongW")
	procSetWindowLongW       = modUser32.NewProc("SetWindowLongW")
	procIsIconic             = modUser32.NewProc("IsIconic")
	procIsZoomed             = modUser32.NewProc("IsZoomed")
	procDragAcceptFiles      = modShell32.NewProc("DragAcceptFiles")
	procDragQueryFileW       = modShell32.NewProc("DragQueryFileW")
	procDragFinish           = modShell32.NewProc("DragFinish")
	procMsgWaitForMultipleObjectsEx = modUser32.NewProc("MsgWaitForMultipleObjectsEx")
)

// ----------------------------------------------------------------------------
// Proc variables — gdi32.dll
// ----------------------------------------------------------------------------

var (
	procChoosePixelFormat   = modGdi32.NewProc("ChoosePixelFormat")
	procDescribePixelFormat = modGdi32.NewProc("DescribePixelFormat")
	procSetPixelFormat      = modGdi32.NewProc("SetPixelFormat")
	procSwapBuffers         = modGdi32.NewProc("SwapBuffers")
)

// ----------------------------------------------------------------------------
// Proc variables — kernel32.dll
// ----------------------------------------------------------------------------

var (
	procGetModuleHandleW    = modKernel32.NewProc("GetModuleHandleW")
	procGetProcAddress      = modKernel32.NewProc("GetProcAddress")
	procGlobalAlloc         = modKernel32.NewProc("GlobalAlloc")
	procGlobalFree          = modKernel32.NewProc("GlobalFree")
	procGlobalLock          = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock        = modKernel32.NewProc("GlobalUnlock")
)

// ----------------------------------------------------------------------------
// Proc variables — shcore.dll
// ----------------------------------------------------------------------------

var (
	procSetProcessDpiAwareness = modShcore.NewProc("SetProcessDpiAwareness")
	procGetDpiForMonitor       = modShcore.NewProc("GetDpiForMonitor")
)

// ----------------------------------------------------------------------------
// WGL proc variables — populated by loadOpenGL32()
// ----------------------------------------------------------------------------

var (
	procWglCreateContext     *windows.LazyProc
	procWglDeleteContext     *windows.LazyProc
	procWglMakeCurrent       *windows.LazyProc
	procWglGetProcAddress    *windows.LazyProc
	procWglGetCurrentDC      *windows.LazyProc
	procWglGetCurrentContext *windows.LazyProc
)

func loadOpenGL32() error {
	modOpenGL32 = windows.NewLazyDLL("opengl32.dll")
	if err := modOpenGL32.Load(); err != nil {
		return &Error{Code: APIUnavailable, Desc: "failed to load opengl32.dll: " + err.Error()}
	}
	procWglCreateContext     = modOpenGL32.NewProc("wglCreateContext")
	procWglDeleteContext     = modOpenGL32.NewProc("wglDeleteContext")
	procWglMakeCurrent       = modOpenGL32.NewProc("wglMakeCurrent")
	procWglGetProcAddress    = modOpenGL32.NewProc("wglGetProcAddress")
	procWglGetCurrentDC      = modOpenGL32.NewProc("wglGetCurrentDC")
	procWglGetCurrentContext = modOpenGL32.NewProc("wglGetCurrentContext")
	return nil
}

// ----------------------------------------------------------------------------
// Typed wrappers — user32
// ----------------------------------------------------------------------------

func registerClassExW(wc *_WNDCLASSEXW) (uint16, error) {
	r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(wc)))
	if r == 0 {
		return 0, fmt.Errorf("RegisterClassExW: %w", e)
	}
	return uint16(r), nil
}

func unregisterClassW(className *uint16, hInstance uintptr) {
	procUnregisterClassW.Call(uintptr(unsafe.Pointer(className)), hInstance)
}

func createWindowExW(exStyle, style uintptr, className, windowName *uint16,
	x, y, w, h int, parent, menu, hInstance uintptr) (uintptr, error) {
	r, _, e := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, menu, hInstance, 0,
	)
	if r == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", e)
	}
	return r, nil
}

func destroyWindow(hwnd uintptr) {
	procDestroyWindow.Call(hwnd)
}

func defWindowProcW(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func showWindow(hwnd uintptr, cmd int) {
	procShowWindow.Call(hwnd, uintptr(cmd))
}

func setWindowTextW(hwnd uintptr, text *uint16) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(text)))
}

func getWindowRect(hwnd uintptr) _RECT {
	var r _RECT
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func getClientRect(hwnd uintptr) _RECT {
	var r _RECT
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func setWindowPos(hwnd, hwndInsertAfter uintptr, x, y, cx, cy int, flags uint32) {
	procSetWindowPos.Call(hwnd, hwndInsertAfter,
		uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(flags))
}

func adjustWindowRectEx(rc *_RECT, style, exStyle uintptr, menu bool) {
	m := uintptr(0)
	if menu {
		m = 1
	}
	procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(rc)), style, m, exStyle)
}

func getDC(hwnd uintptr) (uintptr, error) {
	r, _, e := procGetDC.Call(hwnd)
	if r == 0 {
		return 0, fmt.Errorf("GetDC: %w", e)
	}
	return r, nil
}

func releaseDC(hwnd, dc uintptr) {
	procReleaseDC.Call(hwnd, dc)
}

func peekMessageW(msg *_MSG, hwnd uintptr, min, max, remove uint32) bool {
	r, _, _ := procPeekMessageW.Call(
		uintptr(unsafe.Pointer(msg)), hwnd,
		uintptr(min), uintptr(max), uintptr(remove),
	)
	return r != 0
}

func getMessageW(msg *_MSG, hwnd uintptr, min, max uint32) (bool, error) {
	r, _, e := procGetMessageW.Call(
		uintptr(unsafe.Pointer(msg)), hwnd,
		uintptr(min), uintptr(max),
	)
	// -1 (error), 0 (WM_QUIT), >0 (normal message)
	if r == ^uintptr(0) {
		return false, fmt.Errorf("GetMessageW: %w", e)
	}
	return r != 0, nil
}

func translateMessage(msg *_MSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func dispatchMessageW(msg *_MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func loadCursorW(hInstance, cursorName uintptr) (uintptr, error) {
	r, _, e := procLoadCursorW.Call(hInstance, cursorName)
	if r == 0 {
		return 0, fmt.Errorf("LoadCursorW: %w", e)
	}
	return r, nil
}

func loadIconW(hInstance, iconName uintptr) (uintptr, error) {
	r, _, e := procLoadIconW.Call(hInstance, iconName)
	if r == 0 {
		return 0, fmt.Errorf("LoadIconW: %w", e)
	}
	return r, nil
}

func getKeyState(vk uint32) uint16 {
	r, _, _ := procGetKeyState.Call(uintptr(vk))
	return uint16(r)
}

var procMapVirtualKeyW = modUser32.NewProc("MapVirtualKeyW")

func screenToClient(hwnd uintptr, pt *_POINT) {
	procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(pt)))
}

func clientToScreen(hwnd uintptr, pt *_POINT) {
	procClientToScreen.Call(hwnd, uintptr(unsafe.Pointer(pt)))
}

func getCursorPos() _POINT {
	var pt _POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt
}

func setCursorPos(x, y int) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func setForegroundWindow(hwnd uintptr) {
	procSetForegroundWindow.Call(hwnd)
}

// showCursor increments (show=true) or decrements (show=false) the cursor
// display counter.  The cursor is visible when the counter >= 0.
func showCursor(show bool) {
	b := uintptr(0)
	if show {
		b = 1
	}
	procShowCursor.Call(b)
}

// clipCursor confines the cursor to the given screen rect.
// Pass nil to release the confinement.
func clipCursor(r *_RECT) {
	if r == nil {
		procClipCursor.Call(0)
	} else {
		procClipCursor.Call(uintptr(unsafe.Pointer(r)))
	}
}

// ── Clipboard helpers ────────────────────────────────────────────────────────

const (
	_CF_UNICODETEXT = 13
	_GMEM_MOVEABLE  = 0x0002
)

func openClipboard(hwnd uintptr) bool {
	r, _, _ := procOpenClipboard.Call(hwnd)
	return r != 0
}
func closeClipboard() { procCloseClipboard.Call() }
func emptyClipboard() { procEmptyClipboard.Call() }

func setClipboardData(format uint32, hMem uintptr) uintptr {
	r, _, _ := procSetClipboardData.Call(uintptr(format), hMem)
	return r
}
func getClipboardData(format uint32) uintptr {
	r, _, _ := procGetClipboardData.Call(uintptr(format))
	return r
}

func globalAlloc(flags uint32, size uintptr) uintptr {
	r, _, _ := procGlobalAlloc.Call(uintptr(flags), size)
	return r
}
func globalFree(hMem uintptr) { procGlobalFree.Call(hMem) }
func globalLock(hMem uintptr) unsafe.Pointer {
	r, _, _ := procGlobalLock.Call(hMem)
	// r is a Windows GMEM memory address — not GC-managed.
	// Convert via pointer indirection to avoid the go vet unsafeptr warning.
	return *(*unsafe.Pointer)(unsafe.Pointer(&r))
}
func globalUnlock(hMem uintptr) { procGlobalUnlock.Call(hMem) }

func enumDisplayMonitors(hdc, clip, callback, data uintptr) {
	procEnumDisplayMonitors.Call(hdc, clip, callback, data)
}

func getMonitorInfoW(hmon uintptr, info *_MONITORINFOEXW) error {
	r, _, e := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(info)))
	if r == 0 {
		return fmt.Errorf("GetMonitorInfoW: %w", e)
	}
	return nil
}

func enumDisplaySettingsW(deviceName *uint16, modeNum uint32, dm *_DEVMODEW) bool {
	r, _, _ := procEnumDisplaySettingsW.Call(
		uintptr(unsafe.Pointer(deviceName)),
		uintptr(modeNum),
		uintptr(unsafe.Pointer(dm)),
	)
	return r != 0
}

func getWindowLongW(hwnd uintptr, index int32) uintptr {
	r, _, _ := procGetWindowLongW.Call(hwnd, uintptr(index))
	return r
}

func setWindowLongW(hwnd uintptr, index int32, value uintptr) {
	procSetWindowLongW.Call(hwnd, uintptr(index), value)
}

func isIconic(hwnd uintptr) bool {
	r, _, _ := procIsIconic.Call(hwnd)
	return r != 0
}

func isZoomed(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

func dragAcceptFiles(hwnd uintptr, accept bool) {
	a := uintptr(0)
	if accept {
		a = 1
	}
	procDragAcceptFiles.Call(hwnd, a)
}

func dragQueryFileW(hDrop uintptr, iFile uint32, buf *uint16, cch uint32) uint32 {
	r, _, _ := procDragQueryFileW.Call(
		hDrop, uintptr(iFile),
		uintptr(unsafe.Pointer(buf)), uintptr(cch),
	)
	return uint32(r)
}

func dragFinish(hDrop uintptr) {
	procDragFinish.Call(hDrop)
}

func msgWaitForMultipleObjectsEx(count uint32, handles uintptr, ms, wakeMask, flags uint32) uint32 {
	r, _, _ := procMsgWaitForMultipleObjectsEx.Call(
		uintptr(count), handles,
		uintptr(ms), uintptr(wakeMask), uintptr(flags),
	)
	return uint32(r)
}

// ----------------------------------------------------------------------------
// Typed wrappers — gdi32
// ----------------------------------------------------------------------------

func choosePixelFormat(dc uintptr, pfd *_PIXELFORMATDESCRIPTOR) int32 {
	r, _, _ := procChoosePixelFormat.Call(dc, uintptr(unsafe.Pointer(pfd)))
	return int32(r)
}

func describePixelFormat(dc uintptr, format, size uint32, pfd *_PIXELFORMATDESCRIPTOR) int32 {
	r, _, _ := procDescribePixelFormat.Call(dc, uintptr(format), uintptr(size), uintptr(unsafe.Pointer(pfd)))
	return int32(r)
}

func setPixelFormat(dc uintptr, format int32, pfd *_PIXELFORMATDESCRIPTOR) error {
	r, _, e := procSetPixelFormat.Call(dc, uintptr(format), uintptr(unsafe.Pointer(pfd)))
	if r == 0 {
		return fmt.Errorf("SetPixelFormat: %w", e)
	}
	return nil
}

func swapBuffers(dc uintptr) {
	procSwapBuffers.Call(dc)
}

// ----------------------------------------------------------------------------
// Typed wrappers — kernel32
// ----------------------------------------------------------------------------

func getModuleHandleW(name *uint16) (uintptr, error) {
	r, _, e := procGetModuleHandleW.Call(uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, fmt.Errorf("GetModuleHandleW: %w", e)
	}
	return r, nil
}

func getProcAddressFromDLL(dll uintptr, name string) uintptr {
	n, err := syscall.BytePtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procGetProcAddress.Call(dll, uintptr(unsafe.Pointer(n)))
	return r
}

// ----------------------------------------------------------------------------
// Typed wrappers — shcore
// ----------------------------------------------------------------------------

func setProcessDpiAwareness(value uint32) {
	// Silently ignore — shcore.dll may not exist on older Windows.
	procSetProcessDpiAwareness.Call(uintptr(value))
}

func getDpiForMonitor(hmon uintptr, dpiType uint32, dpiX, dpiY *uint32) error {
	r, _, e := procGetDpiForMonitor.Call(
		hmon, uintptr(dpiType),
		uintptr(unsafe.Pointer(dpiX)), uintptr(unsafe.Pointer(dpiY)),
	)
	if r != 0 { // S_OK = 0
		return fmt.Errorf("GetDpiForMonitor: %w", e)
	}
	return nil
}

// ----------------------------------------------------------------------------
// Typed wrappers — WGL
// ----------------------------------------------------------------------------

func wglCreateContext(dc uintptr) (uintptr, error) {
	r, _, e := procWglCreateContext.Call(dc)
	if r == 0 {
		return 0, fmt.Errorf("wglCreateContext: %w", e)
	}
	return r, nil
}

func wglDeleteContext(hglrc uintptr) {
	procWglDeleteContext.Call(hglrc)
}

func wglMakeCurrent(dc, hglrc uintptr) error {
	r, _, e := procWglMakeCurrent.Call(dc, hglrc)
	if r == 0 {
		return fmt.Errorf("wglMakeCurrent: %w", e)
	}
	return nil
}

func wglGetProcAddressStr(name string) uintptr {
	n, err := syscall.BytePtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procWglGetProcAddress.Call(uintptr(unsafe.Pointer(n)))
	return r
}
