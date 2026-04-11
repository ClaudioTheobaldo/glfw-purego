//go:build windows

package glfw

// PollEvents processes all pending OS events without blocking.
// Must be called from the main OS thread.
func PollEvents() {
	var msg _MSG
	for peekMessageW(&msg, 0, 0, 0, _PM_REMOVE) {
		if msg.Message == _WM_QUIT {
			return
		}
		translateMessage(&msg)
		dispatchMessageW(&msg)
	}
}

// WaitEvents blocks until at least one event is queued, then processes all
// pending events. Must be called from the main OS thread.
func WaitEvents() {
	var msg _MSG
	ok, err := getMessageW(&msg, 0, 0, 0)
	if err != nil || !ok {
		return
	}
	translateMessage(&msg)
	dispatchMessageW(&msg)
	PollEvents()
}

// WaitEventsTimeout blocks for at most timeout seconds, then processes all
// pending events. Must be called from the main OS thread.
func WaitEventsTimeout(timeout float64) {
	ms := uint32(timeout * 1000)
	if ms == 0 {
		ms = 1
	}
	msgWaitForMultipleObjectsEx(0, 0, ms, _QS_ALLINPUT, _MWMO_INPUTAVAILABLE)
	PollEvents()
}
