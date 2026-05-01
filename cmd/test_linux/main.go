//go:build linux

// test_linux exercises the new Group 1-5 APIs on Linux:
//   - GetMonitors / GetPrimaryMonitor  (XRandR struct-layout verification)
//   - VulkanSupported / GetRequiredInstanceExtensions
//   - SetOpacity / GetOpacity          (property round-trip)
//   - RequestAttention                 (no-crash)
//   - SetSizeLimits / SetAspectRatio   (xprop WM_NORMAL_HINTS read-back)
//   - SetClipboardString / GetClipboardString round-trip
//
// Prerequisites:
//   export DISPLAY=:0
//   xprop and xrandr must be on PATH (typically x11-utils package)
//
// Run: go run ./cmd/test_linux  (from repo root, Linux / WSL2)
package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// ── result tracking ──────────────────────────────────────────────────────────

var passed, failed int

func pass(name, detail string) {
	passed++
	if detail != "" {
		fmt.Printf("PASS  %-52s %s\n", name, detail)
	} else {
		fmt.Printf("PASS  %s\n", name)
	}
}

func fail(name, detail string) {
	failed++
	if detail != "" {
		fmt.Printf("FAIL  %-52s %s\n", name, detail)
	} else {
		fmt.Printf("FAIL  %s\n", name)
	}
}

func check(name string, cond bool, detail string) bool {
	if cond {
		pass(name, detail)
	} else {
		fail(name, detail)
	}
	return cond
}

func info(format string, args ...any) {
	fmt.Printf("INFO  "+format+"\n", args...)
}

func skipTest(name, reason string) {
	fmt.Printf("SKIP  %-52s %s\n", name, reason)
}

// ── xrandr parsing ────────────────────────────────────────────────────────────

// xrandrOutput holds what the xrandr CLI reports for one connected output.
type xrandrOutput struct {
	name    string
	primary bool
	w, h    int // current resolution in pixels
	x, y    int // position
	hz      int // refresh rate of the current (*) mode
}

// parseXrandr runs xrandr and parses connected outputs.
// Header format: <name> connected [primary] <W>x<H>+<X>+<Y> ...
// Mode lines:    "   1920x1080     59.95*+"  (entry with * = current)
func parseXrandr() ([]xrandrOutput, error) {
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")

	var outputs []xrandrOutput
	var cur *xrandrOutput

	for _, line := range lines {
		if strings.Contains(line, " connected ") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			o := xrandrOutput{name: parts[0]}
			idx := 2 // parts[1] == "connected"
			if idx < len(parts) && parts[idx] == "primary" {
				o.primary = true
				idx++
			}
			// Geometry token: WxH+X+Y (absent for disabled outputs)
			if idx < len(parts) && strings.Contains(parts[idx], "x") && strings.Contains(parts[idx], "+") {
				geo := parts[idx]
				plusIdx := strings.Index(geo, "+")
				res := geo[:plusIdx]
				rest := geo[plusIdx+1:]
				wh := strings.SplitN(res, "x", 2)
				xy := strings.SplitN(rest, "+", 2)
				if len(wh) == 2 && len(xy) == 2 {
					o.w, _ = strconv.Atoi(wh[0])
					o.h, _ = strconv.Atoi(wh[1])
					o.x, _ = strconv.Atoi(xy[0])
					o.y, _ = strconv.Atoi(xy[1])
				}
			}
			outputs = append(outputs, o)
			cur = &outputs[len(outputs)-1]
			continue
		}

		// Mode line — starts with spaces.
		if cur != nil && strings.HasPrefix(line, "   ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			res := fields[0]
			for _, hz := range fields[1:] {
				if !strings.Contains(hz, "*") {
					continue
				}
				hz = strings.TrimRight(hz, "*+")
				f, parseErr := strconv.ParseFloat(hz, 64)
				if parseErr == nil {
					cur.hz = int(math.Round(f))
					wh := strings.SplitN(res, "x", 2)
					if len(wh) == 2 {
						cur.w, _ = strconv.Atoi(wh[0])
						cur.h, _ = strconv.Atoi(wh[1])
					}
				}
				break
			}
		} else if cur != nil && !strings.HasPrefix(line, " ") {
			cur = nil
		}
	}
	return outputs, nil
}

