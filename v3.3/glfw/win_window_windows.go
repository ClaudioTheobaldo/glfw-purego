//go:build windows

package glfw

import (
	"syscall"
)

// ----------------------------------------------------------------------------
// CreateWindow
// ----------------------------------------------------------------------------

// CreateWindow creates a new window with an associated OpenGL context.
//
// monitor and share are accepted for API compatibility but ignored in this
// initial Windows implementation (fullscreen and context sharing are not yet
// implemented).
func CreateWindow(width, height int, title string, monitor, share *Monitor) (*Window, error) {
	// Snapshot the current hints so concurrent hint changes don't race.
	hints.mu.Lock()
	h := make(map[Hint]int, len(hints.m))
	for k, v := range hints.m {
		h[k] = v
	}
	hints.mu.Unlock()

	style, exStyle := buildWindowStyle(h)

	// Inflate client-area size to include the non-client frame.
	rc := _RECT{0, 0, int32(width), int32(height)}
	adjustWindowRectEx(&rc, style, exStyle, false)
	adjW := int(rc.Right - rc.Left)
	adjH := int(rc.Bottom - rc.Top)

	className, _ := syscall.UTF16PtrFromString(_wndClassName)
	titleUTF16, _ := syscall.UTF16PtrFromString(title)

	hwnd, err := createWindowExW(exStyle, style, className, titleUTF16,
		_CW_USEDEFAULT, _CW_USEDEFAULT, adjW, adjH,
		0, 0, gHInstance)
	if err != nil {
		return nil, &Error{Code: PlatformError, Desc: err.Error()}
	}

	dc, err := getDC(hwnd)
	if err != nil {
		destroyWindow(hwnd)
		return nil, &Error{Code: PlatformError, Desc: err.Error()}
	}

	hglrc, err := createWGLContext(dc, h)
	if err != nil {
		releaseDC(hwnd, dc)
		destroyWindow(hwnd)
		return nil, err
	}

	w := &Window{handle: hwnd, dc: dc, hglrc: hglrc}
	windowByHandle.Store(hwnd, w)

	// Enable drag-and-drop.
	dragAcceptFiles(hwnd, true)

	if h[Visible] != 0 {
		cmd := _SW_SHOW
		if h[Maximized] != 0 {
			cmd = _SW_MAXIMIZE
		}
		showWindow(hwnd, cmd)
		if h[Focused] != 0 {
			setForegroundWindow(hwnd)
		}
	}

	return w, nil
}

// buildWindowStyle derives WS_* and WS_EX_* styles from the hint snapshot.
func buildWindowStyle(h map[Hint]int) (style, exStyle uintptr) {
	if h[Decorated] != 0 {
		style = _WS_CAPTION | _WS_SYSMENU | _WS_MINIMIZEBOX | _WS_CLIPSIBLINGS | _WS_CLIPCHILDREN
		if h[Resizable] != 0 {
			style |= _WS_THICKFRAME | _WS_MAXIMIZEBOX
		}
	} else {
		style = _WS_POPUP | _WS_CLIPSIBLINGS | _WS_CLIPCHILDREN
	}
	exStyle = _WS_EX_APPWINDOW | _WS_EX_ACCEPTFILES
	if h[Floating] != 0 {
		exStyle |= _WS_EX_TOPMOST
	}
	return
}

// ----------------------------------------------------------------------------
// Window destruction
// ----------------------------------------------------------------------------

// destroyWindowPlatform releases the DC, WGL context, and Win32 window.
func destroyWindowPlatform(w *Window) {
	windowByHandle.Delete(w.handle)
	if w.hglrc != 0 {
		wglMakeCurrent(0, 0)
		wglDeleteContext(w.hglrc)
		w.hglrc = 0
	}
	if w.dc != 0 {
		releaseDC(w.handle, w.dc)
		w.dc = 0
	}
	if w.handle != 0 {
		destroyWindow(w.handle)
		w.handle = 0
	}
}

// Destroy releases all resources associated with this window.
func (w *Window) Destroy() {
	destroyWindowPlatform(w)
}

// ----------------------------------------------------------------------------
// Context methods
// ----------------------------------------------------------------------------

// MakeContextCurrent makes the window's OpenGL context current on the
// calling goroutine's OS thread.
func (w *Window) MakeContextCurrent() {
	wglMakeCurrent(w.dc, w.hglrc)
}

