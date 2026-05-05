//go:build linux && wayland

package glfw

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

// ── libwayland-cursor loader ──────────────────────────────────────────────────

var (
	wlCursorLibOnce     sync.Once
	wlCursorLibHandle   uintptr
	wlCursorLibErr      error

	wlCursorThemeLoad    func(name uintptr, size int32, shm uintptr) uintptr
	wlCursorThemeDestroy func(theme uintptr)
	wlCursorThemeGet     func(theme, name uintptr) uintptr
	wlCursorImageGetBuf  func(image uintptr) uintptr

	// Runtime state — valid only after Init().
	wlCursorThemePtr uintptr // wl_cursor_theme*
	wlCursorSurf     uintptr // shared wl_surface* for cursor rendering
)

func loadWaylandCursorLib() error {
	wlCursorLibOnce.Do(func() {
		for _, lib := range []string{"libwayland-cursor.so.0", "libwayland-cursor.so"} {
			wlCursorLibHandle, wlCursorLibErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if wlCursorLibErr == nil {
				break
			}
		}
		if wlCursorLibErr != nil {
			return
		}
		purego.RegisterLibFunc(&wlCursorThemeLoad, wlCursorLibHandle, "wl_cursor_theme_load")
		purego.RegisterLibFunc(&wlCursorThemeDestroy, wlCursorLibHandle, "wl_cursor_theme_destroy")
		purego.RegisterLibFunc(&wlCursorThemeGet, wlCursorLibHandle, "wl_cursor_theme_get_cursor")
		purego.RegisterLibFunc(&wlCursorImageGetBuf, wlCursorLibHandle, "wl_cursor_image_get_buffer")
	})
	return wlCursorLibErr
}

// wlEnsureCursorTheme returns the default cursor theme, loading it on first call.
// Returns 0 if libwayland-cursor is unavailable or wl_shm has not been bound.
func wlEnsureCursorTheme() uintptr {
	if wlCursorThemePtr != 0 {
		return wlCursorThemePtr
	}
	if loadWaylandCursorLib() != nil || wl.shm == 0 {
		return 0
	}
	// NULL name → use $XCURSOR_THEME or the compositor default; 24 = pixel size.
	wlCursorThemePtr = wlCursorThemeLoad(0, 24, wl.shm)
	return wlCursorThemePtr
}

// wlEnsureCursorSurface returns a persistent wl_surface used as the cursor image.
func wlEnsureCursorSurface() uintptr {
	if wlCursorSurf != 0 {
		return wlCursorSurf
	}
	if wl.compositor == 0 {
		return 0
	}
	wlCursorSurf = wlProxyMarshalFlags(wl.compositor, 0, wlSurfaceIfaceAddr, 4, 0, 0)
	return wlCursorSurf
}

// ── wl_cursor / wl_cursor_image struct accessors ──────────────────────────────
//
// wl_cursor layout (64-bit):
//   offset 0:  image_count  uint32
//   offset 4:  (4 bytes padding)
//   offset 8:  images       **wl_cursor_image
//   offset 16: name         *char
//
// wl_cursor_image layout:
//   offset 0:  width     uint32
//   offset 4:  height    uint32
//   offset 8:  hotspot_x uint32
//   offset 12: hotspot_y uint32
//   offset 16: delay     uint32

// wlCursorFirstFrame extracts the first frame's wl_cursor_image* and hotspot
// from a wl_cursor*.  Returns (0,0,0) on any failure.
func wlCursorFirstFrame(cursor uintptr) (image uintptr, hotX, hotY uint32) {
	if cursor == 0 {
		return
	}
	count := *(*uint32)(nativePtrFromUintptr(cursor))
	if count == 0 {
		return
	}
	imagesPtr := *(*uintptr)(unsafe.Add(nativePtrFromUintptr(cursor), 8))
	if imagesPtr == 0 {
		return
	}
	image = *(*uintptr)(nativePtrFromUintptr(imagesPtr))
	if image == 0 {
		return
	}
	hotX = *(*uint32)(unsafe.Add(nativePtrFromUintptr(image), 8))
	hotY = *(*uint32)(unsafe.Add(nativePtrFromUintptr(image), 12))
	return
}

// ── Standard cursor name table ────────────────────────────────────────────────

