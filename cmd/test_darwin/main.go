//go:build darwin

// test_darwin is the comprehensive macOS smoke test.  It exercises every
// public API in v3.3/glfw that's applicable on darwin: library lifecycle,
// monitors, windows, callbacks (all 17), context, input, cursors, clipboard,
// joystick stubs, Vulkan probes, and the native Cocoa handles.
//
// Run: go run ./cmd/test_darwin   (from repo root, on macOS or in CI)
package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── tiny test harness ─────────────────────────────────────────────────────────

var (
	passed int
	failed int
)

func check(label string, ok bool, extra string) {
	if ok {
		passed++
		fmt.Printf("  PASS  %s\n", label)
	} else {
		failed++
		if extra != "" {
			fmt.Printf("  FAIL  %s  (%s)\n", label, extra)
		} else {
			fmt.Printf("  FAIL  %s\n", label)
		}
	}
}

func section(name string) {
	fmt.Printf("── %s ───────────────────────────────────────────\n", name)
}

// ── library + version + timer ─────────────────────────────────────────────────

func testVersionAndHints() {
	section("Version / hints")
	major, minor, _ := glfw.GetVersion()
	check("GetVersion returns 3.3.x", major == 3 && minor == 3,
		fmt.Sprintf("%d.%d", major, minor))
	check("GetVersionString non-empty", glfw.GetVersionString() != "",
		glfw.GetVersionString())

	// Stub APIs — verify no-panic.
	glfw.InitHint(glfw.Focused, 1)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHintString(glfw.Focused, "value")
	glfw.DefaultWindowHints()
	check("InitHint / WindowHint / WindowHintString / DefaultWindowHints: no panic", true, "")

	// Key utilities (darwin returns -1 / "" by design).
	check("GetKeyScancode returns int", true,
		fmt.Sprintf("KeyA=%d", glfw.GetKeyScancode(glfw.KeyA)))
	check("GetKeyName returns string", true,
		fmt.Sprintf("name=%q", glfw.GetKeyName(glfw.KeyA, 0)))
}

func testTimer() {
	section("Timer")
	check("GetTime >= 0", glfw.GetTime() >= 0, fmt.Sprintf("%v", glfw.GetTime()))
	check("GetTimerFrequency > 0", glfw.GetTimerFrequency() > 0,
		fmt.Sprintf("%d", glfw.GetTimerFrequency()))
	check("GetTimerValue > 0", glfw.GetTimerValue() > 0,
		fmt.Sprintf("%d", glfw.GetTimerValue()))

	// SetTime round-trip.
	glfw.SetTime(10.0)
	check("SetTime then GetTime >= 10.0", glfw.GetTime() >= 10.0,
		fmt.Sprintf("%.3f", glfw.GetTime()))
	glfw.SetTime(0)
}

func testFeatureQueries() {
	section("Feature queries")
	supp := glfw.RawMouseMotionSupported()
	check("RawMouseMotionSupported() returns true on macOS", supp, fmt.Sprintf("%v", supp))
}

func testEvents() {
	section("Events")
	glfw.PollEvents()
	check("PollEvents: no panic", true, "")
	glfw.PostEmptyEvent()
	check("PostEmptyEvent: no panic", true, "")
	glfw.WaitEventsTimeout(0.001)
	check("WaitEventsTimeout(1ms): no panic", true, "")
}

// ── monitors ──────────────────────────────────────────────────────────────────

