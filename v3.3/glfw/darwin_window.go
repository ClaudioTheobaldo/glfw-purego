//go:build darwin

// darwin_window.go — macOS NSWindow creation and Window method implementations.

package glfw

import (
	"github.com/ebitengine/purego/objc"
)

// NSWindow style-mask bit constants.
const (
	nsWindowStyleMaskBorderless       uint64 = 0
	nsWindowStyleMaskTitled           uint64 = 1 << 0
	nsWindowStyleMaskClosable         uint64 = 1 << 1
	nsWindowStyleMaskMiniaturizable   uint64 = 1 << 2
	nsWindowStyleMaskResizable        uint64 = 1 << 3
	nsWindowStyleMaskFullSizeContentView uint64 = 1 << 15
)

// NSWindowCollectionBehavior — used for fullscreen support.
const (
	nsWindowCollectionBehaviorFullScreenPrimary uint64 = 1 << 7
)

// NSRequestUserAttentionType
const nsInformationalRequest = 10

// nsWin returns the receiver as an objc.ID (the NSWindow handle).
func (w *Window) nsWin() objc.ID { return objc.ID(w.handle) }

// primaryScreenHeight returns the height of the primary (main) screen in points.
// Used to flip Y coordinates between Cocoa (bottom-left origin) and GLFW
// (top-left origin).
func primaryScreenHeight() float64 {
	screen := objc.ID(objc.GetClass("NSScreen")).Send(selMainScreen)
	if screen == 0 {
		return 0
	}
	f := objc.Send[NSRect](screen, selFrame)
	return f.Size.Height
}

// cocoaToGLFWY converts a Cocoa Y coordinate (bottom-left origin, upward) for a
// window whose Cocoa frame is cocoaFrameRect to the GLFW Y coordinate
// (top-left origin, downward) of the window's content area.
func cocoaToGLFWY(nswin objc.ID, cocoaContentOriginY, contentH float64) int {
	screen := nswin.Send(selScreen)
	if screen == 0 {
		screen = objc.ID(objc.GetClass("NSScreen")).Send(selMainScreen)
	}
	if screen == 0 {
		return int(cocoaContentOriginY)
	}
	sf := objc.Send[NSRect](screen, selVisibleFrame)
	return int(sf.Origin.Y + sf.Size.Height - cocoaContentOriginY - contentH)
}

// glfwToCocoa converts a GLFW (top-left origin) Y coordinate to the Cocoa
// Y origin for `setFrameOrigin:` (position of the bottom-left corner of the
// window frame).
func glfwToCocoa(nswin objc.ID, glfwY int, contentH float64) float64 {
	screen := nswin.Send(selScreen)
	if screen == 0 {
		screen = objc.ID(objc.GetClass("NSScreen")).Send(selMainScreen)
	}
	if screen == 0 {
		return float64(glfwY)
	}
	sf := objc.Send[NSRect](screen, selVisibleFrame)
	return sf.Origin.Y + sf.Size.Height - float64(glfwY) - contentH
}

// ── CreateWindow ──────────────────────────────────────────────────────────────

