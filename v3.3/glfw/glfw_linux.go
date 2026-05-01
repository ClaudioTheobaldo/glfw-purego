//go:build linux

package glfw

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// ----------------------------------------------------------------------------
// Monitor type — real implementation backed by XRandR (linux_x11_xrandr.go)
// ----------------------------------------------------------------------------

// Monitor represents a connected display.
type Monitor struct {
	name                string
	x, y                int
	widthPx, heightPx   int
	widthMM, heightMM   int
	modes               []*VidMode
	currentMode         *VidMode
	crtc, output        uint64 // XRandR RRCrtc / RROutput IDs
}

// GetName returns the monitor's display name as reported by XRandR.
func (m *Monitor) GetName() string { return m.name }

// GetPos returns the monitor's position in virtual screen coordinates.
func (m *Monitor) GetPos() (x, y int) { return m.x, m.y }

// GetWorkarea returns the monitor's usable area.
// On basic X11 we return the full monitor bounds (no taskbar subtraction).
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

// ----------------------------------------------------------------------------
// SetMonitorCallback — monitor connect/disconnect via XRandR RRNotify
// ----------------------------------------------------------------------------

var (
	linuxMonitorCb      func(*Monitor, PeripheralEvent)
	linuxCachedMonitors []*Monitor
)

const _RROutputChangeNotifyMask = int32(1)

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

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
// Requires XRandR; silently ignored if libXrandr is not available.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	linuxMonitorCb = cb
	if cb != nil && x11Display != 0 {
		if loadXrandr() == nil {
			xrrSelectInput(x11Display, x11Root, _RROutputChangeNotifyMask)
		}
		linuxCachedMonitors, _ = GetMonitors()
	}
}

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
		v.(*Window).Destroy()
		return true
	})
	// Free shared invisible cursor before closing the display.
	if x11InvisibleCursor != 0 && x11Display != 0 {
		xFreeCursor(x11Display, x11InvisibleCursor)
		x11InvisibleCursor = 0
	}
	if x11Display != 0 {
		xCloseDisplay(x11Display)
		x11Display = 0
	}
	// Close self-pipe after the display so no racing WaitEvents can write to it.
	if x11PostPipeRead >= 0 {
		syscall.Close(x11PostPipeRead)
		x11PostPipeRead = -1
	}
	if x11PostPipeWrite >= 0 {
		syscall.Close(x11PostPipeWrite)
		x11PostPipeWrite = -1
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
// _MotifWMHints — used by SetAttrib(Decorated, …)
// Each field is a C long (8 bytes on 64-bit Linux); 5 fields = 40 bytes total.
// ----------------------------------------------------------------------------

type _MotifWMHints struct {
	Flags       uint64
	Functions   uint64
	Decorations uint64
	InputMode   int64
	Status      uint64
}

const _MWM_HINTS_DECORATIONS = uint64(2)

// ----------------------------------------------------------------------------
// Window attribute / fullscreen / icon
// ----------------------------------------------------------------------------

// SetMonitor switches the window between fullscreen and windowed mode.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if monitor != nil {
		// Save windowed state if not already fullscreen
		if w.fsMonitor == nil {
			var wa _XWindowAttributes
			xGetWindowAttributes(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&wa)))
			w.savedX = int(wa.X)
			w.savedY = int(wa.Y)
			w.savedW = int(wa.Width)
			w.savedH = int(wa.Height)
		}
		w.fsMonitor = monitor
		// Prefer the monitor's real geometry when available.
		if monitor.widthPx > 0 {
			xpos, ypos = monitor.x, monitor.y
			width, height = monitor.widthPx, monitor.heightPx
		}
		// Move/resize to requested geometry first, then request fullscreen state
		xMoveResizeWindow(x11Display, uint64(w.handle), int32(xpos), int32(ypos), uint32(width), uint32(height))
		w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateFull, 0)
	} else {
		// Exit fullscreen
		w.fsMonitor = nil
		w.sendNETWMState(_NET_WM_STATE_REMOVE, atomNETWMStateFull, 0)
		xMoveResizeWindow(x11Display, uint64(w.handle), int32(w.savedX), int32(w.savedY), uint32(w.savedW), uint32(w.savedH))
	}
	xFlush(x11Display)
}

