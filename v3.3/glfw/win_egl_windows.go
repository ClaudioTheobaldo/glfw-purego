//go:build windows

package glfw

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ----------------------------------------------------------------------------
// EGL library loading
// ----------------------------------------------------------------------------

// loadEGL loads libEGL.dll (ANGLE) and populates all EGL proc pointers.
// Returns APIUnavailable if the DLL cannot be found — this is expected on
// machines that do not have ANGLE installed.
func loadEGL() error {
	modLibEGL = windows.NewLazyDLL("libEGL.dll")
	if err := modLibEGL.Load(); err != nil {
		return &Error{Code: APIUnavailable,
			Desc: "EGL/ANGLE not available (libEGL.dll not found): " + err.Error()}
	}

	procEGLGetDisplay          = modLibEGL.NewProc("eglGetDisplay")
	procEGLInitialize          = modLibEGL.NewProc("eglInitialize")
	procEGLChooseConfig        = modLibEGL.NewProc("eglChooseConfig")
	procEGLCreateWindowSurface = modLibEGL.NewProc("eglCreateWindowSurface")
	procEGLCreateContext       = modLibEGL.NewProc("eglCreateContext")
	procEGLMakeCurrent         = modLibEGL.NewProc("eglMakeCurrent")
	procEGLSwapBuffers         = modLibEGL.NewProc("eglSwapBuffers")
	procEGLSwapInterval        = modLibEGL.NewProc("eglSwapInterval")
	procEGLDestroyContext      = modLibEGL.NewProc("eglDestroyContext")
	procEGLDestroySurface      = modLibEGL.NewProc("eglDestroySurface")
	procEGLTerminate           = modLibEGL.NewProc("eglTerminate")
	procEGLGetProcAddress      = modLibEGL.NewProc("eglGetProcAddress")
	procEGLGetCurrentContext   = modLibEGL.NewProc("eglGetCurrentContext")
	procEGLGetCurrentDisplay   = modLibEGL.NewProc("eglGetCurrentDisplay")
	procEGLGetError            = modLibEGL.NewProc("eglGetError")
	procEGLBindAPI             = modLibEGL.NewProc("eglBindAPI")

	eglLibLoaded = true
	return nil
}

// ----------------------------------------------------------------------------
// Hint inspection
// ----------------------------------------------------------------------------

// shouldUseEGL reports whether the given hint snapshot requests EGL / GLES
// context creation.  Either an explicit ClientAPI=OpenGLESAPI hint or an
// explicit ContextCreationAPIHint=EGLContextAPI is sufficient.
func shouldUseEGL(h map[Hint]int) bool {
	return ClientAPI(h[ClientAPIs]) == OpenGLESAPI ||
		ContextCreationAPI(h[ContextCreationAPIHint]) == EGLContextAPI
}

// ----------------------------------------------------------------------------
// EGL context creation
// ----------------------------------------------------------------------------

// createEGLContext initialises EGL (first call only), selects a matching
// EGLConfig, creates a window surface for hwnd, and creates an EGLContext.
// Returns (surface, ctx) on success.
func createEGLContext(hwnd uintptr, h map[Hint]int) (surface, ctx uintptr, err error) {
	if !eglLibLoaded {
		if err = loadEGL(); err != nil {
			return
		}
	}

	// Initialise the shared EGLDisplay exactly once per process.
	if eglSharedDisplay == 0 {
		dpy, _, _ := procEGLGetDisplay.Call(_EGL_DEFAULT_DISPLAY)
		if dpy == _EGL_NO_DISPLAY {
			err = &Error{Code: PlatformError, Desc: "eglGetDisplay returned EGL_NO_DISPLAY"}
			return
		}
		var major, minor int32
		r, _, _ := procEGLInitialize.Call(
			dpy,
			uintptr(unsafe.Pointer(&major)),
			uintptr(unsafe.Pointer(&minor)),
		)
		if r == 0 {
			code, _, _ := procEGLGetError.Call()
			err = &Error{Code: PlatformError,
				Desc: fmt.Sprintf("eglInitialize failed (EGL error 0x%x)", code)}
			return
		}
		// Bind the OpenGL ES API — required before eglCreateContext.
		// Without this the current rendering API is EGL_NONE and
		// eglCreateContext returns EGL_BAD_MATCH.
		api := _EGL_OPENGL_ES_API
		if ClientAPI(h[ClientAPIs]) == OpenGLAPI {
			api = _EGL_OPENGL_API
		}
		procEGLBindAPI.Call(api)
		eglSharedDisplay = dpy
	}

	// Select an EGLConfig.
	cfgAttribs := buildEGLConfigAttribs(h)
	var config uintptr
	var numConfigs int32
	r, _, _ := procEGLChooseConfig.Call(
		eglSharedDisplay,
		uintptr(unsafe.Pointer(&cfgAttribs[0])),
		uintptr(unsafe.Pointer(&config)),
		1,
		uintptr(unsafe.Pointer(&numConfigs)),
	)
	if r == 0 || numConfigs == 0 {
		code, _, _ := procEGLGetError.Call()
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglChooseConfig failed (EGL error 0x%x, numConfigs=%d)", code, numConfigs)}
		return
	}

	// Create window surface.
	surface, _, _ = procEGLCreateWindowSurface.Call(eglSharedDisplay, config, hwnd, 0)
	if surface == _EGL_NO_SURFACE {
		code, _, _ := procEGLGetError.Call()
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglCreateWindowSurface failed (EGL error 0x%x)", code)}
		return
	}

	// Create context.
	ctxAttribs := buildEGLContextAttribs(h)
	ctx, _, _ = procEGLCreateContext.Call(
		eglSharedDisplay,
		config,
		_EGL_NO_CONTEXT,
		uintptr(unsafe.Pointer(&ctxAttribs[0])),
	)
	if ctx == _EGL_NO_CONTEXT {
		code, _, _ := procEGLGetError.Call()
		procEGLDestroySurface.Call(eglSharedDisplay, surface)
		surface = 0
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglCreateContext failed (EGL error 0x%x)", code)}
		return
	}
	return
}

