//go:build darwin

// darwin_input.go — macOS keyboard, mouse, cursor, and scroll input.
//
// Implements a GlfwView Objective-C class (NSView subclass) that receives all
// keyboard and mouse events and maps them to GLFW callbacks.

package glfw

import (
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ── macOS virtual key code → GLFW Key table ───────────────────────────────────
//
// Source: HIToolbox/Events.h kVK_* constants.
// Index = low 7 bits of NSEvent.keyCode.

var darwinKeyTable = [128]Key{
	// ── ANSI letter / punctuation keys ──────────────────────────────────────
	0x00: KeyA,
	0x01: KeyS,
	0x02: KeyD,
	0x03: KeyF,
	0x04: KeyH,
	0x05: KeyG,
	0x06: KeyZ,
	0x07: KeyX,
	0x08: KeyC,
	0x09: KeyV,
	0x0A: KeyUnknown, // kVK_ISO_Section (ISO keyboards only)
	0x0B: KeyB,
	0x0C: KeyQ,
	0x0D: KeyW,
	0x0E: KeyE,
	0x0F: KeyR,

	0x10: KeyY,
	0x11: KeyT,
	0x12: Key1,
	0x13: Key2,
	0x14: Key3,
	0x15: Key4,
	0x16: Key6,
	0x17: Key5,
	0x18: KeyEqual,
	0x19: Key9,
	0x1A: Key7,
	0x1B: KeyMinus,
	0x1C: Key8,
	0x1D: Key0,
	0x1E: KeyRightBracket,
	0x1F: KeyO,

	0x20: KeyU,
	0x21: KeyLeftBracket,
	0x22: KeyI,
	0x23: KeyP,
	0x24: KeyEnter,
	0x25: KeyL,
	0x26: KeyJ,
	0x27: KeyApostrophe,
	0x28: KeyK,
	0x29: KeySemicolon,
	0x2A: KeyBackslash,
	0x2B: KeyComma,
	0x2C: KeySlash,
	0x2D: KeyN,
	0x2E: KeyM,
	0x2F: KeyPeriod,

	// ── Control / editing keys ───────────────────────────────────────────────
	0x30: KeyTab,
	0x31: KeySpace,
	0x32: KeyGraveAccent,
	0x33: KeyBackspace,
	0x34: KeyUnknown,   // no standard mapping
	0x35: KeyEscape,
	0x36: KeyRightSuper, // kVK_RightCommand
	0x37: KeyLeftSuper,  // kVK_Command
	0x38: KeyLeftShift,
	0x39: KeyCapsLock,
	0x3A: KeyLeftAlt,    // kVK_Option
	0x3B: KeyLeftControl,
	0x3C: KeyRightShift,
	0x3D: KeyRightAlt,   // kVK_RightOption
	0x3E: KeyRightControl,
	0x3F: KeyUnknown, // kVK_Function

	// ── Numpad ───────────────────────────────────────────────────────────────
	0x40: KeyF17,
	0x41: KeyKPDecimal,
	0x42: KeyUnknown,
	0x43: KeyKPMultiply,
	0x44: KeyUnknown,
	0x45: KeyKPAdd,
	0x46: KeyUnknown,
	0x47: KeyNumLock,    // kVK_ANSI_KeypadClear ≈ NumLock
	0x48: KeyUnknown,    // kVK_VolumeUp
	0x49: KeyUnknown,    // kVK_VolumeDown
	0x4A: KeyUnknown,    // kVK_Mute
	0x4B: KeyKPDivide,
	0x4C: KeyKPEnter,
	0x4D: KeyUnknown,
	0x4E: KeyKPSubtract,
	0x4F: KeyF18,

	0x50: KeyF19,
	0x51: KeyKPEqual,
	0x52: KeyKP0,
	0x53: KeyKP1,
	0x54: KeyKP2,
	0x55: KeyKP3,
	0x56: KeyKP4,
	0x57: KeyKP5,
	0x58: KeyKP6,
	0x59: KeyKP7,
	0x5A: KeyF20,
	0x5B: KeyKP8,
	0x5C: KeyKP9,
	0x5D: KeyUnknown,
	0x5E: KeyUnknown,
	0x5F: KeyUnknown,

	// ── Function keys (Fn row) ───────────────────────────────────────────────
	0x60: KeyF5,
	0x61: KeyF6,
	0x62: KeyF7,
	0x63: KeyF3,
	0x64: KeyF8,
	0x65: KeyF9,
	0x66: KeyUnknown,
	0x67: KeyF11,
	0x68: KeyUnknown,
	0x69: KeyF13,
	0x6A: KeyF16,
	0x6B: KeyF14,
	0x6C: KeyUnknown,
	0x6D: KeyF10,
	0x6E: KeyMenu,    // kVK_ContextualMenu
	0x6F: KeyF12,

	// ── Navigation cluster ───────────────────────────────────────────────────
	0x70: KeyUnknown,
	0x71: KeyF15,
	0x72: KeyInsert,  // kVK_Help (acts as Insert on extended keyboards)
	0x73: KeyHome,
	0x74: KeyPageUp,
	0x75: KeyDelete,  // kVK_ForwardDelete (forward-delete, not backspace)
	0x76: KeyF4,
	0x77: KeyEnd,
	0x78: KeyF2,
	0x79: KeyPageDown,
	0x7A: KeyF1,
	0x7B: KeyLeft,
	0x7C: KeyRight,
	0x7D: KeyDown,
	0x7E: KeyUp,
	0x7F: KeyUnknown,
}

// ── Modifier flag bit masks (NSEventModifierFlags) ────────────────────────────

const (
	nsModifierCapsLock = uint64(1 << 16) // NSEventModifierFlagCapsLock
	nsModifierShift    = uint64(1 << 17) // NSEventModifierFlagShift
	nsModifierControl  = uint64(1 << 18) // NSEventModifierFlagControl
	nsModifierOption   = uint64(1 << 19) // NSEventModifierFlagOption  (Alt)
	nsModifierCommand  = uint64(1 << 20) // NSEventModifierFlagCommand (Super)
)

// NSTrackingArea option flags
const (
	nsTrackingMouseEnteredAndExited = uint64(0x01)
	nsTrackingActiveInKeyWindow     = uint64(0x20)
	nsTrackingInVisibleRect         = uint64(0x200)
)

// ── Input SEL cache ───────────────────────────────────────────────────────────

var (
	// NSEvent
	selKeyCode                     = objc.RegisterName("keyCode")
	selModifierFlagsEv             = objc.RegisterName("modifierFlags")
	selLocationInWindow            = objc.RegisterName("locationInWindow")
	selCharacters                  = objc.RegisterName("characters")
	selCharactersIgnoringMods      = objc.RegisterName("charactersIgnoringModifiers")
	selDeltaX                      = objc.RegisterName("deltaX")
	selDeltaY                      = objc.RegisterName("deltaY")
	selScrollingDeltaX             = objc.RegisterName("scrollingDeltaX")
	selScrollingDeltaY             = objc.RegisterName("scrollingDeltaY")
	selHasPreciseScrollingDeltas   = objc.RegisterName("hasPreciseScrollingDeltas")
	selButtonNumber                = objc.RegisterName("buttonNumber")

	// NSView
	selViewWindow         = objc.RegisterName("window")
	selTrackingAreas      = objc.RegisterName("trackingAreas")
	selAddTrackingArea    = objc.RegisterName("addTrackingArea:")
	selRemoveTrackingArea = objc.RegisterName("removeTrackingArea:")

	// NSWindow input helpers
	selMouseLocationOutside = objc.RegisterName("mouseLocationOutsideOfEventStream")
	selConvertRectToScreen  = objc.RegisterName("convertRectToScreen:")

	// NSTrackingArea
	selInitTrackingArea = objc.RegisterName("initWithRect:options:owner:userInfo:")
)

// ── CoreGraphics cursor helpers ───────────────────────────────────────────────

var (
	cgWarpMouseCursorPosition       func(NSPoint) int32
	cgAssociateMouseAndCursorPosition func(bool) int32
)

// initCursorCG loads CoreGraphics functions needed for cursor locking.
func initCursorCG() {
	lib, err := purego.Dlopen(
		"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
		purego.RTLD_GLOBAL|purego.RTLD_LAZY,
	)
	if err != nil {
		return // CoreGraphics not available — SetCursorPos will be a no-op
	}
	purego.RegisterLibFunc(&cgWarpMouseCursorPosition, lib, "CGWarpMouseCursorPosition")
	purego.RegisterLibFunc(&cgAssociateMouseAndCursorPosition, lib, "CGAssociateMouseAndMouseCursorPosition")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// windowFromView returns the *Window that owns the given NSView.
func windowFromView(view objc.ID) *Window {
	nswin := view.Send(selViewWindow)
	return windowFromHandle(nswin)
}

// cocoaModifiers converts NSEventModifierFlags to a GLFW ModifierKey bitmask.
func cocoaModifiers(flags uint64) ModifierKey {
	var mods ModifierKey
	if flags&nsModifierCapsLock != 0 {
		mods |= ModCapsLock
	}
	if flags&nsModifierShift != 0 {
		mods |= ModShift
	}
	if flags&nsModifierControl != 0 {
		mods |= ModControl
	}
	if flags&nsModifierOption != 0 {
		mods |= ModAlt
	}
	if flags&nsModifierCommand != 0 {
		mods |= ModSuper
	}
	return mods
}

// eventCursorPos converts an NSEvent's window-local location (Cocoa bottom-left
// origin) to GLFW window-local coordinates (top-left origin).
func eventCursorPos(ev, nswin objc.ID) (x, y float64) {
	loc := objc.Send[NSPoint](ev, selLocationInWindow)
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	return loc.X, content.Size.Height - loc.Y
}

// fireCursorPos extracts the cursor position from ev and fires fCursorPosHolder.
func fireCursorPos(w *Window, ev objc.ID) {
	if w.fCursorPosHolder == nil {
		return
	}
	x, y := eventCursorPos(ev, w.nsWin())
	w.fCursorPosHolder(w, x, y)
}

// ── Keyboard handlers ─────────────────────────────────────────────────────────

func nsViewAcceptsFirstResponder(self objc.ID, _cmd objc.SEL) bool { return true }

func nsViewKeyDown(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	keyCode := objc.Send[uint16](ev, selKeyCode)
	flags := objc.Send[uint64](ev, selModifierFlagsEv)
	mods := cocoaModifiers(flags)

	var key Key
	if int(keyCode) < len(darwinKeyTable) {
		key = darwinKeyTable[keyCode]
		w.darwinKeyState[keyCode] = Press
	}
	if w.fKeyHolder != nil {
		w.fKeyHolder(w, key, int(keyCode), Press, mods)
	}

	// Character callbacks — filter out private-use function-key codes (0xF700–0xF8FF).
	nsChars := ev.Send(selCharacters)
	if nsChars != 0 {
		for _, r := range goStringFromNS(nsChars) {
			if r >= 0xF700 && r <= 0xF8FF {
				continue
			}
			if r < 32 || r == 127 {
				continue
			}
			if w.fCharHolder != nil {
				w.fCharHolder(w, r)
			}
			if w.fCharModsHolder != nil {
				w.fCharModsHolder(w, r, mods)
			}
		}
	}
}

func nsViewKeyUp(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	keyCode := objc.Send[uint16](ev, selKeyCode)
	flags := objc.Send[uint64](ev, selModifierFlagsEv)
	mods := cocoaModifiers(flags)

	var key Key
	if int(keyCode) < len(darwinKeyTable) {
		key = darwinKeyTable[keyCode]
		w.darwinKeyState[keyCode] = Release
	}
	if w.fKeyHolder != nil {
		w.fKeyHolder(w, key, int(keyCode), Release, mods)
	}
}

func nsViewFlagsChanged(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	keyCode := objc.Send[uint16](ev, selKeyCode)
	flags := objc.Send[uint64](ev, selModifierFlagsEv)
	mods := cocoaModifiers(flags)

	var key Key
	if int(keyCode) < len(darwinKeyTable) {
		key = darwinKeyTable[keyCode]
	}

	// Map each modifier key to its flag bit to determine press vs. release.
	var bit uint64
	switch keyCode {
	case 0x38, 0x3C: // left/right Shift
		bit = nsModifierShift
	case 0x3B, 0x3E: // left/right Control
		bit = nsModifierControl
	case 0x3A, 0x3D: // left/right Option/Alt
		bit = nsModifierOption
	case 0x37, 0x36: // left/right Command/Super
		bit = nsModifierCommand
	case 0x39: // Caps Lock
		bit = nsModifierCapsLock
	default:
		return
	}

	action := Release
	if flags&bit != 0 {
		action = Press
	}
	if int(keyCode) < len(w.darwinKeyState) {
		w.darwinKeyState[keyCode] = action
	}
	if w.fKeyHolder != nil {
		w.fKeyHolder(w, key, int(keyCode), action, mods)
	}
}

// ── Mouse button handlers ─────────────────────────────────────────────────────

func nsViewMouseDown(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[MouseButtonLeft] = Press
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, MouseButtonLeft, Press, mods)
	}
}