func testMonitors() {
	section("Monitors")
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("%v", err))
	check("GetMonitors: at least one", len(monitors) > 0, fmt.Sprintf("n=%d", len(monitors)))

	pm := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor: non-nil", pm != nil, "")
	if pm == nil {
		return
	}

	check("Monitor.GetName non-empty", pm.GetName() != "", pm.GetName())

	x, y := pm.GetPos()
	check("Monitor.GetPos: no panic", true, fmt.Sprintf("(%d,%d)", x, y))

	wx, wy, ww, wh := pm.GetWorkarea()
	check("Monitor.GetWorkarea: width>0 && height>0", ww > 0 && wh > 0,
		fmt.Sprintf("(%d,%d,%dx%d)", wx, wy, ww, wh))

	wmm, hmm := pm.GetPhysicalSize()
	check("Monitor.GetPhysicalSize: no panic", true,
		fmt.Sprintf("%dx%dmm", wmm, hmm))

	sx, sy := pm.GetContentScale()
	check("Monitor.GetContentScale >= 1.0", sx >= 1.0 && sy >= 1.0,
		fmt.Sprintf("(%.2f, %.2f)", sx, sy))

	vm := pm.GetVideoMode()
	check("Monitor.GetVideoMode non-nil", vm != nil, "")
	if vm != nil {
		check("VideoMode.Width>0 && Height>0", vm.Width > 0 && vm.Height > 0,
			fmt.Sprintf("%dx%d", vm.Width, vm.Height))
	}

	modes := pm.GetVideoModes()
	check("Monitor.GetVideoModes: non-empty", len(modes) > 0,
		fmt.Sprintf("n=%d", len(modes)))

	// Gamma APIs are deprecated on modern macOS; just verify no panic.
	pm.SetGamma(1.0)
	pm.GetGammaRamp()
	pm.SetGammaRamp(&glfw.GammaRamp{})
	check("Monitor gamma APIs: no panic", true, "")

	// Monitor user pointer round-trip.
	dummy := unsafe.Pointer(&modes)
	pm.SetUserPointer(dummy)
	check("Monitor.SetUserPointer / GetUserPointer round-trip",
		pm.GetUserPointer() == dummy, "")
	pm.SetUserPointer(nil)

	// Native Cocoa monitor handle.
	check("Monitor.GetCocoaMonitor non-zero", pm.GetCocoaMonitor() != 0,
		fmt.Sprintf("0x%x", pm.GetCocoaMonitor()))

	// SetMonitorCallback set/clear (current API returns no value).
	glfw.SetMonitorCallback(func(_ *glfw.Monitor, _ glfw.PeripheralEvent) {})
	glfw.SetMonitorCallback(nil)
	check("SetMonitorCallback set/clear: no panic", true, "")
}

// ── joystick (no device on CI) ────────────────────────────────────────────────

func testJoystickStubs() {
	section("Joystick (no device on CI)")
	for j := glfw.Joystick1; j <= glfw.Joystick16; j++ {
		if glfw.JoystickPresent(j) {
			fmt.Printf("  INFO  Joystick%d present: %s\n", int(j)+1, glfw.GetJoystickName(j))
		}
	}
	check("JoystickPresent(0) = false", !glfw.JoystickPresent(glfw.Joystick1), "")
	check("GetJoystickAxes nil",     glfw.GetJoystickAxes(glfw.Joystick1) == nil, "")
	check("GetJoystickButtons nil",  glfw.GetJoystickButtons(glfw.Joystick1) == nil, "")
	check("GetJoystickHats nil",     glfw.GetJoystickHats(glfw.Joystick1) == nil, "")
	check("GetJoystickName empty",   glfw.GetJoystickName(glfw.Joystick1) == "", "")
	check("GetJoystickGUID empty",   glfw.GetJoystickGUID(glfw.Joystick1) == "", "")
	check("JoystickIsGamepad false", !glfw.JoystickIsGamepad(glfw.Joystick1), "")
	check("GetGamepadName empty",    glfw.GetGamepadName(glfw.Joystick1) == "", "")

	// Both GetGamepadState forms.
	var gs glfw.GamepadState
	check("GetGamepadState (pkg form) returns false", !glfw.GetGamepadState(glfw.Joystick1, &gs), "")
	check("Joystick.GetGamepadState() returns nil",
		glfw.Joystick1.GetGamepadState() == nil, "")

	// UpdateGamepadMappings: no-op on Cocoa, must not panic.
	check("UpdateGamepadMappings: no panic",
		true, fmt.Sprintf("ok=%v", glfw.UpdateGamepadMappings("")))

	// Joystick user pointer (both forms).
	dummy := unsafe.Pointer(&gs)
	glfw.SetJoystickUserPointer(glfw.Joystick1, dummy)
	check("SetJoystickUserPointer / GetJoystickUserPointer (pkg) round-trip",
		glfw.GetJoystickUserPointer(glfw.Joystick1) == dummy, "")
	glfw.Joystick1.SetUserPointer(nil)
	check("Joystick.SetUserPointer / GetUserPointer (method) round-trip",
		glfw.Joystick1.GetUserPointer() == nil, "")

	// SetJoystickCallback set/clear.
	jcb := func(_ glfw.Joystick, _ glfw.PeripheralEvent) {}
	glfw.SetJoystickCallback(jcb)
	glfw.SetJoystickCallback(nil)
	check("SetJoystickCallback: no panic", true, "")
}

