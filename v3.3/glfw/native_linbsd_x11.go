//go:build linux && !wayland

package glfw

// Native handle accessors for the X11 backend.  Mirrors the upstream
// go-gl/glfw `native_linbsd_x11.go` API surface so that code using the
// same accessors can be ported transparently.
//
// All handles are returned as `uintptr` — callers that need the upstream
// `unsafe.Pointer` form can wrap with `unsafe.Pointer(uintptr(...))`.

// GetX11Display returns the Display* shared by all windows.
func GetX11Display() uintptr { return x11Display }

// GetX11Window returns the X11 Window resource ID cast to uintptr.
func (w *Window) GetX11Window() uintptr { return w.handle }

// GetGLXContext returns the OpenGL context handle for this window.
//
// Note: glfw-purego uses EGL via Mesa for OpenGL on X11 instead of GLX,
// so this is technically an EGLContext rather than a GLXContext.  For
// most consumers (Vulkan integrators, other Go bindings) the value is
// what matters and the precise context-type label does not.
func (w *Window) GetGLXContext() uintptr { return w.eglContext }

// GetGLXWindow returns the X11 Window used for GLX/EGL drawing.
// On glfw-purego this is the same as GetX11Window.
func (w *Window) GetGLXWindow() uintptr { return w.handle }

// GetX11Adapter returns the XRandR RROutput for the monitor's CRTC.
func (m *Monitor) GetX11Adapter() uintptr { return uintptr(m.crtc) }

// GetX11Monitor returns the XRandR RROutput ID for the monitor.
func (m *Monitor) GetX11Monitor() uintptr { return uintptr(m.output) }

// GetX11SelectionString returns the contents of the X11 PRIMARY selection
// (interpreted as the clipboard for compatibility with glfw upstream).
func GetX11SelectionString() string { return GetClipboardString() }

// SetX11SelectionString writes a string to the X11 PRIMARY selection.
func SetX11SelectionString(s string) { SetClipboardString(s) }
