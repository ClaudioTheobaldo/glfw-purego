//go:build linux && wayland

// test_wayland is a self-contained smoke test for the Wayland backend.
// It must be run inside a Wayland session (WAYLAND_DISPLAY must be set).
//
// Run: go run -tags wayland ./cmd/test_wayland   (from repo root)
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

func mustInit() {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "glfw.Init failed: %v\n", err)
		os.Exit(1)
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
}

func testMonitors() {
	fmt.Println("── Monitors ─────────────────────────────────────────")
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("%v", err))
	check("GetMonitors: at least one monitor", len(monitors) > 0,
		fmt.Sprintf("got %d", len(monitors)))
	if len(monitors) > 0 {
		m := monitors[0]
		check("Monitor name non-empty", m.GetName() != "", m.GetName())
		vm := m.GetVideoMode()
		check("VideoMode non-nil", vm != nil, "")
		if vm != nil {
			check("VideoMode width > 0", vm.Width > 0, fmt.Sprintf("width=%d", vm.Width))
			check("VideoMode height > 0", vm.Height > 0, fmt.Sprintf("height=%d", vm.Height))
		}
	}
	pm := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor non-nil", pm != nil, "")
}

func testFeatureQueries() {
	fmt.Println("── Feature queries ──────────────────────────────────")
	// RawMouseMotion is not yet wired on Wayland; should return false without panic.
	supported := glfw.RawMouseMotionSupported()
	check("RawMouseMotionSupported: ran without panic", true,
		fmt.Sprintf("result=%v", supported))
}

