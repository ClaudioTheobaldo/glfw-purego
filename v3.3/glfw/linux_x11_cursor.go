//go:build linux && !wayland

package glfw

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ----------------------------------------------------------------------------
// X11 font cursor shape constants (XC_*)
// ----------------------------------------------------------------------------

const (
	_XC_X_cursor             = uint32(0)
	_XC_bottom_left_corner   = uint32(12)
	_XC_bottom_right_corner  = uint32(14)
	_XC_crosshair            = uint32(34)
	_XC_fleur                = uint32(52)
	_XC_hand2                = uint32(60)
	_XC_left_ptr             = uint32(68)
	_XC_sb_h_double_arrow    = uint32(108)
	_XC_sb_v_double_arrow    = uint32(116)
	_XC_xterm                = uint32(152)
)

// ----------------------------------------------------------------------------
// XcursorImage — mirrors C struct (32 bytes on 64-bit Linux)
// ----------------------------------------------------------------------------

// _XcursorImage mirrors XcursorImage from Xcursor.h.
// 6×uint32 = 24 bytes before Pixels; 24 is already 8-byte aligned, so no padding.
type _XcursorImage struct {
	Version uint32  // offset 0
	Size    uint32  // offset 4
	Width   uint32  // offset 8
	Height  uint32  // offset 12
	XHot    uint32  // offset 16
	YHot    uint32  // offset 20
	Pixels  uintptr // offset 24 — XcursorPixel* (ARGB, 32-bit per pixel)
}

// _XColor mirrors the C XColor struct (16 bytes on 64-bit Linux).
type _XColor struct {
	Pixel          uint64
	Red, Green, Blue uint16
	Flags, Pad     int8
}

// ----------------------------------------------------------------------------
// Global shared invisible cursor (created lazily; freed in Terminate)
// ----------------------------------------------------------------------------

var x11InvisibleCursor uint64

// getInvisibleCursor returns the shared invisible cursor, creating it if needed.
func getInvisibleCursor() uint64 {
	if x11InvisibleCursor == 0 {
		x11InvisibleCursor = createInvisibleCursor()
	}
	return x11InvisibleCursor
}

// createInvisibleCursor creates a fully transparent 1×1 cursor via
// XCreateBitmapFromData + XCreatePixmapCursor.
func createInvisibleCursor() uint64 {
	data := [1]byte{0}
	pixmap := xCreateBitmapFromData(x11Display, x11Root, uintptr(unsafe.Pointer(&data[0])), 1, 1)
	if pixmap == 0 {
		return 0
	}
	defer xFreePixmap(x11Display, pixmap)
	var black _XColor
	return xCreatePixmapCursor(x11Display, pixmap, pixmap,
		uintptr(unsafe.Pointer(&black)), uintptr(unsafe.Pointer(&black)), 0, 0)
}

// ----------------------------------------------------------------------------
// Xcursor lazy loader (libXcursor.so.1 — optional, for custom image cursors)
// ----------------------------------------------------------------------------

var (
	xcursorOnce   sync.Once
	xcursorHandle uintptr
	xcursorErr    error

	xcursorImageCreate     func(width, height int32) uintptr
	xcursorImageDestroy    func(image uintptr)
	xcursorImageLoadCursor func(display, image uintptr) uint64
)

func loadXcursor() error {
	xcursorOnce.Do(func() {
		for _, name := range []string{"libXcursor.so.1", "libXcursor.so"} {
			xcursorHandle, xcursorErr = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if xcursorErr == nil {
				break
			}
		}
		if xcursorErr != nil {
			return
		}
		purego.RegisterLibFunc(&xcursorImageCreate, xcursorHandle, "XcursorImageCreate")
		purego.RegisterLibFunc(&xcursorImageDestroy, xcursorHandle, "XcursorImageDestroy")
		purego.RegisterLibFunc(&xcursorImageLoadCursor, xcursorHandle, "XcursorImageLoadCursor")
	})
	return xcursorErr
}