// SwapBuffers swaps the front and back buffers.
func (w *Window) SwapBuffers() {
	swapBuffers(w.dc)
}

// DetachCurrentContext detaches the current GL context from the calling thread.
func DetachCurrentContext() {
	wglMakeCurrent(0, 0)
}

// GetCurrentContext returns the window whose context is current on this thread,
// or nil if none.
func GetCurrentContext() *Window {
	// wglGetCurrentDC is loaded but not used here; instead we scan the map.
	// This is called infrequently so the linear scan is acceptable.
	var found *Window
	windowByHandle.Range(func(k, v any) bool {
		w := v.(*Window)
		if w.dc != 0 && w.hglrc != 0 {
			// Check if this window's context is the current one.
			r, _, _ := procWglGetCurrentContext.Call()
			if r == w.hglrc {
				found = w
				return false
			}
		}
		return true
	})
	return found
}

// ----------------------------------------------------------------------------
// Window geometry
// ----------------------------------------------------------------------------

// GetSize returns the size of the window's client area in screen coordinates.
func (w *Window) GetSize() (width, height int) {
	rc := getClientRect(w.handle)
	return int(rc.Right - rc.Left), int(rc.Bottom - rc.Top)
}

// SetSize sets the size of the window's client area.
func (w *Window) SetSize(width, height int) {
	style   := getWindowLongW(w.handle, _GWL_STYLE)
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	rc := _RECT{0, 0, int32(width), int32(height)}
	adjustWindowRectEx(&rc, style, exStyle, false)
	setWindowPos(w.handle, 0,
		0, 0, int(rc.Right-rc.Left), int(rc.Bottom-rc.Top),
		_SWP_NOACTIVATE|_SWP_NOZORDER|_SWP_NOMOVE,
	)
}

// GetPos returns the position of the window's client area in screen coordinates.
func (w *Window) GetPos() (x, y int) {
	var pt _POINT
	clientToScreen(w.handle, &pt)
	return int(pt.X), int(pt.Y)
}

// SetPos sets the position of the window's upper-left corner.
func (w *Window) SetPos(x, y int) {
	style   := getWindowLongW(w.handle, _GWL_STYLE)
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	rc := _RECT{int32(x), int32(y), int32(x), int32(y)}
	adjustWindowRectEx(&rc, style, exStyle, false)
	setWindowPos(w.handle, 0,
		int(rc.Left), int(rc.Top), 0, 0,
		_SWP_NOACTIVATE|_SWP_NOZORDER|_SWP_NOSIZE,
	)
}

// GetFramebufferSize returns the size of the framebuffer in pixels.
// On Windows without HiDPI scaling this equals the client area size.
func (w *Window) GetFramebufferSize() (width, height int) {
	return w.GetSize()
}

// GetContentScale returns the DPI scale factors for the monitor the window is on.
func (w *Window) GetContentScale() (x, y float32) {
	m := getWindowMonitor(w.handle)
	if m == 0 {
		return 1, 1
	}
	var dpiX, dpiY uint32
	if err := getDpiForMonitor(m, _MDT_EFFECTIVE_DPI, &dpiX, &dpiY); err != nil {
		return 1, 1
	}
	return float32(dpiX) / 96.0, float32(dpiY) / 96.0
}

// getWindowMonitor returns the HMONITOR for the monitor that most overlaps hwnd.
func getWindowMonitor(hwnd uintptr) uintptr {
	// MonitorFromWindow is the cleanest API for this.
	procMonitorFromWindow := modUser32.NewProc("MonitorFromWindow")
	const _MONITOR_DEFAULTTONEAREST = 2
	r, _, _ := procMonitorFromWindow.Call(hwnd, _MONITOR_DEFAULTTONEAREST)
	return r
}

// ----------------------------------------------------------------------------
// Window state
// ----------------------------------------------------------------------------

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) {
	t, _ := syscall.UTF16PtrFromString(title)
	setWindowTextW(w.handle, t)
}

// Iconify minimises the window.
func (w *Window) Iconify() { showWindow(w.handle, _SW_MINIMIZE) }

// Restore restores an iconified or maximised window to normal.
func (w *Window) Restore() { showWindow(w.handle, _SW_RESTORE) }

// Maximize maximises the window.
func (w *Window) Maximize() { showWindow(w.handle, _SW_MAXIMIZE) }

// Show makes the window visible.
func (w *Window) Show() { showWindow(w.handle, _SW_SHOW) }

