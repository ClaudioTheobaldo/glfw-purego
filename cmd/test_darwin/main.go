//go:build darwin

// test_darwin is a smoke test for the macOS backend.
//
// Unlike the X11 and Wayland tests this one is designed to run in a headless
// CI environment — it only exercises APIs that work without a display server
// (version, timer, joystick stubs, clipboard, proc-address).  Window creation
// tests will be added once the Cocoa backend is implemented.
//
// Run: go run ./cmd/test_darwin   (from repo root)
package main

import (
	"fmt"
	"os"
	"runtime"

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

// ── tests ─────────────────────────────────────────────────────────────────────

func testVersion() {
	fmt.Println("── Version ──────────────────────────────────────────")
	major, minor, rev := glfw.GetVersion()
	check("GetVersion returns 3.3.x",
		major == 3 && minor == 3, fmt.Sprintf("got %d.%d.%d", major, minor, rev))
	vs := glfw.GetVersionString()
	check("GetVersionString non-empty", vs != "", vs)
}

func testTimer() {
	fmt.Println("── Timer ────────────────────────────────────────────")
	t0 := glfw.GetTime()
	check("GetTime >= 0", t0 >= 0, fmt.Sprintf("%v", t0))
	freq := glfw.GetTimerFrequency()
	check("GetTimerFrequency > 0", freq > 0, fmt.Sprintf("%d", freq))
	val := glfw.GetTimerValue()
	check("GetTimerValue > 0", val > 0, fmt.Sprintf("%d", val))
}

func testSetTime() {
	fmt.Println("── SetTime ──────────────────────────────────────────")
	glfw.SetTime(10.0)
	t := glfw.GetTime()
	check("SetTime then GetTime >= 10.0", t >= 10.0, fmt.Sprintf("got %.3f", t))
	glfw.SetTime(0)
}

func testClipboard() {
	fmt.Println("── Clipboard ────────────────────────────────────────")
	const text1 = "glfw-purego darwin clipboard test ✓"
	const text2 = "second value"
	glfw.SetClipboardString(text1)
	got1 := glfw.GetClipboardString()
	check("SetClipboardString / GetClipboardString round-trip",
		got1 == text1, fmt.Sprintf("got %q", got1))
	glfw.SetClipboardString(text2)
	got2 := glfw.GetClipboardString()
	check("Second clipboard value round-trip",
		got2 == text2, fmt.Sprintf("got %q", got2))
}

func testJoystickStubs() {
	fmt.Println("── Joystick (no device connected on CI) ─────────────")
	// CI runners have no physical gamepad; all slots should be empty.
	check("JoystickPresent(0) = false (no device)", !glfw.JoystickPresent(glfw.Joystick1), "")
	check("GetJoystickAxes(0) = nil (no device)", glfw.GetJoystickAxes(glfw.Joystick1) == nil, "")
	check("GetJoystickButtons(0) = nil (no device)", glfw.GetJoystickButtons(glfw.Joystick1) == nil, "")
	check("GetJoystickName(0) = '' (no device)", glfw.GetJoystickName(glfw.Joystick1) == "", "")
	check("JoystickIsGamepad(0) = false (no device)", !glfw.JoystickIsGamepad(glfw.Joystick1), "")
	check("GetGamepadState(0) = false (no device)", !glfw.GetGamepadState(glfw.Joystick1, &glfw.GamepadState{}), "")
}

func testPollEvents() {
	fmt.Println("── PollEvents / WaitEventsTimeout ───────────────────")
	// These are no-ops in the stub but must not panic.
	glfw.PollEvents()
	check("PollEvents: no panic", true, "")
	glfw.WaitEventsTimeout(0.001)
	check("WaitEventsTimeout: no panic", true, "")
	glfw.PostEmptyEvent()
	check("PostEmptyEvent: no panic", true, "")
}

func testInitHints() {
	fmt.Println("── Hints ────────────────────────────────────────────")
	// Stub — just verify no panic.
	glfw.InitHint(glfw.Focused, 1)
	glfw.WindowHintString(glfw.Focused, "value")
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.DefaultWindowHints()
	check("Hint functions: no panic", true, "")
}

func testFeatureQueries() {
	fmt.Println("── Feature queries ──────────────────────────────────")
	supported := glfw.RawMouseMotionSupported()
	check("RawMouseMotionSupported: ran without panic", true,
		fmt.Sprintf("result=%v", supported))
}

func testVulkan() {
	fmt.Println("── Vulkan ───────────────────────────────────────────")
	// MoltenVK is not installed on stock CI runners, so VulkanSupported may
	// be false — that is acceptable.  We only assert no panic and correct
	// behaviour when the loader is absent.
	vs := glfw.VulkanSupported()
	check("VulkanSupported: ran without panic", true,
		fmt.Sprintf("result=%v", vs))
	exts := glfw.GetRequiredInstanceExtensions()
	if vs {
		check("GetRequiredInstanceExtensions: 2 extensions when supported",
			len(exts) == 2, fmt.Sprintf("%v", exts))
	} else {
		check("GetRequiredInstanceExtensions: nil when unsupported",
			exts == nil, fmt.Sprintf("%v", exts))
	}
}

func testMonitors() {
	fmt.Println("── Monitors ─────────────────────────────────────────")
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("%v", err))
	// CI runners have at least one virtual display.
	check("GetMonitors: at least one monitor", len(monitors) > 0, fmt.Sprintf("n=%d", len(monitors)))

	pm := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor: non-nil", pm != nil, "")
	if pm != nil {
		check("GetPrimaryMonitor: name non-empty", pm.GetName() != "", pm.GetName())
		vm := pm.GetVideoMode()
		check("GetPrimaryMonitor: current video mode non-nil", vm != nil, "")
		if vm != nil {
			check("VideoMode width > 0", vm.Width > 0, fmt.Sprintf("w=%d", vm.Width))
			check("VideoMode height > 0", vm.Height > 0, fmt.Sprintf("h=%d", vm.Height))
		}
		vms := pm.GetVideoModes()
		check("GetVideoModes: non-empty", len(vms) > 0, fmt.Sprintf("n=%d", len(vms)))
	}
}

