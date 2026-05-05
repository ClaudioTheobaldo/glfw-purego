//go:build linux && !wayland

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
	"unsafe"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

var dummy int // sentinel address used for user-pointer round-trips

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

func testJoystick() {
	// ── API surface smoke-test ────────────────────────────────────────────────
	// All calls must complete without panicking regardless of whether any
	// joystick is physically connected.
	defer func() {
		if r := recover(); r != nil {
			fail("Joystick API: no panic", fmt.Sprintf("panic: %v", r))
		}
	}()

	// Callback registration — must not panic.
	var cbFired bool
	glfw.SetJoystickCallback(func(joy glfw.Joystick, ev glfw.PeripheralEvent) {
		cbFired = true
		_ = cbFired
	})
	pass("SetJoystickCallback: no panic", "")

	// Scan all 16 slots.
	connected := 0
	for i := range 16 {
		joy := glfw.Joystick(i)

		present := joy.Present()
		if !present {
			continue
		}
		connected++

		name := glfw.GetJoystickName(joy)
		guid := glfw.GetJoystickGUID(joy)
		axes := joy.GetAxes()
		btns := joy.GetButtons()
		hats := joy.GetHats()
		isGP := glfw.JoystickIsGamepad(joy)
		gpName := glfw.GetGamepadName(joy)

		fmt.Printf("      [%d] %-30s  GUID=%-16s  axes=%d  buttons=%d  hats=%d  gamepad=%v\n",
			i, name, guid, len(axes), len(btns), len(hats), isGP)

		// GetGamepadState — valid return only makes sense for a gamepad.
		if isGP {
			var gs glfw.GamepadState
			ok := glfw.GetGamepadState(joy, &gs)
			check(fmt.Sprintf("GetGamepadState[%d]", i), ok, fmt.Sprintf("name=%q", gpName))
		}
	}

	if connected == 0 {
		skipTest("Joystick: presence check", "no joystick connected — skipping device checks")
		// Still verify that all getters return the zero-value for slot 0.
		check("JoystickPresent(0) == false (no device)", !glfw.JoystickPresent(glfw.Joystick1), "")
		check("GetJoystickAxes(0) == nil", glfw.GetJoystickAxes(glfw.Joystick1) == nil, "")
		check("GetJoystickButtons(0) == nil", glfw.GetJoystickButtons(glfw.Joystick1) == nil, "")
		check("GetJoystickHats(0) == nil", glfw.GetJoystickHats(glfw.Joystick1) == nil, "")
		check("GetJoystickName(0) == \"\"", glfw.GetJoystickName(glfw.Joystick1) == "", "")
		check("GetJoystickGUID(0) == \"\"", glfw.GetJoystickGUID(glfw.Joystick1) == "", "")
		check("JoystickIsGamepad(0) == false", !glfw.JoystickIsGamepad(glfw.Joystick1), "")
		var gs glfw.GamepadState
		check("GetGamepadState(0) == false", !glfw.GetGamepadState(glfw.Joystick1, &gs), "")
		pass("UpdateGamepadMappings: no panic", "")
		glfw.UpdateGamepadMappings("")
	} else {
		pass(fmt.Sprintf("Joystick: found %d device(s)", connected), "")
	}
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

// ── version / hints / time / events / feature queries ────────────────────────

func testVersionAndHints() {
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

	check("RawMouseMotionSupported: ran without panic", true,
		fmt.Sprintf("result=%v", glfw.RawMouseMotionSupported()))
}

func testTimer() {
	check("GetTime >= 0", glfw.GetTime() >= 0, "")
	check("GetTimerFrequency > 0", glfw.GetTimerFrequency() > 0, "")
	check("GetTimerValue > 0", glfw.GetTimerValue() > 0, "")

	glfw.SetTime(10.0)
	check("SetTime then GetTime >= 10.0", glfw.GetTime() >= 10.0, "")
	glfw.SetTime(0)
}

func testEvents() {
	glfw.PollEvents()
	glfw.PostEmptyEvent()
	glfw.WaitEventsTimeout(0.001)
	check("PollEvents / PostEmptyEvent / WaitEventsTimeout: no panic", true, "")
}

// ── monitor user-pointer + native handles ─────────────────────────────────────

func testMonitorExtras() {
	pm := glfw.GetPrimaryMonitor()
	if pm == nil {
		return
	}
	dummy := unsafe.Pointer(&dummy)
	pm.SetUserPointer(dummy)
	check("Monitor.SetUserPointer / GetUserPointer round-trip",
		pm.GetUserPointer() == dummy, "")
	pm.SetUserPointer(nil)

	check("Monitor.GetX11Adapter: no panic", true,
		fmt.Sprintf("0x%x", pm.GetX11Adapter()))
	check("Monitor.GetX11Monitor: no panic", true,
		fmt.Sprintf("0x%x", pm.GetX11Monitor()))

	// Gamma APIs are real on X11 via libXxf86vm.
	pm.SetGamma(1.0)
	pm.GetGammaRamp()
	pm.SetGammaRamp(&glfw.GammaRamp{})
	check("Monitor gamma APIs: no panic", true, "")

	glfw.SetMonitorCallback(func(_ *glfw.Monitor, _ glfw.PeripheralEvent) {})
	glfw.SetMonitorCallback(nil)
	check("SetMonitorCallback set/clear: no panic", true, "")
}

// ── window: comprehensive method coverage ─────────────────────────────────────

func testWindowComprehensive(w *glfw.Window) {
	width, height := w.GetSize()
	check("GetSize > 0", width > 0 && height > 0,
		fmt.Sprintf("%dx%d", width, height))

	w.SetSize(500, 350)
	check("SetSize: no panic", true, "")

	x, y := w.GetPos()
	check("GetPos: no panic", true, fmt.Sprintf("(%d,%d)", x, y))
	w.SetPos(120, 120)
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

	w.SetTitle("comprehensive-test")
	check("SetTitle: no panic", true, "")
	check("InternalTitle reflects last SetTitle",
		w.InternalTitle() == "comprehensive-test", w.InternalTitle())

	w.SetIcon(nil)
	glfw.SetIconFromImages(w, nil)
	check("SetIcon / SetIconFromImages: no panic", true, "")

	// Lifecycle.
	check("ShouldClose initial false", !w.ShouldClose(), "")
	w.SetShouldClose(true)
	check("SetShouldClose(true)", w.ShouldClose(), "")
	w.SetShouldClose(false)

	w.Show()
	w.Hide()
	w.Show()
	check("Show / Hide / Show: no panic", true, "")

	for range 3 {
		glfw.PollEvents()
	}
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

	// Attribs.
	_ = w.GetAttrib(glfw.Visible)
	_ = w.GetAttrib(glfw.Iconified)
	_ = w.GetAttrib(glfw.Maximized)
	_ = w.GetAttrib(glfw.Focused)
	_ = w.GetAttrib(glfw.Resizable)
	_ = w.GetAttrib(glfw.Decorated)
	check("GetAttrib(...): no panic", true, "")
	w.SetAttrib(glfw.Resizable, 1)
	w.SetAttrib(glfw.Decorated, 1)
	w.SetAttrib(glfw.Floating, 0)
	check("SetAttrib(Resizable/Decorated/Floating): no panic", true, "")

	// Monitor / fullscreen.
	_ = w.GetMonitor()
	check("GetMonitor: no panic", true, "")
	w.SetMonitor(nil, 0, 0, 320, 240, 0)
	check("SetMonitor(nil): no panic", true, "")

	// Handle / GoWindow.
	h := w.Handle()
	check("Window.Handle non-nil", h != nil, "")
	check("GoWindow(Handle()) == w", glfw.GoWindow(h) == w, "")

	// User pointer.
	dummy := unsafe.Pointer(&width)
	w.SetUserPointer(dummy)
	check("Window.SetUserPointer / GetUserPointer round-trip",
		w.GetUserPointer() == dummy, "")
	glfw.SetWindowUserPointer(w, nil)
	check("SetWindowUserPointer / GetWindowUserPointer round-trip",
		glfw.GetWindowUserPointer(w) == nil, "")

	// Window-scoped clipboard.
	const text = "x11 window clipboard"
	w.SetClipboardString(text)
	for range 3 {
		glfw.PollEvents()
	}
	check("Window.SetClipboardString / GetClipboardString round-trip",
		w.GetClipboardString() == text, w.GetClipboardString())

	// Input.
	check("GetKey(KeyA) initial Release", w.GetKey(glfw.KeyA) == glfw.Release, "")
	check("GetMouseButton(MouseButtonLeft) initial Release",
		w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Release, "")
	cx, cy := w.GetCursorPos()
	check("GetCursorPos: no panic", true, fmt.Sprintf("(%.1f,%.1f)", cx, cy))
	w.SetCursorPos(50, 50)
	check("SetCursorPos: no panic", true, "")

	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	check("GetInputMode == CursorNormal",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorNormal, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
	check("GetInputMode == CursorHidden",
		w.GetInputMode(glfw.CursorMode) == glfw.CursorHidden, "")
	w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	w.SetInputMode(glfw.StickyKeys, 1)
	w.SetInputMode(glfw.StickyMouseButtons, 1)
	w.SetInputMode(glfw.LockKeyMods, 1)
	check("SetInputMode (cursor / sticky / lock-mods): no panic", true, "")

	// Native X11 handles.
	check("Window.GetX11Window non-zero", w.GetX11Window() != 0,
		fmt.Sprintf("0x%x", w.GetX11Window()))
	check("GetX11Display non-zero", glfw.GetX11Display() != 0,
		fmt.Sprintf("0x%x", glfw.GetX11Display()))
	_ = w.GetGLXContext()
	check("Window.GetGLXContext: no panic", true,
		fmt.Sprintf("0x%x", w.GetGLXContext()))
	_ = w.GetGLXWindow()
	check("Window.GetGLXWindow: no panic", true,
		fmt.Sprintf("0x%x", w.GetGLXWindow()))
	const sel = "x11-selection-test"
	glfw.SetX11SelectionString(sel)
	glfw.PollEvents()
	check("SetX11SelectionString / GetX11SelectionString round-trip",
		glfw.GetX11SelectionString() == sel, glfw.GetX11SelectionString())

	// GL context (window was created with NoAPI in this flow — skip these
	// when there's no context).
	if w.GetGLXContext() != 0 {
		w.MakeContextCurrent()
		cur := glfw.GetCurrentContext()
		check("GetCurrentContext returns this window", cur == w, "")
		glfw.SwapInterval(1)
		w.SwapBuffers()
		_ = glfw.GetProcAddress("glClear")
		_ = glfw.ExtensionSupported("GLX_EXT_swap_control")
		glfw.DetachCurrentContext()
		check("MakeContextCurrent / SwapBuffers / DetachCurrentContext: no panic", true, "")
	}

	// Cursors.
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

	// Vulkan surface (graceful failure with nil instance).
	if glfw.VulkanSupported() {
		_, err := w.CreateWindowSurface(nil, nil)
		check("CreateWindowSurface(nil instance): returns error without panic",
			err != nil, fmt.Sprintf("%v", err))
		addr := glfw.GetVulkanGetInstanceProcAddress()
		check("GetVulkanGetInstanceProcAddress non-nil when supported",
			addr != nil, fmt.Sprintf("%v", addr))
	}
}

// ── window callbacks (all 17) ─────────────────────────────────────────────────

func funcID(f any) uintptr {
	if f == nil {
		return 0
	}
	v := *(*[2]uintptr)(unsafe.Pointer(&f))
	return v[1]
}

func testCallbacks(w *glfw.Window) {
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

	fmt.Println("── Version / hints ────────────────────────────────────────────")
	testVersionAndHints()
	fmt.Println()

	fmt.Println("── Timer ──────────────────────────────────────────────────────")
	testTimer()
	fmt.Println()

	fmt.Println("── Events ─────────────────────────────────────────────────────")
	testEvents()
	fmt.Println()

	fmt.Println("── Monitors (XRandR) ──────────────────────────────────────────")
	testMonitors()

	fmt.Println("── Monitor extras (user pointer / native handles / gamma) ─────")
	testMonitorExtras()
	fmt.Println()

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
		fmt.Println()
		fmt.Println("── Window: comprehensive method coverage ─────────────────────")
		testWindowComprehensive(w)
		fmt.Println()
		fmt.Println("── Window callbacks (all 17) ─────────────────────────────────")
		testCallbacks(w)
		w.Destroy()
	}

	fmt.Println("── Joystick ───────────────────────────────────────────────────")
	testJoystick()
	fmt.Println()

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
