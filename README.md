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

## Linux build tags

The Linux backend defaults to **X11**.  To build against the Wayland backend instead, pass `-tags wayland` to every `go build`, `go run`, and `go test` invocation:

```sh
go build -tags wayland ./...
go run  -tags wayland .
go test -tags wayland ./...
```

The tag selects the backend at compile time; the resulting binary only links against the chosen display-server libraries (`libwayland-client.so.0` + `libwayland-egl.so.1` instead of `libX11.so.6`).

### Wayland limitations

| Feature | Status |
|---------|--------|
| `GetPos` / `SetPos` | Always (0, 0) — xdg_toplevel has no position request |
| `Hide` | No-op — use `Iconify` or `Destroy` |
| `SetCursorPos` | No-op — clients cannot warp the pointer |
| `RawMouseMotion` | Always false — zwp_relative_pointer not wired yet |
| Custom cursors (`CreateCursor`) | Stub — wl_shm path not implemented |
| Gamma ramps | No-op — not exposed by Wayland |
| Window opacity | No-op — compositor-side only |

---

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
