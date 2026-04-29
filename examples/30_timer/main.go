//go:build windows

// 30_timer demonstrates GetTimerFrequency and GetTimerValue for high-resolution
// timing, comparing them against the double-precision GetTime.
//
// The window title updates every frame with both measurements so you can
// verify they track each other closely.
//
// Controls:
//
//	R   — reset GetTime base (calls SetTime(0))
//	ESC — quit
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 30_timer.exe .
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

	win, err := glfw.CreateWindow(800, 600, "Timer — glfw-purego", nil, nil)
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

	freq := glfw.GetTimerFrequency()
	fmt.Printf("Timer frequency: %d Hz (%.3f MHz)\n", freq, float64(freq)/1e6)

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyR:
			glfw.SetTime(0)
			fmt.Println("Timer reset via SetTime(0)")
		}
	})

	fmt.Printf("Press R to reset timer, ESC to quit.\n\n")

	frames := 0
	for !win.ShouldClose() {
		frames++

		// High-res raw timer.
		ticks := glfw.GetTimerValue()
		rawSec := float64(ticks) / float64(freq)

		// Double-precision GetTime (same underlying counter, different base).
		getTimeSec := glfw.GetTime()

		if frames%60 == 0 {
			fmt.Printf("frame %5d | GetTimerValue: %10d ticks = %.4f s | GetTime: %.4f s\n",
				frames, ticks, rawSec, getTimeSec)
		}

		win.SetTitle(fmt.Sprintf("Timer | raw: %.3f s | GetTime: %.3f s | freq: %d Hz",
			rawSec, getTimeSec, freq))

		gl.ClearColor(0.05, 0.10, 0.15, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		win.SwapBuffers()
		glfw.PollEvents()
	}
}
