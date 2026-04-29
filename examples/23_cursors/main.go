//go:build windows

// 23_cursors showcases all standard GLFW cursor shapes plus a custom cursor
// created from a procedural RGBA image.  Press Space/Right to cycle forward,
// Left to cycle back, ESC to quit.
//
// Build (CGO disabled):
//
//	CGO_ENABLED=0 go build -o 23_cursors.exe .
package main

import (
	"fmt"
	"log"

	gl   "github.com/ClaudioTheobaldo/gl-purego/v3.3/gl"
	glfw "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
)

// makeCrosshairCursor builds a 16×16 RGBA image with a red crosshair at
// the centre row and column; all other pixels are fully transparent.
func makeCrosshairCursor() *glfw.Image {
	const size = 16
	pixels := make([]uint8, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if x == 8 || y == 8 {
				idx := (y*size + x) * 4
				pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3] = 255, 0, 0, 255
			}
		}
	}
	img := glfw.Image{Width: size, Height: size, Pixels: pixels}
	return &img
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

	win, err := glfw.CreateWindow(800, 600, "Cursor Showcase — glfw-purego", nil, nil)
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

	// Create all 6 standard cursors.
	shapes := []glfw.StandardCursorShape{
		glfw.ArrowCursor,
		glfw.IBeamCursor,
		glfw.CrosshairCursor,
		glfw.HandCursor,
		glfw.HResizeCursor,
		glfw.VResizeCursor,
	}
	cursors := make([]*glfw.Cursor, 0, 7)
	for _, shape := range shapes {
		c, err := glfw.CreateStandardCursor(shape)
		if err != nil {
			log.Fatalf("CreateStandardCursor: %v", err)
		}
		cursors = append(cursors, c)
	}

	// Create custom crosshair cursor.
	img := makeCrosshairCursor()
	customCursor, err := glfw.CreateCursor(img, 8, 8)
	if err != nil {
		log.Fatalf("CreateCursor: %v", err)
	}
	cursors = append(cursors, customCursor)

	cursorNames := []string{"Arrow", "IBeam", "Crosshair", "Hand", "HResize", "VResize", "Custom"}

	currentIdx := 0
	win.SetCursor(cursors[currentIdx])

	setCursor := func(idx int) {
		win.SetCursor(cursors[idx])
		fmt.Printf("Cursor → %s (%d/%d)  [move mouse inside window to see it]\n",
			cursorNames[idx], idx+1, len(cursors))
	}
	setCursor(currentIdx)

	win.SetKeyCallback(func(w *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
		if action != glfw.Press {
			return
		}
		switch key {
		case glfw.KeyEscape:
			w.SetShouldClose(true)
		case glfw.KeyRight, glfw.KeySpace:
			currentIdx = (currentIdx + 1) % len(cursors)
			setCursor(currentIdx)
		case glfw.KeyLeft:
			currentIdx = (currentIdx - 1 + len(cursors)) % len(cursors)
			setCursor(currentIdx)
		}
	})

	fmt.Println("Space/Right = next cursor, Left = previous cursor, ESC = quit")
	fmt.Println("IBeam = text I-bar, Crosshair = thin + symbol — best seen on dark background")

	for !win.ShouldClose() {
		gl.ClearColor(0.08, 0.08, 0.10, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		win.SetTitle(fmt.Sprintf("Cursor Showcase | %s (%d/%d)",
			cursorNames[currentIdx], currentIdx+1, len(cursors)))

		win.SwapBuffers()
		glfw.PollEvents()
	}

	// Destroy only the custom cursor (standard ones are system-owned).
	glfw.DestroyCursor(customCursor)
}
