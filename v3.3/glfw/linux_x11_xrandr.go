//go:build linux

package glfw

import (
	"math"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ----------------------------------------------------------------------------
// XRandR lazy loader
// ----------------------------------------------------------------------------

var (
	xrandrOnce      sync.Once
	xrandrHandle    uintptr
	xrandrErr       error
	xrandrEventBase int32 // used for Group 5 monitor-change events
	xrandrErrorBase int32

	xrrGetScreenResources func(display uintptr, window uint64) uintptr
	xrrFreeScreenResources func(res uintptr)
	xrrGetOutputInfo       func(display, res uintptr, output uint64) uintptr
	xrrFreeOutputInfo      func(info uintptr)
	xrrGetCrtcInfo         func(display, res uintptr, crtc uint64) uintptr
	xrrFreeCrtcInfo        func(info uintptr)
	xrrGetOutputPrimary    func(display uintptr, window uint64) uint64
	xrrSelectInput         func(display uintptr, window uint64, mask int32)
	xrrQueryExtension      func(display uintptr, eventBase, errorBase uintptr) int32
)

func loadXrandr() error {
	xrandrOnce.Do(func() {
		for _, lib := range []string{"libXrandr.so.2", "libXrandr.so"} {
			xrandrHandle, xrandrErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if xrandrErr == nil {
				break
			}
		}
		if xrandrErr != nil {
			return
		}
		purego.RegisterLibFunc(&xrrGetScreenResources, xrandrHandle, "XRRGetScreenResources")
		purego.RegisterLibFunc(&xrrFreeScreenResources, xrandrHandle, "XRRFreeScreenResources")
		purego.RegisterLibFunc(&xrrGetOutputInfo, xrandrHandle, "XRRGetOutputInfo")
		purego.RegisterLibFunc(&xrrFreeOutputInfo, xrandrHandle, "XRRFreeOutputInfo")
		purego.RegisterLibFunc(&xrrGetCrtcInfo, xrandrHandle, "XRRGetCrtcInfo")
		purego.RegisterLibFunc(&xrrFreeCrtcInfo, xrandrHandle, "XRRFreeCrtcInfo")
		purego.RegisterLibFunc(&xrrGetOutputPrimary, xrandrHandle, "XRRGetOutputPrimary")
		purego.RegisterLibFunc(&xrrSelectInput, xrandrHandle, "XRRSelectInput")
		purego.RegisterLibFunc(&xrrQueryExtension, xrandrHandle, "XRRQueryExtension")

		// Query event/error base so Group-5 RRNotify dispatch works.
		xrrQueryExtension(x11Display,
			uintptr(unsafe.Pointer(&xrandrEventBase)),
			uintptr(unsafe.Pointer(&xrandrErrorBase)))
	})
	return xrandrErr
}

// ----------------------------------------------------------------------------
// XRandR C struct mirrors
// All sizes verified against Xrandr.h for 64-bit Linux (unsigned long = 8 bytes).
// ----------------------------------------------------------------------------

// _XRRScreenResources — 64 bytes
type _XRRScreenResources struct {
	Timestamp       uint64  // 0
	ConfigTimestamp uint64  // 8
	NCrtc           int32   // 16
	_               [4]byte // 20
	Crtcs           uintptr // 24  → *RRCrtc (array of uint64)
	NOutput         int32   // 32
	_               [4]byte // 36
	Outputs         uintptr // 40  → *RROutput (array of uint64)
	NMode           int32   // 48
	_               [4]byte // 52
	Modes           uintptr // 56  → *XRRModeInfo
}

// _XRRModeInfo — 80 bytes
type _XRRModeInfo struct {
	Id         uint64  // 0   (RRMode = unsigned long)
	Width      uint32  // 8
	Height     uint32  // 12
	DotClock   uint64  // 16
	HSyncStart uint32  // 24
	HSyncEnd   uint32  // 28
	HTotal     uint32  // 32
	HSkew      uint32  // 36
	VSyncStart uint32  // 40
	VSyncEnd   uint32  // 44
	VTotal     uint32  // 48
	_          [4]byte // 52
	Name       uintptr // 56  → char*
	NameLength uint32  // 64
	_          [4]byte // 68
	ModeFlags  uint64  // 72
} // total: 80

// _XRROutputInfo — 96 bytes
type _XRROutputInfo struct {
	Timestamp     uint64  // 0
	Crtc          uint64  // 8   (RRCrtc = unsigned long; current CRTC, 0 if none)
	Name          uintptr // 16  → char*
	NameLen       int32   // 24
	_             [4]byte // 28
	MmWidth       uint64  // 32
	MmHeight      uint64  // 40
	Connection    uint16  // 48  (0 = RR_Connected)
	SubpixelOrder uint16  // 50
	NCrtc         int32   // 52
	Crtcs         uintptr // 56  → *RRCrtc
	NClone        int32   // 64
	_             [4]byte // 68
	Clones        uintptr // 72  → *RROutput
	NMode         int32   // 80
	NPreferred    int32   // 84
	Modes         uintptr // 88  → *RRMode (array of uint64)
} // total: 96

// _XRRCrtcInfo — 64 bytes
type _XRRCrtcInfo struct {
	Timestamp uint64  // 0
	X         int32   // 8
	Y         int32   // 12
	Width     uint32  // 16
	Height    uint32  // 20
	Mode      uint64  // 24  (RRMode = unsigned long; current mode ID)
	Rotation  uint16  // 32
	_         [2]byte // 34
	NOutput   int32   // 36
	Outputs   uintptr // 40  → *RROutput
	Rotations uint16  // 48
	_         [2]byte // 50
	NPossible int32   // 52
	Possible  uintptr // 56  → *RROutput
} // total: 64

// ----------------------------------------------------------------------------
// GetMonitors — enumerate connected outputs via XRandR
// ----------------------------------------------------------------------------

// GetMonitors returns all currently connected monitors via XRandR.
// Falls back to a single stub monitor when XRandR is unavailable.
func GetMonitors() ([]*Monitor, error) {
	if x11Display == 0 {
		return nil, nil
	}
	if err := loadXrandr(); err != nil {
		return []*Monitor{{name: "default"}}, nil
	}

	resPtr := xrrGetScreenResources(x11Display, x11Root)
	if resPtr == 0 {
		return nil, &Error{Code: PlatformError, Desc: "XRRGetScreenResources failed"}
	}
	defer xrrFreeScreenResources(resPtr)
	sr := (*_XRRScreenResources)(unsafe.Pointer(resPtr))

	primaryOutput := xrrGetOutputPrimary(x11Display, x11Root)

	// Build a map from RRMode ID → VidMode using the resource mode list.
	type modeEntry struct {
		id uint64
		vm *VidMode
	}
	globalModes := make(map[uint64]*VidMode, int(sr.NMode))
	for i := 0; i < int(sr.NMode); i++ {
		mi := (*_XRRModeInfo)(unsafe.Pointer(sr.Modes + uintptr(i)*80))
		globalModes[mi.Id] = &VidMode{
			Width:       int(mi.Width),
			Height:      int(mi.Height),
			RefreshRate: modeRefreshRate(mi.DotClock, mi.HTotal, mi.VTotal),
			RedBits:     8,
			GreenBits:   8,
			BlueBits:    8,
		}
	}

	var monitors []*Monitor
	for i := 0; i < int(sr.NOutput); i++ {
		outputID := *(*uint64)(unsafe.Pointer(sr.Outputs + uintptr(i)*8))
		oiPtr := xrrGetOutputInfo(x11Display, resPtr, outputID)
		if oiPtr == 0 {
			continue
		}
		oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

		if oi.Connection != 0 { // RR_Connected = 0; skip disconnected
			xrrFreeOutputInfo(oiPtr)
			continue
		}

		m := &Monitor{
			output:   outputID,
			widthMM:  int(oi.MmWidth),
			heightMM: int(oi.MmHeight),
		}

		// Copy output name.
		if oi.Name != 0 && oi.NameLen > 0 {
			b := make([]byte, oi.NameLen)
			for j := int32(0); j < oi.NameLen; j++ {
				b[j] = *(*byte)(unsafe.Pointer(oi.Name + uintptr(j)))
			}
			m.name = string(b)
		}
		if m.name == "" {
			m.name = "monitor"
		}

		// Build per-output mode list ordered by the output's mode array.
		type oidMode struct {
			id uint64
			vm *VidMode
		}
		var orderedModes []oidMode
		for j := 0; j < int(oi.NMode); j++ {
			mid := *(*uint64)(unsafe.Pointer(oi.Modes + uintptr(j)*8))
			if vm, ok := globalModes[mid]; ok {
				orderedModes = append(orderedModes, oidMode{mid, vm})
			}
		}
		m.modes = make([]*VidMode, len(orderedModes))
		for j, e := range orderedModes {
			m.modes[j] = e.vm
		}

		// Get CRTC for position, pixel size, and current mode.
		if oi.Crtc != 0 {
			ciPtr := xrrGetCrtcInfo(x11Display, resPtr, oi.Crtc)
			if ciPtr != 0 {
				ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
				m.x = int(ci.X)
				m.y = int(ci.Y)
				m.widthPx = int(ci.Width)
				m.heightPx = int(ci.Height)
				m.crtc = oi.Crtc
				// Find the currently-active VidMode by matching the CRTC's mode ID.
				for _, e := range orderedModes {
					if e.id == ci.Mode {
						m.currentMode = e.vm
						break
					}
				}
				xrrFreeCrtcInfo(ciPtr)
			}
		}

		if len(m.modes) == 0 {
			xrrFreeOutputInfo(oiPtr)
			continue
		}
		if m.currentMode == nil {
			m.currentMode = m.modes[0]
		}

		// Primary output goes first.
		if outputID == primaryOutput {
			monitors = append([]*Monitor{m}, monitors...)
		} else {
			monitors = append(monitors, m)
		}
		xrrFreeOutputInfo(oiPtr)
	}

	return monitors, nil
}

// GetPrimaryMonitor returns the primary monitor (first in the list).
func GetPrimaryMonitor() *Monitor {
	monitors, _ := GetMonitors()
	if len(monitors) == 0 {
		return nil
	}
	return monitors[0]
}

// modeRefreshRate computes the refresh rate in Hz from XRandR mode parameters.
func modeRefreshRate(dotClock uint64, hTotal, vTotal uint32) int {
	if hTotal == 0 || vTotal == 0 || dotClock == 0 {
		return 0
	}
	return int(math.Round(float64(dotClock) / float64(uint64(hTotal)*uint64(vTotal))))
}
