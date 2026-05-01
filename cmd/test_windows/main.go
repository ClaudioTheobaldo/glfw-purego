//go:build windows

// test_windows exercises the new Group 1-5 APIs on Windows:
//   - GetMonitors / GetPrimaryMonitor
//   - VulkanSupported / GetRequiredInstanceExtensions
//   - Clipboard round-trip (SetClipboardString / GetClipboardString)
//
// Run: go run ./cmd/test_windows  (from repo root, Windows)
package main

import (
	"fmt"
	"os"
	"slices"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── helpers ──────────────────────────────────────────────────────────────────

var (
	passed, failed int
)

func pass(name, detail string) {
	passed++
	fmt.Printf("PASS  %-45s %s\n", name, detail)
}

func fail(name, detail string) {
	failed++
	fmt.Printf("FAIL  %-45s %s\n", name, detail)
}

func check(name string, cond bool, detail string) {
	if cond {
		pass(name, detail)
	} else {
		fail(name, detail)
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func testMonitors() {
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("err=%v", err))
	check("GetMonitors: non-empty", len(monitors) > 0, fmt.Sprintf("got %d", len(monitors)))

	for i, m := range monitors {
		vm := m.GetVideoMode()
		x, y := m.GetPos()
		wx, wy, ww, wh := m.GetWorkarea()
		sx, sy := m.GetContentScale()
		wmm, hmm := m.GetPhysicalSize()
		modes := m.GetVideoModes()
		tag := ""
		if m == glfw.GetPrimaryMonitor() {
			tag = " [primary]"
		}
		fmt.Printf("      [%d] %-20s %4dx%-4d @%dHz  pos=(%d,%d)  "+
			"work=(%d,%d,%d,%d)  scale=%.2fx%.2f  phys=%dx%dmm  %d modes%s\n",
			i, m.GetName(),
			vm.Width, vm.Height, vm.RefreshRate,
			x, y,
			wx, wy, ww, wh,
			sx, sy,
			wmm, hmm,
			len(modes),
			tag)
	}

	primary := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor: non-nil", primary != nil, "")
	if primary != nil {
		// GetPrimaryMonitor returns a fresh pointer each call, so compare by name.
		inList := slices.ContainsFunc(monitors, func(m *glfw.Monitor) bool {
			return m.GetName() == primary.GetName()
		})
		check("GetPrimaryMonitor: name in GetMonitors list", inList, primary.GetName())
	}
}

func testRawMouseMotion() {
	// RawMouseMotionSupported must return true on Windows (WM_INPUT is always available).
	check("RawMouseMotionSupported", glfw.RawMouseMotionSupported(), "")

	glfw.WindowHint(glfw.Visible, 0)
	glfw.WindowHint(glfw.ClientAPIs, int(glfw.NoAPI))
	w, err := glfw.CreateWindow(1, 1, "raw-motion-test", nil, nil)
	check("CreateWindow (raw motion prereq)", err == nil, fmt.Sprintf("err=%v", err))
	if err != nil {
		return
	}
	defer w.Destroy()

	// Enable raw mouse motion — no panic expected.
	func() {
		defer func() {
			if r := recover(); r != nil {
				fail("SetInputMode(RawMouseMotion,1): no panic", fmt.Sprintf("panic: %v", r))
			}
		}()
		w.SetInputMode(glfw.RawMouseMotion, 1)
		check("GetInputMode(RawMouseMotion) after enable", w.GetInputMode(glfw.RawMouseMotion) == 1, "")
		pass("SetInputMode(RawMouseMotion,1): no panic", "")
	}()

	// Disable raw mouse motion.
	func() {
		defer func() {
			if r := recover(); r != nil {
				fail("SetInputMode(RawMouseMotion,0): no panic", fmt.Sprintf("panic: %v", r))
			}
		}()
		w.SetInputMode(glfw.RawMouseMotion, 0)
		check("GetInputMode(RawMouseMotion) after disable", w.GetInputMode(glfw.RawMouseMotion) == 0, "")
		pass("SetInputMode(RawMouseMotion,0): no panic", "")
	}()
}

func testVulkan() {
	supported := glfw.VulkanSupported()
	// Just report — may be false if vulkan-1.dll isn't installed.
	check("VulkanSupported: ran without panic", true, fmt.Sprintf("result=%v", supported))

	exts := glfw.GetRequiredInstanceExtensions()
	check("GetRequiredInstanceExtensions: non-nil", exts != nil, fmt.Sprintf("%v", exts))
	if supported {
		wantSurface := slices.Contains(exts, "VK_KHR_surface")
		wantWin32 := slices.Contains(exts, "VK_KHR_win32_surface")
		check("GetRequiredInstanceExtensions: VK_KHR_surface", wantSurface, fmt.Sprintf("%v", exts))
		check("GetRequiredInstanceExtensions: VK_KHR_win32_surface", wantWin32, fmt.Sprintf("%v", exts))
	}
}

func testClipboard() {
	if err := glfw.Init(); err != nil {
		fail("Init (clipboard prereq)", err.Error())
		return
	}
	defer glfw.Terminate()

	// Create a hidden window — Windows clipboard needs a message queue.
	glfw.WindowHint(glfw.Visible, 0)
	glfw.WindowHint(glfw.ClientAPIs, int(glfw.NoAPI))
	w, err := glfw.CreateWindow(1, 1, "clipboard-test", nil, nil)
	check("CreateWindow (clipboard prereq)", err == nil, fmt.Sprintf("err=%v", err))
	if err != nil {
		return
	}
	defer w.Destroy()

	const want = "hello-glfw-test-clipboard-7291"
	glfw.SetClipboardString(want)
	got := glfw.GetClipboardString()
	check("Clipboard: round-trip", got == want,
		fmt.Sprintf("want=%q got=%q", want, got))

	// Second distinct value to make sure we're not reading a stale string.
	const want2 = "second-value-4472"
	glfw.SetClipboardString(want2)
	got2 := glfw.GetClipboardString()
	check("Clipboard: second write", got2 == want2,
		fmt.Sprintf("want=%q got=%q", want2, got2))
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== glfw-purego Windows API test ===")
	fmt.Println()

	fmt.Println("── Monitors ─────────────────────────────────────────────")
	testMonitors()
	fmt.Println()

	fmt.Println("── Vulkan ───────────────────────────────────────────────")
	testVulkan()
	fmt.Println()

	fmt.Println("── Raw Mouse Motion ─────────────────────────────────────")
	if err := glfw.Init(); err != nil {
		fail("Init (raw-motion prereq)", err.Error())
	} else {
		testRawMouseMotion()
		glfw.Terminate()
	}
	fmt.Println()

	fmt.Println("── Clipboard ────────────────────────────────────────────")
	testClipboard()
	fmt.Println()

	fmt.Printf("─────────────────────────────────────────────────────────\n")
	total := passed + failed
	fmt.Printf("Result: %d/%d passed", passed, total)
	if failed == 0 {
		fmt.Println("  ✓ all tests passed")
	} else {
		fmt.Printf("  ✗ %d FAILED\n", failed)
		os.Exit(1)
	}
}
