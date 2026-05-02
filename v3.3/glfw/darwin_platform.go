//go:build darwin

// darwin_platform.go — macOS Cocoa platform layer.
//
// Implements the GLFW platform API for macOS using pure Go via
// github.com/ebitengine/purego/objc. No Cgo required.
//
// Thread model: all Cocoa calls must occur on the OS main thread.
// Callers must call runtime.LockOSThread() before Init.

package glfw

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ── Cocoa geometry types ──────────────────────────────────────────────────────

// NSPoint is a 2D point in Cocoa screen/window coordinates.
type NSPoint struct{ X, Y float64 }

// NSSize is a 2D size in Cocoa screen/window coordinates.
type NSSize struct{ Width, Height float64 }

// NSRect is a rectangle in Cocoa screen/window coordinates.
type NSRect struct {
	Origin NSPoint
	Size   NSSize
}

// NSMakeRect returns an NSRect with the given origin and size.
func NSMakeRect(x, y, w, h float64) NSRect {
	return NSRect{Origin: NSPoint{x, y}, Size: NSSize{w, h}}
}

// ── SEL cache (initialized by package init, after objc.init()) ───────────────

var (
	// NSObject
	selAlloc    = objc.RegisterName("alloc")
	selInit     = objc.RegisterName("init")
	selNew      = objc.RegisterName("new")
	selRelease  = objc.RegisterName("release")
	selDrain    = objc.RegisterName("drain")
	selClass    = objc.RegisterName("class")
	selResponds = objc.RegisterName("respondsToSelector:")

	// NSApplication
	selSharedApplication    = objc.RegisterName("sharedApplication")
	selSetActivationPolicy  = objc.RegisterName("setActivationPolicy:")
	selFinishLaunching      = objc.RegisterName("finishLaunching")
	selNextEventMatchingMask = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent             = objc.RegisterName("sendEvent:")
	selUpdateWindows         = objc.RegisterName("updateWindows")
	selPostEventAtStart      = objc.RegisterName("postEvent:atStart:")
	selActivate              = objc.RegisterName("activateIgnoringOtherApps:")
	selOtherEventWithType    = objc.RegisterName("otherEventWithType:location:modifierFlags:timestamp:windowNumber:context:subtype:data1:data2:")

	// NSDate
	selDistantPast   = objc.RegisterName("distantPast")
	selDistantFuture = objc.RegisterName("distantFuture")
	selDateWithTimeIntervalSinceNow = objc.RegisterName("dateWithTimeIntervalSinceNow:")

	// NSString
	selStringWithUTF8String = objc.RegisterName("stringWithUTF8String:")
	selUTF8String           = objc.RegisterName("UTF8String")
	selLength               = objc.RegisterName("length")

	// NSArray
	selCount          = objc.RegisterName("count")
	selObjectAtIndex  = objc.RegisterName("objectAtIndex:")
	selArrayWithObject = objc.RegisterName("arrayWithObject:")

	// NSNotification
	selObject = objc.RegisterName("object")

	// NSWindow
	selInitWithContentRect      = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	selFrame                    = objc.RegisterName("frame")
	selSetFrameOrigin           = objc.RegisterName("setFrameOrigin:")
	selContentRectForFrameRect  = objc.RegisterName("contentRectForFrameRect:")
	selSetContentSize           = objc.RegisterName("setContentSize:")
	selSetTitle                 = objc.RegisterName("setTitle:")
	selMakeKeyAndOrderFront     = objc.RegisterName("makeKeyAndOrderFront:")
	selOrderOut                 = objc.RegisterName("orderOut:")
	selClose                    = objc.RegisterName("close")
	selCenter                   = objc.RegisterName("center")
	selSetDelegate              = objc.RegisterName("setDelegate:")
	selSetContentView           = objc.RegisterName("setContentView:")
	selContentView              = objc.RegisterName("contentView")
	selSetReleasedWhenClosed    = objc.RegisterName("setReleasedWhenClosed:")
	selSetAcceptsMouseMovedEvents = objc.RegisterName("setAcceptsMouseMovedEvents:")
	selBackingScaleFactor       = objc.RegisterName("backingScaleFactor")
	selIsKeyWindow              = objc.RegisterName("isKeyWindow")
	selIsMiniaturized           = objc.RegisterName("isMiniaturized")
	selIsZoomed                 = objc.RegisterName("isZoomed")
	selMiniaturize              = objc.RegisterName("miniaturize:")
	selDeminiaturize            = objc.RegisterName("deminiaturize:")
	selZoom                     = objc.RegisterName("zoom:")
	selScreen                   = objc.RegisterName("screen")
	selSetOpaque               = objc.RegisterName("setOpaque:")
	selSetBackgroundColor       = objc.RegisterName("setBackgroundColor:")
	selAlphaValue               = objc.RegisterName("alphaValue")
	selSetAlphaValue            = objc.RegisterName("setAlphaValue:")
	selRequestUserAttention     = objc.RegisterName("requestUserAttention:")
	selStyleMask                = objc.RegisterName("styleMask")
	selSetStyleMask             = objc.RegisterName("setStyleMask:")
	selSetHasShadow             = objc.RegisterName("setHasShadow:")
	selInvalidateCursorRects    = objc.RegisterName("invalidateCursorRectsForView:")

	// NSView
	selInitWithFrame    = objc.RegisterName("initWithFrame:")
	selSetWantsLayer   = objc.RegisterName("setWantsLayer:")
	selSetLayer        = objc.RegisterName("setLayer:")

	// NSScreen
	selMainScreen    = objc.RegisterName("mainScreen")
	selScreens       = objc.RegisterName("screens")
	selVisibleFrame  = objc.RegisterName("visibleFrame")

	// NSColor
	selBlackColor = objc.RegisterName("blackColor")

	// NSCursor
	selArrowCursor    = objc.RegisterName("arrowCursor")
	selHide           = objc.RegisterName("hide")
	selUnhide         = objc.RegisterName("unhide")
	selPop            = objc.RegisterName("pop")
	selPush           = objc.RegisterName("push")
	selSet            = objc.RegisterName("set")
)

