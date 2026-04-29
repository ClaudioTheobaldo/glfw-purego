//go:build linux

package glfw

import (
	"runtime"
	"time"
)

// ----------------------------------------------------------------------------
// Monitor type — minimal stub for Linux
// ----------------------------------------------------------------------------

// Monitor represents a connected display.
type Monitor struct {
	name string
}

// GetName returns the monitor's device name.
func (m *Monitor) GetName() string { return m.name }

// GetPos returns the monitor's position in virtual screen coordinates.
func (m *Monitor) GetPos() (x, y int) { return 0, 0 }

// GetWorkarea returns the monitor's work area.
func (m *Monitor) GetWorkarea() (x, y, width, height int) { return 0, 0, 0, 0 }

// GetPhysicalSize returns the monitor's physical dimensions in millimetres.
func (m *Monitor) GetPhysicalSize() (widthMM, heightMM int) { return 0, 0 }

// GetContentScale returns the DPI scale factors.
func (m *Monitor) GetContentScale() (x, y float32) { return 1, 1 }

// GetVideoMode returns the monitor's current video mode.
func (m *Monitor) GetVideoMode() *VidMode { return nil }

// GetVideoModes returns all available video modes for this monitor.
func (m *Monitor) GetVideoModes() []*VidMode { return nil }

// GetMonitors returns all currently connected monitors.
func GetMonitors() ([]*Monitor, error) { return nil, nil }

// GetPrimaryMonitor returns the primary monitor.
func GetPrimaryMonitor() *Monitor { return nil }

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {}

// ----------------------------------------------------------------------------
// Init / Terminate / Time
// ----------------------------------------------------------------------------

var linuxInitTime time.Time

// Init initialises the GLFW subsystem on Linux.
// Locks the calling goroutine to its OS thread.
func Init() error {
	runtime.LockOSThread()
	linuxInitTime = time.Now()
	resetHints()
	if err := initX11Display(); err != nil {
		return err
	}
	return nil
}

// Terminate destroys all windows and closes the X11 display.
func Terminate() {
	windowByHandle.Range(func(k, v any) bool {
		w := v.(*Window)
		w.Destroy()
		return true
	})
	if x11Display != 0 {
		xCloseDisplay(x11Display)
		x11Display = 0
	}
}

// GetTime returns the elapsed time in seconds since Init was called.
func GetTime() float64 {
	return time.Since(linuxInitTime).Seconds()
}

// SetTime resets the timer base so that GetTime returns t immediately after.
func SetTime(t float64) {
	linuxInitTime = time.Now().Add(-time.Duration(t * float64(time.Second)))
}

// ----------------------------------------------------------------------------
// Stubs — still not implemented on Linux
// ----------------------------------------------------------------------------

// SetMonitor switches the window between fullscreen and windowed mode.
// Linux: not yet implemented.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {}

// SetAttrib sets a window attribute at runtime.
// Linux: not yet implemented.
func (w *Window) SetAttrib(attrib Hint, value int) {}

// SetIcon sets the window icon.
// Linux: not yet implemented.
func (w *Window) SetIcon(images []Image) {}

// SetCursor sets the cursor shape for this window.
// Linux: not yet implemented.
func (w *Window) SetCursor(cursor *Cursor) {}

// CreateCursor creates a custom cursor.
// Linux: returns nil (not yet implemented).
func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) { return nil, nil }

// CreateStandardCursor returns a standard system cursor.
// Linux: returns nil (not yet implemented).
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) { return nil, nil }

// DestroyCursor frees a cursor object.
func DestroyCursor(cursor *Cursor) {}

// JoystickPresent returns whether a joystick is connected.
func JoystickPresent(joy Joystick) bool { return false }

// GetJoystickAxes returns the joystick axis values.
func GetJoystickAxes(joy Joystick) []float32 { return nil }

// GetJoystickButtons returns the joystick button states.
func GetJoystickButtons(joy Joystick) []Action { return nil }

// GetJoystickHats returns the joystick hat states.
func GetJoystickHats(joy Joystick) []JoystickHatState { return nil }

// GetJoystickName returns the joystick name.
func GetJoystickName(joy Joystick) string { return "" }

// GetJoystickGUID returns the joystick GUID.
func GetJoystickGUID(joy Joystick) string { return "" }

