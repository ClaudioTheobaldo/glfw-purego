//go:build darwin

package glfw

// Native handle accessors for the Cocoa backend.  These mirror the
// `native_*` files in upstream go-gl/glfw and the existing
// `win_native_windows.go` pattern, returning the underlying ObjC / CG
// handles as `uintptr` so callers can interop with purego or other Go
// libraries that consume the same handle type.

// GetCocoaWindow returns the NSWindow* for this window cast to uintptr.
func (w *Window) GetCocoaWindow() uintptr { return w.handle }

// GetNSGLContext returns the NSOpenGLContext* for this window cast to uintptr.
// Returns 0 if the window was created without an OpenGL context.
func (w *Window) GetNSGLContext() uintptr { return w.nsglContext }

// GetCocoaMonitor returns the CGDirectDisplayID for this monitor.
func (m *Monitor) GetCocoaMonitor() uint32 { return m.cgDisplayID }
