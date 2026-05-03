//go:build linux && wayland

package glfw

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ── CreateWindow ──────────────────────────────────────────────────────────────

// CreateWindow creates a Wayland window with an EGL context.
func CreateWindow(width, height int, title string, monitor, share *Monitor) (*Window, error) {
	hints.mu.Lock()
	h := make(map[Hint]int, len(hints.m))
	for k, v := range hints.m {
		h[k] = v
	}
	hints.mu.Unlock()

	if wl.display == 0 {
		return nil, &Error{Code: PlatformError, Desc: "Wayland: not initialised"}
	}
	if wl.compositor == 0 || wl.wmBase == 0 {
		return nil, &Error{Code: PlatformError, Desc: "Wayland: compositor or xdg_wm_base not available"}
	}
	if err := loadWaylandEGL(); err != nil {
		return nil, &Error{Code: APIUnavailable,
			Desc: "Wayland: libwayland-egl not available: " + err.Error()}
	}

	w := &Window{title: title, useEGL: true}
	w.minW, w.minH = -1, -1
	w.maxW, w.maxH = -1, -1

	// ── wl_surface ────────────────────────────────────────────────────────────
	// wl_compositor.create_surface opcode=0, signature="n"
	w.handle = wlProxyMarshalFlags(wl.compositor, 0, wlSurfaceIfaceAddr, 4, 0, 0)
	if w.handle == 0 {
		return nil, &Error{Code: PlatformError, Desc: "Wayland: wl_compositor.create_surface failed"}
	}
	windowByHandle.Store(w.handle, w)

	// ── xdg_surface ───────────────────────────────────────────────────────────
	// xdg_wm_base.get_xdg_surface opcode=2, signature="no" (new_id + wl_surface)
	// For "no": variadic args are (NULL for new_id, then the object).
	w.wlXdgSurf = wlProxyMarshalFlags(wl.wmBase, 2,
		uintptr(unsafe.Pointer(&xdgSurfaceIface)), 4, 0,
		0, w.handle)
	if w.wlXdgSurf == 0 {
		windowByHandle.Delete(w.handle)
		wlProxyMarshalFlags(w.handle, 0, 0, 4, 1) // destroy + free
		w.handle = 0
		return nil, &Error{Code: PlatformError, Desc: "Wayland: xdg_wm_base.get_xdg_surface failed"}
	}

	// ── xdg_toplevel ──────────────────────────────────────────────────────────
	// xdg_surface.get_toplevel opcode=1, signature="n"
	w.wlXdgTop = wlProxyMarshalFlags(w.wlXdgSurf, 1,
		uintptr(unsafe.Pointer(&xdgToplevelIface)), 4, 0,
		0)
	if w.wlXdgTop == 0 {
		wlProxyMarshalFlags(w.wlXdgSurf, 0, 0, 4, 1)
		windowByHandle.Delete(w.handle)
		wlProxyMarshalFlags(w.handle, 0, 0, 4, 1)
		w.wlXdgSurf = 0
		w.handle = 0
		return nil, &Error{Code: PlatformError, Desc: "Wayland: xdg_surface.get_toplevel failed"}
	}

	// ── xdg_surface listener (configure event) ────────────────────────────────
	win := w // captured by closures below
	w.wlXdgSurfList = new([1]uintptr)
	w.wlXdgSurfList[0] = purego.NewCallback(func(data, xdgSurf uintptr, serial uint32) {
		// Resize the EGL window if the compositor requested a new size.
		if win.wlWidth > 0 && win.wlHeight > 0 && win.wlEGLWin != 0 {
			wlEGLWindowResize(win.wlEGLWin, int32(win.wlWidth), int32(win.wlHeight), 0, 0)
			if win.fFramebufferSizeHolder != nil {
				win.fFramebufferSizeHolder(win, win.wlWidth, win.wlHeight)
			}
			if win.fSizeHolder != nil {
				win.fSizeHolder(win, win.wlWidth, win.wlHeight)
			}
		}
		// Acknowledge the configure, then commit the surface.
		// xdg_surface.ack_configure opcode=4, signature="u"
		wlProxyMarshalFlags(xdgSurf, 4, 0, 4, 0, uintptr(serial))
		// wl_surface.commit opcode=6
		wlProxyMarshalFlags(win.handle, 6, 0, 4, 0)
		wlDisplayFlush(wl.display)
	})
	wlProxyAddListener(w.wlXdgSurf, uintptr(unsafe.Pointer(w.wlXdgSurfList)), 0)

	// ── xdg_toplevel listener (configure + close events) ─────────────────────
	w.wlXdgTopList = new([2]uintptr)
	// event 0: configure(width, height int32, states wl_array*)
	w.wlXdgTopList[0] = purego.NewCallback(func(data, top uintptr, width, height int32, states uintptr) {
		if width > 0 {
			win.wlWidth = int(width)
		}
		if height > 0 {
			win.wlHeight = int(height)
		}
	})
	// event 1: close()
	w.wlXdgTopList[1] = purego.NewCallback(func(data, top uintptr) {
		win.shouldClose = true
		if win.fCloseHolder != nil {
			win.fCloseHolder(win)
		}
	})
	wlProxyAddListener(w.wlXdgTop, uintptr(unsafe.Pointer(w.wlXdgTopList)), 0)

	// ── Title & app-id ────────────────────────────────────────────────────────
	if title != "" {
		wlSetTitle(w, title)
	}
	appID := []byte("glfw-app\x00")
	wlProxyMarshalFlags(w.wlXdgTop, 3, 0, 4, 0,
		uintptr(unsafe.Pointer(&appID[0])))
	runtime.KeepAlive(appID)

	// ── Server-side decorations (optional) ────────────────────────────────────
	if wl.decoMgr != 0 {
		// zxdg_decoration_manager_v1.get_toplevel_decoration opcode=1, signature="no"
		// For "no": variadic args are (NULL for new_id, then the object).
		deco := wlProxyMarshalFlags(wl.decoMgr, 1,
			uintptr(unsafe.Pointer(&xdgTopDecoIface)), 1, 0,
			0, w.wlXdgTop)
		if deco != 0 {
			// zxdg_toplevel_decoration_v1.set_mode opcode=1, value=2 (server-side)
			wlProxyMarshalFlags(deco, 1, 0, 1, 0, uintptr(2))
			// Destroy immediately — we only needed to request the mode.
			wlProxyMarshalFlags(deco, 0, 0, 1, 1)
		}
	}

	// ── Initial commit — triggers the first configure roundtrip ───────────────
	wlProxyMarshalFlags(w.handle, 6, 0, 4, 0) // wl_surface.commit
	wlDisplayFlush(wl.display)
	wlDisplayRoundtrip(wl.display)

	// Use the requested size if the compositor didn't supply one.
	if w.wlWidth == 0 {
		w.wlWidth = width
	}
	if w.wlHeight == 0 {
		w.wlHeight = height
	}

	// ── EGL window + context ──────────────────────────────────────────────────
	w.wlEGLWin = wlEGLWindowCreate(w.handle, int32(w.wlWidth), int32(w.wlHeight))
	if w.wlEGLWin == 0 {
		w.destroyProxies()
		return nil, &Error{Code: PlatformError, Desc: "Wayland: wl_egl_window_create failed"}
	}

	// Pass wl.display as the EGL native display so Mesa picks the Wayland
	// platform (EGL_PLATFORM_WAYLAND_KHR) instead of trying EGL_DEFAULT_DISPLAY.
	surf, ctx, err := createEGLContext(wl.display, w.wlEGLWin, h)
	if err != nil {
		wlEGLWindowDestroy(w.wlEGLWin)
		w.wlEGLWin = 0
		w.destroyProxies()
		return nil, err
	}
	w.eglSurface = surf
	w.eglContext = ctx

	// Apply size constraints from hints (if set).
	if h[Resizable] == 0 {
		w.SetSizeLimits(w.wlWidth, w.wlHeight, w.wlWidth, w.wlHeight)
	}

	return w, nil
}