// SetAttrib sets a window attribute at runtime.
func (w *Window) SetAttrib(attrib Hint, value int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	switch attrib {
	case Decorated:
		var hints _MotifWMHints
		hints.Flags = _MWM_HINTS_DECORATIONS
		if value != 0 {
			hints.Decorations = 1
		}
		// type = atomMOTIFWMHints (self-typed), format=32 (C long = 8 bytes), 5 longs
		xChangeProperty(x11Display, uint64(w.handle),
			atomMOTIFWMHints, atomMOTIFWMHints,
			32, 0,
			uintptr(unsafe.Pointer(&hints)),
			5)
	case Floating:
		if value != 0 {
			w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateAbove, 0)
		} else {
			w.sendNETWMState(_NET_WM_STATE_REMOVE, atomNETWMStateAbove, 0)
		}
	}
	xFlush(x11Display)
}

// SetIcon sets the window icon from a slice of candidate images.
// Pass nil or an empty slice to remove the icon.
func (w *Window) SetIcon(images []Image) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if len(images) == 0 {
		xDeleteProperty(x11Display, uint64(w.handle), atomNETWMIcon)
		xFlush(x11Display)
		return
	}

	// Build CARDINAL array: for each image [ width, height, ARGB pixels… ]
	// On 64-bit Linux, format=32 ⇒ each element is an 8-byte C long.
	var data []uint64
	for i := range images {
		img := &images[i]
		data = append(data, uint64(img.Width), uint64(img.Height))
		for j := 0; j < img.Width*img.Height; j++ {
			r := uint64(img.Pixels[j*4+0])
			g := uint64(img.Pixels[j*4+1])
			b := uint64(img.Pixels[j*4+2])
			a := uint64(img.Pixels[j*4+3])
			data = append(data, (a<<24)|(r<<16)|(g<<8)|b)
		}
	}

	const xaCARDINAL = uint64(6)
	xChangeProperty(x11Display, uint64(w.handle),
		atomNETWMIcon, xaCARDINAL,
		32, 0,
		uintptr(unsafe.Pointer(&data[0])),
		int32(len(data)))
	runtime.KeepAlive(data)
	xFlush(x11Display)
}

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

// applyWMNormalHints pushes the window's size/aspect constraints to the WM
// via XSetWMNormalHints.  Call whenever minW/maxW/aspectNum change.
func (w *Window) applyWMNormalHints() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	var hints _XSizeHints
	if w.minW > 0 || w.minH > 0 {
		hints.Flags |= _PMinSize
		hints.MinWidth = int32(w.minW)
		hints.MinHeight = int32(w.minH)
	}
	if w.maxW > 0 || w.maxH > 0 {
		hints.Flags |= _PMaxSize
		hints.MaxWidth = int32(w.maxW)
		hints.MaxHeight = int32(w.maxH)
	}
	if w.aspectNum > 0 && w.aspectDen > 0 {
		hints.Flags |= _PAspect
		hints.MinAspX = int32(w.aspectNum)
		hints.MinAspY = int32(w.aspectDen)
		hints.MaxAspX = int32(w.aspectNum)
		hints.MaxAspY = int32(w.aspectDen)
	}
	if hints.Flags == 0 {
		return
	}
	xSetWMNormalHints(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&hints)))
	xFlush(x11Display)
}

// SetSizeLimits sets minimum/maximum window dimensions.
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {
	w.minW, w.minH, w.maxW, w.maxH = minWidth, minHeight, maxWidth, maxHeight
	w.applyWMNormalHints()
}

// SetAspectRatio locks the resize aspect ratio (numer:denom). Pass 0,0 to clear.
func (w *Window) SetAspectRatio(numer, denom int) {
	w.aspectNum, w.aspectDen = numer, denom
	w.applyWMNormalHints()
}

