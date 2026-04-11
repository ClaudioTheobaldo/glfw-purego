//go:build windows

package glfw

// GetWin32Window returns the Win32 HWND handle for the window.
// The returned value is an HWND cast to uintptr.
func (w *Window) GetWin32Window() uintptr {
	return w.handle
}

// GetWGLContext returns the WGL rendering context (HGLRC) for the window.
// The returned value is an HGLRC cast to uintptr.
func (w *Window) GetWGLContext() uintptr {
	return w.hglrc
}

// GetWin32Adapter returns the adapter name for the given monitor.
func (m *Monitor) GetWin32Adapter() string {
	return m.name
}

// GetWin32Monitor returns the display name for the given monitor.
func (m *Monitor) GetWin32Monitor() string {
	return m.name
}
