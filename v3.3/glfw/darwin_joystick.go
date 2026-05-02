//go:build darwin

// darwin_joystick.go — Joystick / gamepad support via GameController.framework.
//
// Uses GCController (ObjC API) to enumerate connected controllers and read
// their state.  Connect/disconnect detection is done by polling in PollEvents
// rather than through NSNotification observers, which avoids the complexity of
// registering an extra ObjC class.
//
// Slot management: GLFW Joystick 0-15 map to darwinJoystickSlots[0-15].
// A slot is occupied when slot.handle != 0 (a retained GCController*).

package glfw

import (
	"fmt"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ── SEL cache (GameController-specific) ──────────────────────────────────────

var (
	selGCControllers = objc.RegisterName("controllers") // class method on GCController

	// GCController properties
	selGCVendorName      = objc.RegisterName("vendorName")
	selGCExtendedGamepad = objc.RegisterName("extendedGamepad")

	// GCExtendedGamepad properties
	selGCLeftThumbstick        = objc.RegisterName("leftThumbstick")
	selGCRightThumbstick       = objc.RegisterName("rightThumbstick")
	selGCLeftTrigger           = objc.RegisterName("leftTrigger")
	selGCRightTrigger          = objc.RegisterName("rightTrigger")
	selGCLeftShoulder          = objc.RegisterName("leftShoulder")
	selGCRightShoulder         = objc.RegisterName("rightShoulder")
	selGCButtonA               = objc.RegisterName("buttonA")
	selGCButtonB               = objc.RegisterName("buttonB")
	selGCButtonX               = objc.RegisterName("buttonX")
	selGCButtonY               = objc.RegisterName("buttonY")
	selGCDpad                  = objc.RegisterName("dpad")
	selGCLeftThumbstickButton  = objc.RegisterName("leftThumbstickButton")
	selGCRightThumbstickButton = objc.RegisterName("rightThumbstickButton")
	selGCButtonMenu            = objc.RegisterName("buttonMenu")    // macOS 11+
	selGCButtonOptions         = objc.RegisterName("buttonOptions") // macOS 11+
	selGCButtonHome            = objc.RegisterName("buttonHome")    // macOS 12+

	// GCControllerDirectionPad properties
	selGCXAxis = objc.RegisterName("xAxis")
	selGCYAxis = objc.RegisterName("yAxis")
	selGCUp    = objc.RegisterName("up")
	selGCDown  = objc.RegisterName("down")
	selGCLeft  = objc.RegisterName("left")
	selGCRight = objc.RegisterName("right")

	// GCControllerAxisInput / GCControllerButtonInput
	selGCValue     = objc.RegisterName("value")
	selGCIsPressed = objc.RegisterName("isPressed")
)

// ── Slot state ────────────────────────────────────────────────────────────────

type darwinJoystickSlot struct {
	handle uintptr // retained GCController*; 0 = empty
	name   string
}

var (
	darwinJoystickSlots [16]darwinJoystickSlot
	darwinGCInited      bool
)

// initGameController loads GameController.framework.
// The dlopen call registers all GCController ObjC classes with the runtime.
// No C function bindings are needed — we use ObjC messaging only.
func initGameController() {
	_, err := purego.Dlopen(
		"/System/Library/Frameworks/GameController.framework/GameController",
		purego.RTLD_GLOBAL|purego.RTLD_LAZY,
	)
	if err != nil {
		return // framework not available (rare)
	}
	darwinGCInited = true
}

// ── Polling ───────────────────────────────────────────────────────────────────

// pollJoysticks synchronises darwinJoystickSlots with [GCController controllers].
// Called from PollEvents on every tick.
func pollJoysticks() {
	if !darwinGCInited {
		return
	}

	arr := objc.ID(objc.GetClass("GCController")).Send(selGCControllers)
	n := objc.Send[uint64](arr, selCount)

	// Build set of currently-connected controller handles.
	current := make(map[uintptr]struct{}, n)
	for i := uint64(0); i < n; i++ {
		c := arr.Send(selObjectAtIndex, i)
		current[uintptr(c)] = struct{}{}
	}

	// Detect disconnections: slot occupied but handle gone.
	for slot := range darwinJoystickSlots {
		h := darwinJoystickSlots[slot].handle
		if h == 0 {
			continue
		}
		if _, ok := current[h]; ok {
			continue
		}
		objc.ID(h).Send(selRelease)
		darwinJoystickSlots[slot] = darwinJoystickSlot{}
		if darwinJoystickCb != nil {
			darwinJoystickCb(Joystick(slot), Disconnected)
		}
	}

	// Build set of already-assigned handles to skip in the next loop.
	assigned := make(map[uintptr]struct{}, n)
	for _, s := range darwinJoystickSlots {
		if s.handle != 0 {
			assigned[s.handle] = struct{}{}
		}
	}

	// Detect connections: handle in current but not yet assigned.
	for i := uint64(0); i < n; i++ {
		c := arr.Send(selObjectAtIndex, i)
		h := uintptr(c)
		if _, ok := assigned[h]; ok {
			continue
		}
		// Find the first empty slot.
		for slot := range darwinJoystickSlots {
			if darwinJoystickSlots[slot].handle != 0 {
				continue
			}
			c.Send(selRetain)
			name := goStringFromNS(c.Send(selGCVendorName))
			if name == "" {
				name = fmt.Sprintf("Controller %d", slot+1)
			}
			darwinJoystickSlots[slot] = darwinJoystickSlot{handle: h, name: name}
			if darwinJoystickCb != nil {
				darwinJoystickCb(Joystick(slot), Connected)
			}
			break
		}
	}
}

// ── State readers ─────────────────────────────────────────────────────────────

// gcExtendedGamepad returns the GCExtendedGamepad for the given slot, or 0.
func gcExtendedGamepad(joy Joystick) objc.ID {
	if int(joy) < 0 || int(joy) >= len(darwinJoystickSlots) {
		return 0
	}
	h := darwinJoystickSlots[joy].handle
	if h == 0 {
		return 0
	}
	return objc.ID(h).Send(selGCExtendedGamepad)
}

// buttonAction converts a GCControllerButtonInput's isPressed state to Action.
func buttonAction(btn objc.ID) Action {
	if btn == 0 {
		return Release
	}
	if objc.Send[bool](btn, selGCIsPressed) {
		return Press
	}
	return Release
}

// safeButtonAction calls buttonAction only when btn is non-zero and the
// controller responds to the selector (for optional macOS 11+/12+ buttons).
func safeButtonAction(ctrl objc.ID, sel objc.SEL) Action {
	if ctrl == 0 {
		return Release
	}
	if !objc.Send[bool](ctrl, selResponds, sel) {
		return Release
	}
	btn := ctrl.Send(sel)
	return buttonAction(btn)
}

// ── Public joystick API ───────────────────────────────────────────────────────

// JoystickPresent returns true if the given joystick slot is connected.
func JoystickPresent(joy Joystick) bool {
	if int(joy) < 0 || int(joy) >= len(darwinJoystickSlots) {
		return false
	}
	return darwinJoystickSlots[joy].handle != 0
}

// GetJoystickAxes returns 6 normalised axis values in [-1, 1]:
// left X, left Y, right X, right Y, left trigger, right trigger.
// Y axes are negated so that "up" maps to +1 (GLFW convention).
// Triggers are remapped from [0, 1] to [-1, +1].
func GetJoystickAxes(joy Joystick) []float32 {
	gp := gcExtendedGamepad(joy)
	if gp == 0 {
		return nil
	}

	ls := gp.Send(selGCLeftThumbstick)
	rs := gp.Send(selGCRightThumbstick)

	lx := objc.Send[float32](ls.Send(selGCXAxis), selGCValue)
	ly := objc.Send[float32](ls.Send(selGCYAxis), selGCValue)
	rx := objc.Send[float32](rs.Send(selGCXAxis), selGCValue)
	ry := objc.Send[float32](rs.Send(selGCYAxis), selGCValue)
	lt := objc.Send[float32](gp.Send(selGCLeftTrigger), selGCValue)
	rt := objc.Send[float32](gp.Send(selGCRightTrigger), selGCValue)

	// Negate Y: GCController +Y = up, GLFW +Y = down.
	// Triggers: GCController [0,1] → GLFW [-1,+1].
	return []float32{lx, -ly, rx, -ry, lt*2 - 1, rt*2 - 1}
}

// GetJoystickButtons returns 17 button states in press/release order:
// A, B, X, Y, L1, R1, L2(digital), R2(digital), Options(Back), Menu(Start),
// Home(Guide), LeftThumb, RightThumb, DpadUp, DpadRight, DpadDown, DpadLeft.
func GetJoystickButtons(joy Joystick) []Action {
	gp := gcExtendedGamepad(joy)
	if gp == 0 {
		return nil
	}
	ctrl := objc.ID(darwinJoystickSlots[joy].handle)

	dpad := gp.Send(selGCDpad)

	return []Action{
		buttonAction(gp.Send(selGCButtonA)),
		buttonAction(gp.Send(selGCButtonB)),
		buttonAction(gp.Send(selGCButtonX)),
		buttonAction(gp.Send(selGCButtonY)),
		buttonAction(gp.Send(selGCLeftShoulder)),
		buttonAction(gp.Send(selGCRightShoulder)),
		buttonAction(gp.Send(selGCLeftTrigger)),
		buttonAction(gp.Send(selGCRightTrigger)),
		safeButtonAction(ctrl, selGCButtonOptions),
		safeButtonAction(ctrl, selGCButtonMenu),
		safeButtonAction(ctrl, selGCButtonHome),
		buttonAction(gp.Send(selGCLeftThumbstickButton)),
		buttonAction(gp.Send(selGCRightThumbstickButton)),
		buttonAction(dpad.Send(selGCUp)),
		buttonAction(dpad.Send(selGCRight)),
		buttonAction(dpad.Send(selGCDown)),
		buttonAction(dpad.Send(selGCLeft)),
	}
}

// GetJoystickHats returns one hat representing the d-pad.
func GetJoystickHats(joy Joystick) []JoystickHatState {
	gp := gcExtendedGamepad(joy)
	if gp == 0 {
		return nil
	}
	dpad := gp.Send(selGCDpad)

	var state JoystickHatState = HatCentered
	if objc.Send[bool](dpad.Send(selGCUp), selGCIsPressed) {
		state |= HatUp
	}
	if objc.Send[bool](dpad.Send(selGCDown), selGCIsPressed) {
		state |= HatDown
	}
	if objc.Send[bool](dpad.Send(selGCLeft), selGCIsPressed) {
		state |= HatLeft
	}
	if objc.Send[bool](dpad.Send(selGCRight), selGCIsPressed) {
		state |= HatRight
	}
	return []JoystickHatState{state}
}

// GetJoystickName returns the controller's vendor name.
func GetJoystickName(joy Joystick) string {
	if int(joy) < 0 || int(joy) >= len(darwinJoystickSlots) {
		return ""
	}
	return darwinJoystickSlots[joy].name
}

// GetJoystickGUID returns an opaque device identifier string.
func GetJoystickGUID(joy Joystick) string {
	if !JoystickPresent(joy) {
		return ""
	}
	// GCController has no stable GUID; synthesise from name + slot index.
	return fmt.Sprintf("darwin-%02d-%s", int(joy), darwinJoystickSlots[joy].name)
}

// JoystickIsGamepad returns true when the slot has a GCExtendedGamepad profile.
func JoystickIsGamepad(joy Joystick) bool {
	return gcExtendedGamepad(joy) != 0
}

// GetGamepadName is an alias for GetJoystickName (GLFW API parity).
func GetGamepadName(joy Joystick) string { return GetJoystickName(joy) }

// GetGamepadState fills state with a standardised gamepad snapshot.
// Returns false if the slot is empty or has no extended gamepad profile.
func GetGamepadState(joy Joystick, state *GamepadState) bool {
	gp := gcExtendedGamepad(joy)
	if gp == 0 {
		return false
	}
	ctrl := objc.ID(darwinJoystickSlots[joy].handle)

	ls := gp.Send(selGCLeftThumbstick)
	rs := gp.Send(selGCRightThumbstick)
	dpad := gp.Send(selGCDpad)

	// Axes.
	state.Axes[AxisLeftX] = objc.Send[float32](ls.Send(selGCXAxis), selGCValue)
	state.Axes[AxisLeftY] = -objc.Send[float32](ls.Send(selGCYAxis), selGCValue)
	state.Axes[AxisRightX] = objc.Send[float32](rs.Send(selGCXAxis), selGCValue)
	state.Axes[AxisRightY] = -objc.Send[float32](rs.Send(selGCYAxis), selGCValue)
	lt := objc.Send[float32](gp.Send(selGCLeftTrigger), selGCValue)
	rt := objc.Send[float32](gp.Send(selGCRightTrigger), selGCValue)
	state.Axes[AxisLeftTrigger] = lt*2 - 1
	state.Axes[AxisRightTrigger] = rt*2 - 1

	// Buttons.
	state.Buttons[ButtonA] = buttonAction(gp.Send(selGCButtonA))
	state.Buttons[ButtonB] = buttonAction(gp.Send(selGCButtonB))
	state.Buttons[ButtonX] = buttonAction(gp.Send(selGCButtonX))
	state.Buttons[ButtonY] = buttonAction(gp.Send(selGCButtonY))
	state.Buttons[ButtonLeftBumper] = buttonAction(gp.Send(selGCLeftShoulder))
	state.Buttons[ButtonRightBumper] = buttonAction(gp.Send(selGCRightShoulder))
	state.Buttons[ButtonBack] = safeButtonAction(ctrl, selGCButtonOptions)
	state.Buttons[ButtonStart] = safeButtonAction(ctrl, selGCButtonMenu)
	state.Buttons[ButtonGuide] = safeButtonAction(ctrl, selGCButtonHome)
	state.Buttons[ButtonLeftThumb] = buttonAction(gp.Send(selGCLeftThumbstickButton))
	state.Buttons[ButtonRightThumb] = buttonAction(gp.Send(selGCRightThumbstickButton))
	state.Buttons[ButtonDpadUp] = buttonAction(dpad.Send(selGCUp))
	state.Buttons[ButtonDpadRight] = buttonAction(dpad.Send(selGCRight))
	state.Buttons[ButtonDpadDown] = buttonAction(dpad.Send(selGCDown))
	state.Buttons[ButtonDpadLeft] = buttonAction(dpad.Send(selGCLeft))

	return true
}

// UpdateGamepadMappings accepts SDL-compatible mapping strings.
// GCController uses its own built-in mapping; SDL strings are ignored.
func UpdateGamepadMappings(_ string) bool { return true }