func testWindow() {
	fmt.Println("── Window ───────────────────────────────────────────")
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	w, err := glfw.CreateWindow(320, 240, "Wayland smoke test", nil, nil)
	check("CreateWindow: no error", err == nil, fmt.Sprintf("%v", err))
	if err != nil || w == nil {
		fmt.Println("  SKIP  (remaining window tests require a valid window)")
		return
	}
	defer func() {
		w.Destroy()
		check("Destroy: no panic", true, "")
	}()

	// ── Phase 1-9 parity additions ────────────────────────────────────────
	check("Window.GetWaylandWindow non-zero", w.GetWaylandWindow() != 0,
		fmt.Sprintf("0x%x", w.GetWaylandWindow()))
	check("GetWaylandDisplay non-zero", glfw.GetWaylandDisplay() != 0,
		fmt.Sprintf("0x%x", glfw.GetWaylandDisplay()))
	w.Show()
	check("Window.Show: no panic", true, "")

	sx, sy := w.GetContentScale()
	check("Window.GetContentScale >= 1.0", sx >= 1.0 && sy >= 1.0,
		fmt.Sprintf("(%.2f, %.2f)", sx, sy))

	w.SetClipboardString("wl-scoped")
	check("Window.GetClipboardString round-trip",
		w.GetClipboardString() == "wl-scoped", w.GetClipboardString())

	// Vulkan loader address (may be nil if no Vulkan installed; just no panic).
	procAddr := glfw.GetVulkanGetInstanceProcAddress()
	check("GetVulkanGetInstanceProcAddress: ran without panic", true,
		fmt.Sprintf("addr=%v", procAddr))

	// Size
	width, height := w.GetSize()
	check("GetSize > 0", width > 0 && height > 0,
		fmt.Sprintf("%dx%d", width, height))

	fbW, fbH := w.GetFramebufferSize()
	check("GetFramebufferSize > 0", fbW > 0 && fbH > 0,
		fmt.Sprintf("%dx%d", fbW, fbH))

	// Position (always 0,0 on Wayland — just check no panic)
	x, y := w.GetPos()
	check("GetPos: no panic", true, fmt.Sprintf("(%d,%d)", x, y))

	// Title
	w.SetTitle("Updated title")
	check("SetTitle: no panic", true, "")

	// ShouldClose
	check("ShouldClose initial false", !w.ShouldClose(), "")
	w.SetShouldClose(true)
	check("SetShouldClose(true)", w.ShouldClose(), "")
	w.SetShouldClose(false)

	// Context
	w.MakeContextCurrent()
	cur := glfw.GetCurrentContext()
	check("GetCurrentContext returns this window", cur == w, "")

	// SwapInterval (no-panic)
	glfw.SwapInterval(1)
	check("SwapInterval: no panic", true, "")

	// SwapBuffers (no-panic)
	w.SwapBuffers()
	check("SwapBuffers: no panic", true, "")

	glfw.DetachCurrentContext()

	// Size limits
	w.SetSizeLimits(100, 100, 800, 600)
	check("SetSizeLimits: no panic", true, "")

	// Aspect ratio
	w.SetAspectRatio(16, 9)
	check("SetAspectRatio: no panic", true, "")

	// Input mode
	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	check("GetInputMode CursorNormal",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorNormal, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
	check("GetInputMode CursorHidden",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorHidden, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)

	// Cursor (standard shapes)
	arrow, cerr := glfw.CreateStandardCursor(glfw.ArrowCursor)
	check("CreateStandardCursor: no error", cerr == nil, fmt.Sprintf("%v", cerr))
	if arrow != nil {
		w.SetCursor(arrow)
		check("SetCursor arrow: no panic", true, "")
		glfw.DestroyCursor(arrow)
	}

	// Key / mouse button state
	_ = w.GetKey(glfw.KeyA)
	_ = w.GetMouseButton(glfw.MouseButtonLeft)
	check("GetKey / GetMouseButton: no panic", true, "")

	// Cursor position
	cx, cy := w.GetCursorPos()
	check("GetCursorPos: no panic", true, fmt.Sprintf("(%.1f,%.1f)", cx, cy))

	// Attribs
	_ = w.GetAttrib(glfw.Visible)
	check("GetAttrib: no panic", true, "")

	// Window operations (no panic)
	w.Maximize()
	glfw.PollEvents()
	w.Restore()
	glfw.PollEvents()
	w.Iconify()
	glfw.PollEvents()
	check("Maximize/Restore/Iconify: no panic", true, "")

	// Fullscreen toggle (to nil = windowed)
	w.SetMonitor(nil, 0, 0, 320, 240, 60)
	check("SetMonitor(nil): no panic", true, "")
}

func testClipboard() {
	fmt.Println("── Clipboard ────────────────────────────────────────")
	const text1 = "glfw-purego wayland clipboard test ✓"
	const text2 = "second clipboard value"

	glfw.SetClipboardString(text1)
	got1 := glfw.GetClipboardString()
	check("SetClipboardString / GetClipboardString round-trip",
		got1 == text1, fmt.Sprintf("got %q", got1))

	glfw.SetClipboardString(text2)
	got2 := glfw.GetClipboardString()
	check("Second clipboard value round-trip",
		got2 == text2, fmt.Sprintf("got %q", got2))
}

func testProcAddress() {
	fmt.Println("── ProcAddress ──────────────────────────────────────")
	// Must have called MakeContextCurrent first; skip here (already detached).
	addr := glfw.GetProcAddress("glClear")
	// May be nil if no context is current — just check no panic.
	check("GetProcAddress: no panic", true,
		fmt.Sprintf("glClear=%v", addr != nil))
}

func testMonitorCallback() {
	fmt.Println("── MonitorCallback ──────────────────────────────────")
	fired := false
	glfw.SetMonitorCallback(func(m *glfw.Monitor, e glfw.PeripheralEvent) {
		fired = true
	})
	check("SetMonitorCallback: no panic", true, "")
	// Can't reliably trigger connect/disconnect in a test; just ensure no crash.
	glfw.PollEvents()
	check("PollEvents after SetMonitorCallback: no panic", true,
		fmt.Sprintf("callback fired: %v", fired))
	glfw.SetMonitorCallback(nil) // deregister
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		fmt.Fprintln(os.Stderr,
			"WAYLAND_DISPLAY is not set — run inside a Wayland session.")
		os.Exit(1)
	}

	fmt.Println("=== glfw-purego Wayland smoke test ===")

	mustInit()
	defer glfw.Terminate()

	testVersion()
	testTimer()
	testMonitors()
	testFeatureQueries()
	testWindow()
	testClipboard()
	testProcAddress()
	testMonitorCallback()

	fmt.Println()
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