func nsViewMouseUp(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[MouseButtonLeft] = Release
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, MouseButtonLeft, Release, mods)
	}
}

func nsViewRightMouseDown(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[MouseButtonRight] = Press
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, MouseButtonRight, Press, mods)
	}
}

func nsViewRightMouseUp(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[MouseButtonRight] = Release
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, MouseButtonRight, Release, mods)
	}
}

func nsViewOtherMouseDown(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	btn := MouseButton(objc.Send[int64](ev, selButtonNumber))
	if btn < 0 || int(btn) >= len(w.darwinBtnState) {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[btn] = Press
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, btn, Press, mods)
	}
}

func nsViewOtherMouseUp(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	btn := MouseButton(objc.Send[int64](ev, selButtonNumber))
	if btn < 0 || int(btn) >= len(w.darwinBtnState) {
		return
	}
	mods := cocoaModifiers(objc.Send[uint64](ev, selModifierFlagsEv))
	w.darwinBtnState[btn] = Release
	if w.fMouseButtonHolder != nil {
		w.fMouseButtonHolder(w, btn, Release, mods)
	}
}

// ── Cursor position handlers ──────────────────────────────────────────────────

func nsViewMouseMoved(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	if w := windowFromView(self); w != nil {
		fireCursorPos(w, ev)
	}
}

