//go:build windows

// test_windows is the comprehensive Windows smoke test.  It exercises every
// public API in v3.3/glfw that's applicable on Windows: library lifecycle,
// monitors, windows, all 17 callbacks, context, input, cursors, clipboard,
// joystick, Vulkan probes, and the Win32/WGL native handles.
//
// Run: go run ./cmd/test_windows  (from repo root, on Windows)
package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"
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

func funcID(f any) uintptr {
	if f == nil {
		return 0
	}
	v := *(*[2]uintptr)(unsafe.Pointer(&f))
	return v[1]
}

// ── library / version / time / events / hints ─────────────────────────────────

func testVersionAndHints() {
	section("Version / hints")
	major, minor, _ := glfw.GetVersion()
	check("GetVersion returns 3.3.x", major == 3 && minor == 3,
		fmt.Sprintf("%d.%d", major, minor))
	check("GetVersionString non-empty", glfw.GetVersionString() != "",
		glfw.GetVersionString())

	glfw.InitHint(glfw.Focused, 1)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHintString(glfw.Focused, "value")
	glfw.DefaultWindowHints()
	check("InitHint / WindowHint / WindowHintString / DefaultWindowHints: no panic", true, "")

	check("GetKeyScancode returns int", true,
		fmt.Sprintf("KeyA=%d", glfw.GetKeyScancode(glfw.KeyA)))
	check("GetKeyName returns string", true,
		fmt.Sprintf("name=%q", glfw.GetKeyName(glfw.KeyA, 0)))

	check("RawMouseMotionSupported returns true on Windows",
		glfw.RawMouseMotionSupported(), "")
}

func testTimer() {
	section("Timer")
	check("GetTime >= 0", glfw.GetTime() >= 0, "")
	check("GetTimerFrequency > 0", glfw.GetTimerFrequency() > 0, "")
	check("GetTimerValue > 0", glfw.GetTimerValue() > 0, "")

	glfw.SetTime(10.0)
	check("SetTime then GetTime >= 10.0", glfw.GetTime() >= 10.0, "")
	glfw.SetTime(0)
}

func testEvents() {
	section("Events")
	glfw.PollEvents()
	glfw.PostEmptyEvent()
	glfw.WaitEventsTimeout(0.001)
	check("PollEvents / PostEmptyEvent / WaitEventsTimeout: no panic", true, "")
}

// ── monitors ──────────────────────────────────────────────────────────────────

func testMonitors() {
	section("Monitors")
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("err=%v", err))
	check("GetMonitors: non-empty", len(monitors) > 0, fmt.Sprintf("got %d", len(monitors)))

	pm := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor: non-nil", pm != nil, "")
	if pm == nil {
		return
	}
	inList := slices.ContainsFunc(monitors, func(m *glfw.Monitor) bool {
		return m.GetName() == pm.GetName()
	})
	check("GetPrimaryMonitor: name in GetMonitors list", inList, pm.GetName())

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
		check("VideoMode width>0 && height>0", vm.Width > 0 && vm.Height > 0,
			fmt.Sprintf("%dx%d", vm.Width, vm.Height))
	}

	modes := pm.GetVideoModes()
	check("Monitor.GetVideoModes: non-empty", len(modes) > 0, fmt.Sprintf("n=%d", len(modes)))

	// Gamma APIs: real on Windows via SetDeviceGammaRamp.
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

	// Native Win32 monitor handles.
	check("Monitor.GetWin32Adapter: no panic", true, pm.GetWin32Adapter())
	check("Monitor.GetWin32Monitor: no panic", true, pm.GetWin32Monitor())

	glfw.SetMonitorCallback(func(_ *glfw.Monitor, _ glfw.PeripheralEvent) {})
	glfw.SetMonitorCallback(nil)
	check("SetMonitorCallback set/clear: no panic", true, "")
}

// ── joystick (Windows uses XInput) ────────────────────────────────────────────

