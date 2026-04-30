//go:build windows

package glfw

import (
	"math"
	"syscall"
	"unsafe"
)

// Monitor represents a connected display.
type Monitor struct {
	hmon    uintptr // HMONITOR
	name    string  // device name from MONITORINFOEXW
	primary bool
}

// GetMonitors returns all currently connected monitors.
func GetMonitors() ([]*Monitor, error) {
	monitors := collectMonitors()
	if len(monitors) == 0 {
		return nil, &Error{Code: PlatformError, Desc: "no monitors found"}
	}
	return monitors, nil
}

// GetPrimaryMonitor returns the primary monitor.
func GetPrimaryMonitor() *Monitor {
	for _, m := range collectMonitors() {
		if m.primary {
			return m
		}
	}
	return nil
}

// collectMonitors enumerates all monitors via EnumDisplayMonitors.
func collectMonitors() []*Monitor {
	var monitors []*Monitor
	cb := syscall.NewCallback(func(hmon, _, _, _ uintptr) uintptr {
		monitors = append(monitors, monitorFromHandle(hmon))
		return 1 // continue
	})
	enumDisplayMonitors(0, 0, cb, 0)
	return monitors
}

// monitorFromHandle builds a Monitor from a HMONITOR handle.
func monitorFromHandle(hmon uintptr) *Monitor {
	var info _MONITORINFOEXW
	info.CbSize = uint32(unsafe.Sizeof(info))
	getMonitorInfoW(hmon, &info) //nolint:errcheck — best effort
	return &Monitor{
		hmon:    hmon,
		name:    syscall.UTF16ToString(info.SzDevice[:]),
		primary: info.DwFlags&_MONITORINFOF_PRIMARY != 0,
	}
}

// GetName returns the monitor's device name.
func (m *Monitor) GetName() string { return m.name }

// GetPos returns the monitor's position in virtual screen coordinates.
func (m *Monitor) GetPos() (x, y int) {
	var info _MONITORINFOEXW
	info.CbSize = uint32(unsafe.Sizeof(info))
	getMonitorInfoW(m.hmon, &info)
	return int(info.RcMonitor.Left), int(info.RcMonitor.Top)
}

// GetWorkarea returns the monitor's work area (excludes taskbar/docks).
func (m *Monitor) GetWorkarea() (x, y, width, height int) {
	var info _MONITORINFOEXW
	info.CbSize = uint32(unsafe.Sizeof(info))
	getMonitorInfoW(m.hmon, &info)
	return int(info.RcWork.Left), int(info.RcWork.Top),
		int(info.RcWork.Right - info.RcWork.Left),
		int(info.RcWork.Bottom - info.RcWork.Top)
}

// GetPhysicalSize returns the monitor's physical dimensions in millimetres.
func (m *Monitor) GetPhysicalSize() (widthMM, heightMM int) {
	// Win32 doesn't expose physical size via MONITORINFO; we approximate
	// from the DPI and pixel dimensions.
	var dpiX, dpiY uint32
	getDpiForMonitor(m.hmon, _MDT_EFFECTIVE_DPI, &dpiX, &dpiY)
	if dpiX == 0 {
		dpiX = 96
	}
	if dpiY == 0 {
		dpiY = 96
	}
	vm := m.GetVideoMode()
	if vm == nil {
		return 0, 0
	}
	const inchToMM = 25.4
	return int(float64(vm.Width) / float64(dpiX) * inchToMM),
		int(float64(vm.Height) / float64(dpiY) * inchToMM)
}

// GetContentScale returns the DPI scale factors (1.0 = 96 DPI).
func (m *Monitor) GetContentScale() (x, y float32) {
	var dpiX, dpiY uint32
	if err := getDpiForMonitor(m.hmon, _MDT_EFFECTIVE_DPI, &dpiX, &dpiY); err != nil {
		return 1, 1
	}
	return float32(dpiX) / 96.0, float32(dpiY) / 96.0
}

// GetVideoMode returns the monitor's current video mode.
func (m *Monitor) GetVideoMode() *VidMode {
	name16, _ := syscall.UTF16PtrFromString(m.name)
	var dm _DEVMODEW
	dm.DmSize = uint16(unsafe.Sizeof(dm))
	if !enumDisplaySettingsW(name16, _ENUM_CURRENT_SETTINGS, &dm) {
		return nil
	}
	return &VidMode{
		Width:       int(dm.DmPelsWidth),
		Height:      int(dm.DmPelsHeight),
		RedBits:     8,
		GreenBits:   8,
		BlueBits:    8,
		RefreshRate: int(dm.DmDisplayFrequency),
	}
}

