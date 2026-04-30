//go:build linux

package glfw

import (
	"sync"
	"syscall"
	"unsafe"
)

// ----------------------------------------------------------------------------
// Additional event mask constants needed for _NET_WM_STATE messages
// ----------------------------------------------------------------------------

const (
	_SubstructureNotifyMask   = int64(1 << 19)
	_SubstructureRedirectMask = int64(1 << 20)
	_RevertToParent           = 2
	_CurrentTime              = uint64(0)
)

// ----------------------------------------------------------------------------
// Per-window X11 state (cursor pos, key/button states)
// stored in a sync.Map because Window struct has no build-tag-guarded fields
// ----------------------------------------------------------------------------

type x11WindowState struct {
	cursorX, cursorY float64
	keyStates        [maxKey]bool
	mouseStates      [8]bool
}

var x11States sync.Map // uintptr(handle) → *x11WindowState

func getX11State(handle uintptr) *x11WindowState {
	if v, ok := x11States.Load(handle); ok {
		return v.(*x11WindowState)
	}
	s := &x11WindowState{}
	x11States.Store(handle, s)
	return s
}

// ----------------------------------------------------------------------------
// CreateWindow
// ----------------------------------------------------------------------------

// CreateWindow creates an X11 window with an associated EGL context.
func CreateWindow(width, height int, title string, monitor, share *Monitor) (*Window, error) {
	hints.mu.Lock()
	h := make(map[Hint]int, len(hints.m))
	for k, v := range hints.m {
		h[k] = v
	}
	hints.mu.Unlock()

	if err := initX11Display(); err != nil {
		return nil, err
	}

	// Get default visual and depth
	visual := xDefaultVisual(x11Display, x11Screen)
	depth := xDefaultDepth(x11Display, x11Screen)

	// Set up window attributes
	var attrs _XSetWindowAttributes
	attrs.EventMask = _KeyPressMask | _KeyReleaseMask |
		_ButtonPressMask | _ButtonReleaseMask |
		_EnterWindowMask | _LeaveWindowMask |
		_PointerMotionMask |
		_ExposureMask |
		_StructureNotifyMask |
		_FocusChangeMask

	valueMask := uint64(_CWEventMask | _CWBorderPixel)

	xwin := xCreateWindow(
		x11Display,
		uintptr(x11Root),
		0, 0,
		uint32(width), uint32(height),
		0,                // border width
		int32(depth),
		uint32(_InputOutput),
		visual,
		valueMask,
		uintptr(unsafe.Pointer(&attrs)),
	)
	if xwin == 0 {
		return nil, &Error{Code: PlatformError, Desc: "XCreateWindow failed"}
	}

	// Register WM_DELETE_WINDOW protocol
	proto := atomWMDeleteWindow
	xSetWMProtocols(x11Display, xwin, uintptr(unsafe.Pointer(&proto)), 1)

	// Set title
	if title != "" {
		titlePtr, _ := syscall.BytePtrFromString(title)
		xStoreName(x11Display, xwin, uintptr(unsafe.Pointer(titlePtr)))
		// Also set _NET_WM_NAME (UTF-8)
		xChangeProperty(x11Display, xwin,
			atomNETWMName, atomUTF8String,
			8, 0,
			uintptr(unsafe.Pointer(titlePtr)),
			int32(len(title)))
	}

	// Create EGL context — pass x11Display so EGL can connect to the X server
	surf, ctx, err := createEGLContext(x11Display, uintptr(xwin), h)
	if err != nil {
		xDestroyWindow(x11Display, xwin)
		return nil, err
	}

	w := &Window{
		handle:     uintptr(xwin),
		eglSurface: surf,
		eglContext: ctx,
		useEGL:     true,
	}
	w.title = title
	windowByHandle.Store(uintptr(xwin), w)

	// Initialise per-window state
	_ = getX11State(uintptr(xwin))

	// Show window if Visible hint is set (default: 1)
	if h[Visible] != 0 {
		xMapWindow(x11Display, xwin)
		xFlush(x11Display)
	}

	return w, nil
}

// ----------------------------------------------------------------------------
// Window — context methods
// ----------------------------------------------------------------------------

// MakeContextCurrent makes the window's EGL context current on this thread.
func (w *Window) MakeContextCurrent() {
	eglMakeCurrentWindow(w)
}

