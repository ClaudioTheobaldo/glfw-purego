//go:build linux && !wayland

package glfw

import (
	"net/url"
	"strings"
	"unsafe"
)

// XdndAware version advertised on each window.
const _xdndVersion = uint64(5)

// setXdndAware sets the XdndAware property on xwin so drag sources know
// which XDND protocol version this window supports (version 5).
func setXdndAware(xwin uint64) {
	version := _xdndVersion
	xChangeProperty(x11Display, xwin,
		atomXdndAware, _xaCARDINAL,
		32, 0,
		uintptr(unsafe.Pointer(&version)),
		1)
}

// xdndState holds the transient XDND drag state for one window.
type xdndState struct {
	source    uint64 // XID of the source window
	timestamp uint64 // Time from XdndDrop
}

// xdndStates maps window XID → *xdndState.  Created lazily.
var xdndStates = make(map[uint64]*xdndState)

func getXdndState(xwin uint64) *xdndState {
	if s, ok := xdndStates[xwin]; ok {
		return s
	}
	s := &xdndState{}
	xdndStates[xwin] = s
	return s
}

// handleXdndClientMessage dispatches XDND-related ClientMessage events.
// Called from handleX11Event when cm.MessageType matches an XDND atom.
func handleXdndClientMessage(w *Window, cm *_XClientMessageEvent) {
	switch uint64(cm.MessageType) {
	case atomXdndEnter:
		s := getXdndState(uint64(w.handle))
		s.source = uint64(cm.Data[0])
		s.timestamp = 0

	case atomXdndPosition:
		s := getXdndState(uint64(w.handle))
		s.source = uint64(cm.Data[0])
		// Reply immediately: we accept the drop (XdndActionCopy).
		sendXdndStatus(s.source, uint64(w.handle), true)

	case atomXdndLeave:
		// Nothing to do — drop was cancelled before release.

	case atomXdndDrop:
		s := getXdndState(uint64(w.handle))
		s.timestamp = uint64(cm.Data[2])
		// Retrieve the dropped file list via the X selection mechanism.
		data := readXdndData(w, s.timestamp)
		paths := parseURIList(data)
		if len(paths) > 0 && w.fDropHolder != nil {
			w.fDropHolder(w, paths)
		}
		sendXdndFinished(s.source, uint64(w.handle), len(paths) > 0)
	}
}

// sendXdndStatus sends an XdndStatus message back to the drag source.
func sendXdndStatus(source, target uint64, accept bool) {
	var cm _XClientMessageEvent
	cm.Type = _ClientMessage
	cm.Window = source
	cm.MessageType = atomXdndStatus
	cm.Format = 32
	cm.Data[0] = int64(target) // our window
	if accept {
		cm.Data[1] = 1                      // bit 0: accept drop
		cm.Data[4] = int64(atomXdndActionCopy)
	}
	xSendEvent(x11Display, source, 0, 0, uintptr(unsafe.Pointer(&cm)))
	xFlush(x11Display)
}

// sendXdndFinished informs the source that the drop has been processed.
func sendXdndFinished(source, target uint64, accepted bool) {
	var cm _XClientMessageEvent
	cm.Type = _ClientMessage
	cm.Window = source
	cm.MessageType = atomXdndFinished
	cm.Format = 32
	cm.Data[0] = int64(target)
	if accepted {
		cm.Data[1] = 1 // bit 0: operation succeeded
		cm.Data[2] = int64(atomXdndActionCopy)
	}
	xSendEvent(x11Display, source, 0, 0, uintptr(unsafe.Pointer(&cm)))
	xFlush(x11Display)
}

// readXdndData requests the drag payload as "text/uri-list" via XConvertSelection
// and returns the raw data string.  Returns "" on timeout or error.
func readXdndData(w *Window, timestamp uint64) string {
	xwin := uint64(w.handle)
	xConvertSelection(x11Display,
		atomXdndSelection, atomTextURIList,
		atomGLFWSel, xwin, timestamp)
	xFlush(x11Display)

	// Poll for SelectionNotify (up to 200 ms, 20 × 10 ms intervals).
	// We borrow the same polling loop used by GetClipboardString.
	const maxTries = 20
	for range maxTries {
		var ev _XEvent
		if xCheckTypedEvent(x11Display, _SelectionNotify, uintptr(unsafe.Pointer(&ev))) != 0 {
			se := (*_XSelectionEvent)(unsafe.Pointer(&ev))
			if se.Property == 0 {
				return "" // conversion refused
			}
			return readProperty(xwin, atomGLFWSel)
		}
		// Short sleep via X11 flush + a busy-wait; avoids importing time.
		xFlush(x11Display)
	}
	return ""
}

// readProperty retrieves the full value of a window property as a string.
// Used by readXdndData to pull the text/uri-list selection result.
func readProperty(xwin, prop uint64) string {
	var (
		actualType   uint64
		actualFormat int32
		nItems       uint64
		bytesAfter   uint64
		propPtr      uintptr
	)
	xGetWindowProperty(x11Display, xwin, prop,
		0, 1<<24, 0, 0,
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&propPtr)))
	if propPtr == 0 || nItems == 0 {
		return ""
	}
	defer xFree(propPtr)
	data := make([]byte, nItems)
	copy(data, unsafe.Slice((*byte)(nativePtrFromUintptr(propPtr)), nItems))
	return string(data)
}

// parseURIList converts a text/uri-list string (RFC 2483) into a slice of
// local file paths.  Entries starting with "file://" are decoded; others are
// dropped.
func parseURIList(raw string) []string {
	var paths []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		if u.Scheme == "file" {
			paths = append(paths, u.Path)
		}
	}
	return paths
}
