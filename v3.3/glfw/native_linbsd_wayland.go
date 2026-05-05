//go:build linux && wayland

package glfw

// Native handle accessors for the Wayland backend.  Mirrors the upstream
// go-gl/glfw `native_linbsd_wayland.go` API surface.
//
// All handles are returned as `uintptr` — callers that need the upstream
// `unsafe.Pointer` form can wrap with `unsafe.Pointer(uintptr(...))`.

// GetWaylandDisplay returns the wl_display* connection cast to uintptr.
func GetWaylandDisplay() uintptr { return wl.display }

// GetWaylandWindow returns the wl_surface* for this window cast to uintptr.
func (w *Window) GetWaylandWindow() uintptr { return w.handle }

// GetWaylandMonitor returns the wl_output* proxy for this monitor cast to
// uintptr, or 0 if the corresponding wl_output proxy is no longer registered
// (rare — would only happen during a hot-unplug race).
func (m *Monitor) GetWaylandMonitor() uintptr {
	for _, out := range wl.outputs {
		if out.monitor == m {
			return out.proxy
		}
	}
	return 0
}

// GetEGLDisplay returns the shared EGLDisplay handle.
func GetEGLDisplay() uintptr { return eglSharedDisplay }

// GetEGLContext returns the EGLContext for this window.
// Returns 0 if the window has no EGL context.
func (w *Window) GetEGLContext() uintptr { return w.eglContext }

// GetEGLSurface returns the EGLSurface for this window.
// Returns 0 if the window has no EGL surface.
func (w *Window) GetEGLSurface() uintptr { return w.eglSurface }
