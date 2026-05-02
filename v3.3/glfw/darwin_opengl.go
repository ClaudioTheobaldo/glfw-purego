//go:build darwin

// darwin_opengl.go — macOS OpenGL context creation via NSOpenGLContext.
//
// Uses the (deprecated-but-still-functional) NSOpenGLContext/NSOpenGLPixelFormat
// API, which GLFW 3.4 itself uses and which remains available through macOS 15.
// Metal migration is left for a future phase.

package glfw

import (
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ── NSOpenGLPixelFormatAttribute constants ─────────────────────────────────────
// Source: NSOpenGL.h / OpenGL/CGLTypes.h

const (
	nsoglPFADoubleBuffer  uint32 = 5
	nsoglPFAColorSize     uint32 = 8
	nsoglPFAAlphaSize     uint32 = 11
	nsoglPFADepthSize     uint32 = 12
	nsoglPFAStencilSize   uint32 = 13
	nsoglPFANoRecovery    uint32 = 72
	nsoglPFAAccelerated   uint32 = 73
	nsoglPFASampleBuffers uint32 = 55
	nsoglPFASamples       uint32 = 56
	nsoglPFAOpenGLProfile uint32 = 99

	nsoglProfileVersionLegacy   uint32 = 0x1000 // NSOpenGLProfileVersionLegacy
	nsoglProfileVersion3_2Core  uint32 = 0x3200 // NSOpenGLProfileVersion3_2Core
	nsoglProfileVersion4_1Core  uint32 = 0x4100 // NSOpenGLProfileVersion4_1Core
)

// NSOpenGLContextParameter value for swap interval.
// Deprecated in 10.14 but functional through macOS 15.
const nsoglCPSwapInterval int64 = 222 // NSOpenGLCPSwapInterval

// ── SEL cache (OpenGL-specific) ───────────────────────────────────────────────

var (
	selNSGLInitWithAttributes    = objc.RegisterName("initWithAttributes:")
	selNSGLInitWithFormatShare   = objc.RegisterName("initWithFormat:shareContext:")
	selNSGLSetView               = objc.RegisterName("setView:")
	selNSGLUpdate                = objc.RegisterName("update")
	selNSGLMakeCurrentContext    = objc.RegisterName("makeCurrentContext")
	selNSGLClearCurrentContext   = objc.RegisterName("clearCurrentContext")
	selNSGLFlushBuffer           = objc.RegisterName("flushBuffer")
	selNSGLSetValuesForParameter = objc.RegisterName("setValues:forParameter:")
)

// ── OpenGL.framework handle ───────────────────────────────────────────────────

// openGLLib is the handle to OpenGL.framework, loaded lazily on first
// GetProcAddress call. Protected only by single-threaded Cocoa main thread.
var openGLLib uintptr

// loadOpenGLFramework loads the OpenGL framework if not already loaded.
// Must be called from the Cocoa main thread.
func loadOpenGLFramework() error {
	if openGLLib != 0 {
		return nil
	}
	h, err := purego.Dlopen(
		"/System/Library/Frameworks/OpenGL.framework/OpenGL",
		purego.RTLD_LAZY|purego.RTLD_GLOBAL,
	)
	if err != nil {
		return err
	}
	openGLLib = h
	return nil
}

// darwinGetProcAddress returns the address of the named OpenGL symbol, or nil.
func darwinGetProcAddress(name string) unsafe.Pointer {
	if err := loadOpenGLFramework(); err != nil {
		return nil
	}
	addr, err := purego.Dlsym(openGLLib, name)
	if err != nil || addr == 0 {
		return nil
	}
	return nativePtrFromUintptr(addr)
}

// ── NSOpenGLPixelFormat construction ─────────────────────────────────────────

// buildNSGLPixelFormat creates an NSOpenGLPixelFormat from the given hint map.
// Returns 0 if the requested format is not supported by the hardware.
func buildNSGLPixelFormat(h map[Hint]int) objc.ID {
	var attrs []uint32

	// Double-buffered rendering (almost always desired).
	if h[DoubleBuffer] != 0 {
		attrs = append(attrs, nsoglPFADoubleBuffer)
	}

	// Prefer hardware acceleration; disable software fallback.
	attrs = append(attrs, nsoglPFAAccelerated)
	attrs = append(attrs, nsoglPFANoRecovery)

	// OpenGL profile — driven by ContextVersionMajor hint.
	switch major := h[ContextVersionMajor]; {
	case major >= 4:
		attrs = append(attrs, nsoglPFAOpenGLProfile, nsoglProfileVersion4_1Core)
	case major == 3:
		attrs = append(attrs, nsoglPFAOpenGLProfile, nsoglProfileVersion3_2Core)
	default:
		attrs = append(attrs, nsoglPFAOpenGLProfile, nsoglProfileVersionLegacy)
	}

	// Color, alpha, depth, stencil bit depths.
	if cb := h[RedBits] + h[GreenBits] + h[BlueBits]; cb > 0 {
		attrs = append(attrs, nsoglPFAColorSize, uint32(cb))
	}
	if h[AlphaBits] > 0 {
		attrs = append(attrs, nsoglPFAAlphaSize, uint32(h[AlphaBits]))
	}
	if h[DepthBits] > 0 {
		attrs = append(attrs, nsoglPFADepthSize, uint32(h[DepthBits]))
	}
	if h[StencilBits] > 0 {
		attrs = append(attrs, nsoglPFAStencilSize, uint32(h[StencilBits]))
	}

	// Multisampling.
	if h[Samples] > 0 {
		attrs = append(attrs, nsoglPFASampleBuffers, 1, nsoglPFASamples, uint32(h[Samples]))
	}

	// Null terminator required by NSOpenGLPixelFormat.
	attrs = append(attrs, 0)

	pf := objc.ID(objc.GetClass("NSOpenGLPixelFormat")).Send(selAlloc).Send(
		selNSGLInitWithAttributes, uintptr(unsafe.Pointer(&attrs[0])))
	runtime.KeepAlive(attrs)
	return pf
}

// ── NSOpenGLContext creation ──────────────────────────────────────────────────

// createNSGLContext creates an NSOpenGLContext for the given NSView content view.
// Returns 0 if context creation fails (e.g. on a headless CI runner without GPU).
func createNSGLContext(h map[Hint]int, contentView objc.ID) objc.ID {
	pf := buildNSGLPixelFormat(h)
	if pf == 0 {
		return 0
	}
	defer pf.Send(selRelease)

	ctx := objc.ID(objc.GetClass("NSOpenGLContext")).Send(selAlloc).Send(
		selNSGLInitWithFormatShare, pf, objc.ID(0) /* no share */)
	if ctx == 0 {
		return 0
	}

	// Associate the context with the window's content view.
	ctx.Send(selNSGLSetView, contentView)
	// Synchronise the context with the current view geometry.
	ctx.Send(selNSGLUpdate)

	return ctx
}

// ── SwapInterval ──────────────────────────────────────────────────────────────

// darwinSwapInterval sets the swap interval on the given NSOpenGLContext.
func darwinSwapInterval(ctx objc.ID, interval int) {
	val := int32(interval)
	ctx.Send(selNSGLSetValuesForParameter,
		uintptr(unsafe.Pointer(&val)),
		nsoglCPSwapInterval,
	)
	runtime.KeepAlive(val)
}
