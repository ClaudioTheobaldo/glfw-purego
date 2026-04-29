//go:build linux

package glfw

import "unsafe"

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

// WaitEvents blocks until at least one event is queued, then processes it.
func WaitEvents() {
	if x11Display == 0 {
		return
	}
	var ev _XEvent
	xNextEvent(x11Display, uintptr(unsafe.Pointer(&ev)))
	handleX11Event(&ev)
	PollEvents() // drain remaining
}

// WaitEventsTimeout blocks for at most timeout seconds, then processes events.
// For simplicity this delegates to WaitEvents (a proper timeout needs select+fd).
func WaitEventsTimeout(timeout float64) {
	WaitEvents()
}

// handleX11Event dispatches a single X11 event to the appropriate window.
func handleX11Event(ev *_XEvent) {
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
		if uint64(cm.Data[0]) == atomWMDeleteWindow {
			w.shouldClose = true
			if w.fCloseHolder != nil {
				w.fCloseHolder(w)
			}
		}
	case _DestroyNotify:
		windowByHandle.Delete(uintptr(xwin))
	}
}
