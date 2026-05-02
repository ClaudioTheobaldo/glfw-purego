//go:build linux && wayland

package glfw

import "syscall"

// ── evdev keycode → GLFW Key ──────────────────────────────────────────────────

// evdevToGLFW maps Linux/evdev scancodes (as carried by wl_keyboard.key) to
// GLFW Key constants.  The table is sparse; unsupported entries are KeyUnknown.
var evdevToGLFW [512]Key

func init() {
	for i := range evdevToGLFW {
		evdevToGLFW[i] = KeyUnknown
	}
	evdevToGLFW[1]   = KeyEscape
	evdevToGLFW[2]   = Key1
	evdevToGLFW[3]   = Key2
	evdevToGLFW[4]   = Key3
	evdevToGLFW[5]   = Key4
	evdevToGLFW[6]   = Key5
	evdevToGLFW[7]   = Key6
	evdevToGLFW[8]   = Key7
	evdevToGLFW[9]   = Key8
	evdevToGLFW[10]  = Key9
	evdevToGLFW[11]  = Key0
	evdevToGLFW[12]  = KeyMinus
	evdevToGLFW[13]  = KeyEqual
	evdevToGLFW[14]  = KeyBackspace
	evdevToGLFW[15]  = KeyTab
	evdevToGLFW[16]  = KeyQ
	evdevToGLFW[17]  = KeyW
	evdevToGLFW[18]  = KeyE
	evdevToGLFW[19]  = KeyR
	evdevToGLFW[20]  = KeyT
	evdevToGLFW[21]  = KeyY
	evdevToGLFW[22]  = KeyU
	evdevToGLFW[23]  = KeyI
	evdevToGLFW[24]  = KeyO
	evdevToGLFW[25]  = KeyP
	evdevToGLFW[26]  = KeyLeftBracket
	evdevToGLFW[27]  = KeyRightBracket
	evdevToGLFW[28]  = KeyEnter
	evdevToGLFW[29]  = KeyLeftControl
	evdevToGLFW[30]  = KeyA
	evdevToGLFW[31]  = KeyS
	evdevToGLFW[32]  = KeyD
	evdevToGLFW[33]  = KeyF
	evdevToGLFW[34]  = KeyG
	evdevToGLFW[35]  = KeyH
	evdevToGLFW[36]  = KeyJ
	evdevToGLFW[37]  = KeyK
	evdevToGLFW[38]  = KeyL
	evdevToGLFW[39]  = KeySemicolon
	evdevToGLFW[40]  = KeyApostrophe
	evdevToGLFW[41]  = KeyGraveAccent
	evdevToGLFW[42]  = KeyLeftShift
	evdevToGLFW[43]  = KeyBackslash
	evdevToGLFW[44]  = KeyZ
	evdevToGLFW[45]  = KeyX
	evdevToGLFW[46]  = KeyC
	evdevToGLFW[47]  = KeyV
	evdevToGLFW[48]  = KeyB
	evdevToGLFW[49]  = KeyN
	evdevToGLFW[50]  = KeyM
	evdevToGLFW[51]  = KeyComma
	evdevToGLFW[52]  = KeyPeriod
	evdevToGLFW[53]  = KeySlash
	evdevToGLFW[54]  = KeyRightShift
	evdevToGLFW[55]  = KeyKPMultiply
	evdevToGLFW[56]  = KeyLeftAlt
	evdevToGLFW[57]  = KeySpace
	evdevToGLFW[58]  = KeyCapsLock
	evdevToGLFW[59]  = KeyF1
	evdevToGLFW[60]  = KeyF2
	evdevToGLFW[61]  = KeyF3
	evdevToGLFW[62]  = KeyF4
	evdevToGLFW[63]  = KeyF5
	evdevToGLFW[64]  = KeyF6
	evdevToGLFW[65]  = KeyF7
	evdevToGLFW[66]  = KeyF8
	evdevToGLFW[67]  = KeyF9
	evdevToGLFW[68]  = KeyF10
	evdevToGLFW[69]  = KeyNumLock
	evdevToGLFW[70]  = KeyScrollLock
	evdevToGLFW[71]  = KeyKP7
	evdevToGLFW[72]  = KeyKP8
	evdevToGLFW[73]  = KeyKP9
	evdevToGLFW[74]  = KeyKPSubtract
	evdevToGLFW[75]  = KeyKP4
	evdevToGLFW[76]  = KeyKP5
	evdevToGLFW[77]  = KeyKP6
	evdevToGLFW[78]  = KeyKPAdd
	evdevToGLFW[79]  = KeyKP1
	evdevToGLFW[80]  = KeyKP2
	evdevToGLFW[81]  = KeyKP3
	evdevToGLFW[82]  = KeyKP0
	evdevToGLFW[83]  = KeyKPDecimal
	evdevToGLFW[87]  = KeyF11
	evdevToGLFW[88]  = KeyF12
	evdevToGLFW[96]  = KeyKPEnter
	evdevToGLFW[97]  = KeyRightControl
	evdevToGLFW[98]  = KeyKPDivide
	evdevToGLFW[99]  = KeyPrintScreen
	evdevToGLFW[100] = KeyRightAlt
	evdevToGLFW[102] = KeyHome
	evdevToGLFW[103] = KeyUp
	evdevToGLFW[104] = KeyPageUp
	evdevToGLFW[105] = KeyLeft
	evdevToGLFW[106] = KeyRight
	evdevToGLFW[107] = KeyEnd
	evdevToGLFW[108] = KeyDown
	evdevToGLFW[109] = KeyPageDown
	evdevToGLFW[110] = KeyInsert
	evdevToGLFW[111] = KeyDelete
	evdevToGLFW[117] = KeyKPEqual
	evdevToGLFW[119] = KeyPause
	evdevToGLFW[125] = KeyLeftSuper
	evdevToGLFW[126] = KeyRightSuper
	evdevToGLFW[127] = KeyMenu
	evdevToGLFW[183] = KeyF13
	evdevToGLFW[184] = KeyF14
	evdevToGLFW[185] = KeyF15
	evdevToGLFW[186] = KeyF16
	evdevToGLFW[187] = KeyF17
	evdevToGLFW[188] = KeyF18
	evdevToGLFW[189] = KeyF19
	evdevToGLFW[190] = KeyF20
	evdevToGLFW[191] = KeyF21
	evdevToGLFW[192] = KeyF22
	evdevToGLFW[193] = KeyF23
	evdevToGLFW[194] = KeyF24
}