// SwapBuffers swaps the front and back buffers.
func (w *Window) SwapBuffers() {
	eglSwapBuffersWindow(w)
}

// Destroy releases all resources associated with the window.
func (w *Window) Destroy() {
	if x11Display != 0 && w.handle != 0 {
		eglDestroyWindow(w)
		xDestroyWindow(x11Display, uint64(w.handle))
	}
	windowByHandle.Delete(w.handle)
	x11States.Delete(w.handle)
	w.handle = 0
}

// ----------------------------------------------------------------------------
// Window — geometry
// ----------------------------------------------------------------------------

// GetSize returns the size of the window's client area.
func (w *Window) GetSize() (width, height int) {
	if x11Display == 0 || w.handle == 0 {
		return 0, 0
	}
	var wa _XWindowAttributes
	xGetWindowAttributes(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&wa)))
	return int(wa.Width), int(wa.Height)
}

// SetSize sets the size of the window's client area.
func (w *Window) SetSize(width, height int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xResizeWindow(x11Display, uint64(w.handle), uint32(width), uint32(height))
	xFlush(x11Display)
}

// GetPos returns the position of the window's client area.
func (w *Window) GetPos() (x, y int) {
	if x11Display == 0 || w.handle == 0 {
		return 0, 0
	}
	var wa _XWindowAttributes
	xGetWindowAttributes(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&wa)))
	return int(wa.X), int(wa.Y)
}

// SetPos sets the position of the window's upper-left corner.
func (w *Window) SetPos(x, y int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xMoveWindow(x11Display, uint64(w.handle), int32(x), int32(y))
	xFlush(x11Display)
}

// GetFramebufferSize returns the framebuffer size in pixels (same as client area on X11).
func (w *Window) GetFramebufferSize() (width, height int) {
	return w.GetSize()
}

// GetContentScale returns the DPI scale factors (always 1,1 on basic X11).
func (w *Window) GetContentScale() (x, y float32) {
	return 1, 1
}

// ----------------------------------------------------------------------------
// Window — state
// ----------------------------------------------------------------------------

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	titlePtr, _ := syscall.BytePtrFromString(title)
	xStoreName(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(titlePtr)))
	xChangeProperty(x11Display, uint64(w.handle),
		atomNETWMName, atomUTF8String,
		8, 0,
		uintptr(unsafe.Pointer(titlePtr)),
		int32(len(title)))
	w.title = title
}

// Iconify minimises the window.
func (w *Window) Iconify() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xIconifyWindow(x11Display, uint64(w.handle), x11Screen)
	xFlush(x11Display)
}

// Restore restores an iconified or maximised window.
func (w *Window) Restore() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	// Remove maximized state
	w.sendNETWMState(_NET_WM_STATE_REMOVE, atomNETWMStateMaxH, atomNETWMStateMaxV)
	// Map the window (restore from iconic)
	xMapWindow(x11Display, uint64(w.handle))
	xFlush(x11Display)
}

// Maximize maximises the window.
func (w *Window) Maximize() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateMaxH, atomNETWMStateMaxV)
	xFlush(x11Display)
}

// Show makes the window visible.
func (w *Window) Show() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xMapWindow(x11Display, uint64(w.handle))
	xFlush(x11Display)
}

// Hide hides the window.
func (w *Window) Hide() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xUnmapWindow(x11Display, uint64(w.handle))
	xFlush(x11Display)
}

// Focus brings the window to the foreground.
func (w *Window) Focus() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xRaiseWindow(x11Display, uint64(w.handle))
	xSetInputFocus(x11Display, uint64(w.handle), _RevertToParent, _CurrentTime)
	xFlush(x11Display)
}

// GetAttrib returns the current value of a window attribute.
func (w *Window) GetAttrib(attrib Hint) int {
	if x11Display == 0 || w.handle == 0 {
		return 0
	}
	switch attrib {
	case Visible:
		var wa _XWindowAttributes
		xGetWindowAttributes(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&wa)))
		if wa.MapState == 2 { // IsViewable
			return 1
		}
		return 0
	case Focused:
		return 0 // would need XGetInputFocus
	case Resizable:
		return 1
	case Decorated:
		return 1
	}
	return 0
}

// GetMonitor returns the monitor the window is currently on, or nil.
func (w *Window) GetMonitor() *Monitor { return nil }

// ----------------------------------------------------------------------------
// Window — input
// ----------------------------------------------------------------------------

