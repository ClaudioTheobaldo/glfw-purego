//go:build windows

// 22_icon_attribs demonstrates setting a procedural window icon and toggling
// window attributes (Decorated, Floating, Resizable) at runtime.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 22_icon_attribs.exe .
package main

import (
	"fmt"
	"log"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// makeQuadrantIcon creates a size×size RGBA image split into 4 coloured quadrants.
func makeQuadrantIcon(size int) glfw.Image {
	pixels := make([]uint8, size*size*4)
	half := size / 2
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			idx := (y*size + x) * 4
			top := y < half
			left := x < half
			switch {
			case top && left: // top-left: red
				pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = 255, 0, 0, 255
			case top && !left: // top-right: green
				pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = 0, 255, 0, 255
			case !top && left: // bottom-left: blue
				pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = 0, 0, 255, 255
			default: // bottom-right: yellow
				pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = 255, 255, 0, 255
			}
		}
	}
	return glfw.Image{Width: size, Height: size, Pixels: pixels}
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

	win, err := glfw.CreateWindow(800, 600, "Icon & Attribs — glfw-purego", nil, nil)
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

	// Set procedural icon: 32×32 and 16×16 versions.
	img32 := makeQuadrantIcon(32)
	img16 := makeQuadrantIcon(16)
	win.SetIcon([]glfw.Image{img32, img16})

	// Track attribute states.
	decorated := true
	floating := false
	resizable := true

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyD:
			decorated = !decorated
			val := 0
			if decorated {
				val = 1
			}
			win.SetAttrib(glfw.Decorated, val)
		case glfw.KeyF:
			floating = !floating
			val := 0
			if floating {
				val = 1
			}
			win.SetAttrib(glfw.Floating, val)
		case glfw.KeyR:
			resizable = !resizable
			val := 0
			if resizable {
				val = 1
			}
			win.SetAttrib(glfw.Resizable, val)
		}
	})

	fmt.Println("D = toggle decorated, F = toggle floating (always-on-top), R = toggle resizable, ESC = quit")

	onOff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}

	for !win.ShouldClose() {
		gl.ClearColor(0.1, 0.1, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SetTitle(fmt.Sprintf("Icon & Attribs | D:%s F:%s R:%s",
			onOff(decorated), onOff(floating), onOff(resizable)))

		win.SwapBuffers()
		glfw.PollEvents()
	}
}
