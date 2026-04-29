//go:build linux

package glfw

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ----------------------------------------------------------------------------
// EGL sentinel values
// ----------------------------------------------------------------------------

const (
	_EGL_DEFAULT_DISPLAY uintptr = 0
	_EGL_NO_DISPLAY      uintptr = 0
	_EGL_NO_CONTEXT      uintptr = 0
	_EGL_NO_SURFACE      uintptr = 0
)

// ----------------------------------------------------------------------------
// EGL config and context attribute keys / values
// ----------------------------------------------------------------------------

const (
	_EGL_NONE            = int32(0x3038)
	_EGL_ALPHA_SIZE      = int32(0x3021)
	_EGL_BLUE_SIZE       = int32(0x3022)
	_EGL_GREEN_SIZE      = int32(0x3023)
	_EGL_RED_SIZE        = int32(0x3024)
	_EGL_DEPTH_SIZE      = int32(0x3025)
	_EGL_STENCIL_SIZE    = int32(0x3026)
	_EGL_SAMPLES         = int32(0x3031)
	_EGL_SAMPLE_BUFFERS  = int32(0x3032)
	_EGL_SURFACE_TYPE    = int32(0x3033)
	_EGL_RENDERABLE_TYPE = int32(0x3040)

	_EGL_WINDOW_BIT     = int32(0x0004)
	_EGL_OPENGL_ES2_BIT = int32(0x0004)
	_EGL_OPENGL_ES3_BIT = int32(0x0040)
	_EGL_OPENGL_BIT     = int32(0x0008)

	_EGL_CONTEXT_CLIENT_VERSION = int32(0x3098)
	_EGL_CONTEXT_MINOR_VERSION  = int32(0x30FB)

	// eglBindAPI values
	_EGL_OPENGL_ES_API = uintptr(0x30A0)
	_EGL_OPENGL_API    = uintptr(0x30A2)
)

// ----------------------------------------------------------------------------
// EGL function types and pointers — populated by loadEGL()
// ----------------------------------------------------------------------------

var (
	eglGetDisplay          func(displayID uintptr) uintptr
	eglInitialize          func(dpy uintptr, major, minor *int32) bool
	eglChooseConfig        func(dpy, attribs, configs uintptr, configSize int32, numConfig *int32) bool
	eglCreateWindowSurface func(dpy, config, nativeWindow, attribs uintptr) uintptr
	eglCreateContext       func(dpy, config, shareContext, attribs uintptr) uintptr
	eglMakeCurrent         func(dpy, draw, read, ctx uintptr) bool
	eglSwapBuffers         func(dpy, surface uintptr) bool
	eglSwapInterval        func(dpy uintptr, interval int32) bool
	eglDestroyContext      func(dpy, ctx uintptr) bool
	eglDestroySurface      func(dpy, surface uintptr) bool
	eglTerminate           func(dpy uintptr) bool
	eglGetProcAddress      func(name *byte) uintptr
	eglGetCurrentContext   func() uintptr
	eglGetCurrentDisplay   func() uintptr
	eglGetError            func() int32
	eglBindAPI             func(api uintptr) bool
)

var (
	eglLibLoaded      bool
	eglSharedDisplay  uintptr
	currentEGLDisplay uintptr
	libEGLHandle      uintptr
)

// ----------------------------------------------------------------------------
// Library loading
// ----------------------------------------------------------------------------

