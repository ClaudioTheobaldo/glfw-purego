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
