//go:build linux && !wayland

package glfw

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// ----------------------------------------------------------------------------
// SetMonitorCallback — monitor connect/disconnect via XRandR RRNotify
// ----------------------------------------------------------------------------

var (
	linuxMonitorCb      func(*Monitor, PeripheralEvent)
	linuxCachedMonitors []*Monitor
)

const _RROutputChangeNotifyMask = int32(1)

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
// Requires XRandR; silently ignored if libXrandr is not available.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	linuxMonitorCb = cb
	if cb != nil && x11Display != 0 {
		if loadXrandr() == nil {
			xrrSelectInput(x11Display, x11Root, _RROutputChangeNotifyMask)
		}
		linuxCachedMonitors, _ = GetMonitors()
	}
}

// ----------------------------------------------------------------------------
// Init / Terminate
// ----------------------------------------------------------------------------

// Init initialises the GLFW subsystem on Linux (X11 backend).
// Locks the calling goroutine to its OS thread.
func Init() error {
	runtime.LockOSThread()
	linuxInitTime = time.Now()
	resetHints()
	if err := initX11Display(); err != nil {
		return err
	}
	return nil
}

// Terminate destroys all windows and closes the X11 display.
func Terminate() {
	windowByHandle.Range(func(k, v any) bool {
		v.(*Window).Destroy()
		return true
	})
	// Free shared invisible cursor before closing the display.
	if x11InvisibleCursor != 0 && x11Display != 0 {
		xFreeCursor(x11Display, x11InvisibleCursor)
		x11InvisibleCursor = 0
	}
	if x11Display != 0 {
		xCloseDisplay(x11Display)
		x11Display = 0
	}
	// Close self-pipe after the display so no racing WaitEvents can write to it.
	if x11PostPipeRead >= 0 {
		syscall.Close(x11PostPipeRead)
		x11PostPipeRead = -1
	}
	if x11PostPipeWrite >= 0 {
		syscall.Close(x11PostPipeWrite)
		x11PostPipeWrite = -1
	}
}

// ----------------------------------------------------------------------------
// _MotifWMHints — used by SetAttrib(Decorated, …)
// Each field is a C long (8 bytes on 64-bit Linux); 5 fields = 40 bytes total.
// ----------------------------------------------------------------------------

type _MotifWMHints struct {
	Flags       uint64
	Functions   uint64
	Decorations uint64
	InputMode   int64
	Status      uint64
}

const _MWM_HINTS_DECORATIONS = uint64(2)

// ----------------------------------------------------------------------------
// Window attribute / fullscreen / icon
// ----------------------------------------------------------------------------

// SetMonitor switches the window between fullscreen and windowed mode.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if monitor != nil {
		// Save windowed state if not already fullscreen
		if w.fsMonitor == nil {
			var wa _XWindowAttributes
			xGetWindowAttributes(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&wa)))
			w.savedX = int(wa.X)
			w.savedY = int(wa.Y)
			w.savedW = int(wa.Width)
			w.savedH = int(wa.Height)
		}
		w.fsMonitor = monitor
		// Prefer the monitor's real geometry when available.
		if monitor.widthPx > 0 {
			xpos, ypos = monitor.x, monitor.y
			width, height = monitor.widthPx, monitor.heightPx
		}
		// Move/resize to requested geometry first, then request fullscreen state
		xMoveResizeWindow(x11Display, uint64(w.handle), int32(xpos), int32(ypos), uint32(width), uint32(height))
		w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateFull, 0)
	} else {
		// Exit fullscreen
		w.fsMonitor = nil
		w.sendNETWMState(_NET_WM_STATE_REMOVE, atomNETWMStateFull, 0)
		xMoveResizeWindow(x11Display, uint64(w.handle), int32(w.savedX), int32(w.savedY), uint32(w.savedW), uint32(w.savedH))
	}
	xFlush(x11Display)
}

// SetAttrib sets a window attribute at runtime.
func (w *Window) SetAttrib(attrib Hint, value int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	switch attrib {
	case Decorated:
		var hints _MotifWMHints
		hints.Flags = _MWM_HINTS_DECORATIONS
		if value != 0 {
			hints.Decorations = 1
		}
		xChangeProperty(x11Display, uint64(w.handle),
			atomMOTIFWMHints, atomMOTIFWMHints,
			32, 0,
			uintptr(unsafe.Pointer(&hints)),
			5)
	case Floating:
		if value != 0 {
			w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateAbove, 0)
		} else {
			w.sendNETWMState(_NET_WM_STATE_REMOVE, atomNETWMStateAbove, 0)
		}
	}
	xFlush(x11Display)
}

// SetIcon sets the window icon from a slice of candidate images.
// Pass nil or an empty slice to remove the icon.
func (w *Window) SetIcon(images []Image) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if len(images) == 0 {
		xDeleteProperty(x11Display, uint64(w.handle), atomNETWMIcon)
		xFlush(x11Display)
		return
	}

	// Build CARDINAL array: for each image [ width, height, ARGB pixels… ]
	// On 64-bit Linux, format=32 ⇒ each element is an 8-byte C long.
	var data []uint64
	for i := range images {
		img := &images[i]
		data = append(data, uint64(img.Width), uint64(img.Height))
		for j := 0; j < img.Width*img.Height; j++ {
			r := uint64(img.Pixels[j*4+0])
			g := uint64(img.Pixels[j*4+1])
			b := uint64(img.Pixels[j*4+2])
			a := uint64(img.Pixels[j*4+3])
			data = append(data, (a<<24)|(r<<16)|(g<<8)|b)
		}
	}

	const xaCARDINAL = uint64(6)
	xChangeProperty(x11Display, uint64(w.handle),
		atomNETWMIcon, xaCARDINAL,
		32, 0,
		uintptr(unsafe.Pointer(&data[0])),
		int32(len(data)))
	runtime.KeepAlive(data)
	xFlush(x11Display)
}