// ── US QWERTY character table ─────────────────────────────────────────────────

// evdevCharNorm / evdevCharShift map an evdev scancode to its unshifted / shifted
// character in a US QWERTY layout.  Zero means "no printable character".
var (
	evdevCharNorm  [256]rune
	evdevCharShift [256]rune
)

func init() {
	type pair struct{ n, s rune }
	m := map[int]pair{
		2:  {'1', '!'}, 3: {'2', '@'}, 4: {'3', '#'}, 5: {'4', '$'},
		6:  {'5', '%'}, 7: {'6', '^'}, 8: {'7', '&'}, 9: {'8', '*'},
		10: {'9', '('}, 11: {'0', ')'}, 12: {'-', '_'}, 13: {'=', '+'},
		16: {'q', 'Q'}, 17: {'w', 'W'}, 18: {'e', 'E'}, 19: {'r', 'R'},
		20: {'t', 'T'}, 21: {'y', 'Y'}, 22: {'u', 'U'}, 23: {'i', 'I'},
		24: {'o', 'O'}, 25: {'p', 'P'}, 26: {'[', '{'}, 27: {']', '}'},
		30: {'a', 'A'}, 31: {'s', 'S'}, 32: {'d', 'D'}, 33: {'f', 'F'},
		34: {'g', 'G'}, 35: {'h', 'H'}, 36: {'j', 'J'}, 37: {'k', 'K'},
		38: {'l', 'L'}, 39: {';', ':'}, 40: {'\'', '"'}, 41: {'`', '~'},
		43: {'\\', '|'},
		44: {'z', 'Z'}, 45: {'x', 'X'}, 46: {'c', 'C'}, 47: {'v', 'V'},
		48: {'b', 'B'}, 49: {'n', 'N'}, 50: {'m', 'M'},
		51: {',', '<'}, 52: {'.', '>'}, 53: {'/', '?'},
		57: {' ', ' '},
	}
	for evdev, p := range m {
		evdevCharNorm[evdev] = p.n
		evdevCharShift[evdev] = p.s
	}
}

