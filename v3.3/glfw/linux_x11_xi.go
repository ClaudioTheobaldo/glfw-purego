//go:build linux && !wayland

package glfw

import (
	"syscall"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// XInput2 extension — lazy-loaded on first use.
var (
	xiOnce   sync.Once
	xiHandle uintptr
	xiLoaded bool
	xiOpcode int32 // major X opcode for the XInput extension

	xISelectEvents func(display uintptr, window uint64, masks uintptr, numMasks int32) int32
)

// loadXI opens libXi and queries the XInput extension opcode.
// Returns true only when Xi is available AND XInput2 is present on the server.
func loadXI() bool {
	xiOnce.Do(func() {
		if libX11Handle == 0 {
			return
		}
		var err error
		for _, name := range []string{"libXi.so.6", "libXi.so"} {
			xiHandle, err = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err == nil {
				break
			}
		}
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&xISelectEvents, xiHandle, "XISelectEvents")

		// Query whether the XInput extension is present on the server.
		// The extension name used by XInput2 is "XInputExtension".
		namePtr, _ := syscall.BytePtrFromString("XInputExtension")
		var firstEvent, firstError int32
		if xQueryExtension(x11Display,
			uintptr(unsafe.Pointer(namePtr)),
			uintptr(unsafe.Pointer(&xiOpcode)),
			uintptr(unsafe.Pointer(&firstEvent)),
			uintptr(unsafe.Pointer(&firstError))) != 0 && xiOpcode != 0 {
			xiLoaded = true
		}
	})
	return xiLoaded
}

// RawMouseMotionSupported reports whether raw (unaccelerated) mouse motion
// is available on the current platform.
func RawMouseMotionSupported() bool {
	if x11Display == 0 {
		return false
	}
	return loadXI()
}

// rawMotionWindow is the window that currently receives XI_RawMotion events.
var rawMotionWindow *Window

// enableRawMotion subscribes the root window to XI_RawMotion events and
// records which window should receive the resulting cursor-pos callbacks.
func enableRawMotion(w *Window) {
	if !loadXI() {
		return
	}
	rawMotionWindow = w
	// XI_RawMotion = 17; lives in mask byte 2, bit 1 (17/8=2, 17%8=1).
	mask := [4]byte{0, 0, 1 << (17 - 16), 0}
	em := _XIEventMask{
		DeviceID: _XIAllMasterDevices,
		MaskLen:  int32(len(mask)),
		Mask:     uintptr(unsafe.Pointer(&mask[0])),
	}
	xISelectEvents(x11Display, x11Root, uintptr(unsafe.Pointer(&em)), 1)
	xFlush(x11Display)
}

// disableRawMotion unsubscribes from XI_RawMotion events.
func disableRawMotion(_ *Window) {
	if !xiLoaded {
		return
	}
	rawMotionWindow = nil
	mask := [4]byte{}
	em := _XIEventMask{
		DeviceID: _XIAllMasterDevices,
		MaskLen:  int32(len(mask)),
		Mask:     uintptr(unsafe.Pointer(&mask[0])),
	}
	xISelectEvents(x11Display, x11Root, uintptr(unsafe.Pointer(&em)), 1)
	xFlush(x11Display)
}

// handleGenericEvent processes an XInput2 GenericEvent.
// Called from handleX11Event when ev.eventType() == _GenericEvent.
func handleGenericEvent(ev *_XEvent) {
	if !xiLoaded {
		return
	}
	cookie := (*_XGenericEventCookie)(unsafe.Pointer(ev))
	if cookie.Extension != xiOpcode {
		return
	}
	if xGetEventData(x11Display, uintptr(unsafe.Pointer(cookie))) == 0 {
		return
	}
	defer xFreeEventData(x11Display, uintptr(unsafe.Pointer(cookie)))

	if cookie.Evtype != _XI_RawMotion {
		return
	}
	w := rawMotionWindow
	if w == nil || w.fCursorPosHolder == nil {
		return
	}
	re := (*_XIRawEvent)(nativePtrFromUintptr(cookie.Data))
	dx, dy := readRawDeltas(re)
	w.rawCursorX += dx
	w.rawCursorY += dy
	w.fCursorPosHolder(w, w.rawCursorX, w.rawCursorY)
}

// readRawDeltas extracts the X (axis 0) and Y (axis 1) delta values from an
// XIRawEvent.  The valuators mask is a bitfield: bit N is set when axis N is
// present, and the values array is densely packed for present axes only.
func readRawDeltas(re *_XIRawEvent) (dx, dy float64) {
	v := &re.Valuators
	if v.MaskLen == 0 || v.Mask == 0 || v.Values == 0 {
		return
	}
	mask := (*[8]byte)(nativePtrFromUintptr(v.Mask))
	vals := (*[64]float64)(nativePtrFromUintptr(v.Values))
	idx := 0
	for axis := 0; axis <= 1; axis++ {
		if mask[axis/8]&(1<<uint(axis%8)) != 0 {
			if axis == 0 {
				dx = vals[idx]
			} else {
				dy = vals[idx]
			}
			idx++
		}
	}
	return
}