// ── Vulkan ────────────────────────────────────────────────────────────────────

func testVulkan() {
	section("Vulkan")
	vs := glfw.VulkanSupported()
	check("VulkanSupported: no panic", true, fmt.Sprintf("supported=%v", vs))

	exts := glfw.GetRequiredInstanceExtensions()
	if vs {
		check("GetRequiredInstanceExtensions: 2 extensions", len(exts) == 2,
			fmt.Sprintf("%v", exts))
	} else {
		check("GetRequiredInstanceExtensions: nil when unsupported",
			exts == nil, fmt.Sprintf("%v", exts))
	}

	addr := glfw.GetVulkanGetInstanceProcAddress()
	if vs {
		check("GetVulkanGetInstanceProcAddress non-nil when supported",
			addr != nil, fmt.Sprintf("%v", addr))
	} else {
		check("GetVulkanGetInstanceProcAddress nil when unsupported",
			addr == nil, fmt.Sprintf("%v", addr))
	}
}

// ── window: callback round-trip helpers ───────────────────────────────────────
//
// For each of the 17 window-scoped callback setters we use the same pattern:
// register cb1 and check the previous value is nil, register cb2 and check
// previous == cb1 (using runtime.FuncForPC), then clear with nil.
//
// We intentionally compare via reflection on function identity rather than
// direct equality (Go's == is invalid for func values).

func funcID(f any) uintptr {
	if f == nil {
		return 0
	}
	v := *(*[2]uintptr)(unsafe.Pointer(&f))
	return v[1]
}