// GetKey returns the last known state of the given keyboard key.
func (w *Window) GetKey(key Key) Action {
	if key == KeyUnknown {
		return Release
	}
	s := getX11State(w.handle)
	idx := int(key)
	if idx >= 0 && idx < len(s.keyStates) && s.keyStates[idx] {
		return Press
	}
	return Release
}

// GetMouseButton returns the last known state of the given mouse button.
func (w *Window) GetMouseButton(button MouseButton) Action {
	s := getX11State(w.handle)
	idx := int(button)
	if idx >= 0 && idx < len(s.mouseStates) && s.mouseStates[idx] {
		return Press
	}
	return Release
}

// GetCursorPos returns the cursor position in client coordinates.
func (w *Window) GetCursorPos() (x, y float64) {
	s := getX11State(w.handle)
	return s.cursorX, s.cursorY
}

// SetCursorPos moves the cursor to the given client-area position.
func (w *Window) SetCursorPos(x, y float64) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	xWarpPointer(x11Display, 0, uint64(w.handle), 0, 0, 0, 0, int32(x), int32(y))
	xFlush(x11Display)
	// Update cached position
	s := getX11State(w.handle)
	s.cursorX = x
	s.cursorY = y
}

// GetInputMode returns the current value of an input mode.
func (w *Window) GetInputMode(mode InputMode) int {
	if mode == CursorMode {
		return w.cursorMode
	}
	return 0
}

// SetInputMode sets an input mode on the window.
func (w *Window) SetInputMode(mode InputMode, value int) {
	if mode != CursorMode {
		return
	}
	w.cursorMode = value
	if x11Display == 0 || w.handle == 0 {
		return
	}
	switch value {
	case CursorNormal:
		// Restore the user-set cursor, or the system default if none.
		if w.cursor != 0 {
			xDefineCursor(x11Display, uint64(w.handle), uint64(w.cursor))
		} else {
			xDefineCursor(x11Display, uint64(w.handle), 0)
		}
	case CursorHidden, CursorDisabled:
		invis := getInvisibleCursor()
		if invis != 0 {
			xDefineCursor(x11Display, uint64(w.handle), invis)
		}
	}
	xFlush(x11Display)
}

// ----------------------------------------------------------------------------
// GetProcAddress / SwapInterval / ExtensionSupported
// ----------------------------------------------------------------------------

// GetProcAddress returns the address of the named OpenGL ES function.
func GetProcAddress(name string) unsafe.Pointer {
	return eglGetProcAddr(name)
}

// SwapInterval sets the swap interval for the current EGL display.
func SwapInterval(interval int) {
	eglSwapIntervalNow(interval)
}

// ExtensionSupported reports whether the named extension is available.
func ExtensionSupported(extension string) bool {
	return false
}

// ----------------------------------------------------------------------------
// sendNETWMState — helper for _NET_WM_STATE client messages
// ----------------------------------------------------------------------------

func (w *Window) sendNETWMState(action int64, atom1, atom2 uint64) {
	var ev _XClientMessageEvent
	ev.Type = _ClientMessage
	ev.Window = uint64(w.handle)
	ev.MessageType = atomNETWMState
	ev.Format = 32
	ev.Data[0] = action
	ev.Data[1] = int64(atom1)
	ev.Data[2] = int64(atom2)
	xSendEvent(x11Display, x11Root, 0,
		_SubstructureNotifyMask|_SubstructureRedirectMask,
		uintptr(unsafe.Pointer(&ev)))
}

// ----------------------------------------------------------------------------
// DetachCurrentContext
// ----------------------------------------------------------------------------

// DetachCurrentContext detaches the current EGL context from the calling thread.
func DetachCurrentContext() {
	if eglLibLoaded && currentEGLDisplay != 0 {
		eglMakeCurrent(currentEGLDisplay, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT)
		currentEGLDisplay = 0
	}
}

// GetCurrentContext returns the window whose EGL context is current on this thread.
func GetCurrentContext() *Window {
	if !eglLibLoaded {
		return nil
	}
	cur := eglGetCurrentContext()
	if cur == 0 {
		return nil
	}
	var found *Window
	windowByHandle.Range(func(k, v any) bool {
		ww := v.(*Window)
		if ww.eglContext == cur {
			found = ww
			return false
		}
		return true
	})
	return found
}
