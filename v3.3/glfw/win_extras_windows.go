//go:build windows

package glfw

import (
	"syscall"
	"unsafe"
)

// ----------------------------------------------------------------------------
// SetSizeLimits — constrain the minimum and maximum resize dimensions.
// Pass -1 (GLFW_DONT_CARE) for any limit you don't want to set.
// ----------------------------------------------------------------------------

// SetSizeLimits sets the minimum and maximum client-area dimensions for the
// window. Pass -1 for any value to leave that limit unconstrained.
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {
	w.minW = minWidth
	w.minH = minHeight
	w.maxW = maxWidth
	w.maxH = maxHeight
	// Trigger WM_GETMINMAXINFO immediately by nudging the window.
	setWindowPos(w.handle, 0, 0, 0, 0, 0,
		_SWP_NOMOVE|_SWP_NOSIZE|_SWP_NOZORDER|_SWP_NOACTIVATE|_SWP_FRAMECHANGED)
}

// SetAspectRatio locks the window resize to the given aspect ratio expressed
// as numer:denom. Pass -1/-1 to remove the constraint.
func (w *Window) SetAspectRatio(numer, denom int) {
	w.aspectNum = numer
	w.aspectDen = denom
}

// applyMinMaxInfo fills a MINMAXINFO from the window's size-limit fields.
// Called from wndProc for WM_GETMINMAXINFO.
func (w *Window) applyMinMaxInfo(mmi *_MINMAXINFO) {
	// Track sizes are the window total size, not the client area.
	// Compute the frame border offset so we can adjust correctly.
	style   := getWindowLongW(w.handle, _GWL_STYLE)
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	var rc _RECT
	adjustWindowRectEx(&rc, style, exStyle, false)
	bw := int(rc.Right  - rc.Left) // border width delta
	bh := int(rc.Bottom - rc.Top)  // border height delta

	if w.minW > 0 {
		mmi.PtMinTrackSize.X = int32(w.minW + bw)
	}
	if w.minH > 0 {
		mmi.PtMinTrackSize.Y = int32(w.minH + bh)
	}
	if w.maxW > 0 {
		mmi.PtMaxTrackSize.X = int32(w.maxW + bw)
	}
	if w.maxH > 0 {
		mmi.PtMaxTrackSize.Y = int32(w.maxH + bh)
	}
}

// enforceAspectRatio adjusts the candidate RECT (from WM_SIZING) to maintain
// the stored aspect ratio. edge is the WM_SIZING wParam (which border is being
// dragged).
func (w *Window) enforceAspectRatio(rc *_RECT, edge uintptr) {
	if w.aspectNum <= 0 || w.aspectDen <= 0 {
		return
	}
	// Compute the frame border offsets (same trick as applyMinMaxInfo).
	style   := getWindowLongW(w.handle, _GWL_STYLE)
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	var adj _RECT
	adjustWindowRectEx(&adj, style, exStyle, false)
	bw := int32(adj.Right - adj.Left)
	bh := int32(adj.Bottom - adj.Top)

	// Client area dimensions proposed by the drag.
	cw := (rc.Right - rc.Left) - bw
	ch := (rc.Bottom - rc.Top) - bh

	ratio := float64(w.aspectNum) / float64(w.aspectDen)

	switch edge {
	case _WMSZ_LEFT, _WMSZ_RIGHT, _WMSZ_BOTTOMLEFT, _WMSZ_BOTTOMRIGHT:
		// Width is driving — fix height.
		ch = int32(float64(cw) / ratio)
	default:
		// Height is driving — fix width.
		cw = int32(float64(ch) * ratio)
	}

	// Write back. Anchor the top-left corner.
	rc.Right = rc.Left + cw + bw
	rc.Bottom = rc.Top + ch + bh
}

// ----------------------------------------------------------------------------
// GetOpacity / SetOpacity — window translucency via WS_EX_LAYERED
// ----------------------------------------------------------------------------

// SetOpacity sets the window opacity in the range [0.0, 1.0] where 0 is fully
// transparent and 1 is fully opaque.
func (w *Window) SetOpacity(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	// Ensure WS_EX_LAYERED is on the window.
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	if exStyle&_WS_EX_LAYERED == 0 {
		setWindowLongW(w.handle, _GWL_EXSTYLE, exStyle|_WS_EX_LAYERED)
	}
	alpha := byte(opacity * 255)
	setLayeredWindowAttributes(w.handle, 0, alpha, _LWA_ALPHA)
}

