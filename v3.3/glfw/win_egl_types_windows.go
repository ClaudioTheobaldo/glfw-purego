//go:build windows

package glfw

import "golang.org/x/sys/windows"

// ----------------------------------------------------------------------------
// EGL sentinel values (all zero in the EGL spec).
// ----------------------------------------------------------------------------

const (
	_EGL_DEFAULT_DISPLAY uintptr = 0
	_EGL_NO_DISPLAY      uintptr = 0
	_EGL_NO_CONTEXT      uintptr = 0
	_EGL_NO_SURFACE      uintptr = 0
)

// ----------------------------------------------------------------------------
// EGL config and context attribute keys / values.
// ----------------------------------------------------------------------------

const (
	_EGL_NONE            = int32(0x3038)
	_EGL_BUFFER_SIZE     = int32(0x3020)
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

	// Surface type bits
	_EGL_WINDOW_BIT  = int32(0x0004)
	_EGL_PBUFFER_BIT = int32(0x0001)

	// Renderable type bits
	_EGL_OPENGL_ES_BIT  = int32(0x0001)
	_EGL_OPENGL_ES2_BIT = int32(0x0004)
	_EGL_OPENGL_ES3_BIT = int32(0x0040) // EGL_KHR_create_context / EGL 1.5
	_EGL_OPENGL_BIT     = int32(0x0008)

	// Context creation attributes
	_EGL_CONTEXT_CLIENT_VERSION = int32(0x3098) // EGL 1.2 name; == EGL_CONTEXT_MAJOR_VERSION in EGL 1.5
	_EGL_CONTEXT_MINOR_VERSION  = int32(0x30FB) // EGL 1.5

	// eglBindAPI values
	_EGL_OPENGL_ES_API = uintptr(0x30A0)
	_EGL_OPENGL_API    = uintptr(0x30A2)
)

// ----------------------------------------------------------------------------
// libEGL.dll handle and proc pointers — populated by loadEGL().
// ----------------------------------------------------------------------------

var (
	modLibEGL *windows.LazyDLL

	procEGLGetDisplay          *windows.LazyProc
	procEGLInitialize          *windows.LazyProc
	procEGLChooseConfig        *windows.LazyProc
	procEGLCreateWindowSurface *windows.LazyProc
	procEGLCreateContext       *windows.LazyProc
	procEGLMakeCurrent         *windows.LazyProc
	procEGLSwapBuffers         *windows.LazyProc
	procEGLSwapInterval        *windows.LazyProc
	procEGLDestroyContext      *windows.LazyProc
	procEGLDestroySurface      *windows.LazyProc
	procEGLTerminate           *windows.LazyProc
	procEGLGetProcAddress      *windows.LazyProc
	procEGLGetCurrentContext   *windows.LazyProc
	procEGLGetCurrentDisplay   *windows.LazyProc
	procEGLGetError            *windows.LazyProc
	procEGLBindAPI             *windows.LazyProc
)

// eglLibLoaded is true after loadEGL() has succeeded.
var eglLibLoaded bool

// eglSharedDisplay is the EGLDisplay handle shared across all EGL windows in
// this process.  Set once on the first EGL window creation, never cleared.
var eglSharedDisplay uintptr

// currentEGLDisplay is the EGLDisplay associated with the context that is
// currently current on this thread.  Used by SwapInterval so it can call
// eglSwapInterval with the right display.
var currentEGLDisplay uintptr