// CreateWindow creates a window with an associated OpenGL context.
func CreateWindow(width, height int, title string, monitor, share *Monitor) (*Window, error) {
	if !darwinInitialized {
		return nil, &Error{Code: NotInitialized, Desc: "GLFW has not been initialised"}
	}

	hints.mu.Lock()
	h := make(map[Hint]int, len(hints.m))
	for k, v := range hints.m {
		h[k] = v
	}
	hints.mu.Unlock()

	// Build NSWindowStyleMask from window hints.
	var styleMask uint64
	if h[Decorated] != 0 {
		styleMask = nsWindowStyleMaskTitled |
			nsWindowStyleMaskClosable |
			nsWindowStyleMaskMiniaturizable
		if h[Resizable] != 0 {
			styleMask |= nsWindowStyleMaskResizable
		}
	} else {
		styleMask = nsWindowStyleMaskBorderless
	}

	// Create and configure NSWindow.
	rect := NSMakeRect(0, 0, float64(width), float64(height))
	nsWin := objc.ID(objc.GetClass("NSWindow")).Send(selAlloc).Send(
		selInitWithContentRect,
		rect,
		styleMask,
		uint64(2), // NSBackingStoreBuffered
		false,     // defer: NO
	)
	if nsWin == 0 {
		return nil, &Error{Code: PlatformError, Desc: "NSWindow allocation failed"}
	}

	// Do not let NSWindow release itself when closed; we manage the lifetime.
	nsWin.Send(selSetReleasedWhenClosed, false)

	// Accept mouse-moved events (needed for cursor-pos callbacks in Phase B).
	nsWin.Send(selSetAcceptsMouseMovedEvents, true)

	// Set the window title.
	nsWin.Send(selSetTitle, nsStringFromGoString(title))

	// Create a GlfwView (NSView subclass) as the content view.
	// It handles all keyboard/mouse/scroll events (Phase B).
	contentView := objc.ID(darwinViewClass).Send(selAlloc).Send(
		selInitWithFrame, rect)
	nsWin.Send(selSetContentView, contentView)

	// Make the view layer-backed and attach a CAMetalLayer.
	// Required for Vulkan surface creation; also compatible with NSOpenGLContext.
	// CAMetalLayer lives in QuartzCore, pulled in by AppKit via Cocoa.framework.
	var metalLayerID objc.ID
	contentView.Send(selSetWantsLayer, true)
	if caml := objc.ID(objc.GetClass("CAMetalLayer")).Send(selNew); caml != 0 {
		contentView.Send(selSetLayer, caml)
		metalLayerID = caml
	}

	contentView.Send(selRelease)

	// Attach the window delegate.
	delegate := objc.ID(darwinDelegateClass).Send(selAlloc).Send(selInit)
	nsWin.Send(selSetDelegate, delegate)
	delegate.Send(selRelease)

	// Build the *Window.
	w := &Window{
		handle:     uintptr(nsWin),
		title:      title,
		metalLayer: uintptr(metalLayerID),
	}

	// Register in the global map so delegate callbacks can find it.
	windowByHandle.Store(uintptr(nsWin), w)

	// Create an NSOpenGLContext if OpenGL was requested.
	// On headless CI runners context creation may return 0 — that is non-fatal;
	// the window is still usable for non-rendering tests.
	if ClientAPI(h[ClientAPIs]) == OpenGLAPI {
		if ctx := createNSGLContext(h, contentView); ctx != 0 {
			w.nsglContext = uintptr(ctx)
		}
	}

	// Apply fullscreen mode if a monitor was supplied.
	if monitor != nil {
		w.fsMonitor = monitor
		// TODO Phase D: real fullscreen via CGDisplayCapture.
	}

	// Show the window (if Visible hint is set).
	if h[Visible] != 0 {
		nsWin.Send(selMakeKeyAndOrderFront, objc.ID(0))
		if h[Focused] != 0 {
			nsApp.Send(selActivate, true)
		}
	}

	return w, nil
}

// ── Geometry ──────────────────────────────────────────────────────────────────

// GetPos returns the window's position in screen coordinates (top-left of content area).
func (w *Window) GetPos() (x, y int) {
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	x = int(content.Origin.X)
	y = cocoaToGLFWY(nswin, content.Origin.Y, content.Size.Height)
	return
}

// SetPos moves the window to the given screen coordinates.
func (w *Window) SetPos(xPos, yPos int) {
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	cocoaY := glfwToCocoa(nswin, yPos, content.Size.Height)
	// setFrameOrigin: sets the bottom-left of the window frame.
	// Adjust from content origin to frame origin.
	frameOffsetX := content.Origin.X - frame.Origin.X
	frameOffsetY := content.Origin.Y - frame.Origin.Y
	nswin.Send(selSetFrameOrigin, NSPoint{
		X: float64(xPos) - frameOffsetX,
		Y: cocoaY - frameOffsetY,
	})
}

// GetSize returns the size of the window's client area in screen coordinates.
func (w *Window) GetSize() (width, height int) {
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	return int(content.Size.Width), int(content.Size.Height)
}

// SetSize resizes the window's client area.
func (w *Window) SetSize(width, height int) {
	w.nsWin().Send(selSetContentSize, NSSize{float64(width), float64(height)})
}

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) {
	w.title = title
	w.nsWin().Send(selSetTitle, nsStringFromGoString(title))
}

// GetFramebufferSize returns the size of the window's framebuffer in pixels.
func (w *Window) GetFramebufferSize() (width, height int) {
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	scale := objc.Send[float64](nswin, selBackingScaleFactor)
	return int(content.Size.Width * scale), int(content.Size.Height * scale)
}

// GetFrameSize returns the size of the window decorations surrounding the client area.
func (w *Window) GetFrameSize() (left, top, right, bottom int) {
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	left = int(content.Origin.X - frame.Origin.X)
	bottom = int(content.Origin.Y - frame.Origin.Y)
	right = int((frame.Origin.X + frame.Size.Width) - (content.Origin.X + content.Size.Width))
	top = int((frame.Origin.Y + frame.Size.Height) - (content.Origin.Y + content.Size.Height))
	return
}