// wlCursorNames maps GLFW cursor shapes to ordered lists of XCursor names
// (tried in sequence; first match wins).
var wlCursorNames = map[StandardCursorShape][]string{
	ArrowCursor:        {"default", "left_ptr"},
	IBeamCursor:        {"text", "xterm"},
	CrosshairCursor:    {"crosshair", "cross"},
	HandCursor:         {"pointer", "hand2"},
	HResizeCursor:      {"ew-resize", "col-resize", "size_hor"},
	VResizeCursor:      {"ns-resize", "row-resize", "size_ver"},
	ResizeEWCursor:     {"ew-resize"},
	ResizeNSCursor:     {"ns-resize"},
	ResizeNWSECursor:   {"nwse-resize"},
	ResizeNESWCursor:   {"nesw-resize"},
	ResizeAllCursor:    {"all-scroll", "fleur"},
	PointingHandCursor: {"pointing_hand", "pointer"},
	NotAllowedCursor:   {"not-allowed", "forbidden"},
}

// ── CreateStandardCursor ──────────────────────────────────────────────────────

// CreateStandardCursor loads a system cursor from the XCursor theme via
// libwayland-cursor.  Falls back to a stub cursor if the library or theme is
// unavailable (the build still works; the cursor just won't change shape).
func CreateStandardCursor(shape StandardCursorShape) (*Cursor, error) {
	names, ok := wlCursorNames[shape]
	if !ok {
		return &Cursor{system: true}, nil
	}
	theme := wlEnsureCursorTheme()
	if theme == 0 {
		return &Cursor{system: true}, nil
	}
	for _, name := range names {
		nameC := append([]byte(name), 0)
		cur := wlCursorThemeGet(theme, uintptr(unsafe.Pointer(&nameC[0])))
		runtime.KeepAlive(nameC)
		if cur == 0 {
			continue
		}
		img, hotX, hotY := wlCursorFirstFrame(cur)
		if img == 0 {
			continue
		}
		return &Cursor{
			handle: img, // wl_cursor_image* — lifetime = cursor theme
			system: true,
			wlHotX: int32(hotX),
			wlHotY: int32(hotY),
		}, nil
	}
	return &Cursor{system: true}, nil
}

// ── CreateCursor (custom RGBA image) ─────────────────────────────────────────

// CreateCursor creates a cursor from an arbitrary RGBA image, backed by a
// shared-memory wl_buffer.  The buffer stays mapped for the cursor's lifetime
// and is released by DestroyCursor.  Returns an empty (invisible) cursor if
// wl_shm is unavailable or the underlying memfd/mmap calls fail.
func CreateCursor(image *Image, xhot, yhot int) (*Cursor, error) {
	if image == nil || image.Width <= 0 || image.Height <= 0 ||
		wl.shm == 0 || wlShmPoolIfaceAddr == 0 || wlBufferIfaceAddr == 0 {
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}
	w, h := image.Width, image.Height
	stride := w * 4
	size := stride * h

	// 1. Anonymous shared-memory file via memfd_create (Linux 3.17+).
	fd, err := unix.MemfdCreate("glfw-cursor", unix.MFD_CLOEXEC)
	if err != nil {
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}
	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		unix.Close(fd)
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}
	mmap, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		unix.Close(fd)
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}

	// 2. Convert input RGBA → ARGB8888 little-endian (B,G,R,A bytes) with
	//    pre-multiplied alpha, which is what WL_SHM_FORMAT_ARGB8888 expects.
	src := image.Pixels
	for i := 0; i < w*h; i++ {
		r := uint32(src[i*4+0])
		g := uint32(src[i*4+1])
		b := uint32(src[i*4+2])
		a := uint32(src[i*4+3])
		// Premultiply.
		r = r * a / 255
		g = g * a / 255
		b = b * a / 255
		// Little-endian byte order: [B, G, R, A].
		mmap[i*4+0] = byte(b)
		mmap[i*4+1] = byte(g)
		mmap[i*4+2] = byte(r)
		mmap[i*4+3] = byte(a)
	}

	// 3. wl_shm.create_pool opcode=0, signature="nhi" (new_id, fd, size).
	pool := wlProxyMarshalFlags(wl.shm, 0, wlShmPoolIfaceAddr, 1, 0,
		uintptr(0), uintptr(fd), uintptr(size))
	if pool == 0 {
		_ = unix.Munmap(mmap)
		unix.Close(fd)
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}

	// 4. wl_shm_pool.create_buffer opcode=0,
	//    signature="niiiiu" (new_id, offset, w, h, stride, format).
	const _WL_SHM_FORMAT_ARGB8888 = uint32(0)
	buf := wlProxyMarshalFlags(pool, 0, wlBufferIfaceAddr, 1, 0,
		uintptr(0), uintptr(0), uintptr(w), uintptr(h),
		uintptr(stride), uintptr(_WL_SHM_FORMAT_ARGB8888))

	// 5. Pool is no longer needed once the buffer is created.
	wlProxyMarshalFlags(pool, 1, 0, 1, 1) // wl_shm_pool.destroy + free
	unix.Close(fd)                          // pool/buffer keep the mapping live

	if buf == 0 {
		_ = unix.Munmap(mmap)
		return &Cursor{wlHotX: int32(xhot), wlHotY: int32(yhot)}, nil
	}

	return &Cursor{
		handle:    buf, // wl_buffer*
		system:    false,
		wlHotX:    int32(xhot),
		wlHotY:    int32(yhot),
		wlMmap:    mmap,
		wlSurface: 0, // shared cursor surface is reused via wlEnsureCursorSurface
	}, nil
}