func nsViewMouseDragged(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	if w := windowFromView(self); w != nil {
		fireCursorPos(w, ev)
	}
}

func nsViewRightMouseDragged(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	if w := windowFromView(self); w != nil {
		fireCursorPos(w, ev)
	}
}

func nsViewOtherMouseDragged(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	if w := windowFromView(self); w != nil {
		fireCursorPos(w, ev)
	}
}

// ── Cursor enter / exit ───────────────────────────────────────────────────────

func nsViewMouseEntered(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	if w.fCursorEnterHolder != nil {
		w.fCursorEnterHolder(w, true)
	}
}

func nsViewMouseExited(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	if w.fCursorEnterHolder != nil {
		w.fCursorEnterHolder(w, false)
	}
}

// ── Scroll handler ────────────────────────────────────────────────────────────

func nsViewScrollWheel(self objc.ID, _cmd objc.SEL, ev objc.ID) {
	w := windowFromView(self)
	if w == nil {
		return
	}
	var dx, dy float64
	if bool(objc.Send[bool](ev, selHasPreciseScrollingDeltas)) {
		// Trackpad: use high-resolution deltas (in points)
		dx = objc.Send[float64](ev, selScrollingDeltaX)
		dy = objc.Send[float64](ev, selScrollingDeltaY)
	} else {
		// Traditional scroll wheel
		dx = objc.Send[float64](ev, selDeltaX)
		dy = objc.Send[float64](ev, selDeltaY)
	}
	if dx == 0 && dy == 0 {
		return
	}
	if w.fScrollHolder != nil {
		w.fScrollHolder(w, dx, dy)
	}
}