func testCallbacks() {
	fmt.Println("── Callbacks ────────────────────────────────────────")
	glfw.SetMonitorCallback(func(_ *glfw.Monitor, _ glfw.PeripheralEvent) {})
	check("SetMonitorCallback: no panic", true, "")
	glfw.SetMonitorCallback(nil)

	glfw.SetJoystickCallback(func(_ glfw.Joystick, _ glfw.PeripheralEvent) {})
	check("SetJoystickCallback: no panic", true, "")
	glfw.SetJoystickCallback(nil)
}

func testWindow() {
	fmt.Println("── Window (Phase A) ─────────────────────────────────")
	glfw.WindowHint(glfw.Visible, 0) // invisible — CI has no display
	w, err := glfw.CreateWindow(320, 240, "smoke-test", nil, nil)
	check("CreateWindow: no error", err == nil, fmt.Sprintf("%v", err))
	if w == nil {
		check("CreateWindow: non-nil *Window", false, "got nil")
		return
	}
	check("CreateWindow: non-nil *Window", true, "")

	// GetSize should return the requested dimensions.
	width, height := w.GetSize()
	check("GetSize matches requested width", width == 320, fmt.Sprintf("got %d", width))
	check("GetSize matches requested height", height == 240, fmt.Sprintf("got %d", height))

	// SetTitle must not panic.
	w.SetTitle("updated title")
	check("SetTitle: no panic", true, "")

	// GetFramebufferSize must return positive values (may be 2× on Retina).
	fbW, fbH := w.GetFramebufferSize()
	check("GetFramebufferSize width > 0", fbW > 0, fmt.Sprintf("got %d", fbW))
	check("GetFramebufferSize height > 0", fbH > 0, fmt.Sprintf("got %d", fbH))

	// ShouldClose starts as false.
	check("ShouldClose initially false", !w.ShouldClose(), "")
	w.SetShouldClose(true)
	check("ShouldClose after SetShouldClose(true)", w.ShouldClose(), "")

	// PollEvents must not panic with a live window.
	glfw.PollEvents()
	check("PollEvents with window: no panic", true, "")

	// PostEmptyEvent must not panic.
	glfw.PostEmptyEvent()
	check("PostEmptyEvent: no panic", true, "")

	// WaitEventsTimeout with a very short timeout must not hang.
	glfw.WaitEventsTimeout(0.001)
	check("WaitEventsTimeout(1ms): no panic", true, "")

	w.Destroy()
	check("Destroy: no panic", true, "")

	glfw.DefaultWindowHints() // restore defaults
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	runtime.LockOSThread()

	fmt.Println("=== glfw-purego macOS smoke test (headless) ===")

	// Init must succeed now that the Cocoa backend is implemented (Phase A+).
	if err := glfw.Init(); err != nil {
		fmt.Printf("FATAL glfw.Init() failed: %v\n", err)
		os.Exit(1)
	}
	defer glfw.Terminate()
	check("Init: no error", true, "")

	testVersion()
	testTimer()
	testSetTime()
	testClipboard()
	testJoystickStubs()
	testPollEvents()
	testInitHints()
	testFeatureQueries()
	testVulkan()
	testMonitors()
	testCallbacks()
	testWindow()

	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