// ── xprop helpers ─────────────────────────────────────────────────────────────

func xpropWindow(xid uintptr, propName string) (string, error) {
	hexID := fmt.Sprintf("0x%x", xid)
	out, err := exec.Command("xprop", "-id", hexID, propName).Output()
	return string(out), err
}

// ── monitor tests ─────────────────────────────────────────────────────────────

func testMonitors() {
	monitors, err := glfw.GetMonitors()
	check("GetMonitors: no error", err == nil, fmt.Sprintf("err=%v", err))
	if !check("GetMonitors: non-empty", len(monitors) > 0, fmt.Sprintf("got %d", len(monitors))) {
		return
	}

	fmt.Println()
	for i, m := range monitors {
		vm := m.GetVideoMode()
		x, y := m.GetPos()
		wmm, hmm := m.GetPhysicalSize()
		sx, sy := m.GetContentScale()
		all := m.GetVideoModes()
		primaryMon := glfw.GetPrimaryMonitor()
		tag := ""
		if primaryMon != nil && m.GetName() == primaryMon.GetName() {
			tag = " [primary]"
		}
		fmt.Printf("      [%d] %-15s %4dx%-4d @%dHz  pos=(%d,%d)  "+
			"phys=%dx%dmm  scale=%.2fx%.2f  %d modes%s\n",
			i, m.GetName(),
			vm.Width, vm.Height, vm.RefreshRate,
			x, y,
			wmm, hmm,
			sx, sy,
			len(all),
			tag)
	}
	fmt.Println()

	// Cross-check against xrandr CLI.
	xouts, xErr := parseXrandr()
	if xErr != nil {
		skipTest("XRandR cross-check", "xrandr not available: "+xErr.Error())
		return
	}

	info("xrandr reports %d connected output(s):", len(xouts))
	for _, o := range xouts {
		tag := ""
		if o.primary {
			tag = " [primary]"
		}
		info("  %-15s %dx%d+%d+%d @%dHz%s", o.name, o.w, o.h, o.x, o.y, o.hz, tag)
	}
	fmt.Println()

	// Every xrandr-connected output should appear in GetMonitors().
	glfwNames := make([]string, len(monitors))
	for i, m := range monitors {
		glfwNames[i] = m.GetName()
	}
	allPresent := true
	for _, o := range xouts {
		if !slices.Contains(glfwNames, o.name) {
			fail("GetMonitors: missing output "+o.name, fmt.Sprintf("glfw has %v", glfwNames))
			allPresent = false
		}
	}
	if allPresent {
		pass("GetMonitors: all xrandr outputs present", fmt.Sprintf("%v", glfwNames))
	}

	// Primary monitor resolution / position must match xrandr.
	primary := glfw.GetPrimaryMonitor()
	check("GetPrimaryMonitor: non-nil", primary != nil, "")
	if primary == nil {
		return
	}

	var xprim *xrandrOutput
	for i := range xouts {
		if xouts[i].primary {
			xprim = &xouts[i]
			break
		}
	}
	if xprim == nil && len(xouts) > 0 {
		xprim = &xouts[0] // single-monitor setups may not mark primary
	}
	if xprim == nil {
		return
	}

	vm := primary.GetVideoMode()
	var gW, gH int
	if vm != nil {
		gW, gH = vm.Width, vm.Height
	}
	check("Primary monitor resolution matches xrandr",
		gW == xprim.w && gH == xprim.h,
		fmt.Sprintf("glfw=%dx%d xrandr=%dx%d", gW, gH, xprim.w, xprim.h))

	if vm != nil && xprim.hz != 0 {
		hzDiff := vm.RefreshRate - xprim.hz
		if hzDiff < 0 {
			hzDiff = -hzDiff
		}
		check("Primary monitor refresh rate matches xrandr (±2 Hz)",
			hzDiff <= 2,
			fmt.Sprintf("glfw=%dHz xrandr=%dHz", vm.RefreshRate, xprim.hz))
	}

	px, py := primary.GetPos()
	check("Primary monitor position matches xrandr",
		px == xprim.x && py == xprim.y,
		fmt.Sprintf("glfw=(%d,%d) xrandr=(%d,%d)", px, py, xprim.x, xprim.y))
}