// ── Global state ──────────────────────────────────────────────────────────────

var (
	darwinInitTime     time.Time
	darwinInitialized  bool
	darwinInitOnce     sync.Once
	nsApp              objc.ID  // [NSApplication sharedApplication]
	nsPool             objc.ID  // top-level NSAutoreleasePool
	nsDefaultRunLoopMode objc.ID // NSString "NSDefaultRunLoopMode"
	nsDistantPast      objc.ID  // [NSDate distantPast]
	nsDistantFuture    objc.ID  // [NSDate distantFuture]
	darwinDelegateClass objc.Class

	darwinMonitorCb      func(*Monitor, PeripheralEvent)
	darwinCachedMonitors []*Monitor

	darwinJoystickCb func(Joystick, PeripheralEvent)
	darwinCurrentWindow *Window
)

// ── Monitor ───────────────────────────────────────────────────────────────────

// Monitor represents a connected display.
type Monitor struct {
	cgDisplayID       uint32 // CGDirectDisplayID (macOS)
	name              string
	x, y              int
	widthPx, heightPx int
	widthMM, heightMM int
	modes             []*VidMode
	currentMode       *VidMode
}

func (m *Monitor) GetName() string { return m.name }
func (m *Monitor) GetPos() (x, y int) { return m.x, m.y }
func (m *Monitor) GetWorkarea() (x, y, width, height int) {
	return m.x, m.y, m.widthPx, m.heightPx
}
func (m *Monitor) GetPhysicalSize() (widthMM, heightMM int) { return m.widthMM, m.heightMM }
func (m *Monitor) GetContentScale() (x, y float32)          { return 1, 1 }
func (m *Monitor) GetVideoMode() *VidMode                    { return m.currentMode }
func (m *Monitor) GetVideoModes() []*VidMode                 { return m.modes }
func (m *Monitor) GetGammaRamp() *GammaRamp                  { return nil }
func (m *Monitor) SetGamma(_ float32)                        {}
func (m *Monitor) SetGammaRamp(_ *GammaRamp)                 {}
func (m *Monitor) GetWin32Adapter() string                   { return "" }
func (m *Monitor) GetWin32Monitor() string                   { return "" }