// ── Tracking area (for enter/exit events) ─────────────────────────────────────

func nsViewUpdateTrackingAreas(self objc.ID, _cmd objc.SEL) {
	// Let NSView do its own cleanup first.
	self.SendSuper(_cmd)

	// Remove existing tracking areas before adding a new one.
	areas := self.Send(selTrackingAreas)
	n := int(objc.Send[uint64](areas, selCount))
	existing := make([]objc.ID, n)
	for i := 0; i < n; i++ {
		existing[i] = areas.Send(selObjectAtIndex, uint64(i))
	}
	for _, area := range existing {
		self.Send(selRemoveTrackingArea, area)
	}

	// Add a single tracking area that follows the visible rect automatically.
	opts := nsTrackingMouseEnteredAndExited | nsTrackingActiveInKeyWindow | nsTrackingInVisibleRect
	ta := objc.ID(objc.GetClass("NSTrackingArea")).Send(selAlloc).Send(
		selInitTrackingArea,
		NSMakeRect(0, 0, 0, 0), // rect is ignored when InVisibleRect is set
		opts,
		self,        // owner receives mouseEntered:/mouseExited:
		objc.ID(0),  // userInfo nil
	)
	self.Send(selAddTrackingArea, ta)
	ta.Send(selRelease)
}