// ── Vulkan tests ──────────────────────────────────────────────────────────────

func testVulkan() {
	supported := glfw.VulkanSupported()
	check("VulkanSupported: ran without panic", true, fmt.Sprintf("result=%v", supported))

	exts := glfw.GetRequiredInstanceExtensions()
	check("GetRequiredInstanceExtensions: non-nil", exts != nil, fmt.Sprintf("%v", exts))
	if supported {
		check("GetRequiredInstanceExtensions: VK_KHR_surface",
			slices.Contains(exts, "VK_KHR_surface"), fmt.Sprintf("%v", exts))
		check("GetRequiredInstanceExtensions: VK_KHR_xlib_surface",
			slices.Contains(exts, "VK_KHR_xlib_surface"), fmt.Sprintf("%v", exts))
	}
}

// ── window-dependent tests ────────────────────────────────────────────────────

func testOpacity(w *glfw.Window, xid uintptr) {
	w.SetOpacity(0.5)
	got := w.GetOpacity()
	check("SetOpacity(0.5) → GetOpacity()",
		math.Abs(float64(got-0.5)) < 0.01,
		fmt.Sprintf("got %.4f", got))

	if out, err := xpropWindow(xid, "_NET_WM_WINDOW_OPACITY"); err == nil {
		hasIt := strings.Contains(out, "_NET_WM_WINDOW_OPACITY") && strings.Contains(out, "=")
		check("SetOpacity(0.5): xprop sees _NET_WM_WINDOW_OPACITY", hasIt, strings.TrimSpace(out))
	} else {
		skipTest("SetOpacity(0.5): xprop check", "xprop not available")
	}

	w.SetOpacity(1.0)
	got = w.GetOpacity()
	check("SetOpacity(1.0) → GetOpacity()",
		math.Abs(float64(got-1.0)) < 0.01,
		fmt.Sprintf("got %.4f", got))

	if out, err := xpropWindow(xid, "_NET_WM_WINDOW_OPACITY"); err == nil {
		// xprop prints "not found" when the property doesn't exist.
		gone := strings.Contains(out, "not found")
		check("SetOpacity(1.0): property deleted (xprop)", gone, strings.TrimSpace(out))
	} else {
		skipTest("SetOpacity(1.0): xprop check", "xprop not available")
	}
}

func testAttention(w *glfw.Window) {
	defer func() {
		if r := recover(); r != nil {
			fail("RequestAttention: no panic", fmt.Sprintf("panic: %v", r))
		}
	}()
	w.RequestAttention()
	pass("RequestAttention: no panic", "")
}

func testSizeHints(w *glfw.Window, xid uintptr) {
	w.SetSizeLimits(200, 150, 800, 600)
	glfw.PollEvents()

	if out, err := xpropWindow(xid, "WM_NORMAL_HINTS"); err == nil {
		check("SetSizeLimits: xprop has min 200×150",
			strings.Contains(out, "200") && strings.Contains(out, "150"),
			strings.TrimSpace(out))
		check("SetSizeLimits: xprop has max 800×600",
			strings.Contains(out, "800") && strings.Contains(out, "600"),
			"")
	} else {
		skipTest("SetSizeLimits: xprop check", "xprop not available")
	}

	w.SetAspectRatio(16, 9)
	glfw.PollEvents()

	if out, err := xpropWindow(xid, "WM_NORMAL_HINTS"); err == nil {
		// xprop prints "aspect ratio: 16/9 to 16/9" or shows raw values.
		check("SetAspectRatio(16,9): xprop has 16 and 9",
			strings.Contains(out, "16") && strings.Contains(out, "9"),
			strings.TrimSpace(out))
	} else {
		skipTest("SetAspectRatio: xprop check", "xprop not available")
	}
}