// diffAndFireMonitorCallbacks fires Connected/Disconnected for monitors that
// appeared or disappeared between two snapshots.
func diffAndFireMonitorCallbacks(old, cur []*Monitor, cb func(*Monitor, PeripheralEvent)) {
	oldSet := make(map[string]*Monitor, len(old))
	for _, m := range old {
		oldSet[m.name] = m
	}
	curSet := make(map[string]*Monitor, len(cur))
	for _, m := range cur {
		curSet[m.name] = m
	}
	for _, m := range cur {
		if _, existed := oldSet[m.name]; !existed {
			cb(m, Connected)
		}
	}
	for _, m := range old {
		if _, exists := curSet[m.name]; !exists {
			cb(m, Disconnected)
		}
	}
}

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
// The first non-nil registration also arms the CGDisplay reconfiguration hook.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	darwinMonitorCb = cb
	if cb != nil && darwinCGReconfigCBPtr == 0 {
		registerMonitorReconfigCB()
	}
}

// ── NSString helpers ──────────────────────────────────────────────────────────

// nsStringFromGoString creates an NSString from a Go string.
// The returned ID is autoreleased.
func nsStringFromGoString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8String, s)
}

// goStringFromNS converts an NSString to a Go string.
func goStringFromNS(ns objc.ID) string {
	if ns == 0 {
		return ""
	}
	ptr := objc.Send[uintptr](ns, selUTF8String)
	if ptr == 0 {
		return ""
	}
	// Read null-terminated C string from ptr.
	b := (*[1 << 20]byte)(nativePtrFromUintptr(ptr))
	n := 0
	for b[n] != 0 {
		n++
	}
	return string(b[:n])
}

// ── NSWindowDelegate class ────────────────────────────────────────────────────

// windowFromNotification looks up the *Window from an NSNotification whose
// object is the NSWindow handle.
func windowFromNotification(notification objc.ID) *Window {
	nswin := notification.Send(selObject)
	v, ok := windowByHandle.Load(uintptr(nswin))
	if !ok {
		return nil
	}
	return v.(*Window)
}

// windowFromHandle looks up the *Window from an NSWindow handle (ID).
func windowFromHandle(nswin objc.ID) *Window {
	v, ok := windowByHandle.Load(uintptr(nswin))
	if !ok {
		return nil
	}
	return v.(*Window)
}

// nsWindowShouldClose is called when the user clicks the close button.
// We set shouldClose and fire the callback, but prevent actual closing so
// GLFW-style programs can handle it in their own event loop.
func nsWindowShouldClose(self objc.ID, _cmd objc.SEL, sender objc.ID) bool {
	w := windowFromHandle(sender)
	if w == nil {
		return true
	}
	w.shouldClose = true
	if w.fCloseHolder != nil {
		w.fCloseHolder(w)
	}
	return false
}

func nsWindowDidResize(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	width := int(content.Size.Width)
	height := int(content.Size.Height)
	// Notify NSOpenGLContext of resize so it stays in sync with the view.
	if w.nsglContext != 0 {
		objc.ID(w.nsglContext).Send(selNSGLUpdate)
	}
	if w.fSizeHolder != nil {
		w.fSizeHolder(w, width, height)
	}
	scale := objc.Send[float64](nswin, selBackingScaleFactor)
	fbW := int(float64(width) * scale)
	fbH := int(float64(height) * scale)
	if w.fFramebufferSizeHolder != nil {
		w.fFramebufferSizeHolder(w, fbW, fbH)
	}
}

