//go:build windows

package glfw

// createCursorRGBA is the platform-specific custom-cursor builder; the public
// CreateCursor wrapper in window.go handles the image.Image conversion.
func createCursorRGBA(image *Image, xhot, yhot int) *Cursor {
	if image == nil {
		return nil
	}
	h := createHICONCursor(*image, xhot, yhot, false)
	if h == 0 {
		return nil
	}
	return &Cursor{handle: h}
}

// CreateStandardCursor returns a cursor with one of the standard shapes.
// Mirrors upstream go-gl/glfw v3.3: no error return; nil on failure.
// The caller must call DestroyCursor when done.
func CreateStandardCursor(shape StandardCursorShape) *Cursor {
	var idc uintptr
	switch shape {
	case ArrowCursor:
		idc = _IDC_ARROW
	case IBeamCursor:
		idc = _IDC_IBEAM
	case CrosshairCursor:
		idc = _IDC_CROSS
	case HandCursor:
		idc = _IDC_HAND
	case HResizeCursor:
		idc = _IDC_SIZEWE
	case VResizeCursor:
		idc = _IDC_SIZENS
	case ResizeEWCursor:
		idc = _IDC_SIZEWE
	case ResizeNSCursor:
		idc = _IDC_SIZENS
	case ResizeNWSECursor:
		idc = _IDC_SIZENWSE
	case ResizeNESWCursor:
		idc = _IDC_SIZENESW
	case ResizeAllCursor:
		idc = _IDC_SIZEALL
	case PointingHandCursor:
		idc = _IDC_HAND
	case NotAllowedCursor:
		idc = _IDC_NO
	default:
		return nil
	}
	h, err := loadCursorW(0, idc)
	if err != nil || h == 0 {
		return nil
	}
	// System cursors must NOT be destroyed; mark them with system=true.
	return &Cursor{handle: h, system: true}
}

// DestroyCursor releases resources associated with a cursor created by
// CreateCursor. Do not call this for cursors obtained from CreateStandardCursor.
func DestroyCursor(cursor *Cursor) {
	if cursor == nil || cursor.system || cursor.handle == 0 {
		return
	}
	destroyIcon(cursor.handle)
	cursor.handle = 0
}

// Destroy is a convenience method; it calls DestroyCursor(c).
func (c *Cursor) Destroy() { DestroyCursor(c) }