func loadEGL() error {
	var err error
	for _, name := range []string{"libEGL.so.1", "libEGL.so"} {
		libEGLHandle, err = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}
	if err != nil {
		return &Error{Code: APIUnavailable,
			Desc: "EGL not available (libEGL.so not found): " + err.Error()}
	}

	purego.RegisterLibFunc(&eglGetDisplay, libEGLHandle, "eglGetDisplay")
	purego.RegisterLibFunc(&eglInitialize, libEGLHandle, "eglInitialize")
	purego.RegisterLibFunc(&eglChooseConfig, libEGLHandle, "eglChooseConfig")
	purego.RegisterLibFunc(&eglCreateWindowSurface, libEGLHandle, "eglCreateWindowSurface")
	purego.RegisterLibFunc(&eglCreateContext, libEGLHandle, "eglCreateContext")
	purego.RegisterLibFunc(&eglMakeCurrent, libEGLHandle, "eglMakeCurrent")
	purego.RegisterLibFunc(&eglSwapBuffers, libEGLHandle, "eglSwapBuffers")
	purego.RegisterLibFunc(&eglSwapInterval, libEGLHandle, "eglSwapInterval")
	purego.RegisterLibFunc(&eglDestroyContext, libEGLHandle, "eglDestroyContext")
	purego.RegisterLibFunc(&eglDestroySurface, libEGLHandle, "eglDestroySurface")
	purego.RegisterLibFunc(&eglTerminate, libEGLHandle, "eglTerminate")
	purego.RegisterLibFunc(&eglGetProcAddress, libEGLHandle, "eglGetProcAddress")
	purego.RegisterLibFunc(&eglGetCurrentContext, libEGLHandle, "eglGetCurrentContext")
	purego.RegisterLibFunc(&eglGetCurrentDisplay, libEGLHandle, "eglGetCurrentDisplay")
	purego.RegisterLibFunc(&eglGetError, libEGLHandle, "eglGetError")
	purego.RegisterLibFunc(&eglBindAPI, libEGLHandle, "eglBindAPI")

	eglLibLoaded = true
	return nil
}

// ----------------------------------------------------------------------------
// Hint inspection
// ----------------------------------------------------------------------------

func shouldUseEGL(h map[Hint]int) bool {
	return ClientAPI(h[ClientAPIs]) == OpenGLESAPI ||
		ContextCreationAPI(h[ContextCreationAPIHint]) == EGLContextAPI
}

// ----------------------------------------------------------------------------
// EGL context creation
// ----------------------------------------------------------------------------

