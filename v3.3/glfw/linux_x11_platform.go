//go:build linux

package glfw

import (
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	libX11Handle uintptr
	x11Display   uintptr // Display* — shared connection
	x11Screen    int32
	x11Root      uint64 // root window XID

	// Atoms for WM protocols
	atomWMDeleteWindow   uint64
	atomWMProtocols      uint64
	atomNETWMState       uint64
	atomNETWMStateMaxH   uint64
	atomNETWMStateMaxV   uint64
	atomNETWMStateFull   uint64
	atomNETWMStateHidden uint64
	atomNETWMName        uint64
	atomUTF8String       uint64

	x11Loaded bool
)

// X11 function pointers
var (
	xOpenDisplay               func(name uintptr) uintptr
	xCloseDisplay              func(display uintptr) int32
	xDefaultScreen             func(display uintptr) int32
	xDefaultRootWindow         func(display uintptr) uint64
	xDefaultVisual             func(display uintptr, screen int32) uintptr
	xDefaultDepth              func(display uintptr, screen int32) int32
	xCreateWindow              func(display, parent uintptr, x, y int32, width, height, borderWidth uint32, depth int32, class uint32, visual uintptr, valueMask uint64, attrs uintptr) uint64
	xDestroyWindow             func(display uintptr, window uint64) int32
	xMapWindow                 func(display uintptr, window uint64) int32
	xUnmapWindow               func(display uintptr, window uint64) int32
	xStoreName                 func(display uintptr, window uint64, name uintptr) int32
	xSelectInput               func(display uintptr, window uint64, eventMask int64) int32
	xNextEvent                 func(display uintptr, event uintptr) int32
	xPending                   func(display uintptr) int32
	xFlush                     func(display uintptr) int32
	xSync                      func(display uintptr, discard int32) int32
	xInternAtom                func(display uintptr, name uintptr, onlyIfExists int32) uint64
	xSetWMProtocols            func(display uintptr, window uint64, protocols uintptr, count int32) int32
	xGetWindowAttributes       func(display uintptr, window uint64, attribs uintptr) int32
	xMoveWindow                func(display uintptr, window uint64, x, y int32) int32
	xResizeWindow              func(display uintptr, window uint64, width, height uint32) int32
	xMoveResizeWindow          func(display uintptr, window uint64, x, y int32, width, height uint32) int32
	xIconifyWindow             func(display uintptr, window uint64, screen int32) int32
	xRaiseWindow               func(display uintptr, window uint64) int32
	xSetInputFocus             func(display uintptr, window uint64, revertTo int32, time uint64) int32
	xLookupKeysym              func(event uintptr, index int32) uint64
	xLookupString              func(event uintptr, buf uintptr, nbytes int32, ksym uintptr, status uintptr) int32
	xSendEvent                 func(display uintptr, window uint64, propagate int32, eventMask int64, event uintptr) int32
	xFree                      func(data uintptr) int32
	xChangeProperty            func(display uintptr, window uint64, property, typ uint64, format, mode int32, data uintptr, nElements int32) int32
	xkbSetDetectableAutoRepeat func(display uintptr, detectable int32, supported uintptr) int32
	xGetKeyboardMapping        func(display uintptr, firstKeycode uint32, count int32, symsPerKeycode uintptr) uintptr
	xDisplayKeycodes           func(display uintptr, minKeycodes, maxKeycodes uintptr) int32
	xWarpPointer               func(display uintptr, srcWin, dstWin uint64, srcX, srcY int32, srcWidth, srcHeight uint32, dstX, dstY int32) int32
	xDefineCursor              func(display uintptr, window uint64, cursor uint64) int32
	xCreateFontCursor          func(display uintptr, shape uint32) uint64
	xFreeCursor                func(display uintptr, cursor uint64) int32
)

