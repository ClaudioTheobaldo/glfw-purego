//go:build darwin

// darwin_monitor.go — CoreGraphics monitor enumeration and hotplug detection.
//
// Monitor geometry is read from NSScreen (safe ObjC struct returns).
// Video-mode enumeration uses CGDisplayMode (scalar returns only).
// Monitor connect/disconnect uses CGDisplayRegisterReconfigurationCallback.

package glfw

import (
	"fmt"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// ── SEL cache (monitor-specific) ──────────────────────────────────────────────

var (
	selDeviceDescription = objc.RegisterName("deviceDescription")
	selObjectForKey      = objc.RegisterName("objectForKey:")
	selUnsignedIntValue  = objc.RegisterName("unsignedIntValue")
	selLocalizedName     = objc.RegisterName("localizedName")
)

// ── CoreGraphics bindings ─────────────────────────────────────────────────────
//
// We deliberately avoid CG functions that return structs (CGDisplayBounds,
// CGDisplayScreenSize) because their calling convention on amd64 requires
// special sret handling.  Monitor geometry is obtained via NSScreen instead.

var (
	cgMonMainDisplayID           func() uint32
	cgMonGetActiveDisplayList    func(maxDisplays uint32, activeDisplays *uint32, displayCount *uint32) int32
	cgMonDisplayCopyAllModes     func(display uint32, options uintptr) uintptr  // CFArrayRef
	cgMonDisplayCopyCurrentMode  func(display uint32) uintptr                   // CGDisplayModeRef
	cgMonDisplayModeGetWidth     func(mode uintptr) uint64
	cgMonDisplayModeGetHeight    func(mode uintptr) uint64
	cgMonDisplayModeGetRefresh   func(mode uintptr) float64
	cgMonRegisterReconfigCB      func(callback, userInfo uintptr) int32
	cgMonRemoveReconfigCB        func(callback, userInfo uintptr) int32

	// CoreFoundation array helpers (re-exported by CoreGraphics.framework).
	cfMonArrayGetCount      func(array uintptr) int64
	cfMonArrayGetValueAt    func(array uintptr, idx int64) uintptr
	cfMonRelease            func(obj uintptr)
)

// darwinCGReconfigCBPtr holds the C callback pointer for the display
// reconfiguration callback; it must stay alive for the lifetime of the
// registration to prevent it from being garbage-collected.
var darwinCGReconfigCBPtr uintptr

// initMonitorCG loads CoreGraphics and CoreFoundation display functions.
// Called once from Init() via darwinInitOnce.
func initMonitorCG() {
	lib, err := purego.Dlopen(
		"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
		purego.RTLD_GLOBAL|purego.RTLD_LAZY,
	)
	if err != nil {
		return // CoreGraphics unavailable — monitor APIs will return stubs
	}
	purego.RegisterLibFunc(&cgMonMainDisplayID, lib, "CGMainDisplayID")
	purego.RegisterLibFunc(&cgMonGetActiveDisplayList, lib, "CGGetActiveDisplayList")
	purego.RegisterLibFunc(&cgMonDisplayCopyAllModes, lib, "CGDisplayCopyAllDisplayModes")
	purego.RegisterLibFunc(&cgMonDisplayCopyCurrentMode, lib, "CGDisplayCopyCurrentDisplayMode")
	purego.RegisterLibFunc(&cgMonDisplayModeGetWidth, lib, "CGDisplayModeGetWidth")
	purego.RegisterLibFunc(&cgMonDisplayModeGetHeight, lib, "CGDisplayModeGetHeight")
	purego.RegisterLibFunc(&cgMonDisplayModeGetRefresh, lib, "CGDisplayModeGetRefreshRate")
	purego.RegisterLibFunc(&cgMonRegisterReconfigCB, lib, "CGDisplayRegisterReconfigurationCallback")
	purego.RegisterLibFunc(&cgMonRemoveReconfigCB, lib, "CGDisplayRemoveReconfigurationCallback")
	purego.RegisterLibFunc(&cfMonArrayGetCount, lib, "CFArrayGetCount")
	purego.RegisterLibFunc(&cfMonArrayGetValueAt, lib, "CFArrayGetValueAtIndex")
	purego.RegisterLibFunc(&cfMonRelease, lib, "CFRelease")
}

// ── Monitor struct addition ───────────────────────────────────────────────────
//
// cgDisplayID is appended to the Monitor struct declared in darwin_platform.go
// by re-declaring the type with the extra field.  Go requires the struct to be
// declared in only one place, so we add cgDisplayID directly there; this file
// only documents the intent.

// ── Coordinate helpers ────────────────────────────────────────────────────────

// mainScreenHeightPt returns the logical (point) height of the primary NSScreen.
// Used to convert Cocoa Y (origin bottom-left, up) → GLFW Y (origin top-left, down).
func mainScreenHeightPt() float64 {
	main := objc.ID(objc.GetClass("NSScreen")).Send(selMainScreen)
	if main == 0 {
		return 0
	}
	f := objc.Send[NSRect](main, selFrame)
	return f.Size.Height
}

// cocoaScreenFrameToGLFW converts an NSScreen frame (Cocoa coords) to the
// GLFW virtual-desktop position of the screen's top-left corner.
func cocoaScreenFrameToGLFW(frame NSRect) (x, y int) {
	mainH := mainScreenHeightPt()
	x = int(frame.Origin.X)
	// Cocoa Y is the distance from the bottom of the primary screen to the
	// bottom of this screen.  GLFW Y is measured from the top of the primary
	// screen to the top of this screen.
	y = int(mainH - (frame.Origin.Y + frame.Size.Height))
	return
}

// ── Video-mode helpers ────────────────────────────────────────────────────────

// modesForDisplay returns all video modes for the given CGDirectDisplayID.
// Returns an empty slice if CoreGraphics is not available.
func modesForDisplay(displayID uint32) []*VidMode {
	if cgMonDisplayCopyAllModes == nil {
		return nil
	}
	arr := cgMonDisplayCopyAllModes(displayID, 0 /* options = nil */)
	if arr == 0 {
		return nil
	}
	defer cfMonRelease(arr)

	count := cfMonArrayGetCount(arr)
	modes := make([]*VidMode, 0, count)
	seen := make(map[[3]int]bool, count)

	for i := int64(0); i < count; i++ {
		mode := cfMonArrayGetValueAt(arr, i)
		if mode == 0 {
			continue
		}
		w := int(cgMonDisplayModeGetWidth(mode))
		h := int(cgMonDisplayModeGetHeight(mode))
		hz := int(cgMonDisplayModeGetRefresh(mode) + 0.5)
		if w == 0 || h == 0 {
			continue
		}
		key := [3]int{w, h, hz}
		if seen[key] {
			continue
		}
		seen[key] = true
		modes = append(modes, &VidMode{
			Width: w, Height: h,
			RedBits: 8, GreenBits: 8, BlueBits: 8,
			RefreshRate: hz,
		})
	}
	return modes
}

// currentModeForDisplay returns the current video mode of a display.
// Returns nil if CoreGraphics is not available.
func currentModeForDisplay(displayID uint32) *VidMode {
	if cgMonDisplayCopyCurrentMode == nil {
		return nil
	}
	mode := cgMonDisplayCopyCurrentMode(displayID)
	if mode == 0 {
		return nil
	}
	defer cfMonRelease(mode)
	w := int(cgMonDisplayModeGetWidth(mode))
	h := int(cgMonDisplayModeGetHeight(mode))
	hz := int(cgMonDisplayModeGetRefresh(mode) + 0.5)
	return &VidMode{
		Width: w, Height: h,
		RedBits: 8, GreenBits: 8, BlueBits: 8,
		RefreshRate: hz,
	}
}

// ── NSScreen → Monitor ────────────────────────────────────────────────────────

// monitorFromNSScreen builds a Monitor from an NSScreen object.
func monitorFromNSScreen(screen objc.ID) *Monitor {
	// Retrieve CGDirectDisplayID from the screen's device description.
	desc := screen.Send(selDeviceDescription)
	nsNumKey := nsStringFromGoString("NSScreenNumber")
	nsNum := desc.Send(selObjectForKey, nsNumKey)
	displayID := uint32(objc.Send[uint64](nsNum, selUnsignedIntValue))

	// Screen name (macOS 10.15+).
	name := goStringFromNS(screen.Send(selLocalizedName))
	if name == "" {
		name = fmt.Sprintf("Display %d", displayID)
	}

	// Logical (point) frame — position in the Cocoa virtual desktop.
	frame := objc.Send[NSRect](screen, selFrame)
	x, y := cocoaScreenFrameToGLFW(frame)

	modes := modesForDisplay(displayID)
	currentMode := currentModeForDisplay(displayID)

	// Synthesise a currentMode from the screen frame if CG mode lookup failed.
	if currentMode == nil {
		currentMode = &VidMode{
			Width: int(frame.Size.Width), Height: int(frame.Size.Height),
			RedBits: 8, GreenBits: 8, BlueBits: 8,
		}
	}
	if len(modes) == 0 {
		modes = []*VidMode{currentMode}
	}

	return &Monitor{
		cgDisplayID: displayID,
		name:        name,
		x:           x,
		y:           y,
		widthPx:     int(frame.Size.Width),
		heightPx:    int(frame.Size.Height),
		// Physical size: 0,0 — Phase D stub (needs CGDisplayScreenSize or IOKit).
		widthMM:     0,
		heightMM:    0,
		modes:       modes,
		currentMode: currentMode,
	}
}

// ── Public monitor API ────────────────────────────────────────────────────────

// GetMonitors returns all connected monitors ordered primary-first.
func GetMonitors() ([]*Monitor, error) {
	screens := objc.ID(objc.GetClass("NSScreen")).Send(selScreens)
	count := objc.Send[uint64](screens, selCount)
	if count == 0 {
		return nil, nil
	}

	monitors := make([]*Monitor, 0, count)
	var primary *Monitor

	var mainID uint32
	if cgMonMainDisplayID != nil {
		mainID = cgMonMainDisplayID()
	}

	for i := uint64(0); i < count; i++ {
		screen := screens.Send(selObjectAtIndex, i)
		m := monitorFromNSScreen(screen)
		if m == nil {
			continue
		}
		if m.cgDisplayID == mainID {
			primary = m
		} else {
			monitors = append(monitors, m)
		}
	}

	// Primary monitor goes first.
	if primary != nil {
		monitors = append([]*Monitor{primary}, monitors...)
	}
	return monitors, nil
}

// GetPrimaryMonitor returns the primary (main) monitor.
func GetPrimaryMonitor() *Monitor {
	if cgMonMainDisplayID == nil {
		return nil
	}
	screens := objc.ID(objc.GetClass("NSScreen")).Send(selScreens)
	count := objc.Send[uint64](screens, selCount)
	mainID := cgMonMainDisplayID()

	for i := uint64(0); i < count; i++ {
		screen := screens.Send(selObjectAtIndex, i)
		desc := screen.Send(selDeviceDescription)
		nsNum := desc.Send(selObjectForKey, nsStringFromGoString("NSScreenNumber"))
		displayID := uint32(objc.Send[uint64](nsNum, selUnsignedIntValue))
		if displayID == mainID {
			return monitorFromNSScreen(screen)
		}
	}
	return nil
}

// ── Hotplug callback ──────────────────────────────────────────────────────────

// registerMonitorReconfigCB registers a CGDisplayReconfigurationCallback that
// fires darwinMonitorCb when a display is connected or disconnected.
// It is a no-op if CoreGraphics is not available.
func registerMonitorReconfigCB() {
	if cgMonRegisterReconfigCB == nil {
		return
	}
	cb := purego.NewCallback(func(displayID uint32, flags uint32, _ uintptr) {
		// kCGDisplayAddFlag = 0x100, kCGDisplayRemoveFlag = 0x200
		const addFlag    uint32 = 0x100
		const removeFlag uint32 = 0x200
		if flags&(addFlag|removeFlag) == 0 {
			return // not a connect/disconnect event
		}
		if darwinMonitorCb == nil {
			return
		}
		newMonitors, _ := GetMonitors()
		diffAndFireMonitorCallbacks(darwinCachedMonitors, newMonitors, darwinMonitorCb)
		darwinCachedMonitors = newMonitors
	})
	darwinCGReconfigCBPtr = cb
	cgMonRegisterReconfigCB(cb, 0)
}

// deregisterMonitorReconfigCB removes the display reconfiguration callback.
// Called from Terminate().
func deregisterMonitorReconfigCB() {
	if cgMonRemoveReconfigCB == nil || darwinCGReconfigCBPtr == 0 {
		return
	}
	cgMonRemoveReconfigCB(darwinCGReconfigCBPtr, 0)
	darwinCGReconfigCBPtr = 0
}
