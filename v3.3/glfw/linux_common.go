//go:build linux

package glfw

import "time"

// ----------------------------------------------------------------------------
// Monitor type — shared between X11 and Wayland backends.
// X11-specific fields (crtc, output) are zero on Wayland.
// Wayland-specific field (outputID) is zero on X11.
// ----------------------------------------------------------------------------

// Monitor represents a connected display.
type Monitor struct {
	name              string
	x, y              int
	widthPx, heightPx int
	widthMM, heightMM int
	modes             []*VidMode
	currentMode       *VidMode
	crtc, output      uint64 // XRandR RRCrtc / RROutput IDs (X11 only)
	outputID          uint32 // wl_output object ID (Wayland only)
}

// GetName returns the monitor's display name.
func (m *Monitor) GetName() string { return m.name }

// GetPos returns the monitor's position in virtual screen coordinates.
func (m *Monitor) GetPos() (x, y int) { return m.x, m.y }

// GetWorkarea returns the monitor's usable area.
func (m *Monitor) GetWorkarea() (x, y, width, height int) {
	return m.x, m.y, m.widthPx, m.heightPx
}

// GetPhysicalSize returns the monitor's physical dimensions in millimetres.
func (m *Monitor) GetPhysicalSize() (widthMM, heightMM int) {
	return m.widthMM, m.heightMM
}

// GetContentScale returns the DPI scale factors relative to 96 DPI.
func (m *Monitor) GetContentScale() (x, y float32) {
	if m.widthPx == 0 || m.widthMM == 0 {
		return 1, 1
	}
	const refDPI = 96.0
	dpiX := float32(m.widthPx) / (float32(m.widthMM) / 25.4)
	dpiY := float32(m.heightPx) / (float32(m.heightMM) / 25.4)
	return dpiX / refDPI, dpiY / refDPI
}

// GetVideoMode returns the monitor's current video mode.
func (m *Monitor) GetVideoMode() *VidMode { return m.currentMode }

// GetVideoModes returns all available video modes for this monitor.
func (m *Monitor) GetVideoModes() []*VidMode { return m.modes }

// GetWin32Adapter is Windows-only; returns empty string on Linux.
func (m *Monitor) GetWin32Adapter() string { return "" }

// GetWin32Monitor is Windows-only; returns empty string on Linux.
func (m *Monitor) GetWin32Monitor() string { return "" }

// ----------------------------------------------------------------------------
// diffAndFireMonitorCallbacks — shared helper
// ----------------------------------------------------------------------------

// diffAndFireMonitorCallbacks compares two monitor lists by name and fires
// Connected/Disconnected for monitors that appeared or disappeared.
func diffAndFireMonitorCallbacks(old, cur []*Monitor, cb func(*Monitor, PeripheralEvent)) {
	oldSet := make(map[string]*Monitor, len(old))
	for _, m := range old {
		oldSet[m.name] = m
	}
	curSet := make(map[string]*Monitor, len(cur))
	for _, m := range cur {
		curSet[m.name] = m
	}
	for _, m := range cur {
		if _, existed := oldSet[m.name]; !existed {
			cb(m, Connected)
		}
	}
	for _, m := range old {
		if _, exists := curSet[m.name]; !exists {
			cb(m, Disconnected)
		}
	}
}

// ----------------------------------------------------------------------------
// Shared Linux timer
// ----------------------------------------------------------------------------

var linuxInitTime time.Time

// GetTime returns the elapsed time in seconds since Init was called.
func GetTime() float64 {
	return time.Since(linuxInitTime).Seconds()
}

// SetTime resets the timer base so that GetTime returns t immediately after.
func SetTime(t float64) {
	linuxInitTime = time.Now().Add(-time.Duration(t * float64(time.Second)))
}

// GetTimerFrequency returns the raw timer frequency. Linux uses nanoseconds.
func GetTimerFrequency() uint64 { return 1_000_000_000 }

// GetTimerValue returns the raw timer counter.
func GetTimerValue() uint64 { return uint64(GetTime() * 1e9) }

// ----------------------------------------------------------------------------
// Shared stubs (identical on X11 and Wayland)
// ----------------------------------------------------------------------------

// InitHint sets a hint for the next Init call. Stub — no-op.
func InitHint(hint Hint, value int) {}

// GetVersion returns the compile-time version of the GLFW library.
func GetVersion() (major, minor, revision int) { return 3, 3, 0 }

// GetVersionString returns a human-readable version string.
func GetVersionString() string { return "3.3.0 purego" }

// WindowHintString sets a string-valued window hint. Stub — no-op.
func WindowHintString(hint Hint, value string) {}

// GetKeyScancode returns the platform scancode for a key. Linux: returns -1.
func GetKeyScancode(key Key) int { return -1 }

// GetKeyName returns the localized name of a key. Linux: returns empty string.
func GetKeyName(key Key, scancode int) string { return "" }