// Hide hides the window.
func (w *Window) Hide() { showWindow(w.handle, _SW_HIDE) }

// Focus brings the window to the foreground.
func (w *Window) Focus() { setForegroundWindow(w.handle) }

// GetAttrib returns the current value of a window attribute.
func (w *Window) GetAttrib(attrib Hint) int {
	switch attrib {
	case Iconified:
		if isIconic(w.handle) { return 1 }
	case Maximized:
		if isZoomed(w.handle) { return 1 }
	case Visible:
		style := getWindowLongW(w.handle, _GWL_STYLE)
		if style&_WS_VISIBLE != 0 { return 1 }
	case Decorated:
		style := getWindowLongW(w.handle, _GWL_STYLE)
		if style&_WS_CAPTION != 0 { return 1 }
	case Resizable:
		style := getWindowLongW(w.handle, _GWL_STYLE)
		if style&_WS_THICKFRAME != 0 { return 1 }
	case Floating:
		exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
		if exStyle&_WS_EX_TOPMOST != 0 { return 1 }
	}
	return 0
}

// GetMonitor returns the monitor the window is currently on, or nil.
func (w *Window) GetMonitor() *Monitor {
	hmon := getWindowMonitor(w.handle)
	if hmon == 0 {
		return nil
	}
	return monitorFromHandle(hmon)
}

// ----------------------------------------------------------------------------
// Input
// ----------------------------------------------------------------------------

// GetKey returns the last known state of the given keyboard key.
func (w *Window) GetKey(key Key) Action {
	vk := keyToVK(key)
	if vk == 0 {
		return Release
	}
	if getKeyState(vk)&0x8000 != 0 {
		return Press
	}
	return Release
}

// GetMouseButton returns the last known state of the given mouse button.
func (w *Window) GetMouseButton(button MouseButton) Action {
	var vk uint32
	switch button {
	case MouseButtonLeft:   vk = 0x01
	case MouseButtonRight:  vk = 0x02
	case MouseButtonMiddle: vk = 0x04
	case MouseButton4:      vk = 0x05
	case MouseButton5:      vk = 0x06
	default:                return Release
	}
	if getKeyState(vk)&0x8000 != 0 {
		return Press
	}
	return Release
}

// GetCursorPos returns the cursor position in client coordinates.
func (w *Window) GetCursorPos() (x, y float64) {
	pt := getCursorPos()
	screenToClient(w.handle, &pt)
	return float64(pt.X), float64(pt.Y)
}

// SetCursorPos moves the cursor to the given client-area position.
func (w *Window) SetCursorPos(x, y float64) {
	pt := _POINT{int32(x), int32(y)}
	clientToScreen(w.handle, &pt)
	setCursorPos(int(pt.X), int(pt.Y))
}

// GetInputMode returns the current value of an input mode.
func (w *Window) GetInputMode(mode InputMode) int {
	// Stub — full raw mouse / sticky keys support is a future addition.
	return 0
}

// SetInputMode sets an input mode on the window.
func (w *Window) SetInputMode(mode InputMode, value int) {
	// Stub — full cursor capture / hide support is a future addition.
}

// ----------------------------------------------------------------------------
// Clipboard
// ----------------------------------------------------------------------------

// GetClipboardString returns the current clipboard contents as a string.
// TODO: implement Win32 GlobalAlloc/GlobalLock clipboard round-trip.
func GetClipboardString() string { return "" }

// SetClipboardString sets the clipboard contents to the given string.
// TODO: implement Win32 GlobalAlloc/GlobalLock clipboard round-trip.
func SetClipboardString(s string) {}