func createEGLContext(x11Disp, nativeWindow uintptr, h map[Hint]int) (surface, ctx uintptr, err error) {
	if !eglLibLoaded {
		if err = loadEGL(); err != nil {
			return
		}
	}

	if eglSharedDisplay == 0 {
		displayID := _EGL_DEFAULT_DISPLAY
		if x11Disp != 0 {
			displayID = x11Disp
		}
		dpy := eglGetDisplay(displayID)
		if dpy == _EGL_NO_DISPLAY {
			err = &Error{Code: PlatformError, Desc: "eglGetDisplay returned EGL_NO_DISPLAY"}
			return
		}
		var major, minor int32
		if !eglInitialize(dpy, &major, &minor) {
			code := eglGetError()
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
		eglBindAPI(api)
		eglSharedDisplay = dpy
	}

	cfgAttribs := buildEGLConfigAttribs(h)
	var config uintptr
	var numConfigs int32
	if !eglChooseConfig(eglSharedDisplay,
		uintptr(unsafe.Pointer(&cfgAttribs[0])),
		uintptr(unsafe.Pointer(&config)),
		1, &numConfigs) || numConfigs == 0 {
		code := eglGetError()
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglChooseConfig failed (EGL error 0x%x, numConfigs=%d)", code, numConfigs)}
		return
	}

	surface = eglCreateWindowSurface(eglSharedDisplay, config, nativeWindow, 0)
	if surface == _EGL_NO_SURFACE {
		code := eglGetError()
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglCreateWindowSurface failed (EGL error 0x%x)", code)}
		return
	}

	ctxAttribs := buildEGLContextAttribs(h)
	ctx = eglCreateContext(eglSharedDisplay, config, _EGL_NO_CONTEXT,
		uintptr(unsafe.Pointer(&ctxAttribs[0])))
	if ctx == _EGL_NO_CONTEXT {
		code := eglGetError()
		eglDestroySurface(eglSharedDisplay, surface)
		surface = 0
		err = &Error{Code: PlatformError,
			Desc: fmt.Sprintf("eglCreateContext failed (EGL error 0x%x)", code)}
		return
	}
	return
}

// ----------------------------------------------------------------------------
// Per-window helpers
// ----------------------------------------------------------------------------

func eglMakeCurrentWindow(w *Window) {
	eglMakeCurrent(eglSharedDisplay, w.eglSurface, w.eglSurface, w.eglContext)
	currentEGLDisplay = eglSharedDisplay
}

func eglSwapBuffersWindow(w *Window) {
	eglSwapBuffers(eglSharedDisplay, w.eglSurface)
}

func eglDestroyWindow(w *Window) {
	if eglSharedDisplay == 0 {
		return
	}
	eglMakeCurrent(eglSharedDisplay, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT)
	if w.eglContext != 0 {
		eglDestroyContext(eglSharedDisplay, w.eglContext)
		w.eglContext = 0
	}
	if w.eglSurface != 0 {
		eglDestroySurface(eglSharedDisplay, w.eglSurface)
		w.eglSurface = 0
	}
}

// ----------------------------------------------------------------------------
// GetProcAddress / SwapInterval helpers
// ----------------------------------------------------------------------------

func eglGetProcAddr(name string) unsafe.Pointer {
	if !eglLibLoaded {
		return nil
	}
	b, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil
	}
	addr := eglGetProcAddress(b)
	if addr == 0 {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

func eglSwapIntervalNow(interval int) {
	if eglLibLoaded && currentEGLDisplay != 0 {
		eglSwapInterval(currentEGLDisplay, int32(interval))
	}
}

// ----------------------------------------------------------------------------
// Attribute builders
// ----------------------------------------------------------------------------

func buildEGLConfigAttribs(h map[Hint]int) []int32 {
	redBits := h[RedBits]
	if redBits == 0 {
		redBits = 8
	}
	greenBits := h[GreenBits]
	if greenBits == 0 {
		greenBits = 8
	}
	blueBits := h[BlueBits]
	if blueBits == 0 {
		blueBits = 8
	}
	alphaBits := h[AlphaBits]
	if alphaBits == 0 {
		alphaBits = 8
	}
	depthBits := h[DepthBits]
	if depthBits == 0 {
		depthBits = 24
	}

	renderableType := _EGL_OPENGL_ES2_BIT
	switch {
	case ClientAPI(h[ClientAPIs]) == OpenGLAPI:
		renderableType = _EGL_OPENGL_BIT
	case h[ContextVersionMajor] >= 3:
		renderableType = _EGL_OPENGL_ES3_BIT
	}

	a := []int32{
		_EGL_RED_SIZE, int32(redBits),
		_EGL_GREEN_SIZE, int32(greenBits),
		_EGL_BLUE_SIZE, int32(blueBits),
		_EGL_ALPHA_SIZE, int32(alphaBits),
		_EGL_DEPTH_SIZE, int32(depthBits),
		_EGL_STENCIL_SIZE, int32(h[StencilBits]),
		_EGL_SURFACE_TYPE, _EGL_WINDOW_BIT,
		_EGL_RENDERABLE_TYPE, renderableType,
	}
	if samples := h[Samples]; samples > 0 {
		a = append(a, _EGL_SAMPLE_BUFFERS, 1, _EGL_SAMPLES, int32(samples))
	}
	a = append(a, _EGL_NONE)
	return a
}

func buildEGLContextAttribs(h map[Hint]int) []int32 {
	major := h[ContextVersionMajor]
	minor := h[ContextVersionMinor]
	if major <= 1 {
		major = 2
		minor = 0
	}
	return []int32{
		_EGL_CONTEXT_CLIENT_VERSION, int32(major),
		_EGL_CONTEXT_MINOR_VERSION,  int32(minor),
		_EGL_NONE,
	}
}
