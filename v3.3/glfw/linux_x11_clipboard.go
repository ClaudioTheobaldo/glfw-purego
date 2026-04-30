//go:build linux

package glfw

import (
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// ----------------------------------------------------------------------------
// Clipboard — X11 selection protocol (CLIPBOARD selection)
// ----------------------------------------------------------------------------

// clipboardOnce creates the helper window lazily the first time it is needed.
var (
	clipboardOnce   sync.Once
	clipboardWindow uint64
	clipboardMu     sync.Mutex
	clipboardText   string
)

// getClipboardWindow returns the XID of the clipboard helper window,
// creating it on the first call. Returns 0 if X11 is not available.
func getClipboardWindow() uint64 {
	clipboardOnce.Do(func() {
		if x11Display == 0 {
			return
		}
		clipboardWindow = xCreateSimpleWindow(
			x11Display, x11Root,
			0, 0, 1, 1, 0, 0, 0,
		)
	})
	return clipboardWindow
}

// SetClipboardString places s on the system clipboard by acquiring ownership
// of the CLIPBOARD selection.
func SetClipboardString(s string) {
	if x11Display == 0 {
		return
	}
	cw := getClipboardWindow()
	if cw == 0 {
		return
	}
	clipboardMu.Lock()
	clipboardText = s
	clipboardMu.Unlock()
	xSetSelectionOwner(x11Display, atomCLIPBOARD, cw, _CurrentTime)
	xFlush(x11Display)
}

// GetClipboardString returns the current clipboard contents as a UTF-8 string.
// If we own the selection the cached text is returned immediately.
// Otherwise XConvertSelection is used with a 100 ms poll timeout.
func GetClipboardString() string {
	if x11Display == 0 {
		return ""
	}
	cw := getClipboardWindow()
	if cw == 0 {
		return ""
	}
	// Short-circuit: we own the clipboard.
	owner := xGetSelectionOwner(x11Display, atomCLIPBOARD)
	if owner == cw {
		clipboardMu.Lock()
		s := clipboardText
		clipboardMu.Unlock()
		return s
	}
	if owner == 0 {
		return "" // no owner
	}
	// Request conversion to UTF-8.
	xConvertSelection(x11Display,
		atomCLIPBOARD,
		atomUTF8String,
		atomGLFWSel,
		cw,
		_CurrentTime)
	xFlush(x11Display)

	// Poll for SelectionNotify (max 100 ms).
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		var ev _XEvent
		if xCheckTypedEvent(x11Display, _SelectionNotify, uintptr(unsafe.Pointer(&ev))) != 0 {
			sn := (*_XSelectionEvent)(unsafe.Pointer(&ev))
			if sn.Property == 0 {
				return "" // owner could not convert
			}
			return readClipboardProperty(cw, atomGLFWSel)
		}
		time.Sleep(1 * time.Millisecond)
	}
	return ""
}

// readClipboardProperty reads an 8-bit text property from window/prop,
// deletes it after reading, and returns the string.
func readClipboardProperty(window, prop uint64) string {
	var actualType uint64
	var actualFormat int32
	var nItems, bytesAfter uint64
	var dataPtr uintptr

	ret := xGetWindowProperty(x11Display, window, prop,
		0, 1<<18, // longOffset=0, longLength=256k longs ≈ 1 MB of text
		1,        // delete = true
		0,        // AnyPropertyType
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&dataPtr)))
	if ret != 0 || dataPtr == 0 || nItems == 0 {
		return ""
	}
	// Copy bytes from X11-managed memory into a Go string.
	b := make([]byte, nItems)
	for i := uint64(0); i < nItems; i++ {
		b[i] = *(*byte)(unsafe.Pointer(dataPtr + uintptr(i)))
	}
	xFree(dataPtr)
	return string(b)
}

// handleSelectionRequest is called from the top of handleX11Event when a
// SelectionRequest event arrives. It writes our clipboard text to the
// requestor's window and sends a SelectionNotify reply.
func handleSelectionRequest(ev *_XEvent) {
	sr := (*_XSelectionRequestEvent)(unsafe.Pointer(ev))

	// Build the reply — default to None (conversion failed).
	reply := _XSelectionEvent{
		Type:      _SelectionNotify,
		Requestor: sr.Requestor,
		Selection: sr.Selection,
		Target:    sr.Target,
		Property:  0, // None
		Time:      sr.Time,
	}

	prop := sr.Property
	if prop == 0 {
		prop = sr.Target // old-style requestors that don't set Property
	}

	clipboardMu.Lock()
	text := clipboardText
	clipboardMu.Unlock()

	switch sr.Target {
	case atomTARGETS:
		// Tell the requestor which formats we support.
		targets := []uint64{atomUTF8String, _xaString}
		xChangeProperty(x11Display, sr.Requestor,
			prop, _xaAtom,
			32, 0,
			uintptr(unsafe.Pointer(&targets[0])),
			int32(len(targets)))
		runtime.KeepAlive(targets)
		reply.Property = prop
	case atomUTF8String, _xaString:
		if len(text) > 0 {
			tb := []byte(text)
			xChangeProperty(x11Display, sr.Requestor,
				prop, sr.Target,
				8, 0,
				uintptr(unsafe.Pointer(&tb[0])),
				int32(len(tb)))
			runtime.KeepAlive(tb)
			reply.Property = prop
		}
	}

	// Send SelectionNotify to the requestor.
	var buf _XEvent
	*(*_XSelectionEvent)(unsafe.Pointer(&buf)) = reply
	xSendEvent(x11Display, sr.Requestor, 0, 0, uintptr(unsafe.Pointer(&buf)))
	xFlush(x11Display)
}
