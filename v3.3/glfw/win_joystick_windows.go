//go:build windows

package glfw

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ----------------------------------------------------------------------------
// XInput
// ----------------------------------------------------------------------------

const _XUSER_MAX_COUNT = 4

var (
	modXInput      *windows.LazyDLL
	xInputGetState *windows.LazyProc
	xInputGetCaps  *windows.LazyProc
	xInputOnce     sync.Once
	xInputLoaded   bool
)

type _XINPUT_GAMEPAD struct {
	WButtons      uint16
	BLeftTrigger  byte
	BRightTrigger byte
	SThumbLX      int16
	SThumbLY      int16
	SThumbRX      int16
	SThumbRY      int16
}

type _XINPUT_STATE struct {
	DwPacketNumber uint32
	Gamepad        _XINPUT_GAMEPAD
}

type _XINPUT_CAPABILITIES struct {
	Type      byte
	SubType   byte
	Flags     uint16
	Gamepad   _XINPUT_GAMEPAD
	Vibration [4]byte // XINPUT_VIBRATION
}

const (
	_ERROR_SUCCESS              = 0
	_ERROR_DEVICE_NOT_CONNECTED = 1167
	_XINPUT_DEVSUBTYPE_GAMEPAD  = 0x01
	// XInput button bitmasks
	_XINPUT_GAMEPAD_DPAD_UP        = 0x0001
	_XINPUT_GAMEPAD_DPAD_DOWN      = 0x0002
	_XINPUT_GAMEPAD_DPAD_LEFT      = 0x0004
	_XINPUT_GAMEPAD_DPAD_RIGHT     = 0x0008
	_XINPUT_GAMEPAD_START          = 0x0010
	_XINPUT_GAMEPAD_BACK           = 0x0020
	_XINPUT_GAMEPAD_LEFT_THUMB     = 0x0040
	_XINPUT_GAMEPAD_RIGHT_THUMB    = 0x0080
	_XINPUT_GAMEPAD_LEFT_SHOULDER  = 0x0100
	_XINPUT_GAMEPAD_RIGHT_SHOULDER = 0x0200
	_XINPUT_GAMEPAD_A              = 0x1000
	_XINPUT_GAMEPAD_B              = 0x2000
	_XINPUT_GAMEPAD_X              = 0x4000
	_XINPUT_GAMEPAD_Y              = 0x8000
)

func loadXInput() {
	xInputOnce.Do(func() {
		for _, name := range []string{"xinput1_4.dll", "xinput1_3.dll", "xinput9_1_0.dll"} {
			modXInput = windows.NewLazySystemDLL(name)
			if err := modXInput.Load(); err == nil {
				xInputGetState = modXInput.NewProc("XInputGetState")
				xInputGetCaps  = modXInput.NewProc("XInputGetCapabilities")
				xInputLoaded = true
				return
			}
		}
	})
}

func xinputGetState(userIndex uint32) (_XINPUT_STATE, bool) {
	loadXInput()
	if !xInputLoaded {
		return _XINPUT_STATE{}, false
	}
	var state _XINPUT_STATE
	r, _, _ := xInputGetState.Call(uintptr(userIndex), uintptr(unsafe.Pointer(&state)))
	return state, r == _ERROR_SUCCESS
}

func xinputGetCaps(userIndex uint32) (_XINPUT_CAPABILITIES, bool) {
	loadXInput()
	if !xInputLoaded {
		return _XINPUT_CAPABILITIES{}, false
	}
	var caps _XINPUT_CAPABILITIES
	r, _, _ := xInputGetCaps.Call(uintptr(userIndex), 0, uintptr(unsafe.Pointer(&caps)))
	return caps, r == _ERROR_SUCCESS
}

// xInputIndex maps Joystick (0-15) to XInput user index (0-3).
// Returns false if the joystick is out of XInput's range.
func xInputIndex(joy Joystick) (uint32, bool) {
	if joy < 0 || joy >= 4 {
		return 0, false
	}
	return uint32(joy), true
}

// JoystickPresent returns true if the given joystick is connected.
func JoystickPresent(joy Joystick) bool {
	idx, ok := xInputIndex(joy)
	if !ok {
		return false
	}
	_, present := xinputGetState(idx)
	return present
}

// GetJoystickAxes returns the axis values for the given joystick.
// Returns nil if not connected.
func GetJoystickAxes(joy Joystick) []float32 {
	idx, ok := xInputIndex(joy)
	if !ok {
		return nil
	}
	state, present := xinputGetState(idx)
	if !present {
		return nil
	}
	gp := state.Gamepad
	return []float32{
		normAxisShort(gp.SThumbLX),
		-normAxisShort(gp.SThumbLY), // Y axis flipped for GLFW convention
		normAxisShort(gp.SThumbRX),
		-normAxisShort(gp.SThumbRY),
		normAxisTrigger(gp.BLeftTrigger),
		normAxisTrigger(gp.BRightTrigger),
	}
}

