//go:build linux

package glfw

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// ── ioctl request codes for /dev/input/js* ──────────────────────────────────
//
// Linux ioctl encoding: (dir<<30)|(size<<16)|(type<<8)|nr
//   dir: read=2, write=1, rw=3
//   For JSIOC: type='j'=0x6a
//
// JSIOCGAXES     = _IOR('j', 0x11, 1)  = 0x80016a11
// JSIOCGBUTTONS  = _IOR('j', 0x12, 1)  = 0x80016a12
// JSIOCGID       = _IOR('j', 0x02, 8)  = 0x80086a02
// JSIOCGNAME(128)= _IOR('j', 0x13,128) = 0x80806a13

const (
	_JSIOCGAXES    = uintptr(0x80016a11)
	_JSIOCGBUTTONS = uintptr(0x80016a12)
	_JSIOCGID      = uintptr(0x80086a02)
	_JSIOCGNAME128 = uintptr(0x80806a13)

	_JS_EVENT_BUTTON = uint8(0x01)
	_JS_EVENT_AXIS   = uint8(0x02)
	_JS_EVENT_INIT   = uint8(0x80) // OR'd on synthetic init events
)

// inotify flags (inotify_init1 / inotify_add_watch).
const (
	_IN_NONBLOCK = uintptr(0x800)
	_IN_CLOEXEC  = uintptr(0x80000)
	_IN_CREATE   = uint32(0x100)
	_IN_DELETE   = uint32(0x200)
)

// ── Kernel structs ───────────────────────────────────────────────────────────

// jsEvent mirrors struct js_event (8 bytes).
type jsEvent struct {
	Time   uint32
	Value  int16
	Type   uint8
	Number uint8
}

// jsInputID mirrors struct input_id (8 bytes).
type jsInputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// ── Per-device state ─────────────────────────────────────────────────────────

type linuxJoystick struct {
	fd      int
	name    string
	guid    string
	axes    []float32
	buttons []Action
}

// ── Package-level state ──────────────────────────────────────────────────────

var (
	jsOnce            sync.Once
	jsEverInitialized bool
	jsMu              sync.Mutex
	joysticks         [int(JoystickLast) + 1]*linuxJoystick
	jsCb              JoystickCallback

	inotifyFd = -1
)

// ── Initialisation ───────────────────────────────────────────────────────────

func initJoysticks() {
	jsOnce.Do(func() {
		// Set up inotify to detect hot-plug events on /dev/input.
		if fd, _, errno := syscall.RawSyscall(
			syscall.SYS_INOTIFY_INIT1,
			_IN_NONBLOCK|_IN_CLOEXEC, 0, 0,
		); errno == 0 {
			path, _ := syscall.BytePtrFromString("/dev/input")
			if _, _, errno2 := syscall.RawSyscall6(
				syscall.SYS_INOTIFY_ADD_WATCH,
				fd,
				uintptr(unsafe.Pointer(path)),
				uintptr(_IN_CREATE|_IN_DELETE), 0, 0, 0,
			); errno2 == 0 {
				inotifyFd = int(fd)
			}
		}
		// Probe all 16 slots for already-connected devices.
		for i := range int(JoystickLast) + 1 {
			probeJoystick(Joystick(i))
		}
		jsEverInitialized = true
	})
}

// ── Hot-plug helpers ─────────────────────────────────────────────────────────

// probeJoystick tries to open /dev/input/jsN and register it.
func probeJoystick(joy Joystick) {
	path := fmt.Sprintf("/dev/input/js%d", int(joy))
	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC,
		0,
	)
	if err != nil {
		return
	}

	// Device name (JSIOCGNAME)
	var nameBuf [128]byte
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), _JSIOCGNAME128, uintptr(unsafe.Pointer(&nameBuf[0])))
	name := jsNullStr(nameBuf[:])

	// Axis and button counts
	var nAxes, nButtons uint8
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), _JSIOCGAXES, uintptr(unsafe.Pointer(&nAxes)))
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), _JSIOCGBUTTONS, uintptr(unsafe.Pointer(&nButtons)))

	// GUID from bus/vendor/product/version
	var id jsInputID
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), _JSIOCGID, uintptr(unsafe.Pointer(&id)))
	guid := fmt.Sprintf("%04x%04x%04x%04x00000000",
		id.Bustype, id.Vendor, id.Product, id.Version)

	// Drain the init events the driver sends on open.
	j := &linuxJoystick{
		fd:      fd,
		name:    name,
		guid:    guid,
		axes:    make([]float32, nAxes),
		buttons: make([]Action, nButtons),
	}
	drainJoystickEvents(j) // consume JS_EVENT_INIT events

	jsMu.Lock()
	joysticks[joy] = j
	jsMu.Unlock()

	if jsCb != nil {
		jsCb(joy, Connected)
	}
}

