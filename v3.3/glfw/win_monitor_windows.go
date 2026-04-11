//go:build windows

package glfw

import (
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

// SetMonitorCallback registers a callback for monitor connect/disconnect events.
// Stub — not implemented on Windows yet.
func SetMonitorCallback(cb func(monitor *Monitor, event PeripheralEvent)) {
}
