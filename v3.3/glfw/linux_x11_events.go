//go:build linux

package glfw

import (
	"syscall"
	"unsafe"
)

// PollEvents processes all pending OS events without blocking.
func PollEvents() {
	if x11Display == 0 {
		return
	}
	for xPending(x11Display) > 0 {
		var ev _XEvent
		xNextEvent(x11Display, uintptr(unsafe.Pointer(&ev)))
		handleX11Event(&ev)
	}
}

// WaitEvents blocks until at least one X11 event or a PostEmptyEvent wake-up
// arrives, then processes all pending events.
func WaitEvents() {
	if x11Display == 0 {
		return
	}
	// Fast path: events already queued.
	if xPending(x11Display) > 0 {
		PollEvents()
		return
	}
	xFd := int(xConnectionNumber(x11Display))
	var rfds syscall.FdSet
	rfds.Bits[xFd/64] |= int64(1) << (uint(xFd) % 64)
	maxFd := xFd
	if x11PostPipeRead >= 0 {
		rfds.Bits[x11PostPipeRead/64] |= int64(1) << (uint(x11PostPipeRead) % 64)
		if x11PostPipeRead > maxFd {
			maxFd = x11PostPipeRead
		}
	}
	syscall.Select(maxFd+1, &rfds, nil, nil, nil) //nolint:errcheck
	drainPostPipe(&rfds)
	PollEvents()
}

// WaitEventsTimeout blocks for at most timeout seconds, then processes events.
func WaitEventsTimeout(timeout float64) {
	if x11Display == 0 {
		return
	}
	if xPending(x11Display) > 0 {
		PollEvents()
		return
	}
	xFd := int(xConnectionNumber(x11Display))
	var rfds syscall.FdSet
	rfds.Bits[xFd/64] |= int64(1) << (uint(xFd) % 64)
	maxFd := xFd
	if x11PostPipeRead >= 0 {
		rfds.Bits[x11PostPipeRead/64] |= int64(1) << (uint(x11PostPipeRead) % 64)
		if x11PostPipeRead > maxFd {
			maxFd = x11PostPipeRead
		}
	}
	sec := int64(timeout)
	usec := int64((timeout - float64(sec)) * 1e6)
	tv := syscall.Timeval{Sec: sec, Usec: usec}
	syscall.Select(maxFd+1, &rfds, nil, nil, &tv) //nolint:errcheck
	drainPostPipe(&rfds)
	PollEvents()
}

// PostEmptyEvent wakes up a WaitEvents or WaitEventsTimeout call on any thread.
func PostEmptyEvent() {
	if x11PostPipeWrite >= 0 {
		syscall.Write(x11PostPipeWrite, []byte{0}) //nolint:errcheck
	}
}

// drainPostPipe reads one byte from the self-pipe if it was selected.
func drainPostPipe(rfds *syscall.FdSet) {
	if x11PostPipeRead < 0 {
		return
	}
	if rfds.Bits[x11PostPipeRead/64]&(int64(1)<<(uint(x11PostPipeRead)%64)) != 0 {
		var buf [1]byte
		syscall.Read(x11PostPipeRead, buf[:]) //nolint:errcheck
	}
}

// handleX11Event dispatches a single X11 event to the appropriate window.
func handleX11Event(ev *_XEvent) {
	// Handle clipboard / selection events before the per-window lookup because
	// these are directed at the clipboard helper window which is not in
	// windowByHandle.
	switch ev.eventType() {
	case _SelectionRequest:
		handleSelectionRequest(ev)
		return
	case _SelectionClear:
		// We lost clipboard ownership; discard our cached text.
		clipboardMu.Lock()
		clipboardText = ""
		clipboardMu.Unlock()
		return
	}

	// XInput2 GenericEvent — not directed at a specific window handle.
	if ev.eventType() == _GenericEvent {
		handleGenericEvent(ev)
		return
	}

	xwin := ev.window()
	v, ok := windowByHandle.Load(uintptr(xwin))
	if !ok {
		return
	}
	w := v.(*Window)

	switch ev.eventType() {
	case _KeyPress:
		ke := (*_XKeyEvent)(unsafe.Pointer(ev))
		handleKeyEvent(w, ke, true)
	case _KeyRelease:
		ke := (*_XKeyEvent)(unsafe.Pointer(ev))
		handleKeyEvent(w, ke, false)
	case _ButtonPress:
		be := (*_XButtonEvent)(unsafe.Pointer(ev))
		handleButtonEvent(w, be, true)
	case _ButtonRelease:
		be := (*_XButtonEvent)(unsafe.Pointer(ev))
		handleButtonEvent(w, be, false)
	case _MotionNotify:
		me := (*_XMotionEvent)(unsafe.Pointer(ev))
		s := getX11State(w.handle)
		s.cursorX = float64(me.X)
		s.cursorY = float64(me.Y)
		if w.fCursorPosHolder != nil {
			w.fCursorPosHolder(w, s.cursorX, s.cursorY)
		}
	case _EnterNotify:
		if w.fCursorEnterHolder != nil {
			w.fCursorEnterHolder(w, true)
		}
	case _LeaveNotify:
		if w.fCursorEnterHolder != nil {
			w.fCursorEnterHolder(w, false)
		}
	case _FocusIn:
		if w.fFocusHolder != nil {
			w.fFocusHolder(w, true)
		}
	case _FocusOut:
		if w.fFocusHolder != nil {
			w.fFocusHolder(w, false)
		}
	case _Expose:
		if w.fRefreshHolder != nil {
			w.fRefreshHolder(w)
		}
	case _ConfigureNotify:
		ce := (*_XConfigureEvent)(unsafe.Pointer(ev))
		if w.fSizeHolder != nil {
			w.fSizeHolder(w, int(ce.Width), int(ce.Height))
		}
		if w.fFramebufferSizeHolder != nil {
			w.fFramebufferSizeHolder(w, int(ce.Width), int(ce.Height))
		}
		if w.fPosHolder != nil {
			w.fPosHolder(w, int(ce.X), int(ce.Y))
		}
	case _ClientMessage:
		cm := (*_XClientMessageEvent)(unsafe.Pointer(ev))
		switch cm.MessageType {
		case atomXdndEnter, atomXdndPosition, atomXdndLeave, atomXdndDrop:
			handleXdndClientMessage(w, cm)
		default:
			if uint64(cm.Data[0]) == atomWMDeleteWindow {
				w.shouldClose = true
				if w.fCloseHolder != nil {
					w.fCloseHolder(w)
				}
			}
		}
	case _DestroyNotify:
		windowByHandle.Delete(uintptr(xwin))
	}

	// XRandR RRNotify — fired when a monitor is connected or disconnected.
	// xrandrEventBase+1 = RROutputChangeNotify (from XRandR extension base).
	if xrandrEventBase != 0 && int32(ev.eventType()) == xrandrEventBase+1 {
		if linuxMonitorCb != nil {
			newMonitors, _ := GetMonitors()
			diffAndFireMonitorCallbacks(linuxCachedMonitors, newMonitors, linuxMonitorCb)
			linuxCachedMonitors = newMonitors
		}
	}
}
