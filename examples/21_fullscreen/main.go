//go:build windows

// 21_fullscreen demonstrates toggling between fullscreen and windowed mode
// using SetMonitor and GetVideoMode from glfw-purego.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 21_fullscreen.exe .
package main

import (
	"fmt"
	"log"
	"math"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)

	win, err := glfw.CreateWindow(800, 600, "Fullscreen Toggle — glfw-purego", nil, nil)
	if err != nil {
		log.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	win.MakeContextCurrent()
	glfw.SwapInterval(1)

	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		log.Fatalf("gl.Init: %v", err)
	}

	win.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	fw, fh := win.GetFramebufferSize()
	gl.Viewport(0, 0, int32(fw), int32(fh))

	fullscreen := false

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyF11:
			if fullscreen {
				// Return to windowed mode
				win.SetMonitor(nil, 100, 100, 800, 600, 0)
				fullscreen = false
			} else {
				// Go fullscreen on primary monitor
				monitor := glfw.GetPrimaryMonitor()
				if monitor != nil {
					vidmode := monitor.GetVideoMode()
					if vidmode != nil {
						win.SetMonitor(monitor, 0, 0, vidmode.Width, vidmode.Height, vidmode.RefreshRate)
					}
				}
				fullscreen = true
			}
		}
	})

	fmt.Println("F11 = toggle fullscreen, ESC = quit")

	for !win.ShouldClose() {
		t := glfw.GetTime()
		r := float32(0.5 + 0.5*math.Sin(t*0.4))
		g := float32(0.5 + 0.5*math.Sin(t*0.4+2.094))
		b := float32(0.5 + 0.5*math.Sin(t*0.4+4.189))

		gl.ClearColor(r*0.25, g*0.25, b*0.25, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Update title with current size and mode
		w, h := win.GetSize()
		mode := "windowed"
		if fullscreen {
			mode = "FULLSCREEN"
		}
		win.SetTitle(fmt.Sprintf("Fullscreen Toggle | %dx%d | %s", w, h, mode))

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
