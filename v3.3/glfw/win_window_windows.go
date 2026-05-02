//go:build windows

package glfw

import (
	"syscall"
	"unsafe"
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

	var w *Window

	if shouldUseEGL(h) {
		// EGL path — used for OpenGL ES (via ANGLE on Windows).
		surf, ctx, eglErr := createEGLContext(hwnd, h)
		if eglErr != nil {
			destroyWindow(hwnd)
			return nil, eglErr
		}
		w = &Window{handle: hwnd, eglSurface: surf, eglContext: ctx, useEGL: true}
	} else {
		// WGL path — used for desktop OpenGL (default).
		dc, dcErr := getDC(hwnd)
		if dcErr != nil {
			destroyWindow(hwnd)
			return nil, &Error{Code: PlatformError, Desc: dcErr.Error()}
		}
		hglrc, ctxErr := createWGLContext(dc, h)
		if ctxErr != nil {
			releaseDC(hwnd, dc)
			destroyWindow(hwnd)
			return nil, ctxErr
		}
		w = &Window{handle: hwnd, dc: dc, hglrc: hglrc}
	}
	w.title = title
	windowByHandle.Store(hwnd, w)

	// Record a window handle for PostEmptyEvent (first window wins).
	if gPostHwnd == 0 {
		gPostHwnd = hwnd
	}

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

// destroyWindowPlatform releases the context (WGL or EGL), DC, and Win32 window.
func destroyWindowPlatform(w *Window) {
	windowByHandle.Delete(w.handle)
	if w.useEGL {
		eglDestroyWindow(w)
	} else {
		if w.hglrc != 0 {
			wglMakeCurrent(0, 0)
			wglDeleteContext(w.hglrc)
			w.hglrc = 0
		}
		if w.dc != 0 {
			releaseDC(w.handle, w.dc)
			w.dc = 0
		}
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

// MakeContextCurrent makes the window's OpenGL (or OpenGL ES) context current
// on the calling goroutine's OS thread.
func (w *Window) MakeContextCurrent() {
	if w.useEGL {
		eglMakeCurrentWindow(w)
	} else {
		wglMakeCurrent(w.dc, w.hglrc)
	}
}

// SwapBuffers swaps the front and back buffers.
func (w *Window) SwapBuffers() {
	if w.useEGL {
		eglSwapBuffersWindow(w)
	} else {
		swapBuffers(w.dc)
	}
}

// DetachCurrentContext detaches the current GL (or GLES) context from the
// calling thread.
func DetachCurrentContext() {
	if eglLibLoaded && currentEGLDisplay != 0 {
		procEGLMakeCurrent.Call(currentEGLDisplay, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT)
		currentEGLDisplay = 0
		return
	}
	wglMakeCurrent(0, 0)
}

// GetCurrentContext returns the window whose context is current on this thread,
// or nil if none.
func GetCurrentContext() *Window {
	var found *Window
	windowByHandle.Range(func(k, v any) bool {
		w := v.(*Window)
		if w.useEGL {
			if eglLibLoaded {
				cur, _, _ := procEGLGetCurrentContext.Call()
				if cur != 0 && cur == w.eglContext {
					found = w
					return false
				}
			}
		} else if w.dc != 0 && w.hglrc != 0 {
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

// GetFrameSize returns the size, in screen coordinates, of each edge of the
// frame around the window's client area.
func (w *Window) GetFrameSize() (left, top, right, bottom int) {
	wr := getWindowRect(w.handle) // outer window rect in screen coords
	cr := getClientRect(w.handle) // client rect (always starts at 0,0)
	pt := _POINT{0, 0}
	clientToScreen(w.handle, &pt) // client origin in screen coords
	left   = int(pt.X - wr.Left)
	top    = int(pt.Y - wr.Top)
	right  = int(wr.Right - (pt.X + cr.Right))
	bottom = int(wr.Bottom - (pt.Y + cr.Bottom))
	return
}

// GetWindowFrameSize is a package-level wrapper around (*Window).GetFrameSize.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) {
	return w.GetFrameSize()
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
	w.title = title
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

// SetMonitor switches the window to fullscreen on the given monitor, or back to
// windowed mode when monitor is nil.
//
// When going fullscreen: xpos/ypos are ignored; width/height set the desired
// framebuffer resolution; refreshRate (or -1 for current) sets the display mode.
//
// When going windowed: xpos/ypos set the window position; width/height set the
// client area size.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if monitor != nil {
		// Save current windowed state before first fullscreen transition.
		if w.fsMonitor == nil {
			w.savedX, w.savedY = w.GetPos()
			w.savedW, w.savedH = w.GetSize()
			w.savedStyle   = getWindowLongW(w.handle, _GWL_STYLE)
			w.savedExStyle = getWindowLongW(w.handle, _GWL_EXSTYLE)
		}

		// Change display mode if a specific resolution or refresh rate is requested.
		if width > 0 && height > 0 {
			var dm _DEVMODEW
			dm.DmSize   = uint16(unsafe.Sizeof(dm))
			dm.DmFields = _DM_PELSWIDTH | _DM_PELSHEIGHT | _DM_BITSPERPEL
			dm.DmPelsWidth  = uint32(width)
			dm.DmPelsHeight = uint32(height)
			dm.DmBitsPerPel = 32
			if refreshRate > 0 {
				dm.DmFields |= _DM_DISPLAYFREQUENCY
				dm.DmDisplayFrequency = uint32(refreshRate)
			}
			devName, _ := syscall.UTF16PtrFromString(monitor.name)
			changeDisplaySettingsExW(devName, &dm, _CDS_FULLSCREEN)
		}

		// Get the monitor work rectangle (full screen area).
		var mi _MONITORINFOEXW
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		getMonitorInfoW(monitor.hmon, &mi)
		mr := mi.RcMonitor

		// Switch to borderless popup style.
		newStyle   := _WS_POPUP | _WS_CLIPSIBLINGS | _WS_CLIPCHILDREN
		newExStyle := _WS_EX_APPWINDOW
		setWindowLongW(w.handle, _GWL_STYLE, newStyle)
		setWindowLongW(w.handle, _GWL_EXSTYLE, newExStyle)
		setWindowPos(w.handle, _HWND_TOP,
			int(mr.Left), int(mr.Top),
			int(mr.Right-mr.Left), int(mr.Bottom-mr.Top),
			_SWP_NOACTIVATE|_SWP_FRAMECHANGED,
		)
		showWindow(w.handle, _SW_SHOW)
		w.fsMonitor = monitor

	} else {
		// Restore display mode if we were fullscreen.
		if w.fsMonitor != nil {
			devName, _ := syscall.UTF16PtrFromString(w.fsMonitor.name)
			changeDisplaySettingsExW(devName, nil, 0)
			w.fsMonitor = nil
		}

		// Restore window styles.
		setWindowLongW(w.handle, _GWL_STYLE, w.savedStyle)
		setWindowLongW(w.handle, _GWL_EXSTYLE, w.savedExStyle)

		// Use provided dimensions/position, falling back to saved values.
		x, y := w.savedX, w.savedY
		cw, ch := w.savedW, w.savedH
		if xpos != 0 || ypos != 0 {
			x, y = xpos, ypos
		}
		if width > 0 {
			cw = width
		}
		if height > 0 {
			ch = height
		}

		// Adjust for non-client area.
		rc := _RECT{0, 0, int32(cw), int32(ch)}
		adjustWindowRectEx(&rc, w.savedStyle, w.savedExStyle, false)
		setWindowPos(w.handle, _HWND_TOP,
			x, y,
			int(rc.Right-rc.Left), int(rc.Bottom-rc.Top),
			_SWP_NOACTIVATE|_SWP_FRAMECHANGED,
		)
		showWindow(w.handle, _SW_SHOW)
	}
}

// SetAttrib sets a window attribute at runtime.
// Supported attributes: Decorated, Floating, Resizable.
func (w *Window) SetAttrib(attrib Hint, value int) {
	switch attrib {
	case Decorated:
		style   := getWindowLongW(w.handle, _GWL_STYLE)
		exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
		if value != 0 {
			style |= _WS_CAPTION | _WS_SYSMENU | _WS_MINIMIZEBOX
		} else {
			style &^= _WS_CAPTION | _WS_SYSMENU | _WS_MINIMIZEBOX | _WS_THICKFRAME | _WS_MAXIMIZEBOX
		}
		setWindowLongW(w.handle, _GWL_STYLE, style)
		setWindowLongW(w.handle, _GWL_EXSTYLE, exStyle)
		setWindowPos(w.handle, 0, 0, 0, 0, 0,
			_SWP_NOMOVE|_SWP_NOSIZE|_SWP_NOZORDER|_SWP_NOACTIVATE|_SWP_FRAMECHANGED)

	case Floating:
		exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
		if value != 0 {
			exStyle |= _WS_EX_TOPMOST
		} else {
			exStyle &^= _WS_EX_TOPMOST
		}
		setWindowLongW(w.handle, _GWL_EXSTYLE, exStyle)
		var insertAfter uintptr
		if value != 0 {
			insertAfter = ^uintptr(0) // HWND_TOPMOST = -1
		} else {
			insertAfter = ^uintptr(1) // HWND_NOTOPMOST = -2
		}
		setWindowPos(w.handle, insertAfter, 0, 0, 0, 0,
			_SWP_NOMOVE|_SWP_NOSIZE|_SWP_NOACTIVATE)

	case Resizable:
		style := getWindowLongW(w.handle, _GWL_STYLE)
		if value != 0 {
			style |= _WS_THICKFRAME | _WS_MAXIMIZEBOX
		} else {
			style &^= _WS_THICKFRAME | _WS_MAXIMIZEBOX
		}
		setWindowLongW(w.handle, _GWL_STYLE, style)
		setWindowPos(w.handle, 0, 0, 0, 0, 0,
			_SWP_NOMOVE|_SWP_NOSIZE|_SWP_NOZORDER|_SWP_NOACTIVATE|_SWP_FRAMECHANGED)
	}
}

// SetIcon sets the window icon from a list of images of different sizes.
// The image with the best size for ICON_BIG (~32×32) and ICON_SMALL (~16×16)
// is chosen automatically. Pass nil to revert to the default application icon.
func (w *Window) SetIcon(images []Image) {
	if len(images) == 0 {
		icon, _ := loadIconW(0, _IDI_APPLICATION)
		sendMessageW(w.handle, _WM_SETICON, _ICON_BIG, icon)
		sendMessageW(w.handle, _WM_SETICON, _ICON_SMALL, icon)
		return
	}
	big   := pickIcon(images, 32)
	small := pickIcon(images, 16)
	hBig   := createHICON(big, true)
	hSmall := createHICON(small, true)
	if hBig != 0 {
		sendMessageW(w.handle, _WM_SETICON, _ICON_BIG, hBig)
	}
	if hSmall != 0 {
		sendMessageW(w.handle, _WM_SETICON, _ICON_SMALL, hSmall)
	}
}

// SetCursor sets the cursor shape while the cursor is over the client area.
// Pass nil to revert to the default arrow cursor.
func (w *Window) SetCursor(cursor *Cursor) {
	if cursor == nil {
		w.cursor = 0
		setSysCursor(gDefaultCursor)
	} else {
		w.cursor = cursor.handle
		setSysCursor(cursor.handle)
	}
}

// pickIcon returns the image whose size is closest to target.
func pickIcon(images []Image, target int) Image {
	best := images[0]
	bestDiff := abs(images[0].Width - target)
	for _, img := range images[1:] {
		d := abs(img.Width - target)
		if d < bestDiff {
			best = img
			bestDiff = d
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// createHICON creates a Windows HICON (or HCURSOR when isIcon=false) from an
// RGBA Image using a 32-bit DIB with alpha channel.
func createHICON(img Image, isIcon bool) uintptr {
	return createHICONCursor(img, 0, 0, isIcon)
}

func createHICONCursor(img Image, xhot, yhot int, isIcon bool) uintptr {
	w, h := img.Width, img.Height
	if w <= 0 || h <= 0 || len(img.Pixels) < w*h*4 {
		return 0
	}

	// Create a 32-bit top-down DIB for the colour data.
	bi := _BITMAPV5HEADER{
		BV5Size:        uint32(unsafe.Sizeof(_BITMAPV5HEADER{})),
		BV5Width:       int32(w),
		BV5Height:      -int32(h), // negative = top-down
		BV5Planes:      1,
		BV5BitCount:    32,
		BV5Compression: _BI_BITFIELDS,
		BV5RedMask:     0x00FF0000,
		BV5GreenMask:   0x0000FF00,
		BV5BlueMask:    0x000000FF,
		BV5AlphaMask:   0xFF000000,
		BV5CSType:      _LCS_WINDOWS_COLOR_SPACE,
	}

	var bits unsafe.Pointer
	hColor := createDIBSection(0, &bi, _DIB_RGB_COLORS, &bits)
	if hColor == 0 || bits == nil {
		return 0
	}

	// Copy pixels — source is RGBA; Windows DIB wants BGRA.
	dst := unsafe.Slice((*byte)(bits), w*h*4)
	src := img.Pixels
	for i := 0; i < w*h; i++ {
		dst[i*4+0] = src[i*4+2] // B
		dst[i*4+1] = src[i*4+1] // G
		dst[i*4+2] = src[i*4+0] // R
		dst[i*4+3] = src[i*4+3] // A
	}

	// Create the 1bpp mask bitmap (all zeros → use colour data as-is).
	maskLen := ((w + 31) / 32) * 4 * h
	mask := make([]byte, maskLen)
	hMask := createBitmap(int32(w), int32(h), 1, 1, unsafe.Pointer(&mask[0]))
	if hMask == 0 {
		deleteObject(hColor)
		return 0
	}

	ii := _ICONINFO{
		XHotspot: uint32(xhot),
		YHotspot: uint32(yhot),
		HbmMask:  hMask,
		HbmColor: hColor,
	}
	if isIcon {
		ii.FIcon = 1
	}
	hIcon := createIconIndirect(&ii)

	// The icon/cursor has its own copies of the bitmaps; free the originals.
	deleteObject(hMask)
	deleteObject(hColor)
	return hIcon
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
	switch mode {
	case CursorMode:
		return w.cursorMode
	case RawMouseMotion:
		if w.rawMouseMotion {
			return 1
		}
	}
	return 0
}

// SetInputMode sets an input mode on the window.
//
// Supported modes:
//
//	CursorMode → CursorNormal, CursorHidden, CursorDisabled
//
// CursorDisabled hides the cursor AND clips it to the window client area,
// which is the correct behaviour for first-person mouse-look.
// CursorHidden hides the cursor without clipping.
// CursorNormal restores the cursor to its default visible, unclipped state.
func (w *Window) SetInputMode(mode InputMode, value int) {
	if mode == RawMouseMotion {
		if value == 1 && !w.rawMouseMotion {
			w.rawMouseMotion = true
			w.rawCursorX, w.rawCursorY = 0, 0
			registerRawInputDevices([]_RAWINPUTDEVICE{{
				UsUsagePage: _HID_USAGE_PAGE_GENERIC,
				UsUsage:     _HID_USAGE_GENERIC_MOUSE,
				DwFlags:     _RIDEV_INPUTSINK,
				HwndTarget:  w.handle,
			}})
		} else if value == 0 && w.rawMouseMotion {
			w.rawMouseMotion = false
			registerRawInputDevices([]_RAWINPUTDEVICE{{
				UsUsagePage: _HID_USAGE_PAGE_GENERIC,
				UsUsage:     _HID_USAGE_GENERIC_MOUSE,
				DwFlags:     _RIDEV_REMOVE,
				HwndTarget:  0,
			}})
		}
		return
	}
	if mode != CursorMode {
		return
	}
	prev := w.cursorMode
	next := value
	if prev == next {
		return
	}

	// Restore to normal first so each case starts from a clean slate.
	if prev == CursorDisabled {
		clipCursor(nil)
		showCursor(true)
	} else if prev == CursorHidden {
		showCursor(true)
	}

	switch next {
	case CursorNormal:
		// already restored above

	case CursorHidden:
		showCursor(false)

	case CursorDisabled:
		// Hide and confine the cursor to the window client area.
		showCursor(false)
		rc := getClientRect(w.handle)
		// Convert client-area corners to screen coordinates.
		tl := _POINT{rc.Left, rc.Top}
		br := _POINT{rc.Right, rc.Bottom}
		clientToScreen(w.handle, &tl)
		clientToScreen(w.handle, &br)
		screenRect := _RECT{Left: tl.X, Top: tl.Y, Right: br.X, Bottom: br.Y}
		clipCursor(&screenRect)
	}

	w.cursorMode = next
}

// ----------------------------------------------------------------------------
// Clipboard
// ----------------------------------------------------------------------------

// GetClipboardString returns the current clipboard contents as a UTF-8 string.
// Returns an empty string if the clipboard is empty or does not contain text.
func GetClipboardString() string {
	if !openClipboard(0) {
		return ""
	}
	defer closeClipboard()

	hMem := getClipboardData(_CF_UNICODETEXT)
	if hMem == 0 {
		return ""
	}
	ptr := globalLock(hMem)
	if ptr == nil {
		return ""
	}
	defer globalUnlock(hMem)

	// ptr points to a null-terminated UTF-16LE string.
	// Find its length in uint16 code units.
	p := (*[1 << 28]uint16)(ptr)
	n := 0
	for p[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(p[:n])
}

// SetClipboardString places the given UTF-8 string on the clipboard.
func SetClipboardString(s string) {
	utf16, err := syscall.UTF16FromString(s)
	if err != nil {
		return
	}
	// Allocate a moveable global block large enough for the UTF-16 string + NUL.
	byteLen := uintptr(len(utf16) * 2)
	hMem := globalAlloc(_GMEM_MOVEABLE, byteLen)
	if hMem == 0 {
		return
	}

	// Copy the UTF-16 data into the locked block.
	ptr := globalLock(hMem)
	if ptr == nil {
		globalFree(hMem)
		return
	}
	dst := (*[1 << 28]uint16)(ptr)
	copy(dst[:len(utf16)], utf16)
	globalUnlock(hMem)

	if !openClipboard(0) {
		globalFree(hMem)
		return
	}
	emptyClipboard()
	if setClipboardData(_CF_UNICODETEXT, hMem) == 0 {
		// SetClipboardData failed; we must free the memory ourselves.
		globalFree(hMem)
	}
	// On success the clipboard owns hMem — do NOT free it.
	closeClipboard()
}

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

	case _WM_INPUT:
		if w.rawMouseMotion {
			var ri _RAWINPUT
			sz := uint32(unsafe.Sizeof(ri))
			if getRawInputData(lParam, &ri, &sz) != ^uint32(0) &&
				ri.Header.DwType == _RIM_TYPEMOUSE &&
				ri.Mouse.UsFlags&_MOUSE_MOVE_ABSOLUTE == 0 {
				dx := float64(ri.Mouse.LLastX)
				dy := float64(ri.Mouse.LLastY)
				w.rawCursorX += dx
				w.rawCursorY += dy
				if w.fCursorPosHolder != nil {
					w.fCursorPosHolder(w, w.rawCursorX, w.rawCursorY)
				}
			}
		}
		return 0

	case _WM_MOUSEMOVE:
		// When raw motion is enabled, WM_MOUSEMOVE is suppressed so callers
		// only receive the unaccelerated deltas delivered via WM_INPUT.
		if !w.rawMouseMotion {
			if w.fCursorPosHolder != nil {
				w.fCursorPosHolder(w, float64(getXLParam(lParam)), float64(getYLParam(lParam)))
			}
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

	case _WM_GETMINMAXINFO:
		mmi := (*_MINMAXINFO)(nativePtrFromUintptr(lParam))
		w.applyMinMaxInfo(mmi)
		// Do NOT return 0 here — let DefWindowProc also run so the system
		// maximised/restored defaults are still applied as a baseline.

	case _WM_SIZING:
		rc := (*_RECT)(nativePtrFromUintptr(lParam))
		w.enforceAspectRatio(rc, wParam)
		return 1

	case _WM_SETCURSOR:
		if loword(lParam) == int16(_HTCLIENT) {
			if w.cursor != 0 {
				setSysCursor(w.cursor)
			} else {
				setSysCursor(gDefaultCursor)
			}
			return 1
		}

	case _WM_ERASEBKGND:
		return 1 // prevent flicker

	case _WM_PAINT:
		if w.fRefreshHolder != nil { w.fRefreshHolder(w) }

	case _WM_DISPLAYCHANGE:
		// A monitor was connected or disconnected; fire the monitor callback.
		if winMonitorCb != nil {
			newMonitors, _ := GetMonitors()
			diffAndFireMonitorCallbacks(winCachedMonitors, newMonitors, winMonitorCb)
			winCachedMonitors = newMonitors
		}
		return 0
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