func testWindowCallbacks(w *glfw.Window) {
	section("Window callbacks (17)")

	// Each entry: (label, register-cb1, register-cb2, register-nil-and-return-prev)
	type cbCase struct {
		name     string
		set1     func() any
		set2     func() any
		setNil   func() any
	}
	cases := []cbCase{
		{
			name: "SetPosCallback",
			set1: func() any { return w.SetPosCallback(func(_ *glfw.Window, _, _ int) {}) },
			set2: func() any { return w.SetPosCallback(func(_ *glfw.Window, _, _ int) {}) },
			setNil: func() any { return w.SetPosCallback(nil) },
		},
		{
			name: "SetSizeCallback",
			set1: func() any { return w.SetSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			set2: func() any { return w.SetSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			setNil: func() any { return w.SetSizeCallback(nil) },
		},
		{
			name: "SetFramebufferSizeCallback",
			set1: func() any { return w.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			set2: func() any { return w.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			setNil: func() any { return w.SetFramebufferSizeCallback(nil) },
		},
		{
			name: "SetCloseCallback",
			set1: func() any { return w.SetCloseCallback(func(_ *glfw.Window) {}) },
			set2: func() any { return w.SetCloseCallback(func(_ *glfw.Window) {}) },
			setNil: func() any { return w.SetCloseCallback(nil) },
		},
		{
			name: "SetMaximizeCallback",
			set1: func() any { return w.SetMaximizeCallback(func(_ *glfw.Window, _ bool) {}) },
			set2: func() any { return w.SetMaximizeCallback(func(_ *glfw.Window, _ bool) {}) },
			setNil: func() any { return w.SetMaximizeCallback(nil) },
		},
		{
			name: "SetRefreshCallback",
			set1: func() any { return w.SetRefreshCallback(func(_ *glfw.Window) {}) },
			set2: func() any { return w.SetRefreshCallback(func(_ *glfw.Window) {}) },
			setNil: func() any { return w.SetRefreshCallback(nil) },
		},
		{
			name: "SetFocusCallback",
			set1: func() any { return w.SetFocusCallback(func(_ *glfw.Window, _ bool) {}) },
			set2: func() any { return w.SetFocusCallback(func(_ *glfw.Window, _ bool) {}) },
			setNil: func() any { return w.SetFocusCallback(nil) },
		},
		{
			name: "SetIconifyCallback",
			set1: func() any { return w.SetIconifyCallback(func(_ *glfw.Window, _ bool) {}) },
			set2: func() any { return w.SetIconifyCallback(func(_ *glfw.Window, _ bool) {}) },
			setNil: func() any { return w.SetIconifyCallback(nil) },
		},
		{
			name: "SetContentScaleCallback",
			set1: func() any { return w.SetContentScaleCallback(func(_ *glfw.Window, _, _ float32) {}) },
			set2: func() any { return w.SetContentScaleCallback(func(_ *glfw.Window, _, _ float32) {}) },
			setNil: func() any { return w.SetContentScaleCallback(nil) },
		},
		{
			name: "SetMouseButtonCallback",
			set1: func() any { return w.SetMouseButtonCallback(func(_ *glfw.Window, _ glfw.MouseButton, _ glfw.Action, _ glfw.ModifierKey) {}) },
			set2: func() any { return w.SetMouseButtonCallback(func(_ *glfw.Window, _ glfw.MouseButton, _ glfw.Action, _ glfw.ModifierKey) {}) },
			setNil: func() any { return w.SetMouseButtonCallback(nil) },
		},
		{
			name: "SetCursorPosCallback",
			set1: func() any { return w.SetCursorPosCallback(func(_ *glfw.Window, _, _ float64) {}) },
			set2: func() any { return w.SetCursorPosCallback(func(_ *glfw.Window, _, _ float64) {}) },
			setNil: func() any { return w.SetCursorPosCallback(nil) },
		},
		{
			name: "SetCursorEnterCallback",
			set1: func() any { return w.SetCursorEnterCallback(func(_ *glfw.Window, _ bool) {}) },
			set2: func() any { return w.SetCursorEnterCallback(func(_ *glfw.Window, _ bool) {}) },
			setNil: func() any { return w.SetCursorEnterCallback(nil) },
		},
		{
			name: "SetScrollCallback",
			set1: func() any { return w.SetScrollCallback(func(_ *glfw.Window, _, _ float64) {}) },
			set2: func() any { return w.SetScrollCallback(func(_ *glfw.Window, _, _ float64) {}) },
			setNil: func() any { return w.SetScrollCallback(nil) },
		},
		{
			name: "SetKeyCallback",
			set1: func() any { return w.SetKeyCallback(func(_ *glfw.Window, _ glfw.Key, _ int, _ glfw.Action, _ glfw.ModifierKey) {}) },
			set2: func() any { return w.SetKeyCallback(func(_ *glfw.Window, _ glfw.Key, _ int, _ glfw.Action, _ glfw.ModifierKey) {}) },
			setNil: func() any { return w.SetKeyCallback(nil) },
		},
		{
			name: "SetCharCallback",
			set1: func() any { return w.SetCharCallback(func(_ *glfw.Window, _ rune) {}) },
			set2: func() any { return w.SetCharCallback(func(_ *glfw.Window, _ rune) {}) },
			setNil: func() any { return w.SetCharCallback(nil) },
		},
		{
			name: "SetCharModsCallback",
			set1: func() any { return w.SetCharModsCallback(func(_ *glfw.Window, _ rune, _ glfw.ModifierKey) {}) },
			set2: func() any { return w.SetCharModsCallback(func(_ *glfw.Window, _ rune, _ glfw.ModifierKey) {}) },
			setNil: func() any { return w.SetCharModsCallback(nil) },
		},
		{
			name: "SetDropCallback",
			set1: func() any { return w.SetDropCallback(func(_ *glfw.Window, _ []string) {}) },
			set2: func() any { return w.SetDropCallback(func(_ *glfw.Window, _ []string) {}) },
			setNil: func() any { return w.SetDropCallback(nil) },
		},
	}

	for _, c := range cases {
		// Note: comparing `any == nil` is false when the dynamic type is
		// a non-nil function-type holding a nil value, so use funcID.
		prev1 := c.set1()
		check(c.name+": first-set returns nil func", funcID(prev1) == 0,
			fmt.Sprintf("id=0x%x", funcID(prev1)))
		prev2 := c.set2()
		check(c.name+": second-set returns previous (non-nil)",
			funcID(prev2) != 0, "")
		prevNil := c.setNil()
		check(c.name+": nil-set returns previous (non-nil)",
			funcID(prevNil) != 0, "")
	}
}

// ── window: comprehensive method coverage ─────────────────────────────────────

func testWindow() {
	section("Window")
	glfw.WindowHint(glfw.Visible, 0) // invisible — CI has no display
	glfw.WindowHint(glfw.Resizable, 1)
	glfw.WindowHint(glfw.Decorated, 1)

	w, err := glfw.CreateWindow(320, 240, "smoke-test", nil, nil)
	check("CreateWindow: no error", err == nil, fmt.Sprintf("%v", err))
	if w == nil {
		return
	}

	// ── geometry ────────────────────────────────────────────────────────────
	width, height := w.GetSize()
	check("GetSize matches request", width == 320 && height == 240,
		fmt.Sprintf("%dx%d", width, height))

	w.SetSize(400, 300)
	w2, h2 := w.GetSize()
	check("SetSize then GetSize", w2 == 400 && h2 == 300, fmt.Sprintf("%dx%d", w2, h2))

	x, y := w.GetPos()
	check("GetPos: no panic", true, fmt.Sprintf("(%d,%d)", x, y))
	w.SetPos(100, 100)
	check("SetPos: no panic", true, "")

	fbW, fbH := w.GetFramebufferSize()
	check("GetFramebufferSize positive", fbW > 0 && fbH > 0,
		fmt.Sprintf("%dx%d", fbW, fbH))

	cs1, cs2 := w.GetContentScale()
	check("Window.GetContentScale >= 1.0", cs1 >= 1.0 && cs2 >= 1.0,
		fmt.Sprintf("(%.2f,%.2f)", cs1, cs2))

	l, t, r, b := w.GetFrameSize()
	check("GetFrameSize: no panic", true, fmt.Sprintf("(%d,%d,%d,%d)", l, t, r, b))
	l, t, r, b = glfw.GetWindowFrameSize(w)
	check("GetWindowFrameSize (pkg): no panic", true, fmt.Sprintf("(%d,%d,%d,%d)", l, t, r, b))

	// ── title / icon ────────────────────────────────────────────────────────
	w.SetTitle("updated title")
	check("SetTitle: no panic", true, "")
	check("InternalTitle reflects last SetTitle",
		w.InternalTitle() == "updated title", w.InternalTitle())

	w.SetIcon(nil)
	check("SetIcon(nil): no panic", true, "")
	glfw.SetIconFromImages(w, nil)
	check("SetIconFromImages(nil): no panic", true, "")

	// ── attribs ─────────────────────────────────────────────────────────────
	w.SetSizeLimits(100, 100, 800, 600)
	w.SetAspectRatio(16, 9)
	check("SetSizeLimits / SetAspectRatio: no panic", true, "")

	w.SetAttrib(glfw.Resizable, 1)
	w.SetAttrib(glfw.Decorated, 1)
	w.SetAttrib(glfw.Floating, 1)
	w.SetAttrib(glfw.Floating, 0)
	check("SetAttrib(Resizable/Decorated/Floating): no panic", true, "")

	check("GetAttrib(Resizable) == 1", w.GetAttrib(glfw.Resizable) == 1, "")
	check("GetAttrib(Decorated) == 1", w.GetAttrib(glfw.Decorated) == 1, "")
	_ = w.GetAttrib(glfw.Visible)
	_ = w.GetAttrib(glfw.Iconified)
	_ = w.GetAttrib(glfw.Maximized)
	_ = w.GetAttrib(glfw.Focused)
	check("GetAttrib(Visible/Iconified/Maximized/Focused): no panic", true, "")

	// ── opacity ─────────────────────────────────────────────────────────────
	op := w.GetOpacity()
	check("GetOpacity: 0..1", op >= 0 && op <= 1, fmt.Sprintf("%.2f", op))
	w.SetOpacity(0.5)
	check("SetOpacity: no panic", true, "")
	w.SetOpacity(1.0)

	// ── lifecycle ───────────────────────────────────────────────────────────
	check("ShouldClose initially false", !w.ShouldClose(), "")
	w.SetShouldClose(true)
	check("ShouldClose after SetShouldClose(true)", w.ShouldClose(), "")
	w.SetShouldClose(false)

	w.Show()
	w.Hide()
	check("Show / Hide: no panic", true, "")

	w.Iconify()
	glfw.PollEvents()
	w.Restore()
	glfw.PollEvents()
	w.Maximize()
	glfw.PollEvents()
	w.Restore()
	check("Iconify / Restore / Maximize: no panic", true, "")

	w.Focus()
	check("Focus: no panic", true, "")

	w.RequestAttention()
	check("RequestAttention: no panic", true, "")

	// ── monitor / fullscreen ────────────────────────────────────────────────
	check("GetMonitor before fullscreen == nil", w.GetMonitor() == nil, "")
	w.SetMonitor(nil, 0, 0, 320, 240, 0)
	check("SetMonitor(nil) → windowed: no panic", true, "")

	// ── handle / GoWindow ───────────────────────────────────────────────────
	h := w.Handle()
	check("Window.Handle non-nil", h != nil, fmt.Sprintf("%v", h))
	got := glfw.GoWindow(h)
	check("GoWindow(Handle()) == w", got == w, "")

	// ── user pointer ────────────────────────────────────────────────────────
	dummy := unsafe.Pointer(&width)
	w.SetUserPointer(dummy)
	check("Window.SetUserPointer / GetUserPointer round-trip",
		w.GetUserPointer() == dummy, "")
	glfw.SetWindowUserPointer(w, nil)
	check("SetWindowUserPointer / GetWindowUserPointer round-trip",
		glfw.GetWindowUserPointer(w) == nil, "")

	// ── clipboard (window-scoped methods) ───────────────────────────────────
	const text = "darwin clipboard test"
	w.SetClipboardString(text)
	check("Window.SetClipboardString / Window.GetClipboardString round-trip",
		w.GetClipboardString() == text, w.GetClipboardString())

	// ── input ───────────────────────────────────────────────────────────────
	check("GetKey(KeyA) initial Release", w.GetKey(glfw.KeyA) == glfw.Release, "")
	check("GetMouseButton(MouseButtonLeft) initial Release",
		w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Release, "")

	cx, cy := w.GetCursorPos()
	check("GetCursorPos: no panic", true, fmt.Sprintf("(%.1f,%.1f)", cx, cy))
	w.SetCursorPos(100, 100)
	check("SetCursorPos: no panic", true, "")

	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	check("GetInputMode(CursorMode) == CursorNormal",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorNormal, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
	check("GetInputMode(CursorMode) == CursorHidden",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorHidden, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	check("GetInputMode(CursorMode) == CursorDisabled",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)

	// Sticky + lock-mods modes.
	w.SetInputMode(glfw.StickyKeys, 1)
	w.SetInputMode(glfw.StickyMouseButtons, 1)
	w.SetInputMode(glfw.LockKeyMods, 1)
	w.SetInputMode(glfw.RawMouseMotion, 1)
	check("SetInputMode(StickyKeys/StickyMouseButtons/LockKeyMods/RawMouseMotion): no panic", true, "")
	w.SetInputMode(glfw.RawMouseMotion, 0)

	// ── native handle ───────────────────────────────────────────────────────
	check("GetCocoaWindow non-zero", w.GetCocoaWindow() != 0,
		fmt.Sprintf("0x%x", w.GetCocoaWindow()))
	// NSGLContext is zero unless an OpenGL context was requested; without GL
	// hints CreateWindow still creates a NoAPI window — accept either.
	_ = w.GetNSGLContext()
	check("GetNSGLContext: no panic", true,
		fmt.Sprintf("0x%x", w.GetNSGLContext()))

	// ── context / GL ────────────────────────────────────────────────────────
	w.MakeContextCurrent()
	check("MakeContextCurrent: no panic", true, "")
	cur := glfw.GetCurrentContext()
	check("GetCurrentContext returns this window or nil", cur == w || cur == nil, "")

	glfw.SwapInterval(1)
	check("SwapInterval: no panic", true, "")
	w.SwapBuffers()
	check("SwapBuffers: no panic", true, "")

	addr := glfw.GetProcAddress("glClear")
	check("GetProcAddress(\"glClear\"): no panic", true,
		fmt.Sprintf("addr=%v", addr))
	supported := glfw.ExtensionSupported("GL_ARB_vertex_buffer_object")
	check("ExtensionSupported: no panic", true, fmt.Sprintf("supported=%v", supported))

	glfw.DetachCurrentContext()
	check("DetachCurrentContext: no panic", true, "")

	// ── cursors ─────────────────────────────────────────────────────────────
	for _, shape := range []glfw.StandardCursorShape{
		glfw.ArrowCursor, glfw.IBeamCursor, glfw.CrosshairCursor,
		glfw.HandCursor, glfw.HResizeCursor, glfw.VResizeCursor,
	} {
		c, cerr := glfw.CreateStandardCursor(shape)
		check(fmt.Sprintf("CreateStandardCursor(%v): no error", shape),
			cerr == nil, fmt.Sprintf("%v", cerr))
		if c != nil {
			w.SetCursor(c)
			c.Destroy()
		}
	}

	// Custom cursor (1x1 white pixel).
	pix := []byte{255, 255, 255, 255}
	custom, cerr := glfw.CreateCursor(&glfw.Image{Width: 1, Height: 1, Pixels: pix}, 0, 0)
	check("CreateCursor: no error", cerr == nil, fmt.Sprintf("%v", cerr))
	if custom != nil {
		w.SetCursor(custom)
		check("SetCursor(custom): no panic", true, "")
		glfw.DestroyCursor(custom)
	}
	w.SetCursor(nil)
	check("SetCursor(nil): no panic", true, "")

	// ── callbacks (separate from window methods) ────────────────────────────
	testWindowCallbacks(w)

	// ── Vulkan surface (graceful failure, no instance) ──────────────────────
	if glfw.VulkanSupported() {
		_, err := w.CreateWindowSurface(nil, nil)
		check("CreateWindowSurface(nil instance): returns error without panic",
			err != nil, fmt.Sprintf("%v", err))
	}

	// ── teardown ────────────────────────────────────────────────────────────
	w.Destroy()
	check("Destroy: no panic", true, "")
	glfw.DefaultWindowHints()
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	runtime.LockOSThread()

	fmt.Println("=== glfw-purego macOS comprehensive smoke test ===")

	if err := glfw.Init(); err != nil {
		fmt.Printf("FATAL glfw.Init() failed: %v\n", err)
		os.Exit(1)
	}
	defer glfw.Terminate()
	check("Init: no error", true, "")

	testVersionAndHints()
	testTimer()
	testFeatureQueries()
	testEvents()
	testMonitors()
	testJoystickStubs()
	testVulkan()
	testWindow()

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