// ── GlfwView class registration ───────────────────────────────────────────────

// darwinViewClass is the GlfwView ObjC class; set by registerViewClass().
var darwinViewClass objc.Class

// registerViewClass creates the GlfwView Objective-C class.
// Must be called after Cocoa is loaded (inside darwinInitOnce.Do).
func registerViewClass() {
	class, err := objc.RegisterClass(
		"GlfwView",
		objc.GetClass("NSView"),
		nil, // no protocols needed for basic input
		nil, // no ivars
		[]objc.MethodDef{
			// Responder chain
			{Cmd: objc.RegisterName("acceptsFirstResponder"), Fn: nsViewAcceptsFirstResponder},

			// Keyboard
			{Cmd: objc.RegisterName("keyDown:"), Fn: nsViewKeyDown},
			{Cmd: objc.RegisterName("keyUp:"), Fn: nsViewKeyUp},
			{Cmd: objc.RegisterName("flagsChanged:"), Fn: nsViewFlagsChanged},

			// Mouse buttons
			{Cmd: objc.RegisterName("mouseDown:"), Fn: nsViewMouseDown},
			{Cmd: objc.RegisterName("mouseUp:"), Fn: nsViewMouseUp},
			{Cmd: objc.RegisterName("rightMouseDown:"), Fn: nsViewRightMouseDown},
			{Cmd: objc.RegisterName("rightMouseUp:"), Fn: nsViewRightMouseUp},
			{Cmd: objc.RegisterName("otherMouseDown:"), Fn: nsViewOtherMouseDown},
			{Cmd: objc.RegisterName("otherMouseUp:"), Fn: nsViewOtherMouseUp},

			// Cursor movement
			{Cmd: objc.RegisterName("mouseMoved:"), Fn: nsViewMouseMoved},
			{Cmd: objc.RegisterName("mouseDragged:"), Fn: nsViewMouseDragged},
			{Cmd: objc.RegisterName("rightMouseDragged:"), Fn: nsViewRightMouseDragged},
			{Cmd: objc.RegisterName("otherMouseDragged:"), Fn: nsViewOtherMouseDragged},

			// Enter / exit
			{Cmd: objc.RegisterName("mouseEntered:"), Fn: nsViewMouseEntered},
			{Cmd: objc.RegisterName("mouseExited:"), Fn: nsViewMouseExited},

			// Scroll
			{Cmd: objc.RegisterName("scrollWheel:"), Fn: nsViewScrollWheel},

			// Tracking areas (keeps enter/exit working after resize)
			{Cmd: objc.RegisterName("updateTrackingAreas"), Fn: nsViewUpdateTrackingAreas},

			// Cursor rects (NSCursor support — Phase E)
			{Cmd: objc.RegisterName("resetCursorRects"), Fn: nsViewResetCursorRects},
		},
	)
	if err != nil {
		panic("glfw: failed to register GlfwView: " + err.Error())
	}
	darwinViewClass = class
}
