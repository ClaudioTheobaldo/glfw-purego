//go:build linux && wayland

package glfw

import "unsafe"

// ── C layout types ────────────────────────────────────────────────────────────

// cWlMessage mirrors struct wl_message { const char *name; const char *signature;
// const struct wl_interface **types; } (24 bytes on 64-bit).
type cWlMessage struct {
	name      uintptr // *const char
	signature uintptr // *const char  (arg type codes: u i s o n a f h ?)
	types     uintptr // **const wl_interface (one entry per arg, NULL for non-object)
}

// cWlInterface mirrors struct wl_interface (48 bytes on 64-bit).
// name(8) + version(4) + pad(4) + method_count(4) + pad(4) + methods(8) +
// event_count(4) + pad(4) + events(8) = 48
type cWlInterface2 struct {
	name        uintptr // *const char
	version     int32
	_           [4]byte
	methodCount int32
	_           [4]byte
	methods     uintptr // *const wl_message
	eventCount  int32
	_           [4]byte
	events      uintptr // *const wl_message
}

// wlNullTypes is a shared array of NULL pointers used as the 'types' field
// in all wl_message entries.  Since we never create object proxies from
// incoming events (we handle them ourselves), NULL types are safe.
var wlNullTypes [32]uintptr // all zeros

func wlNullTypesPtr() uintptr { return uintptr(unsafe.Pointer(&wlNullTypes[0])) }

// ── String data (kept as byte slices so the pointer stays valid) ──────────────

var (
	// Method / event names
	bDestroy           = []byte("destroy\x00")
	bCreatePositioner  = []byte("create_positioner\x00")
	bGetXdgSurface     = []byte("get_xdg_surface\x00")
	bPong              = []byte("pong\x00")
	bGetToplevel       = []byte("get_toplevel\x00")
	bGetPopup          = []byte("get_popup\x00")
	bSetWindowGeometry = []byte("set_window_geometry\x00")
	bAckConfigure      = []byte("ack_configure\x00")
	bSetParent         = []byte("set_parent\x00")
	bSetTitle          = []byte("set_title\x00")
	bSetAppId          = []byte("set_app_id\x00")
	bShowWindowMenu    = []byte("show_window_menu\x00")
	bMove              = []byte("move\x00")
	bResize            = []byte("resize\x00")
	bSetMaxSize        = []byte("set_max_size\x00")
	bSetMinSize        = []byte("set_min_size\x00")
	bSetMaximized      = []byte("set_maximized\x00")
	bUnsetMaximized    = []byte("unset_maximized\x00")
	bSetFullscreen     = []byte("set_fullscreen\x00")
	bUnsetFullscreen   = []byte("unset_fullscreen\x00")
	bSetMinimized      = []byte("set_minimized\x00")
	bPing              = []byte("ping\x00")
	bConfigure         = []byte("configure\x00")
	bClose             = []byte("close\x00")
	bGetTopDeco        = []byte("get_toplevel_decoration\x00")
	bSetMode           = []byte("set_mode\x00")
	bUnsetMode         = []byte("unset_mode\x00")

	// Signatures
	sigEmpty     = []byte("\x00")
	sigU         = []byte("u\x00")
	sigI         = []byte("i\x00")
	sigN         = []byte("n\x00")
	sigS         = []byte("s\x00")
	sigNo        = []byte("no\x00")
	sigIiii      = []byte("iiii\x00")
	sigQo        = []byte("?o\x00")
	sigOu        = []byte("ou\x00")
	sigOuu       = []byte("ouu\x00")
	sigIia       = []byte("iia\x00")
	sigOuii      = []byte("ouii\x00")
	sigQoU       = []byte("?ou\x00") // set_fullscreen: optional output + nothing extra? Actually: ?o
	sigQoOnly    = []byte("?o\x00")
)

func bptr(b []byte) uintptr { return uintptr(unsafe.Pointer(&b[0])) }

// ── xdg_wm_base ───────────────────────────────────────────────────────────────

var (
	xdgWmBaseMethods = [4]cWlMessage{
		{bptr(bDestroy), bptr(sigEmpty), wlNullTypesPtr()},
		{bptr(bCreatePositioner), bptr(sigN), wlNullTypesPtr()},
		{bptr(bGetXdgSurface), bptr(sigNo), wlNullTypesPtr()},
		{bptr(bPong), bptr(sigU), wlNullTypesPtr()},
	}
	xdgWmBaseEvents = [1]cWlMessage{
		{bptr(bPing), bptr(sigU), wlNullTypesPtr()},
	}
	xdgWmBaseIface cWlInterface2
)

// ── xdg_surface ───────────────────────────────────────────────────────────────

var (
	xdgSurfaceMethods = [5]cWlMessage{
		{bptr(bDestroy), bptr(sigEmpty), wlNullTypesPtr()},
		{bptr(bGetToplevel), bptr(sigN), wlNullTypesPtr()},
		{bptr(bGetPopup), bptr(sigN), wlNullTypesPtr()}, // simplified
		{bptr(bSetWindowGeometry), bptr(sigIiii), wlNullTypesPtr()},
		{bptr(bAckConfigure), bptr(sigU), wlNullTypesPtr()},
	}
	xdgSurfaceEvents = [1]cWlMessage{
		{bptr(bConfigure), bptr(sigU), wlNullTypesPtr()},
	}
	xdgSurfaceIface cWlInterface2
)

// ── xdg_toplevel ──────────────────────────────────────────────────────────────