// GetJoystickButtons returns the button states for the given joystick.
// Returns nil if not connected.
func GetJoystickButtons(joy Joystick) []Action {
	idx, ok := xInputIndex(joy)
	if !ok {
		return nil
	}
	state, present := xinputGetState(idx)
	if !present {
		return nil
	}
	bits := state.Gamepad.WButtons
	btns := []struct{ mask uint16 }{
		{_XINPUT_GAMEPAD_A},
		{_XINPUT_GAMEPAD_B},
		{_XINPUT_GAMEPAD_X},
		{_XINPUT_GAMEPAD_Y},
		{_XINPUT_GAMEPAD_LEFT_SHOULDER},
		{_XINPUT_GAMEPAD_RIGHT_SHOULDER},
		{_XINPUT_GAMEPAD_BACK},
		{_XINPUT_GAMEPAD_START},
		{_XINPUT_GAMEPAD_LEFT_THUMB},
		{_XINPUT_GAMEPAD_RIGHT_THUMB},
		{_XINPUT_GAMEPAD_DPAD_UP},
		{_XINPUT_GAMEPAD_DPAD_RIGHT},
		{_XINPUT_GAMEPAD_DPAD_DOWN},
		{_XINPUT_GAMEPAD_DPAD_LEFT},
	}
	out := make([]Action, len(btns))
	for i, b := range btns {
		if bits&b.mask != 0 {
			out[i] = Press
		}
	}
	return out
}

// GetJoystickHats returns d-pad hat states. XInput has one hat.
func GetJoystickHats(joy Joystick) []JoystickHatState {
	idx, ok := xInputIndex(joy)
	if !ok {
		return nil
	}
	state, present := xinputGetState(idx)
	if !present {
		return nil
	}
	bits := state.Gamepad.WButtons
	var hat JoystickHatState
	if bits&_XINPUT_GAMEPAD_DPAD_UP != 0 {
		hat |= HatUp
	}
	if bits&_XINPUT_GAMEPAD_DPAD_DOWN != 0 {
		hat |= HatDown
	}
	if bits&_XINPUT_GAMEPAD_DPAD_LEFT != 0 {
		hat |= HatLeft
	}
	if bits&_XINPUT_GAMEPAD_DPAD_RIGHT != 0 {
		hat |= HatRight
	}
	return []JoystickHatState{hat}
}

// GetJoystickName returns the joystick name.
func GetJoystickName(joy Joystick) string {
	if JoystickIsGamepad(joy) {
		return "Xbox Controller"
	}
	return ""
}

// GetJoystickGUID returns a platform-specific GUID string for the joystick.
func GetJoystickGUID(joy Joystick) string {
	if JoystickPresent(joy) {
		return "xinput"
	}
	return ""
}

// JoystickIsGamepad returns true if the joystick is a full gamepad (XInput device).
func JoystickIsGamepad(joy Joystick) bool {
	idx, ok := xInputIndex(joy)
	if !ok {
		return false
	}
	caps, ok := xinputGetCaps(idx)
	return ok && caps.SubType == _XINPUT_DEVSUBTYPE_GAMEPAD
}

// GetGamepadName returns the human-readable name of the gamepad.
func GetGamepadName(joy Joystick) string {
	return GetJoystickName(joy)
}

// GetGamepadState fills state with the current gamepad button/axis state.
// Returns false if the joystick is not connected or not a gamepad.
func GetGamepadState(joy Joystick, state *GamepadState) bool {
	axes := GetJoystickAxes(joy)
	btns := GetJoystickButtons(joy)
	if axes == nil || btns == nil {
		return false
	}
	// Axes: LeftX, LeftY, RightX, RightY, LeftTrigger, RightTrigger
	for i, a := range axes {
		if i < len(state.Axes) {
			state.Axes[i] = a
		}
	}
	// Buttons: A=0,B=1,X=2,Y=3,LB=4,RB=5,Back=6,Start=7,LThumb=8,RThumb=9,...
	for i, a := range btns {
		if i < len(state.Buttons) {
			state.Buttons[i] = a
		}
	}
	return true
}

// UpdateGamepadMappings updates the gamepad mapping database.
// On Windows XInput devices are always recognised; this is a no-op.
func UpdateGamepadMappings(mappings string) bool { return true }

// joystickCallback / joystickConnected drive poll-based connect/disconnect
// detection: WM_DEVICECHANGE requires a message-only window which doesn't fit
// our event loop, so we instead scan XInput slots in PollEvents and fire the
// callback on edge transitions.
var (
	joystickCallback  func(joy Joystick, event PeripheralEvent)
	joystickConnected [4]bool
)

// SetJoystickCallback sets a callback for joystick connect/disconnect events.
// The callback fires from inside PollEvents whenever an XInput slot's
// connection state changes from the previous poll.
func SetJoystickCallback(cb func(joy Joystick, event PeripheralEvent)) {
	joystickCallback = cb
	// Seed the connected state so the very first PollEvents doesn't spuriously
	// report all currently-connected pads as freshly Connected.
	for i := uint32(0); i < 4; i++ {
		_, present := xinputGetState(i)
		joystickConnected[i] = present
	}
}

// pollJoystickConnections re-scans XInput slots and fires the callback on
// state transitions.  Cheap (4 small syscalls); safe to call every frame.
func pollJoystickConnections() {
	if joystickCallback == nil {
		return
	}
	for i := uint32(0); i < 4; i++ {
		_, present := xinputGetState(i)
		if present != joystickConnected[i] {
			joystickConnected[i] = present
			ev := Disconnected
			if present {
				ev = Connected
			}
			joystickCallback(Joystick(i), ev)
		}
	}
}

// normAxisShort normalises a signed int16 thumb-stick value to [-1, 1].
func normAxisShort(v int16) float32 {
	if v >= 0 {
		return float32(v) / 32767.0
	}
	return float32(v) / 32768.0
}

// normAxisTrigger normalises a byte trigger value to [0, 1] then maps to [-1, 1]
// to match the GLFW convention for triggers.
func normAxisTrigger(v byte) float32 {
	return float32(v)/127.5 - 1.0
}