// wlKeyChar returns the Unicode character for the given evdev code and modifiers.
// Returns 0 for non-printable keys.
func wlKeyChar(evdev uint32, mods ModifierKey) rune {
	if int(evdev) >= len(evdevCharNorm) {
		return 0
	}
	var c rune
	if mods&ModShift != 0 {
		c = evdevCharShift[evdev]
	} else {
		c = evdevCharNorm[evdev]
	}
	// CapsLock flips case for Latin letters only
	if mods&ModCapsLock != 0 {
		if c >= 'a' && c <= 'z' {
			c -= 32
		} else if c >= 'A' && c <= 'Z' {
			c += 32
		}
	}
	return c
}

// ── XKB modifier mask → GLFW ModifierKey ─────────────────────────────────────

// xkbToGLFWMods maps the XKB depressed+latched and locked modifier masks to
// the GLFW ModifierKey bitmask.
//
// Standard XKB modifier bits (from xkbcommon):
//   Shift=1  Lock=2  Ctrl=4  Mod1(Alt)=8  Mod2(NumLock)=16  Mod4(Super)=64
func xkbToGLFWMods(depressed, latched, locked uint32) ModifierKey {
	active := depressed | latched
	var m ModifierKey
	if active&1 != 0 {
		m |= ModShift
	}
	if active&4 != 0 {
		m |= ModControl
	}
	if active&8 != 0 {
		m |= ModAlt
	}
	if active&64 != 0 {
		m |= ModSuper
	}
	if locked&2 != 0 {
		m |= ModCapsLock
	}
	if locked&16 != 0 {
		m |= ModNumLock
	}
	return m
}

// ── wl_pointer event handlers ─────────────────────────────────────────────────
// C signatures (all pointers = uintptr, wl_fixed_t = int32):
//   enter:   (data, pointer, serial, surface, sx, sy)
//   leave:   (data, pointer, serial, surface)
//   motion:  (data, pointer, time, sx, sy)
//   button:  (data, pointer, serial, time, button, state)
//   axis:    (data, pointer, time, axis, value)
//   frame:   (data, pointer)
//   axis_source:   (data, pointer, source)
//   axis_stop:     (data, pointer, time, axis)
//   axis_discrete: (data, pointer, axis, discrete)

func wlOnPointerEnter(data, pointer uintptr, serial uint32, surface uintptr, sx, sy int32) {
	wl.ptrSerial = serial
	v, ok := windowByHandle.Load(surface)
	if !ok {
		wl.activeWin = nil
		return
	}
	w := v.(*Window)
	wl.activeWin = w
	w.wlCursorX = float64(sx) / 256.0
	w.wlCursorY = float64(sy) / 256.0
	if w.fCursorEnterHolder != nil {
		w.fCursorEnterHolder(w, true)
	}
	if w.fCursorPosHolder != nil {
		w.fCursorPosHolder(w, w.wlCursorX, w.wlCursorY)
	}
}

func wlOnPointerLeave(data, pointer uintptr, serial uint32, surface uintptr) {
	w := wl.activeWin
	wl.activeWin = nil
	if w != nil && w.fCursorEnterHolder != nil {
		w.fCursorEnterHolder(w, false)
	}
}

func wlOnPointerMotion(data, pointer uintptr, time uint32, sx, sy int32) {
	w := wl.activeWin
	if w == nil {
		return
	}
	w.wlCursorX = float64(sx) / 256.0
	w.wlCursorY = float64(sy) / 256.0
	if w.fCursorPosHolder != nil {
		w.fCursorPosHolder(w, w.wlCursorX, w.wlCursorY)
	}
}

func wlOnPointerButton(data, pointer uintptr, serial, time, evdevBtn, state uint32) {
	// Linux BTN_* evdev codes
	const (
		btnLeft   = uint32(0x110)
		btnRight  = uint32(0x111)
		btnMiddle = uint32(0x112)
		btnSide   = uint32(0x113)
		btnExtra  = uint32(0x114)
	)
	var btn MouseButton
	switch evdevBtn {
	case btnLeft:
		btn = MouseButtonLeft
	case btnRight:
		btn = MouseButtonRight
	case btnMiddle:
		btn = MouseButtonMiddle
	case btnSide:
		btn = MouseButton4
	case btnExtra:
		btn = MouseButton5
	default:
		return
	}
	act := Release
	if state == 1 {
		act = Press
		wl.ptrSerial = serial
	}
	wl.btnState[btn] = act
	w := wl.activeWin
	if w != nil && w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, btn, act, wl.kbMods)
	}
}

