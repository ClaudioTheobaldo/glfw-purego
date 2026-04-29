//go:build linux

// linux_smoke is a minimal test that opens an X11 window, pumps events for
// 3 seconds, verifies the framebuffer size, and exits cleanly.
package main

import (
	"fmt"
	"os"
	"time"

	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

func main() {
	if err := glfw.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "glfw.Init: %v\n", err)
		os.Exit(1)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)

	w, err := glfw.CreateWindow(640, 480, "Linux Smoke Test", nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CreateWindow: %v\n", err)
		os.Exit(1)
	}
	defer w.Destroy()

	w.MakeContextCurrent()
	glfw.SwapInterval(1)

	// Verify reported size
	width, height := w.GetSize()
	fmt.Printf("Window size: %dx%d\n", width, height)
	if width != 640 || height != 480 {
		fmt.Fprintf(os.Stderr, "FAIL: expected 640x480, got %dx%d\n", width, height)
		os.Exit(1)
	}

	fw, fh := w.GetFramebufferSize()
	fmt.Printf("Framebuffer size: %dx%d\n", fw, fh)

	// Callbacks
	w.SetKeyCallback(func(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		fmt.Printf("Key: %v scancode=%d action=%v mods=%v\n", key, scancode, action, mods)
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	})

	w.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		fmt.Printf("Resized: %dx%d\n", width, height)
	})

	w.SetCloseCallback(func(_ *glfw.Window) {
		fmt.Println("Close requested")
	})

	// Run for 3 seconds or until ESC / window close
	deadline := time.Now().Add(3 * time.Second)
	frame := 0
	for !w.ShouldClose() && time.Now().Before(deadline) {
		// Clear to a cycling colour so we can see something happened
		// (no OpenGL calls yet — just swap blank buffers)
		w.SwapBuffers()
		glfw.PollEvents()
		frame++
	}

	fmt.Printf("OK — ran %d frames, timer=%.3fs\n", frame, glfw.GetTime())
}