// ----------------------------------------------------------------------------
// WndProc — the central Win32 message dispatcher
// ----------------------------------------------------------------------------

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	v, ok := windowByHandle.Load(hwnd)
	if !ok {
		return defWindowProcW(hwnd, msg, wParam, lParam)
	}
	w := v.(*Window)

	switch msg {
	case _WM_CLOSE:
		w.shouldClose = true
		if w.fCloseHolder != nil {
			w.fCloseHolder(w)
		}
		return 0 // don't call DestroyWindow — caller decides

	case _WM_SIZE:
		width  := int(loword(lParam))
		height := int(hiword(lParam))
		if w.fFramebufferSizeHolder != nil {
			w.fFramebufferSizeHolder(w, width, height)
		}
		if w.fSizeHolder != nil {
			w.fSizeHolder(w, width, height)
		}
		switch wParam {
		case _SIZE_MAXIMIZED:
			if w.fMaximizeHolder != nil { w.fMaximizeHolder(w, true) }
			if w.fIconifyHolder != nil  { w.fIconifyHolder(w, false) }
		case _SIZE_MINIMIZED:
			if w.fIconifyHolder != nil  { w.fIconifyHolder(w, true) }
			if w.fMaximizeHolder != nil { w.fMaximizeHolder(w, false) }
		case _SIZE_RESTORED:
			if w.fMaximizeHolder != nil { w.fMaximizeHolder(w, false) }
			if w.fIconifyHolder != nil  { w.fIconifyHolder(w, false) }
		}
		return 0

	case _WM_MOVE:
		if w.fPosHolder != nil {
			w.fPosHolder(w, getXLParam(lParam), getYLParam(lParam))
		}
		return 0

	case _WM_SETFOCUS:
		if w.fFocusHolder != nil { w.fFocusHolder(w, true) }
		return 0

	case _WM_KILLFOCUS:
		if w.fFocusHolder != nil { w.fFocusHolder(w, false) }
		return 0

	case _WM_KEYDOWN, _WM_SYSKEYDOWN:
		key, scancode := translateVK(uint32(wParam), lParam)
		mods := currentModifiers()
		action := Press
		if lParam&(1<<30) != 0 { // bit 30: previous key-down state
			action = Repeat
		}
		if w.fKeyHolder != nil {
			w.fKeyHolder(w, key, scancode, action, mods)
		}
		if msg == _WM_SYSKEYDOWN {
			return defWindowProcW(hwnd, msg, wParam, lParam)
		}
		return 0

	case _WM_KEYUP, _WM_SYSKEYUP:
		key, scancode := translateVK(uint32(wParam), lParam)
		if w.fKeyHolder != nil {
			w.fKeyHolder(w, key, scancode, Release, currentModifiers())
		}
		_ = scancode
		if msg == _WM_SYSKEYUP {
			return defWindowProcW(hwnd, msg, wParam, lParam)
		}
		return 0

	case _WM_CHAR, _WM_SYSCHAR:
		r := rune(wParam)
		if r < 32 || (r >= 0xD800 && r <= 0xDFFF) {
			break // skip control chars and lone surrogates
		}
		if w.fCharHolder != nil {
			w.fCharHolder(w, r)
		}
		if w.fCharModsHolder != nil {
			w.fCharModsHolder(w, r, currentModifiers())
		}
		return 0

	case _WM_LBUTTONDOWN:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonLeft, Press, currentModifiers())
		}
		return 0
	case _WM_LBUTTONUP:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonLeft, Release, currentModifiers())
		}
		return 0
	case _WM_RBUTTONDOWN:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonRight, Press, currentModifiers())
		}
		return 0
	case _WM_RBUTTONUP:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonRight, Release, currentModifiers())
		}
		return 0
	case _WM_MBUTTONDOWN:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonMiddle, Press, currentModifiers())
		}
		return 0
	case _WM_MBUTTONUP:
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, MouseButtonMiddle, Release, currentModifiers())
		}
		return 0
	case _WM_XBUTTONDOWN:
		btn := MouseButton4
		if getXButton(wParam) == _XBUTTON2 { btn = MouseButton5 }
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, btn, Press, currentModifiers())
		}
		return 0
	case _WM_XBUTTONUP:
		btn := MouseButton4
		if getXButton(wParam) == _XBUTTON2 { btn = MouseButton5 }
		if w.fMouseButtonHolder != nil {
			w.fMouseButtonHolder(w, btn, Release, currentModifiers())
		}
		return 0

	case _WM_MOUSEMOVE:
		if w.fCursorPosHolder != nil {
			w.fCursorPosHolder(w, float64(getXLParam(lParam)), float64(getYLParam(lParam)))
		}
		return 0

	case _WM_MOUSEWHEEL:
		if w.fScrollHolder != nil {
			w.fScrollHolder(w, 0, getWheelDelta(wParam))
		}
		return 0

	case _WM_MOUSEHWHEEL:
		if w.fScrollHolder != nil {
			w.fScrollHolder(w, getWheelDelta(wParam), 0)
		}
		return 0

	case _WM_DROPFILES:
		handleDropFiles(w, wParam)
		return 0

	case _WM_ERASEBKGND:
		return 1 // prevent flicker

	case _WM_PAINT:
		if w.fRefreshHolder != nil { w.fRefreshHolder(w) }
	}

	return defWindowProcW(hwnd, msg, wParam, lParam)
}