// GetWindowFrameSize is a package-level wrapper around (*Window).GetFrameSize.
func GetWindowFrameSize(w *Window) (left, top, right, bottom int) { return w.GetFrameSize() }

// GetContentScale returns the DPI scale factor relative to 96 DPI.
func (w *Window) GetContentScale() (x, y float32) {
	scale := float32(objc.Send[float64](w.nsWin(), selBackingScaleFactor))
	return scale, scale
}

// ── Monitor / fullscreen ──────────────────────────────────────────────────────

// GetMonitor returns the monitor the window is fullscreened on, or nil.
func (w *Window) GetMonitor() *Monitor { return w.fsMonitor }

// SetMonitor switches the window between fullscreen and windowed mode using
// the modern Cocoa toggleFullScreen: API.  Native macOS fullscreen is a Spaces
// transition rather than a CGDisplayCapture; this matches what upstream GLFW
// 3.3 does on macOS.
//
// Note: per-monitor selection (xpos / ypos / refreshRate) is honoured only as
// far as Cocoa allows — toggleFullScreen: targets the screen the window is
// currently on.  To force a specific monitor, move the window first.
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) {
	nswin := w.nsWin()
	wasFullscreen := w.fsMonitor != nil
	want := monitor != nil

	// Ensure the window can enter native fullscreen.
	if want && !wasFullscreen {
		nswin.Send(selSetCollectionBehavior, nsWindowCollectionBehaviorFullScreenPrimary)
	}

	// Request the transition only if the current state differs from the target.
	if want != wasFullscreen {
		nswin.Send(selToggleFullScreen, objc.ID(0))
	}

	if !want {
		// Restore the requested windowed-mode size & position.
		if width > 0 && height > 0 {
			nswin.Send(selSetContentSize, NSSize{Width: float64(width), Height: float64(height)})
		}
		if xpos != 0 || ypos != 0 {
			nswin.Send(selSetFrameOrigin, NSPoint{
				X: float64(xpos),
				Y: glfwToCocoa(nswin, ypos, float64(height)),
			})
		}
	}
	w.fsMonitor = monitor
}

// ── Window state ──────────────────────────────────────────────────────────────

// Destroy closes the window and releases associated resources.
func (w *Window) Destroy() {
	nswin := w.nsWin()
	windowByHandle.Delete(uintptr(nswin))
	nswin.Send(selClose)
	nswin.Send(selRelease)
	w.handle = 0
}

// Iconify minimises the window.
func (w *Window) Iconify() { w.nsWin().Send(selMiniaturize, objc.ID(0)) }

// Restore restores a minimised or maximised window to its normal state.
func (w *Window) Restore() {
	nswin := w.nsWin()
	if bool(objc.Send[bool](nswin, selIsMiniaturized)) {
		nswin.Send(selDeminiaturize, objc.ID(0))
	} else if bool(objc.Send[bool](nswin, selIsZoomed)) {
		nswin.Send(selZoom, objc.ID(0))
	}
}

// Maximize maximises the window.
func (w *Window) Maximize() {
	if !bool(objc.Send[bool](w.nsWin(), selIsZoomed)) {
		w.nsWin().Send(selZoom, objc.ID(0))
	}
}

// Focus brings the window to the front and gives it keyboard focus.
func (w *Window) Focus() {
	nsApp.Send(selActivate, true)
	w.nsWin().Send(selMakeKeyAndOrderFront, objc.ID(0))
}

// Hide hides the window without destroying it.
func (w *Window) Hide() { w.nsWin().Send(selOrderOut, objc.ID(0)) }

// Show makes the window visible.  Inverse of Hide.
//
// Unlike Focus, Show only orders the window front; it does not steal
// application activation.  Callers wanting both should call Focus.
func (w *Window) Show() { w.nsWin().Send(selMakeKeyAndOrderFront, objc.ID(0)) }

// SetIcon sets the window icon from a slice of candidate images.
// macOS does not support per-window icons; no-op.
func (w *Window) SetIcon(_ []Image) {}

// ── Window attributes ─────────────────────────────────────────────────────────

