//go:build linux && wayland

package glfw

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ── Vulkan surface support (Wayland backend) ──────────────────────────────────

var (
	wlVulkanOnce            sync.Once
	wlVulkanHandle          uintptr
	wlVulkanErr             error
	wlVkGetInstanceProcAddr func(instance uintptr, name uintptr) uintptr
)

func loadWaylandVulkan() error {
	wlVulkanOnce.Do(func() {
		for _, lib := range []string{"libvulkan.so.1", "libvulkan.so"} {
			wlVulkanHandle, wlVulkanErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if wlVulkanErr == nil {
				break
			}
		}
		if wlVulkanErr != nil {
			return
		}
		purego.RegisterLibFunc(&wlVkGetInstanceProcAddr, wlVulkanHandle, "vkGetInstanceProcAddr")
	})
	return wlVulkanErr
}

// VulkanSupported reports whether a Vulkan loader is available on this system.
func VulkanSupported() bool { return loadWaylandVulkan() == nil }

// GetVulkanGetInstanceProcAddress returns the address of vkGetInstanceProcAddr
// (cast to unsafe.Pointer), or nil if the Vulkan loader could not be loaded.
func GetVulkanGetInstanceProcAddress() unsafe.Pointer {
	if loadWaylandVulkan() != nil {
		return nil
	}
	addr, err := purego.Dlsym(wlVulkanHandle, "vkGetInstanceProcAddr")
	if err != nil || addr == 0 {
		return nil
	}
	return nativePtrFromUintptr(addr)
}

// GetRequiredInstanceExtensions returns the Vulkan instance extensions required
// to create a Wayland window surface.
func GetRequiredInstanceExtensions() []string {
	if !VulkanSupported() {
		return nil
	}
	return []string{"VK_KHR_surface", "VK_KHR_wayland_surface"}
}

// VkWaylandSurfaceCreateInfoKHR (structure type 1000006000).
// Layout mirrors the C struct — 40 bytes on 64-bit:
//   offset  0: sType   uint32
//   offset  4: _pad    [4]byte
//   offset  8: pNext   uintptr
//   offset 16: flags   uint32
//   offset 20: _pad    [4]byte
//   offset 24: display uintptr  (wl_display*)
//   offset 32: surface uintptr  (wl_surface*)
type _vkWaylandSurfaceCreateInfo struct {
	sType   uint32
	_       [4]byte
	pNext   uintptr
	flags   uint32
	_       [4]byte
	display uintptr
	surface uintptr
}

const _vkSTypeWaylandSurface = uint32(1000006000)

// CreateWindowSurface creates a VkSurfaceKHR for this Wayland window.
//
// instance must be a VkInstance (cast to unsafe.Pointer).
// allocator may be nil.
// Returns the VkSurfaceKHR as an unsafe.Pointer.
func (w *Window) CreateWindowSurface(instance, allocator unsafe.Pointer) (unsafe.Pointer, error) {
	if err := loadWaylandVulkan(); err != nil {
		return nil, fmt.Errorf("Vulkan not available: %w", err)
	}
	fnName, _ := syscall.BytePtrFromString("vkCreateWaylandSurfaceKHR")
	addr := wlVkGetInstanceProcAddr(uintptr(instance), uintptr(unsafe.Pointer(fnName)))
	if addr == 0 {
		return nil, errors.New("vkGetInstanceProcAddr: vkCreateWaylandSurfaceKHR not found")
	}
	var createFn func(instance, info, allocator, surface uintptr) int32
	purego.RegisterFunc(&createFn, addr)

	info := _vkWaylandSurfaceCreateInfo{
		sType:   _vkSTypeWaylandSurface,
		display: wl.display,
		surface: w.handle,
	}
	var surface uintptr
	r := createFn(
		uintptr(instance),
		uintptr(unsafe.Pointer(&info)),
		uintptr(allocator),
		uintptr(unsafe.Pointer(&surface)),
	)
	if r != 0 {
		return nil, fmt.Errorf("vkCreateWaylandSurfaceKHR: error %d", r)
	}
	return nativePtrFromUintptr(surface), nil
}