func testJoystickStubs() {
	section("Joystick")
	check("JoystickPresent(0) bool", true,
		fmt.Sprintf("present=%v", glfw.JoystickPresent(glfw.Joystick1)))

	for j := glfw.Joystick1; j <= glfw.Joystick16; j++ {
		if glfw.JoystickPresent(j) {
			fmt.Printf("  INFO  Joystick%d present: %s\n", int(j)+1, glfw.GetJoystickName(j))
		}
	}

	// All getters must run without panic — return values depend on whether a
	// device is connected, so we only assert no-panic.
	_ = glfw.GetJoystickAxes(glfw.Joystick1)
	_ = glfw.GetJoystickButtons(glfw.Joystick1)
	_ = glfw.GetJoystickHats(glfw.Joystick1)
	_ = glfw.GetJoystickName(glfw.Joystick1)
	_ = glfw.GetJoystickGUID(glfw.Joystick1)
	_ = glfw.JoystickIsGamepad(glfw.Joystick1)
	_ = glfw.GetGamepadName(glfw.Joystick1)
	check("Joystick getters: no panic", true, "")

	var gs glfw.GamepadState
	_ = glfw.GetGamepadState(glfw.Joystick1, &gs)
	_ = glfw.Joystick1.GetGamepadState()
	check("GetGamepadState (pkg + method): no panic", true, "")

	check("UpdateGamepadMappings: no panic", true,
		fmt.Sprintf("ok=%v", glfw.UpdateGamepadMappings("")))

	dummy := unsafe.Pointer(&gs)
	glfw.SetJoystickUserPointer(glfw.Joystick1, dummy)
	check("SetJoystickUserPointer (pkg) round-trip",
		glfw.GetJoystickUserPointer(glfw.Joystick1) == dummy, "")
	glfw.Joystick1.SetUserPointer(nil)
	check("Joystick.SetUserPointer (method) round-trip",
		glfw.Joystick1.GetUserPointer() == nil, "")

	glfw.SetJoystickCallback(func(_ glfw.Joystick, _ glfw.PeripheralEvent) {})
	glfw.SetJoystickCallback(nil)
	check("SetJoystickCallback: no panic", true, "")
}

// ── Vulkan ────────────────────────────────────────────────────────────────────

func testVulkan() {
	section("Vulkan")
	vs := glfw.VulkanSupported()
	check("VulkanSupported: ran without panic", true, fmt.Sprintf("supported=%v", vs))

	exts := glfw.GetRequiredInstanceExtensions()
	check("GetRequiredInstanceExtensions: non-nil", exts != nil, fmt.Sprintf("%v", exts))
	if vs {
		check("GetRequiredInstanceExtensions has VK_KHR_surface",
			slices.Contains(exts, "VK_KHR_surface"), fmt.Sprintf("%v", exts))
		check("GetRequiredInstanceExtensions has VK_KHR_win32_surface",
			slices.Contains(exts, "VK_KHR_win32_surface"), fmt.Sprintf("%v", exts))

		addr := glfw.GetVulkanGetInstanceProcAddress()
		check("GetVulkanGetInstanceProcAddress non-nil when supported",
			addr != nil, fmt.Sprintf("%v", addr))
	}
}

// ── window callbacks (all 17) ─────────────────────────────────────────────────