// SetAttrib sets a window attribute at runtime.
func (w *Window) SetAttrib(hint Hint, value int) {
	switch hint {
	case Decorated:
		current := objc.Send[uint64](w.nsWin(), selStyleMask)
		if value != 0 {
			current |= nsWindowStyleMaskTitled | nsWindowStyleMaskClosable |
				nsWindowStyleMaskMiniaturizable
		} else {
			current &^= nsWindowStyleMaskTitled | nsWindowStyleMaskClosable |
				nsWindowStyleMaskMiniaturizable
		}
		w.nsWin().Send(selSetStyleMask, current)
	case Floating:
		// NSNormalWindowLevel = 0, NSFloatingWindowLevel = 3.
		// CGWindowLevelKey: kCGNormalWindowLevelKey / kCGFloatingWindowLevelKey.
		level := int64(0)
		if value != 0 {
			level = 3
		}
		w.nsWin().Send(selSetLevel, level)
	case Resizable:
		current := objc.Send[uint64](w.nsWin(), selStyleMask)
		if value != 0 {
			current |= nsWindowStyleMaskResizable
		} else {
			current &^= nsWindowStyleMaskResizable
		}
		w.nsWin().Send(selSetStyleMask, current)
	}
}

// GetAttrib returns the current value of the specified window attribute.
func (w *Window) GetAttrib(hint Hint) int {
	nswin := w.nsWin()
	switch hint {
	case Focused:
		if bool(objc.Send[bool](nswin, selIsKeyWindow)) {
			return 1
		}
	case Iconified:
		if bool(objc.Send[bool](nswin, selIsMiniaturized)) {
			return 1
		}
	case Maximized:
		if bool(objc.Send[bool](nswin, selIsZoomed)) {
			return 1
		}
	case Visible:
		if bool(objc.Send[bool](nswin, selIsVisible)) {
			return 1
		}
	case Floating:
		// Floating: window level > NSNormalWindowLevel (0).
		level := objc.Send[int64](nswin, objc.RegisterName("level"))
		if level > 0 {
			return 1
		}
	case Decorated:
		mask := objc.Send[uint64](nswin, selStyleMask)
		if mask&nsWindowStyleMaskTitled != 0 {
			return 1
		}
	case Resizable:
		mask := objc.Send[uint64](nswin, selStyleMask)
		if mask&nsWindowStyleMaskResizable != 0 {
			return 1
		}
	}
	return 0
}

// ── Size limits / aspect ratio ────────────────────────────────────────────────

// SetSizeLimits sets minimum and maximum content-area dimensions.
// Pass DontCare (-1) to clear a limit.
func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {
	w.minW, w.minH = minWidth, minHeight
	w.maxW, w.maxH = maxWidth, maxHeight
	nswin := w.nsWin()
	// CGFloat is float64 on amd64+arm64; use a large sentinel for DontCare.
	const cgFloatMax = 1.0e9
	minSz := NSSize{Width: cgFloatMax, Height: cgFloatMax}
	if minWidth >= 0 {
		minSz.Width = float64(minWidth)
	} else {
		minSz.Width = 0
	}
	if minHeight >= 0 {
		minSz.Height = float64(minHeight)
	} else {
		minSz.Height = 0
	}
	maxSz := NSSize{Width: cgFloatMax, Height: cgFloatMax}
	if maxWidth >= 0 {
		maxSz.Width = float64(maxWidth)
	}
	if maxHeight >= 0 {
		maxSz.Height = float64(maxHeight)
	}
	nswin.Send(selSetContentMinSize, minSz)
	nswin.Send(selSetContentMaxSize, maxSz)
}

// SetAspectRatio locks the window's content-area resize aspect ratio.
// Pass (0, 0) to disable the constraint.
func (w *Window) SetAspectRatio(numer, denom int) {
	w.aspectNum, w.aspectDen = numer, denom
	nswin := w.nsWin()
	if numer <= 0 || denom <= 0 {
		// Pass NSZeroSize to clear the constraint.
		nswin.Send(selSetContentAspectRatio, NSSize{Width: 0, Height: 0})
		return
	}
	nswin.Send(selSetContentAspectRatio, NSSize{Width: float64(numer), Height: float64(denom)})
}

// ── Opacity ───────────────────────────────────────────────────────────────────

// GetOpacity returns the window's opacity in the range [0, 1].
func (w *Window) GetOpacity() float32 {
	return float32(objc.Send[float64](w.nsWin(), selAlphaValue))
}

// SetOpacity sets the window's opacity.
func (w *Window) SetOpacity(opacity float32) {
	w.nsWin().Send(selSetOpaque, opacity == 1.0)
	w.nsWin().Send(selSetAlphaValue, float64(opacity))
}

// RequestAttention requests user attention (bouncing Dock icon).
func (w *Window) RequestAttention() {
	nsApp.Send(objc.RegisterName("requestUserAttention:"), uint64(10) /* NSInformationalRequest */)
}

