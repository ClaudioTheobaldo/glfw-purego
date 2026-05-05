package glfw

import (
	"sync"
	"unsafe"
)

// ── Joystick user pointer ─────────────────────────────────────────────────────
// Allows arbitrary per-joystick data to be associated with a Joystick slot.

var (
	joystickPtrMu sync.RWMutex
	joystickPtrs  = make(map[Joystick]unsafe.Pointer)
)

// SetJoystickUserPointer stores an arbitrary pointer for the given joystick.
func SetJoystickUserPointer(joy Joystick, ptr unsafe.Pointer) {
	joystickPtrMu.Lock()
	joystickPtrs[joy] = ptr
	joystickPtrMu.Unlock()
}

// GetJoystickUserPointer retrieves the pointer previously set by
// SetJoystickUserPointer.
func GetJoystickUserPointer(joy Joystick) unsafe.Pointer {
	joystickPtrMu.RLock()
	p := joystickPtrs[joy]
	joystickPtrMu.RUnlock()
	return p
}

// SetUserPointer is the method form of SetJoystickUserPointer, matching the
// upstream go-gl/glfw API.
func (joy Joystick) SetUserPointer(ptr unsafe.Pointer) { SetJoystickUserPointer(joy, ptr) }

// GetUserPointer is the method form of GetJoystickUserPointer.
func (joy Joystick) GetUserPointer() unsafe.Pointer { return GetJoystickUserPointer(joy) }

// GetGamepadState is the method form of the package-level GetGamepadState.
// Returns nil if the joystick is not connected or has no gamepad mapping.
//
// This signature matches upstream go-gl/glfw v3.3 (which returns
// *GamepadState).  The package-level (joy, *GamepadState) bool form is also
// retained for backward compatibility with code written against earlier
// glfw-purego versions.
func (joy Joystick) GetGamepadState() *GamepadState {
	var s GamepadState
	if !GetGamepadState(joy, &s) {
		return nil
	}
	return &s
}