// removeJoystick unregisters and closes a joystick.
func removeJoystick(joy Joystick) {
	jsMu.Lock()
	j := joysticks[joy]
	joysticks[joy] = nil
	jsMu.Unlock()
	if j == nil {
		return
	}
	syscall.Close(j.fd)
	if jsCb != nil {
		jsCb(joy, Disconnected)
	}
}

// ── Event polling ────────────────────────────────────────────────────────────

// pollJoystickEvents drains pending events for all joysticks and checks inotify.
// Called from PollEvents() once joystick scanning has been initialised.
func pollJoystickEvents() {
	if inotifyFd >= 0 {
		checkJoystickInotify()
	}

	// Snapshot connected joysticks without holding the lock during I/O.
	jsMu.Lock()
	var active [int(JoystickLast) + 1]*linuxJoystick
	copy(active[:], joysticks[:])
	jsMu.Unlock()

	for i, j := range active {
		if j == nil {
			continue
		}
		if !drainJoystickEvents(j) {
			removeJoystick(Joystick(i))
		}
	}
}

// drainJoystickEvents reads all pending js_events (non-blocking).
// Returns false if the device appears disconnected.
func drainJoystickEvents(j *linuxJoystick) bool {
	var ev jsEvent
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&ev)), unsafe.Sizeof(ev))
	for {
		n, err := syscall.Read(j.fd, buf)
		if n < len(buf) || err != nil {
			return err == syscall.EAGAIN || err == syscall.EWOULDBLOCK
		}
		typ := ev.Type &^ _JS_EVENT_INIT
		switch typ {
		case _JS_EVENT_AXIS:
			if int(ev.Number) < len(j.axes) {
				v := float32(ev.Value) / 32767.0
				if v < -1 {
					v = -1
				}
				j.axes[ev.Number] = v
			}
		case _JS_EVENT_BUTTON:
			if int(ev.Number) < len(j.buttons) {
				if ev.Value != 0 {
					j.buttons[ev.Number] = Press
				} else {
					j.buttons[ev.Number] = Release
				}
			}
		}
	}
}

// checkJoystickInotify processes pending inotify events for /dev/input.
func checkJoystickInotify() {
	var buf [4096]byte
	n, err := syscall.Read(inotifyFd, buf[:])
	if n <= 0 || err != nil {
		return
	}
	// inotify_event layout: wd(4) + mask(4) + cookie(4) + len(4) + name(len)
	for off := 0; off+16 <= n; {
		mask := *(*uint32)(unsafe.Pointer(&buf[off+4]))
		nameLen := int(*(*uint32)(unsafe.Pointer(&buf[off+12])))
		if off+16+nameLen > n {
			break
		}
		name := jsNullStr(buf[off+16 : off+16+nameLen])
		off += 16 + nameLen

		var idx int = -1
		fmt.Sscanf(name, "js%d", &idx)
		if idx < 0 || idx > int(JoystickLast) {
			continue
		}
		joy := Joystick(idx)
		if mask&_IN_CREATE != 0 {
			probeJoystick(joy)
		} else if mask&_IN_DELETE != 0 {
			removeJoystick(joy)
		}
	}
}

// jsNullStr converts a C-string byte slice (null-terminated or full) to a Go string.
func jsNullStr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// ── Public API ───────────────────────────────────────────────────────────────