// destroyProxies tears down the Wayland protocol objects (without EGL cleanup).
func (w *Window) destroyProxies() {
	if w.wlXdgTop != 0 {
		wlProxyMarshalFlags(w.wlXdgTop, 0, 0, 4, 1) // destroy + free
		w.wlXdgTop = 0
	}
	if w.wlXdgSurf != 0 {
		wlProxyMarshalFlags(w.wlXdgSurf, 0, 0, 4, 1)
		w.wlXdgSurf = 0
	}
	if w.handle != 0 {
		windowByHandle.Delete(w.handle)
		wlProxyMarshalFlags(w.handle, 0, 0, 4, 1)
		w.handle = 0
	}
}

// ── Destroy ───────────────────────────────────────────────────────────────────

// Destroy releases all resources associated with the window.
func (w *Window) Destroy() {
	if wlCurrentWindow == w {
		DetachCurrentContext()
	}
	// Clear pointer/keyboard focus references.
	if wl.activeWin == w {
		wl.activeWin = nil
	}
	if wl.kbWin == w {
		wl.kbWin = nil
	}
	// Destroy EGL context and surface.
	eglDestroyWindow(w)
	// Destroy EGL window (native Wayland window).
	if w.wlEGLWin != 0 {
		wlEGLWindowDestroy(w.wlEGLWin)
		w.wlEGLWin = 0
	}
	// Destroy Wayland protocol objects in correct order.
	w.destroyProxies()
	wlDisplayFlush(wl.display)
}