// handleDropFiles extracts file paths from a WM_DROPFILES payload.
func handleDropFiles(w *Window, hDrop uintptr) {
	count := dragQueryFileW(hDrop, 0xFFFFFFFF, nil, 0)
	names := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		n := dragQueryFileW(hDrop, i, nil, 0)
		if n == 0 {
			continue
		}
		buf := make([]uint16, n+1)
		dragQueryFileW(hDrop, i, &buf[0], n+1)
		names = append(names, syscall.UTF16ToString(buf))
	}
	dragFinish(hDrop)
	if w.fDropHolder != nil && len(names) > 0 {
		w.fDropHolder(w, names)
	}
}

// ----------------------------------------------------------------------------
// Key translation
// ----------------------------------------------------------------------------

// translateVK maps a Win32 virtual key + lParam flags to a GLFW Key and scancode.
func translateVK(vk uint32, lParam uintptr) (Key, int) {
	scancode := int((lParam >> 16) & 0xFF)
	extended := lParam&(1<<24) != 0

	// Left/right variants that share a base VK.
	switch vk {
	case _VK_SHIFT:
		// Distinguish with MapVirtualKey.
		r, _, _ := procMapVirtualKeyW.Call(uintptr(scancode), 3 /*MAPVK_VSC_TO_VK_EX*/)
		if r == _VK_RSHIFT { return KeyRightShift, scancode }
		return KeyLeftShift, scancode
	case _VK_CONTROL:
		if extended { return KeyRightControl, scancode }
		return KeyLeftControl, scancode
	case _VK_MENU:
		if extended { return KeyRightAlt, scancode }
		return KeyLeftAlt, scancode
	}

	switch vk {
	case _VK_BACK:    return KeyBackspace, scancode
	case _VK_TAB:     return KeyTab, scancode
	case _VK_RETURN:  if extended { return KeyKPEnter, scancode }; return KeyEnter, scancode
	case _VK_PAUSE:   return KeyPause, scancode
	case _VK_CAPITAL: return KeyCapsLock, scancode
	case _VK_ESCAPE:  return KeyEscape, scancode
	case _VK_SPACE:   return KeySpace, scancode
	case _VK_PRIOR:   if extended { return KeyPageUp, scancode };    return KeyKP9, scancode
	case _VK_NEXT:    if extended { return KeyPageDown, scancode };  return KeyKP3, scancode
	case _VK_END:     if extended { return KeyEnd, scancode };       return KeyKP1, scancode
	case _VK_HOME:    if extended { return KeyHome, scancode };      return KeyKP7, scancode
	case _VK_LEFT:    if extended { return KeyLeft, scancode };      return KeyKP4, scancode
	case _VK_UP:      if extended { return KeyUp, scancode };        return KeyKP8, scancode
	case _VK_RIGHT:   if extended { return KeyRight, scancode };     return KeyKP6, scancode
	case _VK_DOWN:    if extended { return KeyDown, scancode };      return KeyKP2, scancode
	case _VK_SNAPSHOT: return KeyPrintScreen, scancode
	case _VK_INSERT:  if extended { return KeyInsert, scancode };    return KeyKP0, scancode
	case _VK_DELETE:  if extended { return KeyDelete, scancode };    return KeyKPDecimal, scancode
	case _VK_LWIN:    return KeyLeftSuper, scancode
	case _VK_RWIN:    return KeyRightSuper, scancode
	case _VK_APPS:    return KeyMenu, scancode
	case _VK_MULTIPLY: return KeyKPMultiply, scancode
	case _VK_ADD:     return KeyKPAdd, scancode
	case _VK_SUBTRACT: return KeyKPSubtract, scancode
	case _VK_DECIMAL: return KeyKPDecimal, scancode
	case _VK_DIVIDE:  if extended { return KeyKPDivide, scancode }; return KeyKPDivide, scancode
	case _VK_NUMLOCK: return KeyNumLock, scancode
	case _VK_SCROLL:  return KeyScrollLock, scancode
	case _VK_LSHIFT:  return KeyLeftShift, scancode
	case _VK_RSHIFT:  return KeyRightShift, scancode
	case _VK_LCONTROL: return KeyLeftControl, scancode
	case _VK_RCONTROL: return KeyRightControl, scancode
	case _VK_LMENU:   return KeyLeftAlt, scancode
	case _VK_RMENU:   return KeyRightAlt, scancode
	case _VK_OEM_1:   return KeySemicolon, scancode
	case _VK_OEM_PLUS:  return KeyEqual, scancode
	case _VK_OEM_COMMA: return KeyComma, scancode
	case _VK_OEM_MINUS: return KeyMinus, scancode
	case _VK_OEM_PERIOD: return KeyPeriod, scancode
	case _VK_OEM_2:   return KeySlash, scancode
	case _VK_OEM_3:   return KeyGraveAccent, scancode
	case _VK_OEM_4:   return KeyLeftBracket, scancode
	case _VK_OEM_5:   return KeyBackslash, scancode
	case _VK_OEM_6:   return KeyRightBracket, scancode
	case _VK_OEM_7:   return KeyApostrophe, scancode
	}

	// Digit keys 0–9
	if vk >= 0x30 && vk <= 0x39 {
		return Key(vk), scancode
	}
	// Letter keys A–Z
	if vk >= 0x41 && vk <= 0x5A {
		return Key(vk), scancode
	}
	// Numpad 0–9
	if vk >= _VK_NUMPAD0 && vk <= _VK_NUMPAD9 {
		return Key(KeyKP0 + Key(vk-_VK_NUMPAD0)), scancode
	}
	// F1–F25
	if vk >= _VK_F1 && vk <= _VK_F25 {
		return Key(KeyF1 + Key(vk-_VK_F1)), scancode
	}

	return KeyUnknown, scancode
}