// JoystickPresent returns true if the given joystick slot is connected.
func JoystickPresent(joy Joystick) bool {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return false
	}
	initJoysticks()
	pollJoystickEvents()
	jsMu.Lock()
	defer jsMu.Unlock()
	return joysticks[joy] != nil
}

// GetJoystickAxes returns normalised axis values in [-1, 1].
// Returns nil if the joystick is not connected.
func GetJoystickAxes(joy Joystick) []float32 {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return nil
	}
	initJoysticks()
	pollJoystickEvents()
	jsMu.Lock()
	defer jsMu.Unlock()
	j := joysticks[joy]
	if j == nil {
		return nil
	}
	out := make([]float32, len(j.axes))
	copy(out, j.axes)
	return out
}

// GetJoystickButtons returns button press/release states.
// Returns nil if the joystick is not connected.
func GetJoystickButtons(joy Joystick) []Action {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return nil
	}
	initJoysticks()
	pollJoystickEvents()
	jsMu.Lock()
	defer jsMu.Unlock()
	j := joysticks[joy]
	if j == nil {
		return nil
	}
	out := make([]Action, len(j.buttons))
	copy(out, j.buttons)
	return out
}

// GetJoystickHats returns hat (d-pad) states.
// The /dev/input/js* interface does not expose hats as separate events;
// they are typically reported as axes. Returns an empty (non-nil) slice for
// connected devices, nil for disconnected ones.
func GetJoystickHats(joy Joystick) []JoystickHatState {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return nil
	}
	initJoysticks()
	jsMu.Lock()
	j := joysticks[joy]
	jsMu.Unlock()
	if j == nil {
		return nil
	}
	return []JoystickHatState{} // js interface has no dedicated hat events
}

// GetJoystickName returns the human-readable device name, or "" if not connected.
func GetJoystickName(joy Joystick) string {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return ""
	}
	initJoysticks()
	jsMu.Lock()
	defer jsMu.Unlock()
	if j := joysticks[joy]; j != nil {
		return j.name
	}
	return ""
}

// GetJoystickGUID returns a GUID string derived from the device bus/vendor/product/version.
// Returns "" if not connected.
func GetJoystickGUID(joy Joystick) string {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return ""
	}
	initJoysticks()
	jsMu.Lock()
	defer jsMu.Unlock()
	if j := joysticks[joy]; j != nil {
		return j.guid
	}
	return ""
}

// JoystickIsGamepad returns true if the device has enough axes (≥4) and
// buttons (≥10) to be treated as a full gamepad.
func JoystickIsGamepad(joy Joystick) bool {
	if joy < 0 || int(joy) > int(JoystickLast) {
		return false
	}
	initJoysticks()
	jsMu.Lock()
	defer jsMu.Unlock()
	j := joysticks[joy]
	return j != nil && len(j.axes) >= 4 && len(j.buttons) >= 10
}

// GetGamepadName returns the gamepad name (same as the joystick name).
func GetGamepadName(joy Joystick) string {
	return GetJoystickName(joy)
}

// GetGamepadState fills state with the current gamepad axis/button snapshot.
// Returns false if the device is not connected.
func GetGamepadState(joy Joystick, state *GamepadState) bool {
	axes := GetJoystickAxes(joy)
	btns := GetJoystickButtons(joy)
	if axes == nil || btns == nil {
		return false
	}
	for i, a := range axes {
		if i < len(state.Axes) {
			state.Axes[i] = a
		}
	}
	for i, b := range btns {
		if i < len(state.Buttons) {
			state.Buttons[i] = b
		}
	}
	return true
}

// UpdateGamepadMappings accepts SDL-compatible mapping strings.
// The /dev/input/js* implementation does not use external mappings; this is a no-op.
func UpdateGamepadMappings(_ string) bool { return true }

// SetJoystickCallback registers a callback for joystick connect/disconnect events.
// Hot-plug events are delivered during PollEvents once joysticks are initialised.
func SetJoystickCallback(cb func(joy Joystick, event PeripheralEvent)) {
	initJoysticks()
	jsMu.Lock()
	jsCb = cb
	jsMu.Unlock()
}