func nsWindowDidMove(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	x, y := w.GetPos()
	if w.fPosHolder != nil {
		w.fPosHolder(w, x, y)
	}
}

func nsWindowDidMiniaturize(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	if w.fIconifyHolder != nil {
		w.fIconifyHolder(w, true)
	}
}

func nsWindowDidDeminiaturize(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	if w.fIconifyHolder != nil {
		w.fIconifyHolder(w, false)
	}
}

func nsWindowDidBecomeKey(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	if w.fFocusHolder != nil {
		w.fFocusHolder(w, true)
	}
}

func nsWindowDidResignKey(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	if w.fFocusHolder != nil {
		w.fFocusHolder(w, false)
	}
}

func nsWindowDidChangeBackingProperties(self objc.ID, _cmd objc.SEL, notification objc.ID) {
	w := windowFromNotification(notification)
	if w == nil {
		return
	}
	nswin := w.nsWin()
	frame := objc.Send[NSRect](nswin, selFrame)
	content := objc.Send[NSRect](nswin, selContentRectForFrameRect, frame)
	scale := objc.Send[float64](nswin, selBackingScaleFactor)
	scaleF := float32(scale)
	width := int(content.Size.Width)
	height := int(content.Size.Height)
	fbW := int(float64(width) * scale)
	fbH := int(float64(height) * scale)
	if w.fFramebufferSizeHolder != nil {
		w.fFramebufferSizeHolder(w, fbW, fbH)
	}
	if w.fContentScaleHolder != nil {
		w.fContentScaleHolder(w, scaleF, scaleF)
	}
}

// registerDelegateClass creates the GlfwWindowDelegate ObjC class.
// Must be called after Cocoa framework is loaded.
func registerDelegateClass() {
	var protocols []*objc.Protocol
	if p := objc.GetProtocol("NSWindowDelegate"); p != nil {
		protocols = []*objc.Protocol{p}
	}
	class, err := objc.RegisterClass(
		"GlfwWindowDelegate",
		objc.GetClass("NSObject"),
		protocols,
		nil, // no ivars
		[]objc.MethodDef{
			{Cmd: objc.RegisterName("windowShouldClose:"), Fn: nsWindowShouldClose},
			{Cmd: objc.RegisterName("windowDidResize:"), Fn: nsWindowDidResize},
			{Cmd: objc.RegisterName("windowDidMove:"), Fn: nsWindowDidMove},
			{Cmd: objc.RegisterName("windowDidMiniaturize:"), Fn: nsWindowDidMiniaturize},
			{Cmd: objc.RegisterName("windowDidDeminiaturize:"), Fn: nsWindowDidDeminiaturize},
			{Cmd: objc.RegisterName("windowDidBecomeKey:"), Fn: nsWindowDidBecomeKey},
			{Cmd: objc.RegisterName("windowDidResignKey:"), Fn: nsWindowDidResignKey},
			{Cmd: objc.RegisterName("windowDidChangeBackingProperties:"), Fn: nsWindowDidChangeBackingProperties},
		},
	)
	if err != nil {
		panic("glfw: failed to register GlfwWindowDelegate: " + err.Error())
	}
	darwinDelegateClass = class
}

// ── Init / Terminate ──────────────────────────────────────────────────────────