func testRawMouseMotion(w *glfw.Window) {
	supported := glfw.RawMouseMotionSupported()
	check("RawMouseMotionSupported: ran without panic", true, fmt.Sprintf("result=%v", supported))

	if !supported {
		skipTest("SetInputMode(RawMouseMotion,1)", "XInput2 not available")
		skipTest("GetInputMode(RawMouseMotion) after enable", "XInput2 not available")
		skipTest("SetInputMode(RawMouseMotion,0)", "XInput2 not available")
		skipTest("GetInputMode(RawMouseMotion) after disable", "XInput2 not available")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fail("SetInputMode(RawMouseMotion): no panic", fmt.Sprintf("panic: %v", r))
		}
	}()

	w.SetInputMode(glfw.RawMouseMotion, 1)
	check("GetInputMode(RawMouseMotion) after enable",
		w.GetInputMode(glfw.RawMouseMotion) == 1, "")
	pass("SetInputMode(RawMouseMotion,1): no panic", "")

	w.SetInputMode(glfw.RawMouseMotion, 0)
	check("GetInputMode(RawMouseMotion) after disable",
		w.GetInputMode(glfw.RawMouseMotion) == 0, "")
	pass("SetInputMode(RawMouseMotion,0): no panic", "")
}

func testClipboard() {
	const want = "hello-glfw-clipboard-test-3817"
	glfw.SetClipboardString(want)
	glfw.PollEvents()
	got := glfw.GetClipboardString()
	check("Clipboard: round-trip",
		got == want,
		fmt.Sprintf("want=%q got=%q", want, got))

	const want2 = "second-clipboard-value-9954"
	glfw.SetClipboardString(want2)
	glfw.PollEvents()
	got2 := glfw.GetClipboardString()
	check("Clipboard: second write",
		got2 == want2,
		fmt.Sprintf("want=%q got=%q", want2, got2))
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	if os.Getenv("DISPLAY") == "" {
		fmt.Fprintln(os.Stderr, "ERROR: DISPLAY not set — run under X11 (e.g. export DISPLAY=:0)")
		os.Exit(1)
	}

	fmt.Println("=== glfw-purego Linux API test ===")
	fmt.Println()

	if err := glfw.Init(); err != nil {
		fail("Init", err.Error())
		os.Exit(1)
	}
	pass("Init", "")
	defer glfw.Terminate()

	fmt.Println("── Monitors (XRandR) ──────────────────────────────────────────")
	testMonitors()

	fmt.Println("── Vulkan ─────────────────────────────────────────────────────")
	testVulkan()
	fmt.Println()

	fmt.Println("── Window (opacity / size-hints / clipboard) ──────────────────")
	glfw.WindowHint(glfw.ClientAPIs, int(glfw.NoAPI))
	w, err := glfw.CreateWindow(400, 300, "glfw-purego-test", nil, nil)
	if !check("CreateWindow", err == nil, fmt.Sprintf("err=%v", err)) {
		fail("window-dependent tests", "skipped — CreateWindow failed")
	} else {
		// A few event cycles so the WM maps the window and processes atoms.
		for i := 0; i < 3; i++ {
			glfw.PollEvents()
		}
		xid := uintptr(w.Handle())
		info("window XID = 0x%x", xid)
		fmt.Println()

		testOpacity(w, xid)
		fmt.Println()
		testAttention(w)
		fmt.Println()
		testSizeHints(w, xid)
		fmt.Println()
		fmt.Println("── Raw Mouse Motion ───────────────────────────────────────────")
		testRawMouseMotion(w)
		fmt.Println()
		testClipboard()
		w.Destroy()
	}

	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	total := passed + failed
	fmt.Printf("Result: %d/%d passed", passed, total)
	if failed == 0 {
		fmt.Println("  ✓ all tests passed")
	} else {
		fmt.Printf("  ✗ %d FAILED\n", failed)
		os.Exit(1)
	}
}