// ----------------------------------------------------------------------------
// Size / aspect constraints
// ----------------------------------------------------------------------------

// applyWMNormalHints pushes the window's size/aspect constraints to the WM
// via XSetWMNormalHints.  Call whenever minW/maxW/aspectNum change.
func (w *Window) applyWMNormalHints() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	var hints _XSizeHints
	if w.minW > 0 || w.minH > 0 {
		hints.Flags |= _PMinSize
		hints.MinWidth = int32(w.minW)
		hints.MinHeight = int32(w.minH)
	}
	if w.maxW > 0 || w.maxH > 0 {
		hints.Flags |= _PMaxSize
		hints.MaxWidth = int32(w.maxW)
		hints.MaxHeight = int32(w.maxH)
	}
	if w.aspectNum > 0 && w.aspectDen > 0 {
		hints.Flags |= _PAspect
		hints.MinAspX = int32(w.aspectNum)
		hints.MinAspY = int32(w.aspectDen)
		hints.MaxAspX = int32(w.aspectNum)
		hints.MaxAspY = int32(w.aspectDen)
	}
	if hints.Flags == 0 {
		return
	}
	xSetWMNormalHints(x11Display, uint64(w.handle), uintptr(unsafe.Pointer(&hints)))
	xFlush(x11Display)
}

// SetSizeLimits sets minimum/maximum window dimensions.
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {
	w.minW, w.minH, w.maxW, w.maxH = minWidth, minHeight, maxWidth, maxHeight
	w.applyWMNormalHints()
}

// SetAspectRatio locks the resize aspect ratio (numer:denom). Pass 0,0 to clear.
func (w *Window) SetAspectRatio(numer, denom int) {
	w.aspectNum, w.aspectDen = numer, denom
	w.applyWMNormalHints()
}

// ----------------------------------------------------------------------------
// Opacity
// ----------------------------------------------------------------------------

const _netWMWindowOpacityMax = uint64(0xFFFFFFFF)

// GetOpacity returns the window opacity in the range [0, 1].
func (w *Window) GetOpacity() float32 {
	if x11Display == 0 || w.handle == 0 {
		return 1.0
	}
	var actualType uint64
	var actualFormat int32
	var nItems, bytesAfter uint64
	var propPtr uintptr
	ret := xGetWindowProperty(x11Display, uint64(w.handle),
		atomNETWMWindowOpacity,
		0, 1, 0,
		6, // XA_CARDINAL
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&propPtr)))
	if ret != 0 || propPtr == 0 || nItems < 1 {
		return 1.0
	}
	val := *(*uint64)(nativePtrFromUintptr(propPtr)) & 0xFFFFFFFF
	xFree(propPtr)
	return float32(val) / float32(_netWMWindowOpacityMax)
}

// SetOpacity sets the window opacity. A value of 1.0 removes the property.
func (w *Window) SetOpacity(opacity float32) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if opacity >= 1.0 {
		xDeleteProperty(x11Display, uint64(w.handle), atomNETWMWindowOpacity)
	} else {
		val := uint64(opacity * float32(_netWMWindowOpacityMax))
		xChangeProperty(x11Display, uint64(w.handle),
			atomNETWMWindowOpacity, 6,
			32, 0,
			uintptr(unsafe.Pointer(&val)),
			1)
	}
	xFlush(x11Display)
}

// RequestAttention requests user attention via _NET_WM_STATE_DEMANDS_ATTENTION.
func (w *Window) RequestAttention() {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	w.sendNETWMState(_NET_WM_STATE_ADD, atomNETWMStateDemandsAttention, 0)
	xFlush(x11Display)
}

// ----------------------------------------------------------------------------
// GetFrameSize — reads _NET_FRAME_EXTENTS from the WM
// ----------------------------------------------------------------------------

// GetFrameSize returns the size of the decorations around the window's client area.
func (w *Window) GetFrameSize() (left, top, right, bottom int) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	var reqEv _XClientMessageEvent
	reqEv.Type = _ClientMessage
	reqEv.Window = uint64(w.handle)
	reqEv.MessageType = atomNETRequestFrameExtents
	reqEv.Format = 32
	xSendEvent(x11Display, x11Root, 0,
		_SubstructureNotifyMask|_SubstructureRedirectMask,
		uintptr(unsafe.Pointer(&reqEv)))
	xSync(x11Display, 0)

	var actualType uint64
	var actualFormat int32
	var nItems, bytesAfter uint64
	var propPtr uintptr

	ret := xGetWindowProperty(x11Display, uint64(w.handle),
		atomNETFrameExtents,
		0, 4, 0,
		6, // XA_CARDINAL
		uintptr(unsafe.Pointer(&actualType)),
		uintptr(unsafe.Pointer(&actualFormat)),
		uintptr(unsafe.Pointer(&nItems)),
		uintptr(unsafe.Pointer(&bytesAfter)),
		uintptr(unsafe.Pointer(&propPtr)))

	if ret != 0 || propPtr == 0 || nItems < 4 {
		return
	}
	data := (*[4]uint64)(nativePtrFromUintptr(propPtr))
	left   = int(data[0] & 0xFFFFFFFF)
	right  = int(data[1] & 0xFFFFFFFF)
	top    = int(data[2] & 0xFFFFFFFF)
	bottom = int(data[3] & 0xFFFFFFFF)
	xFree(propPtr)
	return
}

// GetWindowFrameSize is a package-level wrapper around (*Window).GetFrameSize.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) { return w.GetFrameSize() }