var (
	xdgToplevelMethods = [14]cWlMessage{
		{bptr(bDestroy), bptr(sigEmpty), wlNullTypesPtr()},        // 0
		{bptr(bSetParent), bptr(sigQo), wlNullTypesPtr()},         // 1
		{bptr(bSetTitle), bptr(sigS), wlNullTypesPtr()},           // 2
		{bptr(bSetAppId), bptr(sigS), wlNullTypesPtr()},           // 3
		{bptr(bShowWindowMenu), bptr(sigOuii), wlNullTypesPtr()},  // 4
		{bptr(bMove), bptr(sigOu), wlNullTypesPtr()},              // 5
		{bptr(bResize), bptr(sigOuu), wlNullTypesPtr()},           // 6
		{bptr(bSetMaxSize), bptr(sigIiii[:4]), wlNullTypesPtr()},  // 7 ii
		{bptr(bSetMinSize), bptr(sigIiii[:4]), wlNullTypesPtr()},  // 8 ii
		{bptr(bSetMaximized), bptr(sigEmpty), wlNullTypesPtr()},   // 9
		{bptr(bUnsetMaximized), bptr(sigEmpty), wlNullTypesPtr()}, // 10
		{bptr(bSetFullscreen), bptr(sigQoOnly), wlNullTypesPtr()}, // 11
		{bptr(bUnsetFullscreen), bptr(sigEmpty), wlNullTypesPtr()},// 12
		{bptr(bSetMinimized), bptr(sigEmpty), wlNullTypesPtr()},   // 13
	}
	xdgToplevelEvents = [2]cWlMessage{
		{bptr(bConfigure), bptr(sigIia), wlNullTypesPtr()}, // 0
		{bptr(bClose), bptr(sigEmpty), wlNullTypesPtr()},   // 1
	}
	xdgToplevelIface cWlInterface2
)

// ── zxdg_decoration_manager_v1 ───────────────────────────────────────────────

var (
	xdgDecoMgrMethods = [2]cWlMessage{
		{bptr(bDestroy), bptr(sigEmpty), wlNullTypesPtr()},
		{bptr(bGetTopDeco), bptr(sigN), wlNullTypesPtr()},
	}
	xdgDecoMgrIface cWlInterface2
)

// ── zxdg_toplevel_decoration_v1 ──────────────────────────────────────────────

var (
	xdgTopDecoMethods = [3]cWlMessage{
		{bptr(bDestroy), bptr(sigEmpty), wlNullTypesPtr()},
		{bptr(bSetMode), bptr(sigU), wlNullTypesPtr()},
		{bptr(bUnsetMode), bptr(sigEmpty), wlNullTypesPtr()},
	}
	xdgTopDecoEvents = [1]cWlMessage{
		{bptr(bConfigure), bptr(sigU), wlNullTypesPtr()},
	}
	xdgTopDecoIface cWlInterface2
)

// ── Two-byte signatures needed for set_max_size / set_min_size ────────────────

var sigII = []byte("ii\x00")

// ── initProtocols wires up all cWlInterface2 structs ─────────────────────────

func initProtocols() {
	// Patch ii-signature for xdg_toplevel methods 7 and 8
	xdgToplevelMethods[7].signature = bptr(sigII)
	xdgToplevelMethods[8].signature = bptr(sigII)

	// xdg_wm_base
	xdgWmBaseIface.name = uintptr(unsafe.Pointer(&xdgWmBaseInterfaceName[0]))
	xdgWmBaseIface.version = 4
	xdgWmBaseIface.methodCount = 4
	xdgWmBaseIface.methods = uintptr(unsafe.Pointer(&xdgWmBaseMethods[0]))
	xdgWmBaseIface.eventCount = 1
	xdgWmBaseIface.events = uintptr(unsafe.Pointer(&xdgWmBaseEvents[0]))

	// xdg_surface
	xdgSurfaceIface.name = uintptr(unsafe.Pointer(&xdgSurfaceInterfaceName[0]))
	xdgSurfaceIface.version = 4
	xdgSurfaceIface.methodCount = 5
	xdgSurfaceIface.methods = uintptr(unsafe.Pointer(&xdgSurfaceMethods[0]))
	xdgSurfaceIface.eventCount = 1
	xdgSurfaceIface.events = uintptr(unsafe.Pointer(&xdgSurfaceEvents[0]))

	// xdg_toplevel
	xdgToplevelIface.name = uintptr(unsafe.Pointer(&xdgToplevelInterfaceName[0]))
	xdgToplevelIface.version = 4
	xdgToplevelIface.methodCount = 14
	xdgToplevelIface.methods = uintptr(unsafe.Pointer(&xdgToplevelMethods[0]))
	xdgToplevelIface.eventCount = 2
	xdgToplevelIface.events = uintptr(unsafe.Pointer(&xdgToplevelEvents[0]))

	// zxdg_decoration_manager_v1
	xdgDecoMgrIface.name = uintptr(unsafe.Pointer(&xdgDecoMgrInterfaceName[0]))
	xdgDecoMgrIface.version = 1
	xdgDecoMgrIface.methodCount = 2
	xdgDecoMgrIface.methods = uintptr(unsafe.Pointer(&xdgDecoMgrMethods[0]))

	// zxdg_toplevel_decoration_v1
	xdgTopDecoIface.name = uintptr(unsafe.Pointer(&xdgTopDecoInterfaceName[0]))
	xdgTopDecoIface.version = 1
	xdgTopDecoIface.methodCount = 3
	xdgTopDecoIface.methods = uintptr(unsafe.Pointer(&xdgTopDecoMethods[0]))
	xdgTopDecoIface.eventCount = 1
	xdgTopDecoIface.events = uintptr(unsafe.Pointer(&xdgTopDecoEvents[0]))
}
