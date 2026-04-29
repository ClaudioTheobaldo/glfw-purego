package glfw

import (
	"sync"
	"unsafe"
)

// ── Monitor user pointer ──────────────────────────────────────────────────────
// Mirrors the Window user pointer pattern: a package-level map keyed by the
// *Monitor value, protected by a read-write mutex.

var (
	monitorPtrMu sync.RWMutex
	monitorPtrs  = make(map[*Monitor]unsafe.Pointer)
)

// SetUserPointer stores an arbitrary pointer associated with this monitor.
func (m *Monitor) SetUserPointer(ptr unsafe.Pointer) {
	monitorPtrMu.Lock()
	monitorPtrs[m] = ptr
	monitorPtrMu.Unlock()
}

// GetUserPointer retrieves the pointer previously stored by SetUserPointer.
func (m *Monitor) GetUserPointer() unsafe.Pointer {
	monitorPtrMu.RLock()
	p := monitorPtrs[m]
	monitorPtrMu.RUnlock()
	return p
}
