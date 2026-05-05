// render_smoke is the end-to-end render-loop integration test.
//
// It verifies the entire chain Init → CreateWindow → MakeContextCurrent →
// glClearColor + glClear + glFinish → glReadPixels → SwapBuffers → Destroy
// works on every backend.  Unlike the cmd/test_* programs (which test
// individual APIs in isolation) this exercises the GL context plumbing
// end-to-end with a real readback assertion: clear to a known colour, read
// the pixel back, and verify the values match within 8-bit rounding
// tolerance.
//
// GL function pointers are resolved through glfw.GetProcAddress (purposefully
// — the procaddr code is the platform glue that has historically been hardest
// to get right) and bound via purego.RegisterFunc, so this binary depends only
// on glfw-purego and ebitengine/purego — no separate gl-purego checkout needed.
//
// Build tags: linux requires either no tag (X11) or `-tags wayland`.  On
// Windows the test creates an OpenGL window via WGL.  On macOS NSOpenGLContext.
// All platforms request a GL 2.1-compatible context (universal).
//
// Run:
//   go run ./cmd/render_smoke               # Linux X11, macOS, Windows
//   go run -tags wayland ./cmd/render_smoke # Linux Wayland
package main

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"

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

// ── GL bindings (resolved via glfw.GetProcAddress at runtime) ─────────────────

const (
	glColorBufferBit uint32 = 0x00004000
	glRGBA           uint32 = 0x1908
	glUnsignedByte   uint32 = 0x1401
	glNoError        uint32 = 0
)

var (
	glClearColor func(r, g, b, a float32)
	glClear      func(mask uint32)
	glFinish     func()
	glReadPixels func(x, y, width, height int32, format, gltype uint32, pixels uintptr)
	glGetError   func() uint32
	glViewport   func(x, y, width, height int32)
)

// resolveGL binds the GL functions we need through glfw.GetProcAddress.
// Returns an error listing any symbols that could not be resolved.
func resolveGL() error {
	type entry struct {
		name string
		fptr any
	}
	bindings := []entry{
		{"glClearColor", &glClearColor},
		{"glClear", &glClear},
		{"glFinish", &glFinish},
		{"glReadPixels", &glReadPixels},
		{"glGetError", &glGetError},
		{"glViewport", &glViewport},
	}
	var missing []string
	for _, b := range bindings {
		addr := glfw.GetProcAddress(b.name)
		if addr == nil {
			missing = append(missing, b.name)
			continue
		}
		purego.RegisterFunc(b.fptr, uintptr(addr))
	}
	if len(missing) > 0 {
		return fmt.Errorf("unresolved GL symbols: %v", missing)
	}
	return nil
}

// ── test ──────────────────────────────────────────────────────────────────────

// roughEq returns true if a and b agree within ±tol — tolerates 8-bit rounding
// when converting from float clear-colour to a unsigned-byte readback.
func roughEq(a, b, tol int) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

func main() {
	runtime.LockOSThread()

	fmt.Println("=== glfw-purego render-loop smoke test ===")

	if err := glfw.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL glfw.Init() failed: %v\n", err)
		os.Exit(1)
	}
	defer glfw.Terminate()
	check("Init: no error", true, "")

	// Request a vanilla 2.1-class context on every platform.  Every backend's
	// software-rendering fallback supports at least GL 2.1 / GLES 2.0.
	//
	// We deliberately do NOT pass Visible=0 — on macOS, NSOpenGLContext's
	// drawable framebuffer is only allocated once the window's view is laid
	// out by AppKit, which requires the window to be on-screen.  In CI the
	// runners are headless or use software display servers, so a brief flash
	// is invisible to humans and the framebuffer becomes valid.
	glfw.WindowHint(glfw.Resizable, 0)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	w, err := glfw.CreateWindow(64, 64, "render-smoke", nil, nil)
	check("CreateWindow with GL context: no error", err == nil,
		fmt.Sprintf("%v", err))
	if w == nil {
		fmt.Println("FATAL: cannot continue without a window")
		os.Exit(1)
	}
	defer w.Destroy()

	// Pump events so the window is mapped / laid out before we read pixels.
	w.Show()
	for range 5 {
		glfw.PollEvents()
	}

	w.MakeContextCurrent()
	check("MakeContextCurrent: no panic", true, "")
	check("GetCurrentContext == w", glfw.GetCurrentContext() == w, "")

	if err := resolveGL(); err != nil {
		check("Resolve GL function pointers", false, err.Error())
		// Continue — the rest of the test will report nothing-resolved bugs as
		// nil-deref panics caught by the runtime.
		return
	}
	check("Resolve GL function pointers (glClear/Color/Finish/ReadPixels/GetError/Viewport)",
		true, "")

	// The framebuffer size may differ from the window size on HiDPI macs.
	fbW, fbH := w.GetFramebufferSize()
	check("Framebuffer size > 0", fbW > 0 && fbH > 0,
		fmt.Sprintf("%dx%d", fbW, fbH))
	glViewport(0, 0, int32(fbW), int32(fbH))

	// Clear to a distinctive non-grey colour: (0.25, 0.50, 0.75, 1.00).
	// 8-bit readback values: 64, 128, 191, 255.
	const r, g, b, a float32 = 0.25, 0.50, 0.75, 1.00
	glClearColor(r, g, b, a)
	glClear(glColorBufferBit)
	glFinish()

	check("glGetError after clear == GL_NO_ERROR",
		glGetError() == glNoError, "")

	// Read one pixel from the back buffer's bottom-left corner.
	var pixel [4]byte
	glReadPixels(0, 0, 1, 1, glRGBA, glUnsignedByte,
		uintptr(unsafe.Pointer(&pixel[0])))
	check("glGetError after readback == GL_NO_ERROR",
		glGetError() == glNoError, "")

	expR, expG, expB, expA := 64, 128, 191, 255
	got := [4]int{int(pixel[0]), int(pixel[1]), int(pixel[2]), int(pixel[3])}
	check("Pixel readback matches glClearColor (within ±2 of 64/128/191/255)",
		roughEq(got[0], expR, 2) &&
			roughEq(got[1], expG, 2) &&
			roughEq(got[2], expB, 2) &&
			roughEq(got[3], expA, 2),
		fmt.Sprintf("got (%d,%d,%d,%d) want (~%d,%d,%d,%d)",
			got[0], got[1], got[2], got[3],
			expR, expG, expB, expA))

	// Present the back buffer.  We don't read the front; just verify no panic.
	w.SwapBuffers()
	check("SwapBuffers: no panic", true, "")

	// One PollEvents tick to drain anything the compositor / WM queued.
	glfw.PollEvents()
	check("PollEvents post-render: no panic", true, "")

	glfw.DetachCurrentContext()
	check("DetachCurrentContext: no panic", true, "")

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
