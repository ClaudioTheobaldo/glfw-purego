//go:build linux

package glfw

import "unsafe"

// ----------------------------------------------------------------------------
// X11 event type constants
// ----------------------------------------------------------------------------

const (
	_KeyPress        = 2
	_KeyRelease      = 3
	_ButtonPress     = 4
	_ButtonRelease   = 5
	_MotionNotify    = 6
	_EnterNotify     = 7
	_LeaveNotify     = 8
	_FocusIn         = 9
	_FocusOut        = 10
	_Expose          = 12
	_UnmapNotify     = 18
	_MapNotify       = 19
	_ConfigureNotify = 22
	_PropertyNotify  = 28
	_ClientMessage   = 33
	_DestroyNotify   = 17
)

// ----------------------------------------------------------------------------
// XSelectInput event masks
// ----------------------------------------------------------------------------

const (
	_KeyPressMask        = int64(1 << 0)
	_KeyReleaseMask      = int64(1 << 1)
	_ButtonPressMask     = int64(1 << 2)
	_ButtonReleaseMask   = int64(1 << 3)
	_EnterWindowMask     = int64(1 << 4)
	_LeaveWindowMask     = int64(1 << 5)
	_PointerMotionMask   = int64(1 << 6)
	_ExposureMask        = int64(1 << 15)
	_StructureNotifyMask = int64(1 << 17)
	_PropertyChangeMask  = int64(1 << 22)
	_FocusChangeMask     = int64(1 << 21)
)

// ----------------------------------------------------------------------------
// XSetWindowAttributes value mask flags
// ----------------------------------------------------------------------------

const (
	_CWEventMask   = int64(1 << 11)
	_CWColormap    = int64(1 << 13)
	_CWBorderPixel = int64(1 << 3)
	_CWCursor      = int64(1 << 14)
)

// ----------------------------------------------------------------------------
// Window class
// ----------------------------------------------------------------------------

const (
	_InputOutput = 1
)

// ----------------------------------------------------------------------------
// _NET_WM_STATE action constants
// ----------------------------------------------------------------------------

const (
	_NET_WM_STATE_REMOVE = 0
	_NET_WM_STATE_ADD    = 1
	_NET_WM_STATE_TOGGLE = 2
)

// ----------------------------------------------------------------------------
// XEvent — raw 192-byte buffer (24 × int64)
// ----------------------------------------------------------------------------

type _XEvent [24]int64

func (e *_XEvent) eventType() int32 {
	return *(*int32)(unsafe.Pointer(e))
}

// window returns the XID of the window the event belongs to.
// In XAnyEvent the window field is at offset 32 (after type[4], pad[4], serial[8], send_event[4], pad[4], display[8]).
func (e *_XEvent) window() uint64 {
	return *(*uint64)(unsafe.Pointer(uintptr(unsafe.Pointer(e)) + 32))
}

// ----------------------------------------------------------------------------
// XKeyEvent (96 bytes)
// ----------------------------------------------------------------------------

type _XKeyEvent struct {
	Type       int32
	_pad0      int32
	Serial     uint64
	SendEvent  int32
	_pad1      int32
	Display    uintptr
	Window     uint64
	Root       uint64
	Subwindow  uint64
	Time       uint64
	X, Y       int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Keycode    uint32
	SameScreen int32
	_pad2      int32
}

// ----------------------------------------------------------------------------
// _XButtonEvent (96 bytes)
// ----------------------------------------------------------------------------

type _XButtonEvent struct {
	Type       int32
	_pad0      int32
	Serial     uint64
	SendEvent  int32
	_pad1      int32
	Display    uintptr
	Window     uint64
	Root       uint64
	Subwindow  uint64
	Time       uint64
	X, Y       int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Button     uint32
	SameScreen int32
	_pad2      int32
}

// ----------------------------------------------------------------------------
// _XMotionEvent (96 bytes)
// ----------------------------------------------------------------------------

type _XMotionEvent struct {
	Type       int32
	_pad0      int32
	Serial     uint64
	SendEvent  int32
	_pad1      int32
	Display    uintptr
	Window     uint64
	Root       uint64
	Subwindow  uint64
	Time       uint64
	X, Y       int32
	XRoot      int32
	YRoot      int32
	State      uint32
	IsHint     int8
	_pad2      [3]int8
	SameScreen int32
	_pad3      int32
}

// ----------------------------------------------------------------------------
// _XCrossingEvent
// ----------------------------------------------------------------------------

type _XCrossingEvent struct {
	Type       int32
	_pad0      int32
	Serial     uint64
	SendEvent  int32
	_pad1      int32
	Display    uintptr
	Window     uint64
	Root       uint64
	Subwindow  uint64
	Time       uint64
	X, Y       int32
	XRoot      int32
	YRoot      int32
	Mode       int32
	Detail     int32
	SameScreen int32
	Focus      int32
	State      uint32
	_pad2      int32
}

// ----------------------------------------------------------------------------
// _XFocusChangeEvent
// ----------------------------------------------------------------------------

type _XFocusChangeEvent struct {
	Type      int32
	_pad0     int32
	Serial    uint64
	SendEvent int32
	_pad1     int32
	Display   uintptr
	Window    uint64
	Mode      int32
	Detail    int32
}

// ----------------------------------------------------------------------------
// _XConfigureEvent
// ----------------------------------------------------------------------------

type _XConfigureEvent struct {
	Type             int32
	_pad0            int32
	Serial           uint64
	SendEvent        int32
	_pad1            int32
	Display          uintptr
	Event            uint64
	Window           uint64
	X, Y             int32
	Width            int32
	Height           int32
	BorderWidth      int32
	_pad2            int32
	Above            uint64
	OverrideRedirect int32
	_pad3            int32
}

// ----------------------------------------------------------------------------
// _XClientMessageEvent
// ----------------------------------------------------------------------------

type _XClientMessageEvent struct {
	Type        int32
	_pad0       int32
	Serial      uint64
	SendEvent   int32
	_pad1       int32
	Display     uintptr
	Window      uint64
	MessageType uint64 // Atom
	Format      int32
	_pad2       int32
	Data        [5]int64 // union: long[5]
}

// ----------------------------------------------------------------------------
// _XSetWindowAttributes
// ----------------------------------------------------------------------------

type _XSetWindowAttributes struct {
	BackgroundPixmap uint64
	BackgroundPixel  uint64
	BorderPixmap     uint64
	BorderPixel      uint64
	BitGravity       int32
	WinGravity       int32
	BackingStore     int32
	_pad0            int32
	BackingPlanes    uint64
	BackingPixel     uint64
	SaveUnder        int32
	_pad1            int32
	EventMask        int64
	DoNotPropagate   int64
	OverrideRedirect int32
	_pad2            int32
	Colormap         uint64
	Cursor           uint64
}

// ----------------------------------------------------------------------------
// _XWindowAttributes
// ----------------------------------------------------------------------------

type _XWindowAttributes struct {
	X, Y               int32
	Width, Height      int32
	BorderWidth        int32
	Depth              int32
	Visual             uintptr
	Root               uint64
	Class              int32
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      uint64
	BackingPixel       uint64
	SaveUnder          int32
	_pad0              int32
	Colormap           uint64
	MapInstalled       int32
	MapState           int32
	AllEventMasks      int64
	YourEventMask      int64
	DoNotPropagateMask int64
	OverrideRedirect   int32
	_pad1              int32
	Screen             uintptr
}
