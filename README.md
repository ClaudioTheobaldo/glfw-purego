# glfw-purego

CGO-less GLFW 3.3 bindings for Go — a drop-in replacement for [`github.com/go-gl/glfw/v3.3/glfw`](https://github.com/go-gl/glfw).

Rather than wrapping the GLFW C library, this package reimplements the GLFW Go API surface directly over native platform APIs — no C compiler required.

- **Windows**: Win32 + WGL via `golang.org/x/sys/windows`
- **macOS**: Cocoa + CGL via `github.com/ebitengine/purego` + ObjC runtime
- **Linux X11**: X11 via `github.com/jezek/xgb` + GLX via purego
- **Linux Wayland**: Wayland + EGL via purego

## Supported platforms

| Platform | Backend | Status |
|----------|---------|--------|
| Windows (amd64, arm64) | Win32 + WGL | 🚧 |
| macOS (amd64, arm64) | Cocoa + CGL | 🚧 |
| Linux X11 (amd64, arm64) | XGB + GLX | 🚧 |
| Linux Wayland | Wayland + EGL | 🚧 |

## Usage

```go
import "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"

func main() {
    if err := glfw.Init(); err != nil {
        log.Fatal(err)
    }
    defer glfw.Terminate()

    glfw.WindowHint(glfw.ContextVersionMajor, 2)
    glfw.WindowHint(glfw.ContextVersionMinor, 1)

    window, err := glfw.CreateWindow(800, 600, "Hello", nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    window.MakeContextCurrent()

    for !window.ShouldClose() {
        // render ...
        window.SwapBuffers()
        glfw.PollEvents()
    }
}
```

## Drop-in replacement

```go
// Before
import "github.com/go-gl/glfw/v3.3/glfw"

// After
import "github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw"
```

Or via `go.mod` replace directive:

```
replace github.com/go-gl/glfw/v3.3/glfw => github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw
```

## Acknowledgements

This repository was built in collaboration with [Claude Code](https://claude.ai/claude-code) (Anthropic Claude Sonnet 4.6).