// Init initialises the GLFW subsystem on macOS.
// Must be called from the main OS thread.
func Init() error {
	runtime.LockOSThread()
	darwinInitTime = time.Now()
	resetHints()

	// Load Cocoa umbrella framework (includes AppKit + Foundation).
	if _, err := purego.Dlopen(
		"/System/Library/Frameworks/Cocoa.framework/Cocoa",
		purego.RTLD_GLOBAL|purego.RTLD_LAZY,
	); err != nil {
		return fmt.Errorf("glfw: failed to load Cocoa framework: %w", err)
	}

	// Top-level autorelease pool (drained in Terminate).
	nsPool = objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selAlloc).Send(selInit)

	// NSApplication singleton.
	nsApp = objc.ID(objc.GetClass("NSApplication")).Send(selSharedApplication)

	// Show the app in the Dock and menu bar.
	nsApp.Send(selSetActivationPolicy, 0 /* NSApplicationActivationPolicyRegular */)

	// Prepare the app for event dispatch without starting a run loop.
	nsApp.Send(selFinishLaunching)

	// Disable mouse-event coalescing so we get every mouse-moved event.
	objc.ID(objc.GetClass("NSEvent")).Send(
		objc.RegisterName("setMouseCoalescingEnabled:"), false)

	// Cache frequently-used NSDate sentinels.
	nsDistantPast = objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
	nsDistantFuture = objc.ID(objc.GetClass("NSDate")).Send(selDistantFuture)

	// Cache the default run-loop mode string.
	nsDefaultRunLoopMode = nsStringFromGoString("NSDefaultRunLoopMode")

	// Register all Objective-C classes and load native libraries (once per process).
	darwinInitOnce.Do(func() {
		registerDelegateClass()
		registerViewClass()
		initCursorCG()
		initMonitorCG()
		initGameController()
	})

	// Snapshot the connected monitors for hotplug diffing.
	darwinCachedMonitors, _ = GetMonitors()

	darwinInitialized = true
	return nil
}

// Terminate cleans up all GLFW resources.
func Terminate() {
	darwinInitialized = false
	deregisterMonitorReconfigCB()
	if nsPool != 0 {
		nsPool.Send(selDrain)
		nsPool = 0
	}
}

// ── Event loop ────────────────────────────────────────────────────────────────

// PollEvents processes all pending events without blocking.
func PollEvents() {
	if !darwinInitialized {
		return
	}
	pollJoysticks()
	for {
		// nextEventMatchingMask:untilDate:inMode:dequeue: with distantPast
		// returns immediately if no event is pending.
		ev := nsApp.Send(
			selNextEventMatchingMask,
			uint64(0xFFFFFFFFFFFFFFFF), // NSEventMaskAny
			nsDistantPast,
			nsDefaultRunLoopMode,
			true, // dequeue
		)
		if ev == 0 {
			break
		}
		nsApp.Send(selSendEvent, ev)
	}
	nsApp.Send(selUpdateWindows)
}

// WaitEvents blocks until at least one event is available, then drains the queue.
func WaitEvents() {
	if !darwinInitialized {
		return
	}
	// Block until one event arrives (distantFuture = effectively forever).
	ev := nsApp.Send(
		selNextEventMatchingMask,
		uint64(0xFFFFFFFFFFFFFFFF),
		nsDistantFuture,
		nsDefaultRunLoopMode,
		true,
	)
	if ev != 0 {
		nsApp.Send(selSendEvent, ev)
	}
	// Drain any remaining events.
	PollEvents()
}

// WaitEventsTimeout waits up to timeout seconds for an event, then processes pending events.
func WaitEventsTimeout(timeout float64) {
	if !darwinInitialized {
		return
	}
	deadline := objc.ID(objc.GetClass("NSDate")).Send(
		selDateWithTimeIntervalSinceNow, timeout)
	ev := nsApp.Send(
		selNextEventMatchingMask,
		uint64(0xFFFFFFFFFFFFFFFF),
		deadline,
		nsDefaultRunLoopMode,
		true,
	)
	if ev != 0 {
		nsApp.Send(selSendEvent, ev)
	}
	PollEvents()
}

// PostEmptyEvent wakes up a blocked WaitEvents call from another goroutine.
func PostEmptyEvent() {
	if !darwinInitialized {
		return
	}
	// Post a synthetic NSApplicationDefined event (type = 15) to unblock WaitEvents.
	event := objc.ID(objc.GetClass("NSEvent")).Send(
		selOtherEventWithType,
		uint64(15),    // NSEventTypeApplicationDefined
		NSPoint{0, 0}, // location
		uint64(0),     // modifierFlags
		float64(0),    // timestamp
		uint64(0),     // windowNumber
		objc.ID(0),    // context (nil)
		int16(0),      // subtype
		int64(0),      // data1
		int64(0),      // data2
	)
	nsApp.Send(selPostEventAtStart, event, true)
}

