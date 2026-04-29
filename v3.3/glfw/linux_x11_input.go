//go:build linux

package glfw

import "unsafe"

// maxKey is one past the highest Key value (KeyLast = KeyMenu = 348).
const maxKey = int(KeyLast) + 1

// handleKeyEvent processes a key press or release.
func handleKeyEvent(w *Window, ke *_XKeyEvent, pressed bool) {
	action := Release
	if pressed {
		action = Press
	}

	ksym := xLookupKeysym(uintptr(unsafe.Pointer(ke)), 0)
	key := keysymToKey(ksym)
	scancode := int(ke.Keycode)
	mods := x11ModsToGLFW(ke.State)

	s := getX11State(w.handle)
	idx := int(key)
	if idx >= 0 && idx < len(s.keyStates) {
		s.keyStates[idx] = pressed
	}

	if w.fKeyHolder != nil {
		w.fKeyHolder(w, key, scancode, action, mods)
	}

	// Character callback — only on press
	if pressed && w.fCharHolder != nil {
		var buf [8]byte
		n := xLookupString(uintptr(unsafe.Pointer(ke)), uintptr(unsafe.Pointer(&buf[0])), 8, 0, 0)
		if n > 0 {
			str := string(buf[:n])
			for _, r := range str {
				if r > 0 && r != 127 { // skip DEL
					w.fCharHolder(w, r)
					if w.fCharModsHolder != nil {
						w.fCharModsHolder(w, r, mods)
					}
				}
			}
		}
	}
}

// handleButtonEvent processes a mouse button press or release.
func handleButtonEvent(w *Window, be *_XButtonEvent, pressed bool) {
	// Buttons 4, 5, 6, 7 = scroll wheel
	if pressed && be.Button >= 4 && be.Button <= 7 {
		var xoff, yoff float64
		switch be.Button {
		case 4:
			yoff = 1
		case 5:
			yoff = -1
		case 6:
			xoff = 1
		case 7:
			xoff = -1
		}
		if w.fScrollHolder != nil {
			w.fScrollHolder(w, xoff, yoff)
		}
		return
	}

	btn := x11ButtonToGLFW(be.Button)
	action := Release
	if pressed {
		action = Press
	}
	mods := x11ModsToGLFW(be.State)

	s := getX11State(w.handle)
	idx := int(btn)
	if idx >= 0 && idx < len(s.mouseStates) {
		s.mouseStates[idx] = pressed
	}

	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, btn, action, mods)
	}
}

func x11ButtonToGLFW(button uint32) MouseButton {
	switch button {
	case 1:
		return MouseButtonLeft
	case 2:
		return MouseButtonMiddle
	case 3:
		return MouseButtonRight
	default:
		return MouseButton(button - 1)
	}
}

func x11ModsToGLFW(state uint32) ModifierKey {
	var mods ModifierKey
	if state&(1<<0) != 0 {
		mods |= ModShift
	}
	if state&(1<<2) != 0 {
		mods |= ModControl
	}
	if state&(1<<3) != 0 {
		mods |= ModAlt
	}
	if state&(1<<6) != 0 {
		mods |= ModSuper
	}
	return mods
}

