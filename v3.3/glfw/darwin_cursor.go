//go:build darwin

// darwin_cursor.go — NSCursor standard and custom cursor support.

package glfw

import (
	"github.com/ebitengine/purego/objc"
)

// ── SEL cache (cursor-specific) ───────────────────────────────────────────────

var (
	// NSCursor shape class methods
	selArrowCursorSel          = objc.RegisterName("arrowCursor")
	selIBeamCursorSel          = objc.RegisterName("IBeamCursor")
	selCrosshairCursorSel      = objc.RegisterName("crosshairCursor")
	selPointingHandCursorSel   = objc.RegisterName("pointingHandCursor")
	selResizeLRCursorSel       = objc.RegisterName("resizeLeftRightCursor")
	selResizeUDCursorSel       = objc.RegisterName("resizeUpDownCursor")
	selClosedHandCursorSel     = objc.RegisterName("closedHandCursor")
	selOperationNotAllowedSel  = objc.RegisterName("operationNotAllowedCursor")

	// Private diagonal resize cursors (stable since macOS 10.8, used by GLFW 3.4)
	selResizeNWSESel = objc.RegisterName("_windowResizeNorthWestSouthEastCursor")
	selResizeNESWSel = objc.RegisterName("_windowResizeNorthEastSouthWestCursor")

	// NSView cursor rect
	selAddCursorRect = objc.RegisterName("addCursorRect:cursor:")
	selBounds        = objc.RegisterName("bounds")

	// NSBitmapImageRep
	selInitBitmapRep   = objc.RegisterName("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:")
	selBitmapData      = objc.RegisterName("bitmapData")

	// NSImage
	selInitWithSize       = objc.RegisterName("initWithSize:")
	selAddRepresentation  = objc.RegisterName("addRepresentation:")

	// NSCursor init
	selInitWithImageHotSpot = objc.RegisterName("initWithImage:hotSpot:")
)

// ── Standard cursor map ────────────────────────────────────────────────────────

// darwinStdCursorSel maps each standard cursor shape to the NSCursor class-method
// selector that returns it.
var darwinStdCursorSel = map[StandardCursorShape]objc.SEL{
	ArrowCursor:        selArrowCursorSel,
	IBeamCursor:        selIBeamCursorSel,
	CrosshairCursor:    selCrosshairCursorSel,
	HandCursor:         selPointingHandCursorSel,
	PointingHandCursor: selPointingHandCursorSel,
	HResizeCursor:      selResizeLRCursorSel,
	VResizeCursor:      selResizeUDCursorSel,
	ResizeEWCursor:     selResizeLRCursorSel,
	ResizeNSCursor:     selResizeUDCursorSel,
	ResizeNWSECursor:   selResizeNWSESel,
	ResizeNESWCursor:   selResizeNESWSel,
	ResizeAllCursor:    selClosedHandCursorSel,
	NotAllowedCursor:   selOperationNotAllowedSel,
}

// ── Public cursor API ─────────────────────────────────────────────────────────

// selRetain is the ObjC retain selector, used to keep singletons alive.
var selRetain = objc.RegisterName("retain")

// CreateStandardCursor returns a Cursor for a built-in system shape.
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) {
	sel, ok := darwinStdCursorSel[shape]
	if !ok {
		sel = selArrowCursorSel // fall back to arrow
	}
	nsCursor := objc.ID(objc.GetClass("NSCursor")).Send(sel)
	// Class-method cursors are autoreleased singletons; retain so they survive
	// beyond the current autorelease pool drain.
	nsCursor.Send(selRetain)
	return &Cursor{handle: uintptr(nsCursor), system: true}, nil
}

// CreateCursor creates a cursor from an RGBA image with the given hot-spot.
func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) {
	w, h := image.Width, image.Height

	// 1. Allocate an NSBitmapImageRep with NULL planes (AppKit allocates the buffer).
	nsDeviceRGB := nsStringFromGoString("NSDeviceRGBColorSpace")
	rep := objc.ID(objc.GetClass("NSBitmapImageRep")).Send(selAlloc).Send(
		selInitBitmapRep,
		uintptr(0),   // bitmapDataPlanes: NULL
		int64(w),     // pixelsWide
		int64(h),     // pixelsHigh
		int64(8),     // bitsPerSample
		int64(4),     // samplesPerPixel (RGBA)
		true,         // hasAlpha
		false,        // isPlanar
		nsDeviceRGB,  // colorSpaceName
		int64(w*4),   // bytesPerRow
		int64(32),    // bitsPerPixel
	)
	if rep == 0 {
		return &Cursor{}, nil
	}
	defer rep.Send(selRelease)

	// 2. Copy RGBA pixels into the rep's buffer.
	dataPtr := objc.Send[uintptr](rep, selBitmapData)
	if dataPtr != 0 {
		dst := (*[1 << 26]byte)(nativePtrFromUintptr(dataPtr))
		copy(dst[:len(image.Pixels)], image.Pixels)
	}

	// 3. Build an NSImage and add the rep.
	nsImg := objc.ID(objc.GetClass("NSImage")).Send(selAlloc).Send(
		selInitWithSize, NSSize{float64(w), float64(h)})
	if nsImg == 0 {
		return &Cursor{}, nil
	}
	nsImg.Send(selAddRepresentation, rep)

	// 4. Create the NSCursor with the image and hot-spot.
	nsCursor := objc.ID(objc.GetClass("NSCursor")).Send(selAlloc).Send(
		selInitWithImageHotSpot,
		nsImg,
		NSPoint{float64(xhot), float64(yhot)},
	)
	nsImg.Send(selRelease)
	if nsCursor == 0 {
		return &Cursor{}, nil
	}
	return &Cursor{handle: uintptr(nsCursor), system: false}, nil
}

// DestroyCursor releases the NSCursor.
func DestroyCursor(c *Cursor) {
	if c == nil || c.handle == 0 {
		return
	}
	objc.ID(c.handle).Send(selRelease)
	c.handle = 0
}

// Destroy is the method form of DestroyCursor — matches go-gl/glfw.
func (c *Cursor) Destroy() { DestroyCursor(c) }

// ── Window.SetCursor ──────────────────────────────────────────────────────────

// SetCursor sets the cursor shape for the window.
// The new cursor takes effect immediately if the pointer is inside the window.
func (w *Window) SetCursor(c *Cursor) {
	if c != nil {
		w.cursor = c.handle
	} else {
		w.cursor = 0
	}
	// Invalidate the cursor rects so AppKit calls resetCursorRects on the view.
	nswin := w.nsWin()
	contentView := nswin.Send(selContentView)
	nswin.Send(selInvalidateCursorRects, contentView)
}

// ── GlfwView resetCursorRects ─────────────────────────────────────────────────

// nsViewResetCursorRects is the GlfwView implementation of -resetCursorRects.
// AppKit calls this whenever cursor-rect tracking needs to be rebuilt.
// We cover the entire view with the window's current cursor.
func nsViewResetCursorRects(self objc.ID, _ objc.SEL) {
	w := windowFromView(self)
	if w == nil {
		return
	}

	bounds := objc.Send[NSRect](self, selBounds)

	var cursor objc.ID
	if w.cursor != 0 {
		cursor = objc.ID(w.cursor)
	} else {
		cursor = objc.ID(objc.GetClass("NSCursor")).Send(selArrowCursorSel)
	}
	self.Send(selAddCursorRect, bounds, cursor)
}

// ── NSImage / NSBitmapImageRep helpers ────────────────────────────────────────