// ── DestroyCursor ─────────────────────────────────────────────────────────────

// DestroyCursor frees a cursor created by CreateCursor.
// Cursors created by CreateStandardCursor are owned by the cursor theme and
// are not freed individually.
func DestroyCursor(c *Cursor) {
	if c == nil || c.system || c.handle == 0 {
		return
	}
	// Custom cursor: the handle is a wl_buffer* we allocated via wl_shm.
	// wl_buffer.destroy opcode=0.
	wlProxyMarshalFlags(c.handle, 0, 0, 1, 1) // destroy + free
	c.handle = 0
	if c.wlMmap != nil {
		_ = unix.Munmap(c.wlMmap)
		c.wlMmap = nil
	}
}

// ── SetCursor ─────────────────────────────────────────────────────────────────

// SetCursor sets the cursor shape for the window.
// Passing nil hides the cursor (equivalent to CursorHidden mode).
func (w *Window) SetCursor(c *Cursor) {
	wlApplyCursor(c)
}

// wlApplyCursor updates wl_pointer with the given cursor (nil = hide).
// Called from both SetCursor and SetInputMode(CursorMode, ...).
func wlApplyCursor(c *Cursor) {
	if wl.pointer == 0 {
		return
	}
	if c == nil || c.handle == 0 {
		// wl_pointer.set_cursor opcode=0 with surface=NULL → hide cursor.
		wlProxyMarshalFlags(wl.pointer, 0, 0, 1, 0,
			uintptr(wl.ptrSerial), 0, 0, 0)
		wlDisplayFlush(wl.display)
		return
	}

	surf := wlEnsureCursorSurface()
	if surf == 0 {
		return
	}

	// For system cursors, c.handle is a wl_cursor_image*; resolve it to a
	// wl_buffer*.  For custom cursors created via CreateCursor, c.handle is
	// already a wl_buffer*.
	var buf uintptr
	if c.system {
		buf = wlCursorImageGetBuf(c.handle)
	} else {
		buf = c.handle
	}
	if buf == 0 {
		return
	}

	// Attach the buffer to the cursor surface and commit.
	// wl_surface.attach opcode=1, signature="?oii" (buffer, dx, dy)
	wlProxyMarshalFlags(surf, 1, 0, 4, 0, buf, 0, 0)
	// wl_surface.commit opcode=6
	wlProxyMarshalFlags(surf, 6, 0, 4, 0)
	// wl_pointer.set_cursor opcode=0, signature="u?oii" (serial, surface, hx, hy)
	wlProxyMarshalFlags(wl.pointer, 0, 0, 1, 0,
		uintptr(wl.ptrSerial),
		surf,
		uintptr(uint32(c.wlHotX)),
		uintptr(uint32(c.wlHotY)))
	wlDisplayFlush(wl.display)
}
