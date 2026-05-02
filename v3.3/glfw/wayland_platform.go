//go:build linux && wayland

package glfw

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ── libwayland-client ─────────────────────────────────────────────────────────

var (
	wlClientOnce   sync.Once
	wlClientHandle uintptr
	wlClientErr    error

	wlDisplayConnect         func(name uintptr) uintptr
	wlDisplayDisconnect      func(display uintptr)
	wlDisplayGetFd           func(display uintptr) int32
	wlDisplayFlush           func(display uintptr) int32
	wlDisplayRoundtrip       func(display uintptr) int32
	wlDisplayDispatch        func(display uintptr) int32
	wlDisplayDispatchPending func(display uintptr) int32
	wlDisplayPrepareRead     func(display uintptr) int32
	wlDisplayReadEvents      func(display uintptr) int32
	wlDisplayCancelRead      func(display uintptr)
	wlProxyAddListener           func(proxy, listener, data uintptr) int32
	wlProxyGetId                 func(proxy uintptr) uint32
	wlProxyGetVersion            func(proxy uintptr) uint32
	wlProxyDestroy               func(proxy uintptr)
	wlProxyMarshalFlags          func(proxy uintptr, opcode uint32, iface uintptr,
		version, flags uint32, args ...uintptr) uintptr
	// wl_proxy_marshal_constructor is the low-level function underlying
	// wl_display_get_registry (and similar protocol inlines).  Those inlines
	// are static in wayland-client-protocol.h and therefore NOT exported from
	// the .so — we must reimplement them here via this exported symbol.
	wlProxyMarshalConstructor func(proxy uintptr, opcode uint32, iface uintptr, args ...uintptr) uintptr

	// Built-in libwayland interface descriptors
	wlCompositorIfaceAddr uintptr // &wl_compositor_interface
	wlSeatIfaceAddr       uintptr // &wl_seat_interface
	wlOutputIfaceAddr     uintptr // &wl_output_interface
	wlShmIfaceAddr        uintptr // &wl_shm_interface
	wlDDMgrIfaceAddr      uintptr // &wl_data_device_manager_interface
	wlDDDeviceIfaceAddr   uintptr // &wl_data_device_interface
	wlDDSourceIfaceAddr   uintptr // &wl_data_source_interface
	wlDDOfferIfaceAddr    uintptr // &wl_data_offer_interface
	wlSurfaceIfaceAddr    uintptr // &wl_surface_interface
	wlPointerIfaceAddr    uintptr // &wl_pointer_interface
	wlKeyboardIfaceAddr   uintptr // &wl_keyboard_interface
	wlRegistryIfaceAddr   uintptr // &wl_registry_interface
)