// ── Timer ─────────────────────────────────────────────────────────────────────

// GetTime returns the elapsed time in seconds since Init was called.
func GetTime() float64 { return time.Since(darwinInitTime).Seconds() }

// SetTime resets the timer base so that GetTime returns t immediately after.
func SetTime(t float64) {
	darwinInitTime = time.Now().Add(-time.Duration(t * float64(time.Second)))
}

// GetTimerFrequency returns the raw timer frequency (nanoseconds on macOS).
func GetTimerFrequency() uint64 { return 1_000_000_000 }

// GetTimerValue returns the raw timer counter.
func GetTimerValue() uint64 { return uint64(GetTime() * 1e9) }

// ── Version / hints ───────────────────────────────────────────────────────────

// GetVersion returns the compile-time GLFW version.
func GetVersion() (major, minor, revision int) { return 3, 3, 0 }

// GetVersionString returns a human-readable version string.
func GetVersionString() string { return "3.3.0 purego/darwin" }

// InitHint sets a hint for the next Init call. Stub — no-op.
func InitHint(_ Hint, _ int) {}

// WindowHintString sets a string-valued window hint. Stub — no-op.
func WindowHintString(_ Hint, _ string) {}

// GetKeyScancode returns the macOS virtual key code for the given GLFW key,
// or -1 if there is no mapping.
func GetKeyScancode(key Key) int {
	for vkc, k := range darwinKeyTable {
		if k == key {
			return vkc
		}
	}
	return -1
}

// GetKeyName returns the localized name of a key. Darwin: returns empty string.
func GetKeyName(_ Key, _ int) string { return "" }

// SetJoystickCallback registers a callback for joystick connect/disconnect events.
func SetJoystickCallback(cb func(joy Joystick, event PeripheralEvent)) {
	darwinJoystickCb = cb
}

// ── Input features ────────────────────────────────────────────────────────────

// RawMouseMotionSupported returns true if raw (unaccelerated) mouse motion is
// available. macOS supports it via IOHIDManager; stub returns false until wired.
func RawMouseMotionSupported() bool { return false }

// ── Context ───────────────────────────────────────────────────────────────────

// GetCurrentContext returns the window whose context is current on this thread.
func GetCurrentContext() *Window { return darwinCurrentWindow }

// DetachCurrentContext removes any current OpenGL context from this thread.
func DetachCurrentContext() {
	if darwinCurrentWindow != nil && darwinCurrentWindow.nsglContext != 0 {
		// clearCurrentContext is a class method on NSOpenGLContext.
		objc.ID(objc.GetClass("NSOpenGLContext")).Send(selNSGLClearCurrentContext)
	}
	darwinCurrentWindow = nil
}

// SwapInterval sets the minimum number of video frame periods per buffer swap.
func SwapInterval(interval int) {
	if darwinCurrentWindow == nil || darwinCurrentWindow.nsglContext == 0 {
		return
	}
	darwinSwapInterval(objc.ID(darwinCurrentWindow.nsglContext), interval)
}

// GetProcAddress returns the address of the named OpenGL symbol via dlsym.
func GetProcAddress(name string) unsafe.Pointer { return darwinGetProcAddress(name) }

// ExtensionSupported returns true if the named OpenGL extension is available.
// Checks via dlsym: works for function-name queries; extension string queries
// require an active context (not yet implemented).
func ExtensionSupported(name string) bool { return darwinGetProcAddress(name) != nil }

// ── EGL helpers (called from shared context paths) ────────────────────────────

func eglDestroyWindow(_ *Window)               {}
func eglMakeCurrentWindow(_ *Window)           {}
func eglSwapBuffersWindow(_ *Window)           {}
func eglSwapIntervalNow(_ int)                 {}
func eglGetProcAddr(_ string) unsafe.Pointer  { return nil }

