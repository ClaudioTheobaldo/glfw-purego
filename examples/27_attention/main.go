//go:build windows

// 27_attention demonstrates RequestAttention and PostEmptyEvent.
//
// RequestAttention flashes the taskbar button to alert the user.
// PostEmptyEvent wakes up a blocked WaitEvents from a background goroutine.
//
// Controls:
//
//	F — flash taskbar (RequestAttention)
//	W — switch to WaitEvents mode (background goroutine posts every 2 s)
//	P — switch back to PollEvents mode
//	ESC — quit
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 27_attention.exe .
package main

import (
	"fmt"
	"log"
	"time"

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

	win, err := glfw.CreateWindow(800, 600, "Attention & PostEmptyEvent — glfw-purego", nil, nil)
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

	useWait := false
	frames := 0

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyF:
			win.RequestAttention()
			fmt.Println("RequestAttention called — check the taskbar button!")
		case glfw.KeyW:
			useWait = true
			fmt.Println("Switched to WaitEvents — background goroutine will post every 2 s")
		case glfw.KeyP:
			useWait = false
			fmt.Println("Switched back to PollEvents")
		}
	})

	// Background goroutine that keeps WaitEvents alive by posting every 2 s.
	go func() {
		for {
			time.Sleep(2 * time.Second)
			glfw.PostEmptyEvent()
		}
	}()

	fmt.Println("F = flash taskbar, W = WaitEvents mode, P = PollEvents mode, ESC = quit")

	for !win.ShouldClose() {
		frames++
		mode := "PollEvents"
		if useWait {
			mode = "WaitEvents"
		}
		win.SetTitle(fmt.Sprintf("Attention | frame %d | mode: %s", frames, mode))

		gl.ClearColor(0.08, 0.15, 0.08, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		win.SwapBuffers()

		if useWait {
			glfw.WaitEvents()
		} else {
			glfw.PollEvents()
		}
	}
}
