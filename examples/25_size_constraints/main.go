//go:build windows

// 25_size_constraints demonstrates SetSizeLimits and SetAspectRatio.
//
// Controls:
//
//	L — toggle size limits (min 400×300, max 1200×800)
//	A — cycle aspect ratio (free / 16:9 / 4:3 / 1:1)
//	ESC — quit
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 25_size_constraints.exe .
package main

import (
	"fmt"
	"log"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

type aspectMode int

const (
	aspectFree aspectMode = iota
	aspect16x9
	aspect4x3
	aspect1x1
	aspectCount
)

func (a aspectMode) String() string {
	return [...]string{"Free", "16:9", "4:3", "1:1"}[a]
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalf("glfw.Init: %v", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfileHint, int(glfw.CoreProfile))
	glfw.WindowHint(glfw.OpenGLForwardCompatible, 1)
	glfw.WindowHint(glfw.Resizable, 1)

	win, err := glfw.CreateWindow(800, 600, "Size Constraints — glfw-purego", nil, nil)
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

	limitsOn := false
	aspect := aspectFree

	applyConstraints := func() {
		if limitsOn {
			win.SetSizeLimits(400, 300, 1200, 800)
		} else {
			win.SetSizeLimits(-1, -1, -1, -1)
		}
		switch aspect {
		case aspect16x9:
			win.SetAspectRatio(16, 9)
		case aspect4x3:
			win.SetAspectRatio(4, 3)
		case aspect1x1:
			win.SetAspectRatio(1, 1)
		default:
			win.SetAspectRatio(-1, -1)
		}
		limits := "OFF"
		if limitsOn {
			limits = "ON (400×300 – 1200×800)"
		}
		fmt.Printf("Limits: %-28s  Aspect: %s\n", limits, aspect.String())
	}

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyL:
			limitsOn = !limitsOn
			applyConstraints()
		case glfw.KeyA:
			aspect = (aspect + 1) % aspectCount
			applyConstraints()
		}
	})

	fmt.Println("L = toggle size limits, A = cycle aspect ratio, ESC = quit")
	applyConstraints()

	for !win.ShouldClose() {
		w, h := win.GetSize()
		win.SetTitle(fmt.Sprintf("Size Constraints | %dx%d | limits:%v aspect:%s",
			w, h, limitsOn, aspect.String()))

		gl.ClearColor(0.10, 0.12, 0.18, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