// keyToVK maps a GLFW Key back to a Win32 virtual key for GetKey polling.
func keyToVK(k Key) uint32 {
	switch {
	case k >= Key0 && k <= Key9:   return uint32(k)
	case k >= KeyA && k <= KeyZ:   return uint32(k)
	case k >= KeyF1 && k <= KeyF25: return uint32(_VK_F1 + (k - KeyF1))
	case k >= KeyKP0 && k <= KeyKP9: return uint32(_VK_NUMPAD0 + (k - KeyKP0))
	}
	switch k {
	case KeySpace:     return _VK_SPACE
	case KeyEscape:    return _VK_ESCAPE
	case KeyEnter:     return _VK_RETURN
	case KeyTab:       return _VK_TAB
	case KeyBackspace: return _VK_BACK
	case KeyInsert:    return _VK_INSERT
	case KeyDelete:    return _VK_DELETE
	case KeyRight:     return _VK_RIGHT
	case KeyLeft:      return _VK_LEFT
	case KeyDown:      return _VK_DOWN
	case KeyUp:        return _VK_UP
	case KeyPageUp:    return _VK_PRIOR
	case KeyPageDown:  return _VK_NEXT
	case KeyHome:      return _VK_HOME
	case KeyEnd:       return _VK_END
	case KeyCapsLock:  return _VK_CAPITAL
	case KeyScrollLock: return _VK_SCROLL
	case KeyNumLock:   return _VK_NUMLOCK
	case KeyPause:     return _VK_PAUSE
	case KeyLeftShift: return _VK_LSHIFT
	case KeyRightShift: return _VK_RSHIFT
	case KeyLeftControl: return _VK_LCONTROL
	case KeyRightControl: return _VK_RCONTROL
	case KeyLeftAlt:   return _VK_LMENU
	case KeyRightAlt:  return _VK_RMENU
	case KeyLeftSuper: return _VK_LWIN
	case KeyRightSuper: return _VK_RWIN
	case KeyMenu:      return _VK_APPS
	}
	return 0
}

// currentModifiers returns the modifier key bitmask from current key state.
// GetKeyState high bit (bit 15) = key is down. As uint16 that's 0x8000.
func currentModifiers() ModifierKey {
	var m ModifierKey
	if getKeyState(_VK_SHIFT)   &0x8000 != 0 { m |= ModShift }
	if getKeyState(_VK_CONTROL) &0x8000 != 0 { m |= ModControl }
	if getKeyState(_VK_MENU)    &0x8000 != 0 { m |= ModAlt }
	if getKeyState(_VK_LWIN)    &0x8000 != 0 { m |= ModSuper }
	if getKeyState(_VK_RWIN)    &0x8000 != 0 { m |= ModSuper }
	return m
}

// pressed reports whether a virtual key's GetKeyState high bit is set.
func pressed(vk uint32) bool { return getKeyState(vk)&0x8000 != 0 }