// GetVideoModes returns all available video modes for this monitor.
func (m *Monitor) GetVideoModes() []*VidMode {
	name16, _ := syscall.UTF16PtrFromString(m.name)
	var modes []*VidMode
	seen := map[[3]int]bool{}

	for i := uint32(0); ; i++ {
		var dm _DEVMODEW
		dm.DmSize = uint16(unsafe.Sizeof(dm))
		if !enumDisplaySettingsW(name16, i, &dm) {
			break
		}
		key := [3]int{int(dm.DmPelsWidth), int(dm.DmPelsHeight), int(dm.DmDisplayFrequency)}
		if seen[key] {
			continue
		}
		seen[key] = true
		modes = append(modes, &VidMode{
			Width:       int(dm.DmPelsWidth),
			Height:      int(dm.DmPelsHeight),
			RedBits:     8,
			GreenBits:   8,
			BlueBits:    8,
			RefreshRate: int(dm.DmDisplayFrequency),
		})
	}
	return modes
}

// ----------------------------------------------------------------------------
// SetMonitorCallback — monitor connect/disconnect notifications
// ----------------------------------------------------------------------------

var (
	winMonitorCb     func(*Monitor, PeripheralEvent)
	winCachedMonitors []*Monitor
)

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
// The callback is fired from WM_DISPLAYCHANGE in wndProc.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
	winMonitorCb = cb
	if cb != nil {
		winCachedMonitors, _ = GetMonitors()
	}
}

// diffAndFireMonitorCallbacks compares two monitor lists by name and fires
// Connected/Disconnected events for monitors that appeared or disappeared.
func diffAndFireMonitorCallbacks(old, cur []*Monitor, cb func(*Monitor, PeripheralEvent)) {
	oldSet := make(map[string]*Monitor, len(old))
	for _, m := range old {
		oldSet[m.name] = m
	}
	curSet := make(map[string]*Monitor, len(cur))
	for _, m := range cur {
		curSet[m.name] = m
	}
	for _, m := range cur {
		if _, existed := oldSet[m.name]; !existed {
			cb(m, Connected)
		}
	}
	for _, m := range old {
		if _, exists := curSet[m.name]; !exists {
			cb(m, Disconnected)
		}
	}
}

// ── Monitor gamma ─────────────────────────────────────────────────────────────

// GetGammaRamp returns the current gamma ramp for this monitor.
func (m *Monitor) GetGammaRamp() *GammaRamp {
	name16, _ := syscall.UTF16PtrFromString(m.name)
	hdc := createDCW(nil, name16, nil, 0)
	if hdc == 0 {
		return nil
	}
	defer deleteDC(hdc)
	var ramp _GAMMA_RAMP
	if !getDeviceGammaRamp(hdc, &ramp) {
		return nil
	}
	result := &GammaRamp{
		Red:   make([]uint16, 256),
		Green: make([]uint16, 256),
		Blue:  make([]uint16, 256),
	}
	for i := 0; i < 256; i++ {
		result.Red[i] = ramp[0][i]
		result.Green[i] = ramp[1][i]
		result.Blue[i] = ramp[2][i]
	}
	return result
}

// SetGammaRamp sets the gamma ramp for this monitor.
// ramp must contain exactly 256 entries in each channel.
func (m *Monitor) SetGammaRamp(ramp *GammaRamp) {
	if ramp == nil || len(ramp.Red) != 256 || len(ramp.Green) != 256 || len(ramp.Blue) != 256 {
		return
	}
	name16, _ := syscall.UTF16PtrFromString(m.name)
	hdc := createDCW(nil, name16, nil, 0)
	if hdc == 0 {
		return
	}
	defer deleteDC(hdc)
	var gr _GAMMA_RAMP
	for i := 0; i < 256; i++ {
		gr[0][i] = ramp.Red[i]
		gr[1][i] = ramp.Green[i]
		gr[2][i] = ramp.Blue[i]
	}
	setDeviceGammaRamp(hdc, &gr)
}

// SetGamma sets the monitor's gamma by computing a standard power-law ramp
// and applying it via SetGammaRamp.
func (m *Monitor) SetGamma(gamma float32) {
	if gamma <= 0 {
		return
	}
	inv := 1.0 / float64(gamma)
	ramp := &GammaRamp{
		Red:   make([]uint16, 256),
		Green: make([]uint16, 256),
		Blue:  make([]uint16, 256),
	}
	for i := 0; i < 256; i++ {
		v := math.Pow(float64(i)/255.0, inv)
		if v > 1.0 {
			v = 1.0
		}
		val := uint16(v*65535.0 + 0.5)
		ramp.Red[i] = val
		ramp.Green[i] = val
		ramp.Blue[i] = val
	}
	m.SetGammaRamp(ramp)
}