func loadX11() error {
	var err error
	for _, name := range []string{"libX11.so.6", "libX11.so"} {
		libX11Handle, err = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}
	if err != nil {
		return &Error{Code: APIUnavailable, Desc: "X11 not available: " + err.Error()}
	}
	purego.RegisterLibFunc(&xOpenDisplay, libX11Handle, "XOpenDisplay")
	purego.RegisterLibFunc(&xCloseDisplay, libX11Handle, "XCloseDisplay")
	purego.RegisterLibFunc(&xDefaultScreen, libX11Handle, "XDefaultScreen")
	purego.RegisterLibFunc(&xDefaultRootWindow, libX11Handle, "XDefaultRootWindow")
	purego.RegisterLibFunc(&xDefaultVisual, libX11Handle, "XDefaultVisual")
	purego.RegisterLibFunc(&xDefaultDepth, libX11Handle, "XDefaultDepth")
	purego.RegisterLibFunc(&xCreateWindow, libX11Handle, "XCreateWindow")
	purego.RegisterLibFunc(&xDestroyWindow, libX11Handle, "XDestroyWindow")
	purego.RegisterLibFunc(&xMapWindow, libX11Handle, "XMapWindow")
	purego.RegisterLibFunc(&xUnmapWindow, libX11Handle, "XUnmapWindow")
	purego.RegisterLibFunc(&xStoreName, libX11Handle, "XStoreName")
	purego.RegisterLibFunc(&xSelectInput, libX11Handle, "XSelectInput")
	purego.RegisterLibFunc(&xNextEvent, libX11Handle, "XNextEvent")
	purego.RegisterLibFunc(&xPending, libX11Handle, "XPending")
	purego.RegisterLibFunc(&xFlush, libX11Handle, "XFlush")
	purego.RegisterLibFunc(&xSync, libX11Handle, "XSync")
	purego.RegisterLibFunc(&xInternAtom, libX11Handle, "XInternAtom")
	purego.RegisterLibFunc(&xSetWMProtocols, libX11Handle, "XSetWMProtocols")
	purego.RegisterLibFunc(&xGetWindowAttributes, libX11Handle, "XGetWindowAttributes")
	purego.RegisterLibFunc(&xMoveWindow, libX11Handle, "XMoveWindow")
	purego.RegisterLibFunc(&xResizeWindow, libX11Handle, "XResizeWindow")
	purego.RegisterLibFunc(&xMoveResizeWindow, libX11Handle, "XMoveResizeWindow")
	purego.RegisterLibFunc(&xIconifyWindow, libX11Handle, "XIconifyWindow")
	purego.RegisterLibFunc(&xRaiseWindow, libX11Handle, "XRaiseWindow")
	purego.RegisterLibFunc(&xSetInputFocus, libX11Handle, "XSetInputFocus")
	purego.RegisterLibFunc(&xLookupKeysym, libX11Handle, "XLookupKeysym")
	purego.RegisterLibFunc(&xLookupString, libX11Handle, "XLookupString")
	purego.RegisterLibFunc(&xSendEvent, libX11Handle, "XSendEvent")
	purego.RegisterLibFunc(&xFree, libX11Handle, "XFree")
	purego.RegisterLibFunc(&xChangeProperty, libX11Handle, "XChangeProperty")
	purego.RegisterLibFunc(&xkbSetDetectableAutoRepeat, libX11Handle, "XkbSetDetectableAutoRepeat")
	purego.RegisterLibFunc(&xGetKeyboardMapping, libX11Handle, "XGetKeyboardMapping")
	purego.RegisterLibFunc(&xDisplayKeycodes, libX11Handle, "XDisplayKeycodes")
	purego.RegisterLibFunc(&xWarpPointer, libX11Handle, "XWarpPointer")
	purego.RegisterLibFunc(&xDefineCursor, libX11Handle, "XDefineCursor")
	purego.RegisterLibFunc(&xCreateFontCursor, libX11Handle, "XCreateFontCursor")
	purego.RegisterLibFunc(&xFreeCursor, libX11Handle, "XFreeCursor")
	x11Loaded = true
	return nil
}

func initX11Display() error {
	if x11Display != 0 {
		return nil
	}
	if !x11Loaded {
		if err := loadX11(); err != nil {
			return err
		}
	}
	x11Display = xOpenDisplay(0) // NULL = use DISPLAY env var
	if x11Display == 0 {
		return &Error{Code: PlatformError, Desc: "XOpenDisplay failed — is DISPLAY set?"}
	}
	x11Screen = xDefaultScreen(x11Display)
	x11Root = xDefaultRootWindow(x11Display)

	// Intern WM atoms
	atomWMProtocols = internAtom("WM_PROTOCOLS", false)
	atomWMDeleteWindow = internAtom("WM_DELETE_WINDOW", false)
	atomNETWMState = internAtom("_NET_WM_STATE", false)
	atomNETWMStateMaxH = internAtom("_NET_WM_STATE_MAXIMIZED_HORZ", false)
	atomNETWMStateMaxV = internAtom("_NET_WM_STATE_MAXIMIZED_VERT", false)
	atomNETWMStateFull = internAtom("_NET_WM_STATE_FULLSCREEN", false)
	atomNETWMStateHidden = internAtom("_NET_WM_STATE_HIDDEN", false)
	atomNETWMName = internAtom("_NET_WM_NAME", false)
	atomUTF8String = internAtom("UTF8_STRING", false)

	// Enable detectable auto-repeat so we get clean key repeat events
	xkbSetDetectableAutoRepeat(x11Display, 1, 0)

	return nil
}

func internAtom(name string, onlyIfExists bool) uint64 {
	ptr, _ := syscall.BytePtrFromString(name)
	oie := int32(0)
	if onlyIfExists {
		oie = 1
	}
	return xInternAtom(x11Display, uintptr(unsafe.Pointer(ptr)), oie)
}
