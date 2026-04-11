package glfw

import "sync"

// hints holds the current window / context creation hints set via WindowHint.
// Reset to defaults by Init() and by DefaultWindowHints().
var hints struct {
	mu sync.Mutex
	m  map[Hint]int
}

func init() {
	resetHints()
}

func resetHints() {
	hints.m = map[Hint]int{
		Resizable:              1,
		Visible:                1,
		Decorated:              1,
		Focused:                1,
		AutoIconify:            1,
		Floating:               0,
		Maximized:              0,
		CenterCursor:           1,
		TransparentFramebuffer: 0,
		FocusOnShow:            1,
		RedBits:                8,
		GreenBits:              8,
		BlueBits:               8,
		AlphaBits:              8,
		DepthBits:              24,
		StencilBits:            8,
		Samples:                0,
		SRGBCapable:            0,
		DoubleBuffer:           1,
		RefreshRate:            -1, // GLFW_DONT_CARE
		ClientAPI:              int(OpenGLAPI),
		ContextVersionMajor:    1,
		ContextVersionMinor:    0,
		OpenGLForwardCompatible: 0,
		OpenGLDebugContext:      0,
		OpenGLProfileHint:       int(AnyProfile),
	}
}

// WindowHint sets a window/context creation hint for the next CreateWindow call.
func WindowHint(target Hint, hint int) {
	hints.mu.Lock()
	hints.m[target] = hint
	hints.mu.Unlock()
}

// DefaultWindowHints resets all window hints to their default values.
func DefaultWindowHints() {
	hints.mu.Lock()
	resetHints()
	hints.mu.Unlock()
}

// Window represents an OS window with an associated OpenGL context.
//
// All methods must be called from the main OS thread (the goroutine that
// called glfw.Init). This matches the contract of the original GLFW library.
type Window struct {
	// handle is the platform-specific window handle.
	// Type and meaning varies by platform implementation.
	handle uintptr

	// shouldClose is set to true when the user requests the window to close.
	shouldClose bool

	// --- Callback holders (identical fields to go-gl/glfw) ---
	fPosHolder             func(w *Window, xpos, ypos int)
	fSizeHolder            func(w *Window, width, height int)
	fFramebufferSizeHolder func(w *Window, width, height int)
	fCloseHolder           func(w *Window)
	fMaximizeHolder        func(w *Window, maximized bool)
	fRefreshHolder         func(w *Window)
	fFocusHolder           func(w *Window, focused bool)
	fIconifyHolder         func(w *Window, iconified bool)
	fContentScaleHolder    func(w *Window, x, y float32)
	fMouseButtonHolder     func(w *Window, button MouseButton, action Action, mod ModifierKey)
	fCursorPosHolder       func(w *Window, xpos, ypos float64)
	fCursorEnterHolder     func(w *Window, entered bool)
	fScrollHolder          func(w *Window, xoff, yoff float64)
	fKeyHolder             func(w *Window, key Key, scancode int, action Action, mods ModifierKey)
	fCharHolder            func(w *Window, char rune)
	fCharModsHolder        func(w *Window, char rune, mods ModifierKey)
	fDropHolder            func(w *Window, names []string)
}

// ShouldClose returns true if the window has been requested to close.
func (w *Window) ShouldClose() bool { return w.shouldClose }

// SetShouldClose sets the close flag on the window.
func (w *Window) SetShouldClose(value bool) { w.shouldClose = value }

// --- Callback setters (identical signatures to go-gl/glfw) ---

func (w *Window) SetPosCallback(cb func(w *Window, xpos, ypos int)) {
	w.fPosHolder = cb
}
func (w *Window) SetSizeCallback(cb func(w *Window, width, height int)) {
	w.fSizeHolder = cb
}
func (w *Window) SetFramebufferSizeCallback(cb func(w *Window, width, height int)) {
	w.fFramebufferSizeHolder = cb
}
func (w *Window) SetCloseCallback(cb func(w *Window)) {
	w.fCloseHolder = cb
}
func (w *Window) SetMaximizeCallback(cb func(w *Window, maximized bool)) {
	w.fMaximizeHolder = cb
}
func (w *Window) SetRefreshCallback(cb func(w *Window)) {
	w.fRefreshHolder = cb
}
func (w *Window) SetFocusCallback(cb func(w *Window, focused bool)) {
	w.fFocusHolder = cb
}
func (w *Window) SetIconifyCallback(cb func(w *Window, iconified bool)) {
	w.fIconifyHolder = cb
}
func (w *Window) SetContentScaleCallback(cb func(w *Window, x, y float32)) {
	w.fContentScaleHolder = cb
}
func (w *Window) SetMouseButtonCallback(cb func(w *Window, button MouseButton, action Action, mod ModifierKey)) {
	w.fMouseButtonHolder = cb
}
func (w *Window) SetCursorPosCallback(cb func(w *Window, xpos, ypos float64)) {
	w.fCursorPosHolder = cb
}
func (w *Window) SetCursorEnterCallback(cb func(w *Window, entered bool)) {
	w.fCursorEnterHolder = cb
}
func (w *Window) SetScrollCallback(cb func(w *Window, xoff, yoff float64)) {
	w.fScrollHolder = cb
}
func (w *Window) SetKeyCallback(cb func(w *Window, key Key, scancode int, action Action, mods ModifierKey)) {
	w.fKeyHolder = cb
}
func (w *Window) SetCharCallback(cb func(w *Window, char rune)) {
	w.fCharHolder = cb
}
func (w *Window) SetCharModsCallback(cb func(w *Window, char rune, mods ModifierKey)) {
	w.fCharModsHolder = cb
}
func (w *Window) SetDropCallback(cb func(w *Window, names []string)) {
	w.fDropHolder = cb
}

// windowByHandle is the global registry mapping platform handles → *Window.
// Keyed by uintptr (HWND on Windows, X Window ID on Linux, NSWindow ID on macOS).
var windowByHandle sync.Map