func loadWaylandClient() error {
	wlClientOnce.Do(func() {
		for _, lib := range []string{"libwayland-client.so.0", "libwayland-client.so"} {
			wlClientHandle, wlClientErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if wlClientErr == nil {
				break
			}
		}
		if wlClientErr != nil {
			return
		}
		purego.RegisterLibFunc(&wlDisplayConnect, wlClientHandle, "wl_display_connect")
		purego.RegisterLibFunc(&wlDisplayDisconnect, wlClientHandle, "wl_display_disconnect")
		purego.RegisterLibFunc(&wlDisplayGetFd, wlClientHandle, "wl_display_get_fd")
		purego.RegisterLibFunc(&wlDisplayFlush, wlClientHandle, "wl_display_flush")
		purego.RegisterLibFunc(&wlDisplayRoundtrip, wlClientHandle, "wl_display_roundtrip")
		purego.RegisterLibFunc(&wlDisplayDispatch, wlClientHandle, "wl_display_dispatch")
		purego.RegisterLibFunc(&wlDisplayDispatchPending, wlClientHandle, "wl_display_dispatch_pending")
		purego.RegisterLibFunc(&wlDisplayPrepareRead, wlClientHandle, "wl_display_prepare_read")
		purego.RegisterLibFunc(&wlDisplayReadEvents, wlClientHandle, "wl_display_read_events")
		purego.RegisterLibFunc(&wlDisplayCancelRead, wlClientHandle, "wl_display_cancel_read")
		purego.RegisterLibFunc(&wlProxyAddListener, wlClientHandle, "wl_proxy_add_listener")
		purego.RegisterLibFunc(&wlProxyGetId, wlClientHandle, "wl_proxy_get_id")
		purego.RegisterLibFunc(&wlProxyGetVersion, wlClientHandle, "wl_proxy_get_version")
		purego.RegisterLibFunc(&wlProxyDestroy, wlClientHandle, "wl_proxy_destroy")
		purego.RegisterLibFunc(&wlProxyMarshalFlags, wlClientHandle, "wl_proxy_marshal_flags")
		purego.RegisterLibFunc(&wlProxyMarshalConstructor, wlClientHandle, "wl_proxy_marshal_constructor")

		wlCompositorIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_compositor_interface")
		wlSeatIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_seat_interface")
		wlOutputIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_output_interface")
		wlShmIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_shm_interface")
		wlDDMgrIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_data_device_manager_interface")
		wlDDDeviceIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_data_device_interface")
		wlDDSourceIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_data_source_interface")
		wlDDOfferIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_data_offer_interface")
		wlSurfaceIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_surface_interface")
		wlPointerIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_pointer_interface")
		wlKeyboardIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_keyboard_interface")
		wlRegistryIfaceAddr, _ = purego.Dlsym(wlClientHandle, "wl_registry_interface")
	})
	return wlClientErr
}

// wlDisplayGetRegistry reimplements the static inline wl_display_get_registry
// from wayland-client-protocol.h.  That inline is not exported from the .so,
// so we call the underlying wl_proxy_marshal_constructor directly.
// WL_DISPLAY_GET_REGISTRY is opcode 1 in the wl_display interface.
func wlDisplayGetRegistry(display uintptr) uintptr {
	const WL_DISPLAY_GET_REGISTRY = 1
	return wlProxyMarshalConstructor(display, WL_DISPLAY_GET_REGISTRY, wlRegistryIfaceAddr, 0)
}

// ── libwayland-egl ────────────────────────────────────────────────────────────

var (
	wlEGLOnce          sync.Once
	wlEGLHandle        uintptr
	wlEGLWindowCreate  func(surface uintptr, width, height int32) uintptr
	wlEGLWindowDestroy func(eglWindow uintptr)
	wlEGLWindowResize  func(eglWindow uintptr, width, height, dx, dy int32)
)

func loadWaylandEGL() error {
	var loadErr error
	wlEGLOnce.Do(func() {
		for _, lib := range []string{"libwayland-egl.so.1", "libwayland-egl.so"} {
			wlEGLHandle, loadErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if loadErr == nil {
				break
			}
		}
		if loadErr == nil {
			purego.RegisterLibFunc(&wlEGLWindowCreate, wlEGLHandle, "wl_egl_window_create")
			purego.RegisterLibFunc(&wlEGLWindowDestroy, wlEGLHandle, "wl_egl_window_destroy")
			purego.RegisterLibFunc(&wlEGLWindowResize, wlEGLHandle, "wl_egl_window_resize")
		}
	})
	return loadErr
}

// ── Interface name byte slices (used by both wayland_platform.go and wayland_protocol.go) ──

var (
	xdgWmBaseInterfaceName   = []byte("xdg_wm_base\x00")
	xdgSurfaceInterfaceName  = []byte("xdg_surface\x00")
	xdgToplevelInterfaceName = []byte("xdg_toplevel\x00")
	xdgDecoMgrInterfaceName  = []byte("zxdg_decoration_manager_v1\x00")
	xdgTopDecoInterfaceName  = []byte("zxdg_toplevel_decoration_v1\x00")
)

// ── Global Wayland state ──────────────────────────────────────────────────────

// wlOutput holds per-monitor Wayland state.
type wlOutput struct {
	proxy    uintptr
	pending  Monitor
	monitor  *Monitor
	listener *[4]uintptr // kept alive — Go's non-moving GC keeps address stable
	cbs      [4]uintptr  // purego callback values (must stay referenced)
}

