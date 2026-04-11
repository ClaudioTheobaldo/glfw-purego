// Package glfw provides CGO-less GLFW 3.3 bindings for Go.
//
// This package is a drop-in replacement for github.com/go-gl/glfw/v3.3/glfw.
// It reimplements the GLFW Go API surface directly over native platform APIs
// without wrapping the GLFW C library, using:
//
//   - golang.org/x/sys/windows  — Win32 + WGL on Windows
//   - github.com/ebitengine/purego + ObjC runtime — Cocoa + CGL on macOS
//   - github.com/jezek/xgb + purego dlopen(libGL) — X11 + GLX on Linux
//   - purego dlopen(libwayland-client, libEGL) — Wayland + EGL on Linux
//
// No C compiler is required. CGO_ENABLED=0 builds work out of the box.
//
// # Usage
//
//	if err := glfw.Init(); err != nil {
//	    log.Fatal(err)
//	}
//	defer glfw.Terminate()
//
//	glfw.WindowHint(glfw.ContextVersionMajor, 2)
//	glfw.WindowHint(glfw.ContextVersionMinor, 1)
//
//	window, err := glfw.CreateWindow(800, 600, "Hello", nil, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	window.MakeContextCurrent()
//
//	for !window.ShouldClose() {
//	    // render ...
//	    window.SwapBuffers()
//	    glfw.PollEvents()
//	}
//
// The API is identical to github.com/go-gl/glfw/v3.3/glfw.
package glfw