// ----------------------------------------------------------------------------
// Per-window EGL helpers (called from win_window_windows.go)
// ----------------------------------------------------------------------------

// eglMakeCurrentWindow makes w's EGL context current on this thread.
func eglMakeCurrentWindow(w *Window) {
	procEGLMakeCurrent.Call(eglSharedDisplay, w.eglSurface, w.eglSurface, w.eglContext)
	currentEGLDisplay = eglSharedDisplay
}

// eglSwapBuffersWindow swaps the front/back buffers of w's EGL surface.
func eglSwapBuffersWindow(w *Window) {
	procEGLSwapBuffers.Call(eglSharedDisplay, w.eglSurface)
}

// eglDestroyWindow releases the EGL context and surface owned by w.
func eglDestroyWindow(w *Window) {
	if eglSharedDisplay == 0 {
		return
	}
	// Detach before destroying to avoid "context is current" errors.
	procEGLMakeCurrent.Call(eglSharedDisplay, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT)
	if w.eglContext != 0 {
		procEGLDestroyContext.Call(eglSharedDisplay, w.eglContext)
		w.eglContext = 0
	}
	if w.eglSurface != 0 {
		procEGLDestroySurface.Call(eglSharedDisplay, w.eglSurface)
		w.eglSurface = 0
	}
}

// ----------------------------------------------------------------------------
// GetProcAddress / SwapInterval helpers
// ----------------------------------------------------------------------------

// eglGetProcAddr resolves a GL/GLES function name via eglGetProcAddress.
// Returns nil if EGL is not loaded or the symbol is not found.
func eglGetProcAddr(name string) unsafe.Pointer {
	if !eglLibLoaded {
		return nil
	}
	b, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil
	}
	r, _, _ := procEGLGetProcAddress.Call(uintptr(unsafe.Pointer(b)))
	if r == 0 {
		return nil
	}
	return nativePtrFromUintptr(r)
}

// eglSwapIntervalNow calls eglSwapInterval for the currently bound display.
func eglSwapIntervalNow(interval int) {
	if eglLibLoaded && currentEGLDisplay != 0 {
		procEGLSwapInterval.Call(currentEGLDisplay, uintptr(interval))
	}
}

// ----------------------------------------------------------------------------
// Attribute array builders
// ----------------------------------------------------------------------------

// buildEGLConfigAttribs returns a null-terminated EGLint attribute list for
// eglChooseConfig, derived from the window hint snapshot.
func buildEGLConfigAttribs(h map[Hint]int) []int32 {
	redBits   := h[RedBits];   if redBits   == 0 { redBits   = 8 }
	greenBits := h[GreenBits]; if greenBits == 0 { greenBits = 8 }
	blueBits  := h[BlueBits];  if blueBits  == 0 { blueBits  = 8 }
	alphaBits := h[AlphaBits]; if alphaBits == 0 { alphaBits = 8 }
	depthBits := h[DepthBits]; if depthBits == 0 { depthBits = 24 }

	// Choose the renderable type bit based on API and requested version.
	renderableType := _EGL_OPENGL_ES2_BIT
	switch {
	case ClientAPI(h[ClientAPIs]) == OpenGLAPI:
		renderableType = _EGL_OPENGL_BIT
	case h[ContextVersionMajor] >= 3:
		renderableType = _EGL_OPENGL_ES3_BIT
	}

	a := []int32{
		_EGL_RED_SIZE,        int32(redBits),
		_EGL_GREEN_SIZE,      int32(greenBits),
		_EGL_BLUE_SIZE,       int32(blueBits),
		_EGL_ALPHA_SIZE,      int32(alphaBits),
		_EGL_DEPTH_SIZE,      int32(depthBits),
		_EGL_STENCIL_SIZE,    int32(h[StencilBits]),
		_EGL_SURFACE_TYPE,    _EGL_WINDOW_BIT,
		_EGL_RENDERABLE_TYPE, renderableType,
	}
	if samples := h[Samples]; samples > 0 {
		a = append(a, _EGL_SAMPLE_BUFFERS, 1, _EGL_SAMPLES, int32(samples))
	}
	a = append(a, _EGL_NONE)
	return a
}

// buildEGLContextAttribs returns a null-terminated EGLint attribute list for
// eglCreateContext, derived from the window hint snapshot.
func buildEGLContextAttribs(h map[Hint]int) []int32 {
	major := h[ContextVersionMajor]
	minor := h[ContextVersionMinor]
	// GLES 1.x is legacy; if the caller left the version at the GL default (1)
	// or unset (0), upgrade to GLES 2.0 as a sane baseline.
	if major <= 1 {
		major = 2
		minor = 0
	}
	// Always emit EGL_CONTEXT_MINOR_VERSION so the driver creates the exact
	// GLES version the caller requested.  Without it ANGLE defaults to minor=0
	// (i.e. GLES 3.0) even when 3.1 or 3.2 are available.
	// EGL_CONTEXT_MINOR_VERSION is valid in EGL 1.5 and via
	// EGL_KHR_create_context, both of which ANGLE supports.
	return []int32{
		_EGL_CONTEXT_CLIENT_VERSION, int32(major),
		_EGL_CONTEXT_MINOR_VERSION,  int32(minor),
		_EGL_NONE,
	}
}