var wl struct {
	display  uintptr // wl_display*
	registry uintptr // wl_registry*
	fd       int     // display socket fd

	compositor uintptr // wl_compositor*
	seat       uintptr // wl_seat*
	shm        uintptr // wl_shm*
	wmBase     uintptr // xdg_wm_base*
	ddManager  uintptr // wl_data_device_manager*
	dataDevice uintptr // wl_data_device*
	decoMgr    uintptr // zxdg_decoration_manager_v1*

	pointer  uintptr // wl_pointer*
	keyboard uintptr // wl_keyboard*

	outputs []*wlOutput

	// Input state
	activeWin   *Window // window with pointer focus
	kbWin       *Window // window with keyboard focus
	ptrSerial   uint32
	kbSerial    uint32
	kbMods      ModifierKey
	kbDepressed uint32
	kbLocked    uint32

	// Clipboard
	clipboardText string
	clipboardSrc  uintptr // wl_data_source* when we own clipboard
	lastOffer     uintptr // most recent wl_data_offer*

	// Self-pipe for PostEmptyEvent
	pipeR, pipeW int

	// Listener vtables (pinned arrays, each kept in a *[N]uintptr field)
	regListener  *[2]uintptr // wl_registry listener
	seatListener *[2]uintptr // wl_seat listener
	ptrListener  *[9]uintptr // wl_pointer listener
	kbListener   *[6]uintptr // wl_keyboard listener
	wmListener   *[1]uintptr // xdg_wm_base listener
	ddListener   *[6]uintptr // wl_data_device listener

	// Monitor callbacks
	monitorCb      func(*Monitor, PeripheralEvent)
	cachedMonitors []*Monitor

	// Input state — shared across all windows
	keyState [349]Action // indexed by GLFW Key constant
	btnState [8]Action   // indexed by MouseButton constant

	// Clipboard: latest offer from wl_data_device.data_offer event
	pendingOffer        uintptr    // wl_data_offer* awaiting selection event
	clipboardSrcListener *[3]uintptr // keeps the data-source listener alive

	// Per-frame discrete-scroll tracking (reset in wl_pointer.frame)
	axisDiscrete [2]bool
}

// ── Init / Terminate ──────────────────────────────────────────────────────────