// ── Context ───────────────────────────────────────────────────────────────────

var wlCurrentWindow *Window

// MakeContextCurrent makes the window's EGL context current on this thread.
func (w *Window) MakeContextCurrent() {
	eglMakeCurrentWindow(w)
	wlCurrentWindow = w
}

// SwapBuffers presents the rendered frame to the compositor.
func (w *Window) SwapBuffers() {
	eglSwapBuffersWindow(w)
}

// DetachCurrentContext detaches the EGL context from the calling thread.
func DetachCurrentContext() {
	if eglLibLoaded && currentEGLDisplay != 0 {
		eglMakeCurrent(currentEGLDisplay, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT)
		currentEGLDisplay = 0
	}
	wlCurrentWindow = nil
}

// GetCurrentContext returns the window whose EGL context is current, or nil.
func GetCurrentContext() *Window {
	if !eglLibLoaded {
		return nil
	}
	cur := eglGetCurrentContext()
	if cur == 0 {
		return nil
	}
	var found *Window
	windowByHandle.Range(func(_, v any) bool {
		ww := v.(*Window)
		if ww.eglContext == cur {
			found = ww
			return false
		}
		return true
	})
	return found
}

// ── Geometry ─────────────────────────────────────────────────────────────────

// GetSize returns the current window dimensions (as last set by compositor configure).
func (w *Window) GetSize() (width, height int) {
	return w.wlWidth, w.wlHeight
}

// SetSize requests a new window size.  On Wayland the compositor may ignore it;
// we optimistically resize the EGL window and fire callbacks.
func (w *Window) SetSize(width, height int) {
	w.wlWidth = width
	w.wlHeight = height
	if w.wlEGLWin != 0 {
		wlEGLWindowResize(w.wlEGLWin, int32(width), int32(height), 0, 0)
	}
	if w.fSizeHolder != nil {
		w.fSizeHolder(w, width, height)
	}
	if w.fFramebufferSizeHolder != nil {
		w.fFramebufferSizeHolder(w, width, height)
	}
}

// GetFramebufferSize returns the framebuffer size (same as the window size on Wayland).
func (w *Window) GetFramebufferSize() (width, height int) {
	return w.wlWidth, w.wlHeight
}

// GetContentScale returns the DPI scale (always 1,1 — HiDPI is not yet wired up).
func (w *Window) GetContentScale() (x, y float32) { return 1, 1 }

// GetPos always returns (0, 0) on Wayland.
//
// The Wayland protocol deliberately hides window positions from clients; only
// the compositor knows where a surface is placed on screen.  Code that depends
// on the absolute position of an xdg_toplevel will not work correctly on
// Wayland and should be guarded with a platform check.
func (w *Window) GetPos() (x, y int) { return 0, 0 }

// SetPos is a no-op on Wayland.
//
// xdg_toplevel surfaces are positioned exclusively by the compositor; there is
// no protocol request to place a window at a specific screen coordinate.
func (w *Window) SetPos(x, y int) {}

// ── Visibility / state ───────────────────────────────────────────────────────

// Show commits the surface to make it visible to the compositor.
func (w *Window) Show() {
	if w.handle != 0 {
		wlProxyMarshalFlags(w.handle, 6, 0, 4, 0) // wl_surface.commit
		wlDisplayFlush(wl.display)
	}
}

// Hide is a no-op on Wayland.
//
// The xdg_toplevel protocol has no request to unmap or hide a surface without
// destroying it.  Use Iconify to minimise the window, or Destroy + CreateWindow
// if you need to suppress it from the taskbar entirely.
func (w *Window) Hide() {}

// Focus is a no-op on Wayland — clients cannot steal input focus.
func (w *Window) Focus() {}

// Iconify requests the compositor to minimise the window.
func (w *Window) Iconify() {
	if w.wlXdgTop != 0 {
		wlProxyMarshalFlags(w.wlXdgTop, 13, 0, 4, 0) // set_minimized
		wlProxyMarshalFlags(w.handle, 6, 0, 4, 0)
		wlDisplayFlush(wl.display)
	}
}

// Restore un-maximises the window (minimise-restore is not directly available).
func (w *Window) Restore() {
	if w.wlXdgTop != 0 {
		wlProxyMarshalFlags(w.wlXdgTop, 10, 0, 4, 0) // unset_maximized
		wlProxyMarshalFlags(w.handle, 6, 0, 4, 0)
		wlDisplayFlush(wl.display)
	}
}

