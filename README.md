# glfw-purego

[![smoke](https://github.com/ClaudioTheobaldo/TheClassicsWithOpenGLPurego/actions/workflows/smoke.yml/badge.svg)](https://github.com/ClaudioTheobaldo/TheClassicsWithOpenGLPurego/actions/workflows/smoke.yml) — Windows ✓ · Linux X11 ✓ · Linux Wayland ✓ · macOS ✓

CGO-less GLFW 3.3 bindings for Go — a drop-in replacement for [`github.com/go-gl/glfw/v3.3/glfw`](https://github.com/go-gl/glfw).

Rather than wrapping the GLFW C library, this package reimplements the GLFW Go API surface directly over native platform APIs — no C compiler required.

- **Windows**: Win32 + WGL via `golang.org/x/sys/windows`
- **macOS**: Cocoa + CGL via `github.com/ebitengine/purego` + ObjC runtime
- **Linux X11**: X11 via `github.com/jezek/xgb` + GLX via purego
- **Linux Wayland**: Wayland + EGL via purego

## Supported platforms

| Platform | Backend | Status |
|----------|---------|--------|
| Windows (amd64, arm64) | Win32 + WGL/EGL | ✅ |
| macOS (amd64, arm64) | Cocoa + NSOpenGLContext | ✅ |
| Linux X11 (amd64, arm64) | XGB + GLX/EGL | ✅ |
| Linux Wayland (amd64, arm64) | xdg-shell + EGL | ✅ |

CI passes on all four platforms (macOS arm64, Linux X11, Linux Wayland, Windows cross-compile).

## Linux build tags

The Linux backend defaults to **X11**.  To build against the Wayland backend instead, pass `-tags wayland` to every `go build`, `go run`, and `go test` invocation:

```sh
go build -tags wayland ./...
go run  -tags wayland .
go test -tags wayland ./...
```

The tag selects the backend at compile time; the resulting binary only links against the chosen display-server libraries (`libwayland-client.so.0` + `libwayland-egl.so.1` instead of `libX11.so.6`).

### Wayland limitations

These reflect protocol-level constraints and won't be fixed:

| Feature | Status |
|---------|--------|
| `GetPos` / `SetPos` | Always (0, 0) — xdg_toplevel has no position request |
| `Hide` | No-op — use `Iconify` or `Destroy` |
| `Focus` | No-op — clients cannot steal focus |
| `SetCursorPos` | No-op — clients cannot warp the pointer |
| Gamma ramps | No-op — not exposed by Wayland |
| Window opacity | No-op — compositor-side only |
| `RawMouseMotion` | Always false — `zwp_relative_pointer` not wired |
| `RequestAttention` | Best-effort — uses `xdg_activation_v1` when advertised, otherwise no-op |

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
replace github.com/go-gl/glfw/v3.3/glfw => github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw v1.3.0
```

## Versioning

Pin to an explicit `v1.x` tag for stability.  `@latest` works but
new features land regularly:

```
go get github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw@v1.3.0
```

Each release is a Git tag of the form `v3.3/glfw/vX.Y.Z` for the
submodule and a parallel `vX.Y.Z` for the parent; both resolve to
the same commit.

## Verification

Exercised on every push by
[TheClassicsWithOpenGLPurego](https://github.com/ClaudioTheobaldo/TheClassicsWithOpenGLPurego) —
**18 consumer programs** (games + diagnostics) running on Windows,
Linux X11, Linux Wayland, and macOS via the linked smoke workflow.

A 2-hour soak test ran **9.36 million iterations** with **3.12 million
cursor create/destroy round-trips**, polling all monitors and joysticks
every frame.  Process-wide GDI handles flat at 17, Go heap oscillated
1–3 MB with steady GC, goroutines stable at 2.  No leaks.

## Bugs caught and fixed by real-consumer testing

- `GetMonitors()` recompiled a Win32 callback every call — would crash
  any 60 FPS consumer after ~30 seconds.  Fixed in v1.0.1.
- `share *Window` was a documented stub on Windows; wired up via
  `wglShareLists` / `wglCreateContextAttribsARB` in v1.1.0.
- Same stub on Linux X11 / Wayland / macOS; wired up via
  `eglCreateContext`'s share argument and Cocoa's
  `initWithFormat:shareContext:` in v1.2.0.
- `SetErrorCallback` was missing entirely; added in v1.3.0 with
  `WindowHint` emitting `InvalidEnum` for unrecognised hints.
- The `OpenGLProfile` hint constant was named `OpenGLProfileHint`;
  source ported from `go-gl/glfw` wouldn't compile.  Both names now
  resolve to the same value.

## Acknowledgements

This repository was built in collaboration with [Claude Code](https://claude.ai/claude-code) (Anthropic Claude Sonnet 4.6).