// ----------------------------------------------------------------------------
// StandardCursorShape → XC_* mapping
// ----------------------------------------------------------------------------

func standardCursorXShape(shape StandardCursorShape) uint32 {
	switch shape {
	case ArrowCursor:
		return _XC_left_ptr
	case IBeamCursor:
		return _XC_xterm
	case CrosshairCursor:
		return _XC_crosshair
	case HandCursor, PointingHandCursor:
		return _XC_hand2
	case HResizeCursor, ResizeEWCursor:
		return _XC_sb_h_double_arrow
	case VResizeCursor, ResizeNSCursor:
		return _XC_sb_v_double_arrow
	case ResizeNWSECursor:
		return _XC_bottom_right_corner
	case ResizeNESWCursor:
		return _XC_bottom_left_corner
	case ResizeAllCursor:
		return _XC_fleur
	case NotAllowedCursor:
		return _XC_X_cursor
	default:
		return _XC_left_ptr
	}
}

// ----------------------------------------------------------------------------
// Public cursor API
// ----------------------------------------------------------------------------

// CreateStandardCursor returns a cursor with the given built-in shape.
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) {
	if err := initX11Display(); err != nil {
		return nil, err
	}
	handle := xCreateFontCursor(x11Display, standardCursorXShape(shape))
	if handle == 0 {
		return nil, &Error{Code: PlatformError, Desc: "XCreateFontCursor failed"}
	}
	return &Cursor{handle: uintptr(handle), system: true}, nil
}

// CreateCursor creates a cursor from a custom RGBA image.
func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) {
	if err := initX11Display(); err != nil {
		return nil, err
	}
	if err := loadXcursor(); err != nil {
		return nil, &Error{Code: APIUnavailable, Desc: "libXcursor not available: " + err.Error()}
	}

	// XcursorImageCreate allocates the image and its pixel buffer in C memory.
	imgPtr := xcursorImageCreate(int32(image.Width), int32(image.Height))
	if imgPtr == 0 {
		return nil, &Error{Code: PlatformError, Desc: "XcursorImageCreate failed"}
	}
	defer xcursorImageDestroy(imgPtr)

	ci := (*_XcursorImage)(nativePtrFromUintptr(imgPtr))
	ci.XHot = uint32(xhot)
	ci.YHot = uint32(yhot)

	// Fill pixels from C-allocated buffer: convert RGBA → ARGB.
	pixSlice := unsafe.Slice((*uint32)(nativePtrFromUintptr(ci.Pixels)), image.Width*image.Height)
	src := image.Pixels
	for i := range pixSlice {
		r := uint32(src[i*4+0])
		g := uint32(src[i*4+1])
		b := uint32(src[i*4+2])
		a := uint32(src[i*4+3])
		pixSlice[i] = (a << 24) | (r << 16) | (g << 8) | b
	}
	runtime.KeepAlive(image)

	handle := xcursorImageLoadCursor(x11Display, imgPtr)
	if handle == 0 {
		return nil, &Error{Code: PlatformError, Desc: "XcursorImageLoadCursor failed"}
	}
	return &Cursor{handle: uintptr(handle), system: false}, nil
}

// DestroyCursor frees a cursor object.
func DestroyCursor(cursor *Cursor) {
	if cursor == nil || cursor.handle == 0 || x11Display == 0 {
		return
	}
	xFreeCursor(x11Display, uint64(cursor.handle))
	cursor.handle = 0
}

// Destroy is a convenience method that calls DestroyCursor.
func (c *Cursor) Destroy() { DestroyCursor(c) }

// SetCursor sets the cursor shape for the window.
// Pass nil to revert to the system default arrow.
func (w *Window) SetCursor(cursor *Cursor) {
	if x11Display == 0 || w.handle == 0 {
		return
	}
	if cursor == nil {
		w.cursor = 0
		xDefineCursor(x11Display, uint64(w.handle), 0)
	} else {
		w.cursor = cursor.handle
		xDefineCursor(x11Display, uint64(w.handle), uint64(cursor.handle))
	}
	xFlush(x11Display)
}