// Init initialises the Wayland GLFW subsystem.
func Init() error {
	runtime.LockOSThread()
	linuxInitTime = time.Now()
	resetHints()

	if err := loadWaylandClient(); err != nil {
		return &Error{Code: PlatformError,
			Desc: "Wayland: libwayland-client not available: " + err.Error()}
	}
	initProtocols() // set up custom wl_interface descriptors

	wl.pipeR = -1
	wl.pipeW = -1
	var pipeFDs [2]int
	if err := syscall.Pipe2(pipeFDs[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC); err == nil {
		wl.pipeR = pipeFDs[0]
		wl.pipeW = pipeFDs[1]
	}

	// Connect to Wayland display (NULL → use WAYLAND_DISPLAY env var)
	wl.display = wlDisplayConnect(0)
	if wl.display == 0 {
		disp := os.Getenv("WAYLAND_DISPLAY")
		if disp == "" {
			disp = "wayland-0"
		}
		return &Error{Code: PlatformError,
			Desc: fmt.Sprintf("Wayland: cannot connect to display %q", disp)}
	}
	wl.fd = int(wlDisplayGetFd(wl.display))

	// Get registry and bind globals
	wl.registry = wlDisplayGetRegistry(wl.display)
	wl.regListener = new([2]uintptr)
	wl.regListener[0] = purego.NewCallback(wlOnRegistryGlobal)
	wl.regListener[1] = purego.NewCallback(wlOnRegistryGlobalRemove)
	wlProxyAddListener(wl.registry, uintptr(unsafe.Pointer(wl.regListener)), 0)

	// First roundtrip: collect all globals
	wlDisplayRoundtrip(wl.display)
	// Second roundtrip: collect wl_output geometry/mode/done events
	wlDisplayRoundtrip(wl.display)

	if wl.compositor == 0 {
		return &Error{Code: PlatformError,
			Desc: "Wayland: wl_compositor not advertised"}
	}
	if wl.wmBase == 0 {
		return &Error{Code: PlatformError,
			Desc: "Wayland: xdg_wm_base not advertised (compositor too old)"}
	}

	// Init clipboard data device
	if wl.seat != 0 && wl.ddManager != 0 {
		wlInitDataDevice()
	}

	return nil
}

// Terminate destroys all windows and closes the Wayland connection.
func Terminate() {
	windowByHandle.Range(func(k, v any) bool {
		v.(*Window).Destroy()
		return true
	})
	if wl.dataDevice != 0 {
		wlProxyDestroy(wl.dataDevice)
		wl.dataDevice = 0
	}
	if wl.keyboard != 0 {
		wlProxyDestroy(wl.keyboard)
		wl.keyboard = 0
	}
	if wl.pointer != 0 {
		wlProxyDestroy(wl.pointer)
		wl.pointer = 0
	}
	if wl.registry != 0 {
		wlProxyDestroy(wl.registry)
		wl.registry = 0
	}
	if wl.display != 0 {
		wlDisplayDisconnect(wl.display)
		wl.display = 0
	}
	if wl.pipeR >= 0 {
		syscall.Close(wl.pipeR)
		wl.pipeR = -1
	}
	if wl.pipeW >= 0 {
		syscall.Close(wl.pipeW)
		wl.pipeW = -1
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

// wlOnRegistryGlobal is called by libwayland for each global advertised by the compositor.
// C signature: void(void *data, wl_registry*, uint32_t name, const char *iface, uint32_t version)
func wlOnRegistryGlobal(data, registry uintptr, name uint32, ifacePtr uintptr, version uint32) {
	iface := cString(ifacePtr)
	switch iface {
	case "wl_compositor":
		wl.compositor = wlBind(registry, name, wlCompositorIfaceAddr, version, 4)
	case "wl_shm":
		wl.shm = wlBind(registry, name, wlShmIfaceAddr, version, 1)
	case "wl_seat":
		wl.seat = wlBind(registry, name, wlSeatIfaceAddr, version, 5)
		wl.seatListener = new([2]uintptr)
		wl.seatListener[0] = purego.NewCallback(wlOnSeatCapabilities)
		wl.seatListener[1] = purego.NewCallback(wlOnSeatName)
		wlProxyAddListener(wl.seat, uintptr(unsafe.Pointer(wl.seatListener)), 0)
	case "wl_output":
		out := &wlOutput{}
		out.proxy = wlBind(registry, name, wlOutputIfaceAddr, version, 2)
		wl.outputs = append(wl.outputs, out)
		out.listener = new([4]uintptr)
		outRef := out // capture for closure
		out.cbs[0] = purego.NewCallback(func(d, p uintptr, x, y, w, h, subpixel, mk, model uintptr, transform uint32) {
			wlOnOutputGeometry(outRef, x, y, w, h, mk, model)
		})
		out.cbs[1] = purego.NewCallback(func(d, p uintptr, flags, width, height, refresh uint32) {
			wlOnOutputMode(outRef, flags, width, height, refresh)
		})
		out.cbs[2] = purego.NewCallback(func(d, p uintptr) {
			wlOnOutputDone(outRef)
		})
		out.cbs[3] = purego.NewCallback(func(d, p uintptr, factor int32) {})
		copy(out.listener[:], out.cbs[:])
		wlProxyAddListener(out.proxy, uintptr(unsafe.Pointer(out.listener)), 0)
	case "xdg_wm_base":
		wl.wmBase = wlBind(registry, name,
			uintptr(unsafe.Pointer(&xdgWmBaseIface)), version, 4)
		wl.wmListener = new([1]uintptr)
		wl.wmListener[0] = purego.NewCallback(wlOnWmBasePing)
		wlProxyAddListener(wl.wmBase, uintptr(unsafe.Pointer(wl.wmListener)), 0)
	case "wl_data_device_manager":
		wl.ddManager = wlBind(registry, name, wlDDMgrIfaceAddr, version, 3)
	case "zxdg_decoration_manager_v1":
		wl.decoMgr = wlBind(registry, name,
			uintptr(unsafe.Pointer(&xdgDecoMgrIface)), version, 1)
	}
}

func wlOnRegistryGlobalRemove(data, registry uintptr, name uint32) {
	// TODO: handle output hot-unplug
}

// wlBind calls wl_registry.bind to obtain a proxy for a global.
// iface is a pointer to a wl_interface struct (from libwayland or our custom one).
// The first field of wl_interface is the name string pointer.
func wlBind(registry uintptr, name uint32, iface uintptr, have, want uint32) uintptr {
	v := want
	if have < want {
		v = have
	}
	// First field of wl_interface is *const char (the interface name)
	ifaceName := *(*uintptr)(nativePtrFromUintptr(iface))
	return wlProxyMarshalFlags(registry, 0 /* WL_REGISTRY_BIND */,
		iface, v, 0,
		uintptr(name), ifaceName, uintptr(v), 0)
}

// ── xdg_wm_base ping ─────────────────────────────────────────────────────────

// wlOnWmBasePing responds to compositor keep-alive pings.
// C: void(void *data, xdg_wm_base*, uint32_t serial)
func wlOnWmBasePing(data, wmBase uintptr, serial uint32) {
	// xdg_wm_base.pong opcode = 3
	wlProxyMarshalFlags(wmBase, 3,
		uintptr(unsafe.Pointer(&xdgWmBaseIface)), 1, 0, uintptr(serial))
}

// ── wl_seat ───────────────────────────────────────────────────────────────────

// wlOnSeatCapabilities handles changes in the seat's input device capabilities.
// C: void(void *data, wl_seat*, uint32_t capabilities)
func wlOnSeatCapabilities(data, seat uintptr, caps uint32) {
	const (
		capPointer  = uint32(1)
		capKeyboard = uint32(2)
	)
	// Pointer
	if caps&capPointer != 0 && wl.pointer == 0 {
		// wl_seat.get_pointer opcode = 0, returns wl_pointer
		wl.pointer = wlProxyMarshalFlags(seat, 0, wlPointerIfaceAddr, 1, 0, 0)
		if wl.pointer != 0 {
			wl.ptrListener = new([9]uintptr)
			wl.ptrListener[0] = purego.NewCallback(wlOnPointerEnter)
			wl.ptrListener[1] = purego.NewCallback(wlOnPointerLeave)
			wl.ptrListener[2] = purego.NewCallback(wlOnPointerMotion)
			wl.ptrListener[3] = purego.NewCallback(wlOnPointerButton)
			wl.ptrListener[4] = purego.NewCallback(wlOnPointerAxis)
			wl.ptrListener[5] = purego.NewCallback(wlOnPointerFrame)
			wl.ptrListener[6] = purego.NewCallback(wlOnPointerAxisSource)
			wl.ptrListener[7] = purego.NewCallback(wlOnPointerAxisStop)
			wl.ptrListener[8] = purego.NewCallback(wlOnPointerAxisDiscrete)
			wlProxyAddListener(wl.pointer, uintptr(unsafe.Pointer(wl.ptrListener)), 0)
		}
	} else if caps&capPointer == 0 && wl.pointer != 0 {
		wlProxyDestroy(wl.pointer)
		wl.pointer = 0
	}
	// Keyboard
	if caps&capKeyboard != 0 && wl.keyboard == 0 {
		// wl_seat.get_keyboard opcode = 1, returns wl_keyboard
		wl.keyboard = wlProxyMarshalFlags(seat, 1, wlKeyboardIfaceAddr, 1, 0, 0)
		if wl.keyboard != 0 {
			wl.kbListener = new([6]uintptr)
			wl.kbListener[0] = purego.NewCallback(wlOnKeyboardKeymap)
			wl.kbListener[1] = purego.NewCallback(wlOnKeyboardEnter)
			wl.kbListener[2] = purego.NewCallback(wlOnKeyboardLeave)
			wl.kbListener[3] = purego.NewCallback(wlOnKeyboardKey)
			wl.kbListener[4] = purego.NewCallback(wlOnKeyboardModifiers)
			wl.kbListener[5] = purego.NewCallback(wlOnKeyboardRepeatInfo)
			wlProxyAddListener(wl.keyboard, uintptr(unsafe.Pointer(wl.kbListener)), 0)
		}
	} else if caps&capKeyboard == 0 && wl.keyboard != 0 {
		wlProxyDestroy(wl.keyboard)
		wl.keyboard = 0
	}
}

func wlOnSeatName(data, seat, namePtr uintptr) {} // no-op

// ── wl_output ─────────────────────────────────────────────────────────────────

func wlOnOutputGeometry(out *wlOutput, x, y, w, h, makePtr, modelPtr uintptr) {
	name := cString(makePtr)
	model := cString(modelPtr)
	if model != "" && name != "" {
		name = name + " " + model
	} else if model != "" {
		name = model
	}
	out.pending.name = name
	out.pending.x = int(int32(x))
	out.pending.y = int(int32(y))
	out.pending.widthMM = int(int32(w))
	out.pending.heightMM = int(int32(h))
}

func wlOnOutputMode(out *wlOutput, flags, width, height, refresh uint32) {
	const modeCurrent = uint32(2)
	mode := &VidMode{
		Width:       int(width),
		Height:      int(height),
		RefreshRate: int(math.Round(float64(refresh) / 1000.0)),
	}
	out.pending.modes = append(out.pending.modes, mode)
	if flags&modeCurrent != 0 {
		out.pending.currentMode = mode
		out.pending.widthPx = int(width)
		out.pending.heightPx = int(height)
	}
}

func wlOnOutputDone(out *wlOutput) {
	prev := out.monitor
	if out.monitor == nil {
		out.monitor = new(Monitor)
	}
	*out.monitor = out.pending
	out.monitor.outputID = wlProxyGetId(out.proxy)
	out.pending = Monitor{}

	if wl.monitorCb != nil {
		if prev == nil {
			wl.monitorCb(out.monitor, Connected)
		}
	}
}

// ── PollEvents / WaitEvents ───────────────────────────────────────────────────

// PollEvents processes all pending events without blocking.
func PollEvents() {
	if wl.display == 0 {
		return
	}
	wlDisplayFlush(wl.display)
	// Drain events that are already queued in libwayland's buffer
	for wlDisplayPrepareRead(wl.display) != 0 {
		wlDisplayDispatchPending(wl.display)
	}
	// Non-blocking check for new events on the socket
	var rfds syscall.FdSet
	rfds.Bits[wl.fd/64] |= 1 << (uint(wl.fd) % 64)
	tv := syscall.Timeval{}
	n, _ := syscall.Select(wl.fd+1, &rfds, nil, nil, &tv)
	if n > 0 && rfds.Bits[wl.fd/64]&(1<<(uint(wl.fd)%64)) != 0 {
		wlDisplayReadEvents(wl.display)
	} else {
		wlDisplayCancelRead(wl.display)
	}
	wlDisplayDispatchPending(wl.display)
	wlDisplayFlush(wl.display)

	if jsEverInitialized {
		pollJoystickEvents()
	}
}

// WaitEvents blocks until an event arrives, then processes all pending events.
func WaitEvents() {
	if wl.display == 0 {
		return
	}
	wlDisplayFlush(wl.display)
	wlSelectWait(nil)
	PollEvents()
}

// WaitEventsTimeout blocks for at most timeout seconds.
func WaitEventsTimeout(timeout float64) {
	if wl.display == 0 {
		return
	}
	wlDisplayFlush(wl.display)
	sec := int64(timeout)
	usec := int64((timeout - float64(sec)) * 1e6)
	tv := syscall.Timeval{Sec: sec, Usec: usec}
	wlSelectWait(&tv)
	PollEvents()
}

func wlSelectWait(tv *syscall.Timeval) {
	var rfds syscall.FdSet
	rfds.Bits[wl.fd/64] |= 1 << (uint(wl.fd) % 64)
	maxFd := wl.fd
	if wl.pipeR >= 0 {
		rfds.Bits[wl.pipeR/64] |= 1 << (uint(wl.pipeR) % 64)
		if wl.pipeR > maxFd {
			maxFd = wl.pipeR
		}
	}
	syscall.Select(maxFd+1, &rfds, nil, nil, tv) //nolint:errcheck
	if wl.pipeR >= 0 {
		if rfds.Bits[wl.pipeR/64]&(1<<(uint(wl.pipeR)%64)) != 0 {
			var buf [1]byte
			syscall.Read(wl.pipeR, buf[:]) //nolint:errcheck
		}
	}
}

// PostEmptyEvent wakes up WaitEvents from another thread.
func PostEmptyEvent() {
	if wl.pipeW >= 0 {
		syscall.Write(wl.pipeW, []byte{0}) //nolint:errcheck
	}
}

// ── Monitor API ───────────────────────────────────────────────────────────────

// GetMonitors returns all monitors that have received a wl_output.done event.
func GetMonitors() ([]*Monitor, error) {
	var result []*Monitor
	for _, out := range wl.outputs {
		if out.monitor != nil {
			result = append(result, out.monitor)
		}
	}
	return result, nil
}

// GetPrimaryMonitor returns the first known monitor.
func GetPrimaryMonitor() *Monitor {
	for _, out := range wl.outputs {
		if out.monitor != nil {
			return out.monitor
		}
	}
	return nil
}

// SetMonitorCallback registers a connect/disconnect callback for monitors.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	wl.monitorCb = cb
	if cb != nil {
		wl.cachedMonitors, _ = GetMonitors()
	}
}

// ── Feature query stubs ───────────────────────────────────────────────────────

// RawMouseMotionSupported reports whether raw (unaccelerated) mouse motion is
// available.  Wayland pointer-constraints/relative-pointer are not yet wired up.
func RawMouseMotionSupported() bool { return false }

// ── Gamma (stub) ──────────────────────────────────────────────────────────────

func (m *Monitor) SetGamma(gamma float32)         {}
func (m *Monitor) SetGammaRamp(ramp *GammaRamp)   {}
func (m *Monitor) GetGammaRamp() *GammaRamp        { return nil }

// ── Window stubs for features unavailable on Wayland ─────────────────────────

func (w *Window) SetIcon(images []Image)                              {}
func (w *Window) GetFrameSize() (left, top, right, bottom int)       { return 0, 0, 0, 0 }
func GetWindowFrameSize(w *Window) (left, top, right, bottom int)    { return 0, 0, 0, 0 }
func (w *Window) GetOpacity() float32                                 { return 1.0 }
func (w *Window) SetOpacity(_ float32)                                {}
func (w *Window) RequestAttention()                                   {}

func (w *Window) SetSizeLimits(minWidth, minHeight, maxWidth, maxHeight int) {
	w.minW, w.minH, w.maxW, w.maxH = minWidth, minHeight, maxWidth, maxHeight
	if w.wlXdgTop != 0 {
		// xdg_toplevel.set_min_size opcode = 8
		wlProxyMarshalFlags(w.wlXdgTop, 8, 0, 4, 0,
			uintptr(uint32(minWidth)), uintptr(uint32(minHeight)))
		// xdg_toplevel.set_max_size opcode = 7
		wlProxyMarshalFlags(w.wlXdgTop, 7, 0, 4, 0,
			uintptr(uint32(maxWidth)), uintptr(uint32(maxHeight)))
		// wl_surface.commit opcode = 6
		wlProxyMarshalFlags(w.handle, 6, 0, 1, 0)
		wlDisplayFlush(wl.display)
	}
}

func (w *Window) SetAspectRatio(numer, denom int) {
	w.aspectNum, w.aspectDen = numer, denom
	// Wayland has no aspect ratio hint in xdg-shell; this is a best-effort no-op.
}

// ── Utility helpers ───────────────────────────────────────────────────────────

// cString converts a C null-terminated string pointer to a Go string.
func cString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	b := (*[1 << 20]byte)(nativePtrFromUintptr(ptr))
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return ""
}
