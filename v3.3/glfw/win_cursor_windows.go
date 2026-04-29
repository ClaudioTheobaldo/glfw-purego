//go:build windows

package glfw

// CreateCursor creates a custom cursor from an RGBA image.
// xhot and yhot specify the cursor hotspot in pixels from the top-left corner.
// The caller must call DestroyCursor when the cursor is no longer needed.
func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) {
	if image == nil {
		return nil, &Error{Code: InvalidValue, Desc: "nil image"}
	}
	h := createHICONCursor(*image, xhot, yhot, false)
	if h == 0 {
		return nil, &Error{Code: PlatformError, Desc: "CreateCursor: CreateIconIndirect failed"}
	}
	return &Cursor{handle: h}, nil
}

// CreateStandardCursor returns a cursor with one of the standard shapes.
// The caller must call DestroyCursor when done.
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) {
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
		return nil, &Error{Code: InvalidValue, Desc: "unknown cursor shape"}
	}
	h, err := loadCursorW(0, idc)
	if err != nil || h == 0 {
		return nil, &Error{Code: PlatformError, Desc: "CreateStandardCursor: LoadCursor failed"}
	}
	// System cursors must NOT be destroyed; mark them with system=true.
	return &Cursor{handle: h, system: true}, nil
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
