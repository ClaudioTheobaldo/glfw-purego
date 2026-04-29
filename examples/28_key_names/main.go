//go:build windows

// 28_key_names demonstrates GetKeyName and GetKeyScancode.
//
// Press any key and the window title + stdout show its GLFW key constant,
// platform scancode, and the OS-localized printable name.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 28_key_names.exe .
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

	win, err := glfw.CreateWindow(800, 600, "Key Names — glfw-purego", nil, nil)
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

	lastInfo := "Press any key…"

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, _ glfw.ModifierKey) {
		if action == glfw.Release {
			return
		}
		if key == glfw.KeyEscape {
			w.SetShouldClose(true)
			return
		}

		// GetKeyScancode from the key constant.
		sc := glfw.GetKeyScancode(key)

		// GetKeyName using the key constant (takes precedence over scancode arg).
		name := glfw.GetKeyName(key, scancode)
		if name == "" {
			name = "(no printable name)"
		}

		lastInfo = fmt.Sprintf("key=%-4d  scancode=%-4d  name=%q", int(key), sc, name)
		fmt.Println(lastInfo)
	})

	fmt.Println("Press keys to see their GLFW code, scancode, and OS name. ESC = quit.")

	for !win.ShouldClose() {
		win.SetTitle("Key Names | " + lastInfo)

		gl.ClearColor(0.05, 0.05, 0.15, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
