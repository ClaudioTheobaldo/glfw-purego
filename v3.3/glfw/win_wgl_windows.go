//go:build windows

package glfw

import (
	"syscall"
	"unsafe"
)

// ----------------------------------------------------------------------------
// WGL extension function pointers — loaded during dummy context phase.
// ----------------------------------------------------------------------------

var (
	wglCreateContextAttribsARB uintptr
	wglChoosePixelFormatARB    uintptr
	wglSwapIntervalEXT         uintptr
	wglGetSwapIntervalEXT      uintptr
)

// wglExtLoaded is true once the extension loading phase has run.
var wglExtLoaded bool

// ----------------------------------------------------------------------------
// Public API
// ----------------------------------------------------------------------------

// GetProcAddress returns the address of the named OpenGL function.
// Must be called while a GL context is current.
func GetProcAddress(name string) uintptr {
	addr := wglGetProcAddressStr(name)
	if addr != 0 {
		return addr
	}
	// Fallback: base GL 1.1 functions live in opengl32.dll itself.
	return getProcAddressFromDLL(modOpenGL32.Handle(), name)
}

// SwapInterval sets the minimum number of video frame periods per buffer swap.
func SwapInterval(interval int) {
	if wglSwapIntervalEXT != 0 {
		syscall.SyscallN(wglSwapIntervalEXT, uintptr(interval))
	}
}

// ExtensionSupported reports whether the named WGL or GL extension is available.
func ExtensionSupported(extension string) bool {
	// For simplicity, report false until we need full extension string parsing.
	// This is sufficient for Fyne's usage.
	return false
}

// ----------------------------------------------------------------------------
// Context creation
// ----------------------------------------------------------------------------

// createWGLContext performs the two-phase WGL bootstrap:
//  1. Dummy invisible window → legacy context → load extension procs
//  2. Real window DC → wglChoosePixelFormatARB or legacy → wglCreateContextAttribsARB
func createWGLContext(dc uintptr, h map[Hint]int) (uintptr, error) {
	if !wglExtLoaded {
		if err := loadWGLExtensions(); err != nil {
			return 0, err
		}
	}

	// --- Phase 2: choose pixel format for the real DC ---
	format, err := chooseFormat(dc, h)
	if err != nil {
		return 0, err
	}

	var pfd _PIXELFORMATDESCRIPTOR
	pfd.NSize = uint16(unsafe.Sizeof(pfd))
	describePixelFormat(dc, uint32(format), uint32(unsafe.Sizeof(pfd)), &pfd)

	if err := setPixelFormat(dc, format, &pfd); err != nil {
		return 0, &Error{Code: PlatformError, Desc: err.Error()}
	}

	// --- Phase 2: create real rendering context ---
	if wglCreateContextAttribsARB != 0 {
		attribs := buildContextAttribs(h)
		hglrc, _, _ := syscall.SyscallN(wglCreateContextAttribsARB,
			dc, 0, uintptr(unsafe.Pointer(&attribs[0])))
		if hglrc == 0 {
			// Driver rejected the attrib list — fall back to legacy context.
			return wglLegacyContext(dc)
		}
		return hglrc, nil
	}
	return wglLegacyContext(dc)
}

// loadWGLExtensions creates a throwaway 1×1 window, makes a legacy GL context
// current on it, loads the ARB extension proc addresses, then tears it all down.
func loadWGLExtensions() error {
	className, _ := syscall.UTF16PtrFromString(_wndClassName)
	dummyHWND, err := createWindowExW(0, _WS_POPUP, className, nil,
		0, 0, 1, 1, 0, 0, gHInstance)
	if err != nil {
		return &Error{Code: PlatformError, Desc: "dummy window: " + err.Error()}
	}
	defer destroyWindow(dummyHWND)

	dummyDC, err := getDC(dummyHWND)
	if err != nil {
		return &Error{Code: PlatformError, Desc: "dummy DC: " + err.Error()}
	}
	defer releaseDC(dummyHWND, dummyDC)

	pfd := basicPFD()
	fmt := choosePixelFormat(dummyDC, &pfd)
	if fmt == 0 {
		return &Error{Code: PlatformError, Desc: "dummy ChoosePixelFormat returned 0"}
	}
	if err := setPixelFormat(dummyDC, fmt, &pfd); err != nil {
		return &Error{Code: PlatformError, Desc: err.Error()}
	}

	dummyRC, err := wglCreateContext(dummyDC)
	if err != nil {
		return &Error{Code: PlatformError, Desc: err.Error()}
	}
	if err := wglMakeCurrent(dummyDC, dummyRC); err != nil {
		wglDeleteContext(dummyRC)
		return &Error{Code: PlatformError, Desc: err.Error()}
	}

	// Load extension function pointers.
	wglCreateContextAttribsARB = wglGetProcAddressStr("wglCreateContextAttribsARB")
	wglChoosePixelFormatARB    = wglGetProcAddressStr("wglChoosePixelFormatARB")
	wglSwapIntervalEXT         = wglGetProcAddressStr("wglSwapIntervalEXT")
	wglGetSwapIntervalEXT      = wglGetProcAddressStr("wglGetSwapIntervalEXT")

	wglMakeCurrent(0, 0)
	wglDeleteContext(dummyRC)
	wglExtLoaded = true
	return nil
}