func wlOnPointerAxis(data, pointer uintptr, time, axis uint32, value int32) {
	// Skip if a discrete event already fired for this axis in this frame.
	if uint(axis) < 2 && wl.axisDiscrete[axis] {
		return
	}
	w := wl.activeWin
	if w == nil || w.fScrollHolder == nil {
		return
	}
	// Normalize: ~120 wl_fixed_t units ≈ one scroll notch on a typical wheel.
	delta := -float64(value) / (120.0 * 256.0)
	if axis == 0 { // vertical
		w.fScrollHolder(w, 0, delta)
	} else { // horizontal
		w.fScrollHolder(w, delta, 0)
	}
}

func wlOnPointerFrame(data, pointer uintptr) {
	wl.axisDiscrete[0] = false
	wl.axisDiscrete[1] = false
}

func wlOnPointerAxisSource(data, pointer uintptr, source uint32) {}

func wlOnPointerAxisStop(data, pointer uintptr, time, axis uint32) {}

func wlOnPointerAxisDiscrete(data, pointer uintptr, axis uint32, discrete int32) {
	if uint(axis) < 2 {
		wl.axisDiscrete[axis] = true
	}
	w := wl.activeWin
	if w == nil || w.fScrollHolder == nil {
		return
	}
	// One discrete unit = one scroll notch (GLFW convention: ±1.0).
	if axis == 0 { // vertical
		w.fScrollHolder(w, 0, float64(-discrete))
	} else { // horizontal
		w.fScrollHolder(w, float64(discrete), 0)
	}
}

// ── wl_keyboard event handlers ────────────────────────────────────────────────
// C signatures:
//   keymap:     (data, keyboard, format, fd, size)    — fd is int32
//   enter:      (data, keyboard, serial, surface, keys_array)
//   leave:      (data, keyboard, serial, surface)
//   key:        (data, keyboard, serial, time, key, state)
//   modifiers:  (data, keyboard, serial, dep, lat, loc, group)
//   repeat_info:(data, keyboard, rate, delay)

func wlOnKeyboardKeymap(data, keyboard uintptr, format uint32, fd int32, size uint32) {
	// We don't use libxkbcommon; just close the fd to prevent a leak.
	syscall.Close(int(fd))
}

func wlOnKeyboardEnter(data, keyboard uintptr, serial uint32, surface, keysArray uintptr) {
	wl.kbSerial = serial
	v, ok := windowByHandle.Load(surface)
	if !ok {
		wl.kbWin = nil
		return
	}
	wl.kbWin = v.(*Window)
	if w := wl.kbWin; w.fFocusHolder != nil {
		w.fFocusHolder(w, true)
	}
}

func wlOnKeyboardLeave(data, keyboard uintptr, serial uint32, surface uintptr) {
	w := wl.kbWin
	wl.kbWin = nil
	if w != nil && w.fFocusHolder != nil {
		w.fFocusHolder(w, false)
	}
}

func wlOnKeyboardKey(data, keyboard uintptr, serial, time, evdev, state uint32) {
	if int(evdev) >= len(evdevToGLFW) {
		return
	}
	key := evdevToGLFW[evdev]

	var act Action
	switch state {
	case 1:
		act = Press
	case 2:
		act = Repeat
	default:
		act = Release
	}

	// Track press/release (not repeat) in global key state.
	if key != KeyUnknown && int(key) < len(wl.keyState) && act != Repeat {
		wl.keyState[key] = act
	}

	w := wl.kbWin
	if w == nil {
		return
	}
	if w.fKeyHolder != nil {
		w.fKeyHolder(w, key, int(evdev), act, wl.kbMods)
	}
	if (act == Press || act == Repeat) && key != KeyUnknown {
		if c := wlKeyChar(evdev, wl.kbMods); c != 0 {
			if w.fCharHolder != nil {
				w.fCharHolder(w, c)
			}
			if w.fCharModsHolder != nil {
				w.fCharModsHolder(w, c, wl.kbMods)
			}
		}
	}
}

func wlOnKeyboardModifiers(data, keyboard uintptr, serial, depressed, latched, locked, group uint32) {
	wl.kbMods = xkbToGLFWMods(depressed, latched, locked)
	wl.kbDepressed = depressed
	wl.kbLocked = locked
}

func wlOnKeyboardRepeatInfo(data, keyboard uintptr, rate, delay int32) {
	// Key-repeat is handled by the compositor; no action needed here.
}