func testWindowCallbacks(w *glfw.Window) {
	section("Window callbacks (17)")
	type cbCase struct {
		name string
		set1 func() any
		set2 func() any
		setNil func() any
	}
	cases := []cbCase{
		{"SetPosCallback",
			func() any { return w.SetPosCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetPosCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetPosCallback(nil) }},
		{"SetSizeCallback",
			func() any { return w.SetSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetSizeCallback(nil) }},
		{"SetFramebufferSizeCallback",
			func() any { return w.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetFramebufferSizeCallback(func(_ *glfw.Window, _, _ int) {}) },
			func() any { return w.SetFramebufferSizeCallback(nil) }},
		{"SetCloseCallback",
			func() any { return w.SetCloseCallback(func(_ *glfw.Window) {}) },
			func() any { return w.SetCloseCallback(func(_ *glfw.Window) {}) },
			func() any { return w.SetCloseCallback(nil) }},
		{"SetMaximizeCallback",
			func() any { return w.SetMaximizeCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetMaximizeCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetMaximizeCallback(nil) }},
		{"SetRefreshCallback",
			func() any { return w.SetRefreshCallback(func(_ *glfw.Window) {}) },
			func() any { return w.SetRefreshCallback(func(_ *glfw.Window) {}) },
			func() any { return w.SetRefreshCallback(nil) }},
		{"SetFocusCallback",
			func() any { return w.SetFocusCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetFocusCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetFocusCallback(nil) }},
		{"SetIconifyCallback",
			func() any { return w.SetIconifyCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetIconifyCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetIconifyCallback(nil) }},
		{"SetContentScaleCallback",
			func() any { return w.SetContentScaleCallback(func(_ *glfw.Window, _, _ float32) {}) },
			func() any { return w.SetContentScaleCallback(func(_ *glfw.Window, _, _ float32) {}) },
			func() any { return w.SetContentScaleCallback(nil) }},
		{"SetMouseButtonCallback",
			func() any { return w.SetMouseButtonCallback(func(_ *glfw.Window, _ glfw.MouseButton, _ glfw.Action, _ glfw.ModifierKey) {}) },
			func() any { return w.SetMouseButtonCallback(func(_ *glfw.Window, _ glfw.MouseButton, _ glfw.Action, _ glfw.ModifierKey) {}) },
			func() any { return w.SetMouseButtonCallback(nil) }},
		{"SetCursorPosCallback",
			func() any { return w.SetCursorPosCallback(func(_ *glfw.Window, _, _ float64) {}) },
			func() any { return w.SetCursorPosCallback(func(_ *glfw.Window, _, _ float64) {}) },
			func() any { return w.SetCursorPosCallback(nil) }},
		{"SetCursorEnterCallback",
			func() any { return w.SetCursorEnterCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetCursorEnterCallback(func(_ *glfw.Window, _ bool) {}) },
			func() any { return w.SetCursorEnterCallback(nil) }},
		{"SetScrollCallback",
			func() any { return w.SetScrollCallback(func(_ *glfw.Window, _, _ float64) {}) },
			func() any { return w.SetScrollCallback(func(_ *glfw.Window, _, _ float64) {}) },
			func() any { return w.SetScrollCallback(nil) }},
		{"SetKeyCallback",
			func() any { return w.SetKeyCallback(func(_ *glfw.Window, _ glfw.Key, _ int, _ glfw.Action, _ glfw.ModifierKey) {}) },
			func() any { return w.SetKeyCallback(func(_ *glfw.Window, _ glfw.Key, _ int, _ glfw.Action, _ glfw.ModifierKey) {}) },
			func() any { return w.SetKeyCallback(nil) }},
		{"SetCharCallback",
			func() any { return w.SetCharCallback(func(_ *glfw.Window, _ rune) {}) },
			func() any { return w.SetCharCallback(func(_ *glfw.Window, _ rune) {}) },
			func() any { return w.SetCharCallback(nil) }},
		{"SetCharModsCallback",
			func() any { return w.SetCharModsCallback(func(_ *glfw.Window, _ rune, _ glfw.ModifierKey) {}) },
			func() any { return w.SetCharModsCallback(func(_ *glfw.Window, _ rune, _ glfw.ModifierKey) {}) },
			func() any { return w.SetCharModsCallback(nil) }},
		{"SetDropCallback",
			func() any { return w.SetDropCallback(func(_ *glfw.Window, _ []string) {}) },
			func() any { return w.SetDropCallback(func(_ *glfw.Window, _ []string) {}) },
			func() any { return w.SetDropCallback(nil) }},
	}
	for _, c := range cases {
		prev1 := c.set1()
		check(c.name+": first-set returns nil", prev1 == nil,
			fmt.Sprintf("got %v", prev1))
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
	glfw.WindowHint(glfw.Visible, 0)
	glfw.WindowHint(glfw.ClientAPIs, int(glfw.NoAPI))
	glfw.WindowHint(glfw.Resizable, 1)

	w, err := glfw.CreateWindow(320, 240, "smoke-test", nil, nil)
	check("CreateWindow: no error", err == nil, fmt.Sprintf("%v", err))
	if w == nil {
		return
	}
	defer w.Destroy()

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
	check("GetFramebufferSize > 0", fbW > 0 && fbH > 0,
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
	glfw.SetIconFromImages(w, nil)
	check("SetIcon / SetIconFromImages: no panic", true, "")

	// ── attribs / size limits / aspect ──────────────────────────────────────
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
	w.SetOpacity(1.0)
	check("SetOpacity: no panic", true, "")

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
	check("SetMonitor(nil): no panic", true, "")

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
	const text = "windows clipboard test"
	w.SetClipboardString(text)
	check("Window.SetClipboardString / GetClipboardString round-trip",
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
	check("GetInputMode == CursorNormal",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorNormal, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
	check("GetInputMode == CursorHidden",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorHidden, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	check("GetInputMode == CursorDisabled",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorDisabled, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)

	w.SetInputMode(glfw.StickyKeys, 1)
	w.SetInputMode(glfw.StickyMouseButtons, 1)
	w.SetInputMode(glfw.LockKeyMods, 1)
	w.SetInputMode(glfw.RawMouseMotion, 1)
	check("GetInputMode(RawMouseMotion) == 1",
		w.GetInputMode(glfw.RawMouseMotion) == 1, "")
	w.SetInputMode(glfw.RawMouseMotion, 0)
	check("SetInputMode (sticky / lock / raw): no panic", true, "")

	// ── native Win32 handles ────────────────────────────────────────────────
	check("Window.GetWin32Window non-zero", w.GetWin32Window() != 0,
		fmt.Sprintf("0x%x", w.GetWin32Window()))
	// WGL context is zero with NoAPI hint; just verify no panic.
	_ = w.GetWGLContext()
	check("Window.GetWGLContext: no panic", true,
		fmt.Sprintf("0x%x", w.GetWGLContext()))

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

	// ── callbacks ───────────────────────────────────────────────────────────
	testWindowCallbacks(w)

	// ── Vulkan surface (graceful failure with nil instance) ─────────────────
	if glfw.VulkanSupported() {
		_, err := w.CreateWindowSurface(nil, nil)
		check("CreateWindowSurface(nil instance): returns error without panic",
			err != nil, fmt.Sprintf("%v", err))
	}

	// ── package-level clipboard ─────────────────────────────────────────────
	const want = "pkg-clipboard-7291"
	glfw.SetClipboardString(want)
	check("SetClipboardString / GetClipboardString round-trip",
		glfw.GetClipboardString() == want, glfw.GetClipboardString())
	const want2 = "second-value-4472"
	glfw.SetClipboardString(want2)
	check("Clipboard: second write",
		glfw.GetClipboardString() == want2, glfw.GetClipboardString())

	glfw.DefaultWindowHints()
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	runtime.LockOSThread()

	fmt.Println("=== glfw-purego Windows comprehensive smoke test ===")

	if err := glfw.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL glfw.Init() failed: %v\n", err)
		os.Exit(1)
	}
	defer glfw.Terminate()
	check("Init: no error", true, "")

	testVersionAndHints()
	testTimer()
	testEvents()
	testMonitors()
	testJoystickStubs()
	testVulkan()
	testWindow()

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Result: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