// ── OpenGL context ────────────────────────────────────────────────────────────

// MakeContextCurrent makes this window's OpenGL context current on this thread.
func (w *Window) MakeContextCurrent() {
	if w.nsglContext != 0 {
		objc.ID(w.nsglContext).Send(selNSGLMakeCurrentContext)
	}
	darwinCurrentWindow = w
}

// SwapBuffers swaps the front and back buffers of the window.
func (w *Window) SwapBuffers() {
	if w.nsglContext != 0 {
		objc.ID(w.nsglContext).Send(selNSGLFlushBuffer)
	}
}

// ── Input mode ────────────────────────────────────────────────────────────────

// GetInputMode returns the current value of the specified input mode.
func (w *Window) GetInputMode(mode InputMode) int {
	switch mode {
	case CursorMode:
		return w.cursorMode
	case RawMouseMotion:
		if w.rawMouseMotion {
			return 1
		}
	}
	return 0
}

// SetInputMode sets the value of the specified input mode.
func (w *Window) SetInputMode(mode InputMode, value int) {
	switch mode {
	case CursorMode:
		prev := w.cursorMode
		w.cursorMode = value
		nsCursorClass := objc.ID(objc.GetClass("NSCursor"))
		switch value {
		case CursorNormal:
			// Un-hide cursor if it was hidden or disabled.
			if prev == CursorHidden || prev == CursorDisabled {
				nsCursorClass.Send(selUnhide)
			}
			// Re-enable cursor position association if disabled.
			if prev == CursorDisabled && cgAssociateMouseAndCursorPosition != nil {
				cgAssociateMouseAndCursorPosition(true)
			}
		case CursorHidden:
			// Hide if was normal; if was disabled, just re-enable position tracking.
			if prev == CursorNormal {
				nsCursorClass.Send(selHide)
			} else if prev == CursorDisabled && cgAssociateMouseAndCursorPosition != nil {
				cgAssociateMouseAndCursorPosition(true)
			}
		case CursorDisabled:
			// Hide the cursor and decouple mouse position from cursor position.
			if prev == CursorNormal {
				nsCursorClass.Send(selHide)
			}
			if cgAssociateMouseAndCursorPosition != nil {
				cgAssociateMouseAndCursorPosition(false)
			}
		}
	case RawMouseMotion:
		w.rawMouseMotion = value != 0
	}
}

// ── Key / mouse state ─────────────────────────────────────────────────────────

// GetKey returns the last reported state of a keyboard key.
// Performs a linear scan of darwinKeyTable to map GLFW Key → VKC,
// then returns the entry from darwinKeyState.
func (w *Window) GetKey(key Key) Action {
	for vkc, k := range darwinKeyTable {
		if k == key {
			return w.darwinKeyState[vkc]
		}
	}
	return Release
}

// GetMouseButton returns the last reported state of a mouse button.
func (w *Window) GetMouseButton(button MouseButton) Action {
	if button < 0 || int(button) >= len(w.darwinBtnState) {
		return Release
	}
	return w.darwinBtnState[button]
}

// GetCursorPos returns the cursor position relative to the window's top-left
// content-area origin, using mouseLocationOutsideOfEventStream.
func (w *Window) GetCursorPos() (x, y float64) {
	nswin := w.nsWin()
	loc := objc.Send[NSPoint](nswin, selMouseLocationOutside)
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	return loc.X, content.Size.Height - loc.Y
}

// SetCursorPos warps the cursor to the given window-local position.
// Converts GLFW (top-left) window coords to CG global screen coords.
func (w *Window) SetCursorPos(xPos, yPos float64) {
	if cgWarpMouseCursorPosition == nil {
		return
	}
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	// Cocoa global Y of the target point (Cocoa = bottom-left origin, Y up).
	cocoaGlobalY := content.Origin.Y + (content.Size.Height - yPos)
	// CG global coordinates have Y=0 at top of primary display, Y increases down.
	primH := primaryScreenHeight()
	cgWarpMouseCursorPosition(NSPoint{
		X: content.Origin.X + xPos,
		Y: primH - cocoaGlobalY,
	})
}

// ── Platform-native handles ───────────────────────────────────────────────────

// GetWGLContext is Windows-only; returns 0 on macOS.
func (w *Window) GetWGLContext() uintptr { return 0 }

// GetWin32Window is Windows-only; returns 0 on macOS.
func (w *Window) GetWin32Window() uintptr { return 0 }