// keysymToKey maps an X11 KeySym to a GLFW Key constant.
// Only KeySym values that map to defined Key constants are included.
func keysymToKey(ksym uint64) Key {
	switch ksym {
	case 0x0020:
		return KeySpace
	case 0x0027:
		return KeyApostrophe
	case 0x002C:
		return KeyComma
	case 0x002D:
		return KeyMinus
	case 0x002E:
		return KeyPeriod
	case 0x002F:
		return KeySlash
	case 0x0030:
		return Key0
	case 0x0031:
		return Key1
	case 0x0032:
		return Key2
	case 0x0033:
		return Key3
	case 0x0034:
		return Key4
	case 0x0035:
		return Key5
	case 0x0036:
		return Key6
	case 0x0037:
		return Key7
	case 0x0038:
		return Key8
	case 0x0039:
		return Key9
	case 0x003B:
		return KeySemicolon
	case 0x003D:
		return KeyEqual
	case 0x0041, 0x0061:
		return KeyA
	case 0x0042, 0x0062:
		return KeyB
	case 0x0043, 0x0063:
		return KeyC
	case 0x0044, 0x0064:
		return KeyD
	case 0x0045, 0x0065:
		return KeyE
	case 0x0046, 0x0066:
		return KeyF
	case 0x0047, 0x0067:
		return KeyG
	case 0x0048, 0x0068:
		return KeyH
	case 0x0049, 0x0069:
		return KeyI
	case 0x004A, 0x006A:
		return KeyJ
	case 0x004B, 0x006B:
		return KeyK
	case 0x004C, 0x006C:
		return KeyL
	case 0x004D, 0x006D:
		return KeyM
	case 0x004E, 0x006E:
		return KeyN
	case 0x004F, 0x006F:
		return KeyO
	case 0x0050, 0x0070:
		return KeyP
	case 0x0051, 0x0071:
		return KeyQ
	case 0x0052, 0x0072:
		return KeyR
	case 0x0053, 0x0073:
		return KeyS
	case 0x0054, 0x0074:
		return KeyT
	case 0x0055, 0x0075:
		return KeyU
	case 0x0056, 0x0076:
		return KeyV
	case 0x0057, 0x0077:
		return KeyW
	case 0x0058, 0x0078:
		return KeyX
	case 0x0059, 0x0079:
		return KeyY
	case 0x005A, 0x007A:
		return KeyZ
	case 0x005B:
		return KeyLeftBracket
	case 0x005C:
		return KeyBackslash
	case 0x005D:
		return KeyRightBracket
	case 0x0060:
		return KeyGraveAccent
	case 0xFF08:
		return KeyBackspace
	case 0xFF09:
		return KeyTab
	case 0xFF0D:
		return KeyEnter
	case 0xFF13:
		return KeyPause
	case 0xFF14:
		return KeyScrollLock
	case 0xFF1B:
		return KeyEscape
	case 0xFF50:
		return KeyHome
	case 0xFF51:
		return KeyLeft
	case 0xFF52:
		return KeyUp
	case 0xFF53:
		return KeyRight
	case 0xFF54:
		return KeyDown
	case 0xFF55:
		return KeyPageUp
	case 0xFF56:
		return KeyPageDown
	case 0xFF57:
		return KeyEnd
	case 0xFF61:
		return KeyPrintScreen
	case 0xFF63:
		return KeyInsert
	case 0xFF67:
		return KeyMenu
	case 0xFF7F:
		return KeyNumLock
	case 0xFF8D:
		return KeyKPEnter
	case 0xFFAA:
		return KeyKPMultiply
	case 0xFFAB:
		return KeyKPAdd
	case 0xFFAD:
		return KeyKPSubtract
	case 0xFFAE:
		return KeyKPDecimal
	case 0xFFAF:
		return KeyKPDivide
	case 0xFFB0:
		return KeyKP0
	case 0xFFB1:
		return KeyKP1
	case 0xFFB2:
		return KeyKP2
	case 0xFFB3:
		return KeyKP3
	case 0xFFB4:
		return KeyKP4
	case 0xFFB5:
		return KeyKP5
	case 0xFFB6:
		return KeyKP6
	case 0xFFB7:
		return KeyKP7
	case 0xFFB8:
		return KeyKP8
	case 0xFFB9:
		return KeyKP9
	case 0xFFBE:
		return KeyF1
	case 0xFFBF:
		return KeyF2
	case 0xFFC0:
		return KeyF3
	case 0xFFC1:
		return KeyF4
	case 0xFFC2:
		return KeyF5
	case 0xFFC3:
		return KeyF6
	case 0xFFC4:
		return KeyF7
	case 0xFFC5:
		return KeyF8
	case 0xFFC6:
		return KeyF9
	case 0xFFC7:
		return KeyF10
	case 0xFFC8:
		return KeyF11
	case 0xFFC9:
		return KeyF12
	case 0xFFCA:
		return KeyF13
	case 0xFFCB:
		return KeyF14
	case 0xFFCC:
		return KeyF15
	case 0xFFCD:
		return KeyF16
	case 0xFFCE:
		return KeyF17
	case 0xFFCF:
		return KeyF18
	case 0xFFD0:
		return KeyF19
	case 0xFFD1:
		return KeyF20
	case 0xFFD2:
		return KeyF21
	case 0xFFD3:
		return KeyF22
	case 0xFFD4:
		return KeyF23
	case 0xFFD5:
		return KeyF24
	case 0xFFD6:
		return KeyF25
	case 0xFFE1:
		return KeyLeftShift
	case 0xFFE2:
		return KeyRightShift
	case 0xFFE3:
		return KeyLeftControl
	case 0xFFE4:
		return KeyRightControl
	case 0xFFE7:
		return KeyLeftSuper
	case 0xFFE8:
		return KeyRightSuper
	case 0xFFE9:
		return KeyLeftAlt
	case 0xFFEA:
		return KeyRightAlt
	case 0xFFE5:
		return KeyCapsLock
	case 0xFFFF:
		return KeyDelete
	}
	return KeyUnknown
}
