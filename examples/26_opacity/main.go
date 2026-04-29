//go:build windows

// 26_opacity demonstrates GetOpacity and SetOpacity for window translucency.
//
// Controls:
//
//	Up / Down arrow — increase / decrease opacity by 5%
//	R              — reset to fully opaque
//	ESC            — quit
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 26_opacity.exe .
package main

import (
	"fmt"
	"log"

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

	win, err := glfw.CreateWindow(800, 600, "Opacity — glfw-purego", nil, nil)
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

	opacity := float32(1.0)
	const step = 0.05

	setOpacity := func(v float32) {
		if v < 0.1 {
			v = 0.1
		}
		if v > 1.0 {
			v = 1.0
		}
		opacity = v
		win.SetOpacity(opacity)
		fmt.Printf("Opacity: %.0f%%\n", opacity*100)
	}

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press && action != glfw.Repeat {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyUp:
			setOpacity(win.GetOpacity() + step)
		case glfw.KeyDown:
			setOpacity(win.GetOpacity() - step)
		case glfw.KeyR:
			setOpacity(1.0)
		}
	})

	fmt.Println("Up/Down = opacity ±5%, R = reset, ESC = quit")
	fmt.Printf("Initial opacity: %.0f%%\n", opacity*100)

	for !win.ShouldClose() {
		win.SetTitle(fmt.Sprintf("Opacity | %.0f%% — use Up/Down arrows", win.GetOpacity()*100))

		// Bright background so translucency is obvious.
		gl.ClearColor(0.20, 0.60, 0.90, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
