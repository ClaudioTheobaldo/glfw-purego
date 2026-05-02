//go:build linux && !wayland

package glfw

import (
	"math"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ----------------------------------------------------------------------------
// XF86VidMode gamma — libXxf86vm.so.1 (lazy-loaded)
// ----------------------------------------------------------------------------

var (
	xxf86vmOnce   sync.Once
	xxf86vmHandle uintptr
	xxf86vmErr    error

	xf86VidModeGetGammaRamp func(display uintptr, screen, size int32, red, green, blue uintptr) int32
	xf86VidModeSetGammaRamp func(display uintptr, screen, size int32, red, green, blue uintptr) int32
)

func loadXxf86vm() error {
	xxf86vmOnce.Do(func() {
		for _, name := range []string{"libXxf86vm.so.1", "libXxf86vm.so"} {
			xxf86vmHandle, xxf86vmErr = purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if xxf86vmErr == nil {
				break
			}
		}
		if xxf86vmErr != nil {
			return
		}
		purego.RegisterLibFunc(&xf86VidModeGetGammaRamp, xxf86vmHandle, "XF86VidModeGetGammaRamp")
		purego.RegisterLibFunc(&xf86VidModeSetGammaRamp, xxf86vmHandle, "XF86VidModeSetGammaRamp")
	})
	return xxf86vmErr
}

// GetGammaRamp returns the monitor's current gamma ramp (256 entries per channel).
func (m *Monitor) GetGammaRamp() *GammaRamp {
	if err := loadXxf86vm(); err != nil {
		return nil
	}
	if x11Display == 0 {
		return nil
	}
	const size = 256
	red := make([]uint16, size)
	green := make([]uint16, size)
	blue := make([]uint16, size)
	ok := xf86VidModeGetGammaRamp(x11Display, x11Screen, size,
		uintptr(unsafe.Pointer(&red[0])),
		uintptr(unsafe.Pointer(&green[0])),
		uintptr(unsafe.Pointer(&blue[0])))
	runtime.KeepAlive(red)
	runtime.KeepAlive(green)
	runtime.KeepAlive(blue)
	if ok == 0 {
		return nil
	}
	return &GammaRamp{Red: red, Green: green, Blue: blue}
}

// SetGammaRamp sets the monitor's gamma ramp.
// The ramp must have equal-length Red, Green, Blue slices.
func (m *Monitor) SetGammaRamp(ramp *GammaRamp) {
	if ramp == nil || len(ramp.Red) == 0 {
		return
	}
	if err := loadXxf86vm(); err != nil {
		return
	}
	if x11Display == 0 {
		return
	}
	xf86VidModeSetGammaRamp(x11Display, x11Screen, int32(len(ramp.Red)),
		uintptr(unsafe.Pointer(&ramp.Red[0])),
		uintptr(unsafe.Pointer(&ramp.Green[0])),
		uintptr(unsafe.Pointer(&ramp.Blue[0])))
	runtime.KeepAlive(ramp)
}

// SetGamma builds a standard power-law gamma ramp and applies it.
func (m *Monitor) SetGamma(gamma float32) {
	const size = 256
	red := make([]uint16, size)
	green := make([]uint16, size)
	blue := make([]uint16, size)
	for i := 0; i < size; i++ {
		v := math.Pow(float64(i)/255.0, 1.0/float64(gamma))
		u := uint16(math.Round(v * 65535.0))
		red[i] = u
		green[i] = u
		blue[i] = u
	}
	m.SetGammaRamp(&GammaRamp{Red: red, Green: green, Blue: blue})
}