// GetOpacity returns the current window opacity in the range [0.0, 1.0].
// Returns 1.0 if the layered attribute has not been set.
func (w *Window) GetOpacity() float32 {
	exStyle := getWindowLongW(w.handle, _GWL_EXSTYLE)
	if exStyle&_WS_EX_LAYERED == 0 {
		return 1.0
	}
	var alpha byte
	var flags uint32
	if !getLayeredWindowAttributes(w.handle, nil, &alpha, &flags) {
		return 1.0
	}
	if flags&uint32(_LWA_ALPHA) == 0 {
		return 1.0
	}
	return float32(alpha) / 255.0
}

// ----------------------------------------------------------------------------
// RequestAttention — flash the taskbar button
// ----------------------------------------------------------------------------

// RequestAttention requests the user's attention by flashing the window's
// taskbar button. The flashing stops when the window is focused.
func (w *Window) RequestAttention() {
	fwi := _FLASHWINFO{
		DwFlags: _FLASHW_ALL | _FLASHW_TIMERNOEDIT,
		Hwnd:    w.handle,
		UCount:  0, // flash until focused
	}
	fwi.CbSize = uint32(unsafe.Sizeof(fwi))
	flashWindowEx(&fwi)
}

// ----------------------------------------------------------------------------
// PostEmptyEvent — wake up WaitEvents from another goroutine
// ----------------------------------------------------------------------------

// gPostHwnd is the HWND to which PostEmptyEvent posts WM_NULL.
// Populated with the first window ever created and never cleared, because we
// only need any valid HWND that belongs to our thread's message queue.
var gPostHwnd uintptr

// PostEmptyEvent posts an empty event to the event queue, causing a blocked
// WaitEvents call to return. It is safe to call from any goroutine.
func PostEmptyEvent() {
	hwnd := gPostHwnd
	if hwnd == 0 {
		return
	}
	postMessageW(hwnd, _WM_NULL, 0, 0)
}

// ----------------------------------------------------------------------------
// GetKeyName / GetKeyScancode
// ----------------------------------------------------------------------------

// GetKeyScancode returns the platform-specific scancode for the given key.
// Returns -1 for keys that have no scancode on this platform.
func GetKeyScancode(key Key) int {
	vk := keyToVK(key)
	if vk == 0 {
		return -1
	}
	sc := mapVirtualKeyW(vk, _MAPVK_VK_TO_VSC)
	if sc == 0 {
		return -1
	}
	return int(sc)
}

// GetKeyName returns the localized, printable name of the given key or
// scancode. If key is not KeyUnknown, it takes precedence; otherwise scancode
// is used directly. Returns an empty string if no name is available.
func GetKeyName(key Key, scancode int) string {
	var sc uint32
	if key != KeyUnknown {
		vk := keyToVK(key)
		if vk == 0 {
			return ""
		}
		sc = mapVirtualKeyW(vk, _MAPVK_VK_TO_VSC)
		// Mark extended keys (right-hand modifiers, numpad enter, arrows, etc.)
		switch key {
		case KeyRight, KeyLeft, KeyDown, KeyUp,
			KeyPageUp, KeyPageDown, KeyHome, KeyEnd, KeyInsert, KeyDelete,
			KeyNumLock, KeyRightShift, KeyRightControl, KeyRightAlt, KeyRightSuper,
			KeyPrintScreen:
			sc |= 0x100 // KF_EXTENDED flag in high byte
		}
	} else {
		sc = uint32(scancode)
	}
	if sc == 0 {
		return ""
	}
	// GetKeyNameTextW takes: (scancode << 16) | (extended_bit << 24)
	var extended uintptr
	if sc&0x100 != 0 {
		extended = 1 << 24
		sc &^= 0x100
	}
	lParam := uintptr(sc)<<16 | extended

	buf := make([]uint16, 64)
	n := getKeyNameTextW(lParam, buf)
	if n <= 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

// ----------------------------------------------------------------------------
// GetTimerFrequency / GetTimerValue
// ----------------------------------------------------------------------------

// GetTimerFrequency returns the frequency (ticks per second) of the raw
// high-resolution timer. This is the denominator for GetTimerValue.
func GetTimerFrequency() uint64 {
	return uint64(timeFrequency())
}

// GetTimerValue returns the current value of the raw high-resolution timer.
// Divide by GetTimerFrequency to get seconds elapsed since Init.
func GetTimerValue() uint64 {
	return uint64(timeGetTicks())
}