const _netWMWindowOpacityMax = uint64(0xFFFFFFFF)

// GetOpacity returns the window opacity in the range [0, 1].
// Reads _NET_WM_WINDOW_OPACITY from the window property.
func (w *Window) GetOpacity() float32 {
	if x11Display == 0 || w.handle == 0 {
		return 1.0
	}
	var actualType uint64
	var actualFormat int32
	var nItems, bytesAfter uint64
	var propPtr uintptr
	ret := xGetWindowProperty(x11Display, uint64(w.handle),
		atomNETWMWindowOpacity,
		0, 1, 0,
		6, // XA_CARDINAL
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&propPtr)))
	if ret != 0 || propPtr == 0 || nItems < 1 {
		return 1.0
	}
	// Xlib sign-extends format=32 CARD32 values to native long (8 bytes) on
	// 64-bit systems. Mask to the lower 32 bits to recover the original CARDINAL.
	val := *(*uint64)(unsafe.Pointer(propPtr)) & 0xFFFFFFFF
	xFree(propPtr)
	return float32(val) / float32(_netWMWindowOpacityMax)
}

// SetOpacity sets the window opacity. A value of 1.0 removes the property
// (fully opaque); values < 1.0 set _NET_WM_WINDOW_OPACITY.
func (w *Window) SetOpacity(opacity float32) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if opacity >= 1.0 {
		xDeleteProperty(x11Display, uint64(w.handle), atomNETWMWindowOpacity)
	} else {
		val := uint64(opacity * float32(_netWMWindowOpacityMax))
		xChangeProperty(x11Display, uint64(w.handle),
			atomNETWMWindowOpacity, 6, // XA_CARDINAL
			32, 0,
			uintptr(unsafe.Pointer(&val)),
			1)
	}
	xFlush(x11Display)
}

// RequestAttention requests user attention via _NET_WM_STATE_DEMANDS_ATTENTION.
func (w *Window) RequestAttention() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateDemandsAttention, 0)
	xFlush(x11Display)
}


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

// ── Monitor gamma — implemented in linux_x11_gamma.go ────────────────────────

// ── Window.GetFrameSize — reads _NET_FRAME_EXTENTS from the WM ───────────────

// GetFrameSize returns the size of the decorations around the window's client
// area as reported by the WM via _NET_FRAME_EXTENTS. Returns zeros if the
// property is not available yet (e.g. before the window has been decorated).
func (w *Window) GetFrameSize() (left, top, right, bottom int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	// Request that the WM set the property if it hasn't already, then flush.
	var reqEv _XClientMessageEvent
	reqEv.Type = _ClientMessage
	reqEv.Window = uint64(w.handle)
	reqEv.MessageType = atomNETRequestFrameExtents
	reqEv.Format = 32
	xSendEvent(x11Display, x11Root, 0,
		_SubstructureNotifyMask|_SubstructureRedirectMask,
		uintptr(unsafe.Pointer(&reqEv)))
	xSync(x11Display, 0)

	var actualType uint64
	var actualFormat int32
	var nItems, bytesAfter uint64
	var propPtr uintptr

	ret := xGetWindowProperty(x11Display, uint64(w.handle),
		atomNETFrameExtents,
		0, 4, 0,
		6, // XA_CARDINAL
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&propPtr)))

	if ret != 0 || propPtr == 0 || nItems < 4 {
		return
	}
	// _NET_FRAME_EXTENTS: left, right, top, bottom — each a CARDINAL (uint32).
	// Xlib sign-extends format=32 values to 64-bit longs; mask to recover the
	// original 32-bit unsigned value.
	data := (*[4]uint64)(unsafe.Pointer(propPtr))
	left   = int(data[0] & 0xFFFFFFFF)
	right  = int(data[1] & 0xFFFFFFFF)
	top    = int(data[2] & 0xFFFFFFFF)
	bottom = int(data[3] & 0xFFFFFFFF)
	xFree(propPtr)
	return
}

// GetWindowFrameSize is a package-level wrapper around (*Window).GetFrameSize.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) { return w.GetFrameSize() }
