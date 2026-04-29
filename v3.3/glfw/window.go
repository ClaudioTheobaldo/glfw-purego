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
		ClientAPIs:             int(OpenGLAPI),
		ContextVersionMajor:    1,
		ContextVersionMinor:    0,
		OpenGLForwardCompatible: 0,
		OpenGLDebugContext:      0,
		OpenGLProfileHint:       int(AnyProfile),
		ContextCreationAPIHint: int(NativeContextAPI),
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
	// HWND on Windows, X Window ID on Linux, NSWindow ID on macOS.
	handle uintptr

	// Platform context handles — zero on platforms where not applicable.
	dc    uintptr // HDC   (Windows/WGL)
	hglrc uintptr // HGLRC (Windows/WGL)
	hmon  uintptr // HMONITOR (Windows)

	// EGL context handles — populated when useEGL is true, zero otherwise.
	eglSurface uintptr // EGLSurface (Windows/EGL via ANGLE)
	eglContext uintptr // EGLContext (Windows/EGL via ANGLE)
	useEGL     bool    // true when the context was created via EGL

	// shouldClose is set to true when the user requests the window to close.
	shouldClose bool

	// cursorMode tracks the current cursor visibility/capture state.
	// One of CursorNormal, CursorHidden, CursorDisabled.
	cursorMode int

	// Cursor set on this window; 0 = use system default (IDC_ARROW)
	cursor uintptr

	// Fullscreen tracking
	fsMonitor            *Monitor // non-nil when window is in fullscreen mode
	savedX, savedY       int      // saved windowed position
	savedW, savedH       int      // saved windowed client size
	savedStyle           uintptr  // saved WS_* style
	savedExStyle         uintptr  // saved WS_EX_* style

	// Size constraints (0 = unconstrained, -1 = GLFW_DONT_CARE)
	minW, minH int
	maxW, maxH int

	// Aspect ratio constraint (0/0 = none)
	aspectNum, aspectDen int

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

// --- Callback setters — each returns the previous callback (go-gl/glfw API parity) ---

func (w *Window) SetPosCallback(cb func(w *Window, xpos, ypos int)) func(w *Window, xpos, ypos int) {
	prev := w.fPosHolder; w.fPosHolder = cb; return prev
}
func (w *Window) SetSizeCallback(cb func(w *Window, width, height int)) func(w *Window, width, height int) {
	prev := w.fSizeHolder; w.fSizeHolder = cb; return prev
}
func (w *Window) SetFramebufferSizeCallback(cb func(w *Window, width, height int)) func(w *Window, width, height int) {
	prev := w.fFramebufferSizeHolder; w.fFramebufferSizeHolder = cb; return prev
}
func (w *Window) SetCloseCallback(cb func(w *Window)) func(w *Window) {
	prev := w.fCloseHolder; w.fCloseHolder = cb; return prev
}
func (w *Window) SetMaximizeCallback(cb func(w *Window, maximized bool)) func(w *Window, maximized bool) {
	prev := w.fMaximizeHolder; w.fMaximizeHolder = cb; return prev
}
func (w *Window) SetRefreshCallback(cb func(w *Window)) func(w *Window) {
	prev := w.fRefreshHolder; w.fRefreshHolder = cb; return prev
}
func (w *Window) SetFocusCallback(cb func(w *Window, focused bool)) func(w *Window, focused bool) {
	prev := w.fFocusHolder; w.fFocusHolder = cb; return prev
}
func (w *Window) SetIconifyCallback(cb func(w *Window, iconified bool)) func(w *Window, iconified bool) {
	prev := w.fIconifyHolder; w.fIconifyHolder = cb; return prev
}
func (w *Window) SetContentScaleCallback(cb func(w *Window, x, y float32)) func(w *Window, x, y float32) {
	prev := w.fContentScaleHolder; w.fContentScaleHolder = cb; return prev
}
func (w *Window) SetMouseButtonCallback(cb func(w *Window, button MouseButton, action Action, mod ModifierKey)) func(w *Window, button MouseButton, action Action, mod ModifierKey) {
	prev := w.fMouseButtonHolder; w.fMouseButtonHolder = cb; return prev
}
func (w *Window) SetCursorPosCallback(cb func(w *Window, xpos, ypos float64)) func(w *Window, xpos, ypos float64) {
	prev := w.fCursorPosHolder; w.fCursorPosHolder = cb; return prev
}
func (w *Window) SetCursorEnterCallback(cb func(w *Window, entered bool)) func(w *Window, entered bool) {
	prev := w.fCursorEnterHolder; w.fCursorEnterHolder = cb; return prev
}
func (w *Window) SetScrollCallback(cb func(w *Window, xoff, yoff float64)) func(w *Window, xoff, yoff float64) {
	prev := w.fScrollHolder; w.fScrollHolder = cb; return prev
}
func (w *Window) SetKeyCallback(cb func(w *Window, key Key, scancode int, action Action, mods ModifierKey)) func(w *Window, key Key, scancode int, action Action, mods ModifierKey) {
	prev := w.fKeyHolder; w.fKeyHolder = cb; return prev
}
func (w *Window) SetCharCallback(cb func(w *Window, char rune)) func(w *Window, char rune) {
	prev := w.fCharHolder; w.fCharHolder = cb; return prev
}
func (w *Window) SetCharModsCallback(cb func(w *Window, char rune, mods ModifierKey)) func(w *Window, char rune, mods ModifierKey) {
	prev := w.fCharModsHolder; w.fCharModsHolder = cb; return prev
}
func (w *Window) SetDropCallback(cb func(w *Window, names []string)) func(w *Window, names []string) {
	prev := w.fDropHolder; w.fDropHolder = cb; return prev
}

// windowByHandle is the global registry mapping platform handles → *Window.
// Keyed by uintptr (HWND on Windows, X Window ID on Linux, NSWindow ID on macOS).
var windowByHandle sync.Map