// chooseFormat selects the best pixel format for the given DC.
// Uses wglChoosePixelFormatARB when available, otherwise falls back to
// the legacy ChoosePixelFormat.
func chooseFormat(dc uintptr, h map[Hint]int) (int32, error) {
	if wglChoosePixelFormatARB != 0 {
		attribs := buildPixelFormatAttribs(h)
		var pf int32
		var numFormats uint32
		r, _, _ := syscall.SyscallN(wglChoosePixelFormatARB,
			dc,
			uintptr(unsafe.Pointer(&attribs[0])),
			0,
			1,
			uintptr(unsafe.Pointer(&pf)),
			uintptr(unsafe.Pointer(&numFormats)),
		)
		if r != 0 && numFormats > 0 {
			return pf, nil
		}
	}
	// Legacy fallback.
	pfd := basicPFD()
	redBits   := h[RedBits];   if redBits   == 0 { redBits   = 8 }
	greenBits := h[GreenBits]; if greenBits == 0 { greenBits = 8 }
	blueBits  := h[BlueBits];  if blueBits  == 0 { blueBits  = 8 }
	alphaBits := h[AlphaBits]; if alphaBits == 0 { alphaBits = 8 }
	depthBits := h[DepthBits]; if depthBits == 0 { depthBits = 24 }
	pfd.CColorBits  = byte(redBits + greenBits + blueBits)
	pfd.CAlphaBits  = byte(alphaBits)
	pfd.CDepthBits  = byte(depthBits)
	pfd.CStencilBits = byte(h[StencilBits])
	fmt := choosePixelFormat(dc, &pfd)
	if fmt == 0 {
		return 0, &Error{Code: PlatformError, Desc: "ChoosePixelFormat returned 0"}
	}
	return fmt, nil
}

// basicPFD returns a sane default PIXELFORMATDESCRIPTOR for the dummy window.
func basicPFD() _PIXELFORMATDESCRIPTOR {
	return _PIXELFORMATDESCRIPTOR{
		NSize:        uint16(unsafe.Sizeof(_PIXELFORMATDESCRIPTOR{})),
		NVersion:     1,
		DwFlags:      _PFD_DRAW_TO_WINDOW | _PFD_SUPPORT_OPENGL | _PFD_DOUBLEBUFFER,
		IPixelType:   _PFD_TYPE_RGBA,
		CColorBits:   32,
		CDepthBits:   24,
		CStencilBits: 8,
		ILayerType:   _PFD_MAIN_PLANE,
	}
}

// buildContextAttribs builds the attrib int32 array for wglCreateContextAttribsARB.
func buildContextAttribs(h map[Hint]int) []int32 {
	var a []int32

	major := h[ContextVersionMajor]
	minor := h[ContextVersionMinor]
	if major == 0 {
		major = 1
	}
	a = append(a, _WGL_CONTEXT_MAJOR_VERSION_ARB, int32(major))
	a = append(a, _WGL_CONTEXT_MINOR_VERSION_ARB, int32(minor))

	flags := int32(0)
	if h[OpenGLForwardCompatible] != 0 {
		flags |= _WGL_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB
	}
	if h[OpenGLDebugContext] != 0 {
		flags |= _WGL_CONTEXT_DEBUG_BIT_ARB
	}
	if flags != 0 {
		a = append(a, _WGL_CONTEXT_FLAGS_ARB, flags)
	}

	switch OpenGLProfile(h[OpenGLProfileHint]) {
	case CoreProfile:
		a = append(a, _WGL_CONTEXT_PROFILE_MASK_ARB, _WGL_CONTEXT_CORE_PROFILE_BIT_ARB)
	case CompatibilityProfile:
		a = append(a, _WGL_CONTEXT_PROFILE_MASK_ARB, _WGL_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB)
	}

	a = append(a, 0) // terminate
	return a
}

// buildPixelFormatAttribs builds the attrib array for wglChoosePixelFormatARB.
func buildPixelFormatAttribs(h map[Hint]int) []int32 {
	redBits   := h[RedBits];   if redBits   == 0 { redBits   = 8 }
	greenBits := h[GreenBits]; if greenBits == 0 { greenBits = 8 }
	blueBits  := h[BlueBits];  if blueBits  == 0 { blueBits  = 8 }
	alphaBits := h[AlphaBits]; if alphaBits == 0 { alphaBits = 8 }
	depthBits := h[DepthBits]; if depthBits == 0 { depthBits = 24 }

	a := []int32{
		_WGL_DRAW_TO_WINDOW_ARB, 1,
		_WGL_SUPPORT_OPENGL_ARB, 1,
		_WGL_ACCELERATION_ARB,   _WGL_FULL_ACCELERATION_ARB,
		_WGL_PIXEL_TYPE_ARB,     _WGL_TYPE_RGBA_ARB,
		_WGL_RED_BITS_ARB,       int32(redBits),
		_WGL_GREEN_BITS_ARB,     int32(greenBits),
		_WGL_BLUE_BITS_ARB,      int32(blueBits),
		_WGL_ALPHA_BITS_ARB,     int32(alphaBits),
		_WGL_DEPTH_BITS_ARB,     int32(depthBits),
		_WGL_STENCIL_BITS_ARB,   int32(h[StencilBits]),
	}

	if h[DoubleBuffer] != 0 {
		a = append(a, _WGL_DOUBLE_BUFFER_ARB, 1)
	}
	if samples := h[Samples]; samples > 0 {
		a = append(a,
			_WGL_SAMPLE_BUFFERS_ARB, 1,
			_WGL_SAMPLES_ARB, int32(samples),
		)
	}
	if h[SRGBCapable] != 0 {
		a = append(a, _WGL_FRAMEBUFFER_SRGB_CAPABLE_ARB, 1)
	}

	a = append(a, 0) // terminate
	return a
}

// wglLegacyContext creates a plain legacy OpenGL context (no attrib list).
func wglLegacyContext(dc uintptr) (uintptr, error) {
	hglrc, err := wglCreateContext(dc)
	if err != nil {
		return 0, &Error{Code: PlatformError, Desc: err.Error()}
	}
	return hglrc, nil
}