// JoystickIsGamepad returns whether the joystick is a full gamepad.
func JoystickIsGamepad(joy Joystick) bool { return false }

// GetGamepadName returns the gamepad name.
func GetGamepadName(joy Joystick) string { return "" }

// GetGamepadState fills state with gamepad button/axis data.
func GetGamepadState(joy Joystick, state *GamepadState) bool { return false }

// UpdateGamepadMappings updates the gamepad mapping database.
func UpdateGamepadMappings(mappings string) bool { return false }

// SetJoystickCallback sets a callback for joystick connect/disconnect.
func SetJoystickCallback(cb func(joy Joystick, event PeripheralEvent)) {}

// ----------------------------------------------------------------------------
// New APIs — Linux stubs
// ----------------------------------------------------------------------------

// SetSizeLimits sets minimum/maximum window dimensions. Linux: not yet implemented.
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {}

// SetAspectRatio locks the resize aspect ratio. Linux: not yet implemented.
func (w *Window) SetAspectRatio(numer, denom int) {}

// GetOpacity returns the window opacity (always 1 on Linux stub).
func (w *Window) GetOpacity() float32 { return 1.0 }

// SetOpacity sets the window opacity. Linux: not yet implemented.
func (w *Window) SetOpacity(opacity float32) {}

// RequestAttention requests user attention. Linux: not yet implemented.
func (w *Window) RequestAttention() {}

// PostEmptyEvent wakes up a blocked WaitEvents. Linux: not yet implemented.
func PostEmptyEvent() {}

// GetKeyScancode returns the platform scancode for a key. Linux: returns -1.
func GetKeyScancode(key Key) int { return -1 }

// GetKeyName returns the localized name of a key. Linux: returns empty string.
func GetKeyName(key Key, scancode int) string { return "" }

// GetTimerFrequency returns the raw timer frequency. Linux: uses nanoseconds.
func GetTimerFrequency() uint64 { return 1_000_000_000 }

// GetTimerValue returns the raw timer counter. Linux: nanoseconds since Init.
func GetTimerValue() uint64 {
	return uint64(GetTime() * 1e9)
}

// GetWin32Adapter is Windows-only; returns empty string on Linux.
func (m *Monitor) GetWin32Adapter() string { return "" }

// GetWin32Monitor is Windows-only; returns empty string on Linux.
func (m *Monitor) GetWin32Monitor() string { return "" }

// ── Version / init hints ──────────────────────────────────────────────────────

// InitHint sets a hint for the next Init call.
// Stub — hint storage is not needed in the purego implementation.
func InitHint(hint Hint, value int) {}

// GetVersion returns the compile-time version of the GLFW library.
func GetVersion() (major, minor, revision int) { return 3, 3, 0 }

// GetVersionString returns a human-readable version string.
func GetVersionString() string { return "3.3.0 purego" }

// RawMouseMotionSupported reports whether raw (unscaled, unaccelerated) mouse
// motion is supported on the current platform.
func RawMouseMotionSupported() bool { return false }

// WindowHintString sets a string-valued window or context creation hint.
// Stub — no string hints are used in the purego implementation.
func WindowHintString(hint Hint, value string) {}

// ── Monitor gamma — Linux stubs ───────────────────────────────────────────────

// GetGammaRamp returns the monitor's gamma ramp.
// Linux: not yet implemented; returns nil.
func (m *Monitor) GetGammaRamp() *GammaRamp { return nil }

// SetGammaRamp sets the monitor's gamma ramp.
// Linux: not yet implemented.
func (m *Monitor) SetGammaRamp(ramp *GammaRamp) {}

// SetGamma sets the monitor's gamma by computing a standard power-law ramp.
// Linux: not yet implemented.
func (m *Monitor) SetGamma(gamma float32) {}

// ── Cursor.Destroy — Linux ────────────────────────────────────────────────────

// Destroy is a convenience method; it calls DestroyCursor(c).
func (c *Cursor) Destroy() { DestroyCursor(c) }

// ── Window.GetFrameSize — Linux stub ─────────────────────────────────────────

// GetFrameSize returns the size of each edge of the frame around the window's
// client area. Linux: returns zeros (stub).
func (w *Window) GetFrameSize() (left, top, right, bottom int) { return 0, 0, 0, 0 }

// GetWindowFrameSize is a package-level wrapper around (*Window).GetFrameSize.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) { return w.GetFrameSize() }
