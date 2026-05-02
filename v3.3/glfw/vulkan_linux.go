//go:build linux && !wayland

package glfw

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ----------------------------------------------------------------------------
// Vulkan loader — libvulkan.so.1
// ----------------------------------------------------------------------------

var (
	vulkanOnce            sync.Once
	vulkanHandle          uintptr
	vulkanErr             error
	vkGetInstanceProcAddr func(instance uintptr, name uintptr) uintptr
)

func loadVulkan() error {
	vulkanOnce.Do(func() {
		for _, lib := range []string{"libvulkan.so.1", "libvulkan.so"} {
			vulkanHandle, vulkanErr = purego.Dlopen(lib, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if vulkanErr == nil {
				break
			}
		}
		if vulkanErr != nil {
			return
		}
		purego.RegisterLibFunc(&vkGetInstanceProcAddr, vulkanHandle, "vkGetInstanceProcAddr")
	})
	return vulkanErr
}

// VulkanSupported reports whether a Vulkan loader is available on this system.
func VulkanSupported() bool { return loadVulkan() == nil }

// GetRequiredInstanceExtensions returns the Vulkan instance extensions required
// by glfw-purego to create an Xlib window surface.
func GetRequiredInstanceExtensions() []string {
	if !VulkanSupported() {
		return nil
	}
	return []string{"VK_KHR_surface", "VK_KHR_xlib_surface"}
}

// ----------------------------------------------------------------------------
// VkXlibSurfaceCreateInfoKHR — 40 bytes
// sType  uint32 @0   (pad4 @4)
// pNext  uintptr @8
// flags  uint32 @16  (pad4 @20)
// dpy    uintptr @24  (Display*)
// window uint64 @32   (XID = unsigned long)
// ----------------------------------------------------------------------------

type _vkXlibSurfaceCreateInfo struct {
	sType  uint32
	_      [4]byte
	pNext  uintptr
	flags  uint32
	_      [4]byte
	dpy    uintptr
	window uint64
}

const _vkSTypeXlibSurface = uint32(1000004000)

// CreateWindowSurface creates a VkSurfaceKHR for this window.
//
// instance must be a VkInstance (cast to unsafe.Pointer).
// allocator may be nil.
// Returns the VkSurfaceKHR as an unsafe.Pointer.
func (w *Window) CreateWindowSurface(instance, allocator unsafe.Pointer) (unsafe.Pointer, error) {
	if err := loadVulkan(); err != nil {
		return nil, fmt.Errorf("Vulkan not available: %w", err)
	}
	fnName, _ := syscall.BytePtrFromString("vkCreateXlibSurfaceKHR")
	addr := vkGetInstanceProcAddr(uintptr(instance), uintptr(unsafe.Pointer(fnName)))
	if addr == 0 {
		return nil, errors.New("vkGetInstanceProcAddr: vkCreateXlibSurfaceKHR not found")
	}
	var createFn func(instance, info, allocator, surface uintptr) int32
	purego.RegisterFunc(&createFn, addr)

	info := _vkXlibSurfaceCreateInfo{
		sType:  _vkSTypeXlibSurface,
		dpy:    x11Display,
		window: uint64(w.handle),
	}
	var surface uintptr
	r := createFn(
		uintptr(instance),
		uintptr(unsafe.Pointer(&info)),
		uintptr(allocator),
		uintptr(unsafe.Pointer(&surface)),
	)
	if r != 0 {
		return nil, fmt.Errorf("vkCreateXlibSurfaceKHR: error %d", r)
	}
	return nativePtrFromUintptr(surface), nil
}