// Maximize requests the compositor to maximise the window.
func (w *Window) Maximize() {
	if w.wlXdgTop != 0 {
		wlProxyMarshalFlags(w.wlXdgTop, 9, 0, 4, 0) // set_maximized
		wlProxyMarshalFlags(w.handle, 6, 0, 4, 0)
		wlDisplayFlush(wl.display)
	}
}

// SetTitle updates the window title.
func (w *Window) SetTitle(title string) {
	w.title = title
	wlSetTitle(w, title)
}

// wlSetTitle sends xdg_toplevel.set_title.
func wlSetTitle(w *Window, title string) {
	if w.wlXdgTop == 0 {
		return
	}
	titleC := append([]byte(title), 0)
	wlProxyMarshalFlags(w.wlXdgTop, 2, 0, 4, 0,
		uintptr(unsafe.Pointer(&titleC[0])))
	runtime.KeepAlive(titleC)
	wlProxyMarshalFlags(w.handle, 6, 0, 4, 0)
	wlDisplayFlush(wl.display)
}

// ── Fullscreen ───────────────────────────────────────────────────────────────

// SetMonitor switches between fullscreen (monitor != nil) and windowed mode.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if w.wlXdgTop == 0 {
		return
	}
	if monitor != nil {
		// Find the wl_output proxy for this monitor.
		var outputProxy uintptr
		for _, out := range wl.outputs {
			if out.monitor == monitor {
				outputProxy = out.proxy
				break
			}
		}
		// xdg_toplevel.set_fullscreen opcode=11, signature="?o"
		wlProxyMarshalFlags(w.wlXdgTop, 11, 0, 4, 0, outputProxy)
		w.fsMonitor = monitor
	} else {
		// xdg_toplevel.unset_fullscreen opcode=12
		wlProxyMarshalFlags(w.wlXdgTop, 12, 0, 4, 0)
		w.fsMonitor = nil
	}
	wlProxyMarshalFlags(w.handle, 6, 0, 4, 0) // commit
	wlDisplayFlush(wl.display)
}

// GetMonitor returns the monitor the window is fullscreen on, or nil.
func (w *Window) GetMonitor() *Monitor { return w.fsMonitor }

// ── Attributes ───────────────────────────────────────────────────────────────

// GetAttrib returns the value of a window attribute.
func (w *Window) GetAttrib(attrib Hint) int {
	switch attrib {
	case Visible:
		return 1
	case Resizable:
		if w.minW > 0 && w.minW == w.maxW && w.minH > 0 && w.minH == w.maxH {
			return 0
		}
		return 1
	case Decorated:
		return 1
	case Focused:
		if wl.kbWin == w {
			return 1
		}
		return 0
	}
	return 0
}

// SetAttrib updates a window attribute at runtime.
func (w *Window) SetAttrib(attrib Hint, value int) {
	// Decorated: server-side decoration toggle via zxdg_decoration_manager_v1
	// is a protocol extension; not all compositors support it.  No-op here.
}

// ── Input ─────────────────────────────────────────────────────────────────────

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
	switch value {
	case CursorHidden, CursorDisabled:
		// Hide the cursor while it is over this window.
		wlApplyCursor(nil)
	case CursorNormal:
		// Restore the window's current cursor (or the system default arrow).
		if w.cursor != 0 {
			wlApplyCursor(&Cursor{handle: w.cursor, system: true})
		} else {
			c, _ := CreateStandardCursor(ArrowCursor)
			wlApplyCursor(c)
		}
	}
}

// GetKey returns the last known state of a keyboard key.
func (w *Window) GetKey(key Key) Action {
	if key != KeyUnknown && int(key) < len(wl.keyState) {
		return wl.keyState[key]
	}
	return Release
}

// GetMouseButton returns the last known state of a mouse button.
func (w *Window) GetMouseButton(button MouseButton) Action {
	if int(button) < len(wl.btnState) {
		return wl.btnState[button]
	}
	return Release
}

// GetCursorPos returns the cursor position within the window's client area.
func (w *Window) GetCursorPos() (x, y float64) {
	return w.wlCursorX, w.wlCursorY
}

// SetCursorPos is a no-op — Wayland does not allow clients to warp the pointer.
func (w *Window) SetCursorPos(x, y float64) {}

// ── OpenGL / EGL helpers ─────────────────────────────────────────────────────

// GetProcAddress returns the address of the named OpenGL function.
func GetProcAddress(name string) unsafe.Pointer {
	return eglGetProcAddr(name)
}

// SwapInterval sets the swap interval for the current EGL display.
func SwapInterval(interval int) {
	eglSwapIntervalNow(interval)
}

// ExtensionSupported reports whether an OpenGL extension is available (stub).
func ExtensionSupported(extension string) bool { return false }
