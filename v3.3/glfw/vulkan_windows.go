//go:build windows

package glfw

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ----------------------------------------------------------------------------
// Vulkan loader — vulkan-1.dll
// ----------------------------------------------------------------------------

var (
	vulkanOnce              sync.Once
	modVulkan               *windows.LazyDLL
	vulkanErr               error
	procVkGetInstProcAddr   *windows.LazyProc
)

func loadVulkan() error {
	vulkanOnce.Do(func() {
		modVulkan = windows.NewLazyDLL("vulkan-1.dll")
		vulkanErr = modVulkan.Load()
		if vulkanErr == nil {
			procVkGetInstProcAddr = modVulkan.NewProc("vkGetInstanceProcAddr")
		}
	})
	return vulkanErr
}

// VulkanSupported reports whether a Vulkan loader is available on this system.
func VulkanSupported() bool { return loadVulkan() == nil }

// GetRequiredInstanceExtensions returns the Vulkan instance extensions required
// by glfw-purego to create a Win32 window surface.
func GetRequiredInstanceExtensions() []string {
	if !VulkanSupported() {
		return nil
	}
	return []string{"VK_KHR_surface", "VK_KHR_win32_surface"}
}

// ----------------------------------------------------------------------------
// VkWin32SurfaceCreateInfoKHR — 40 bytes
// sType   uint32 @0   (pad4 @4)
// pNext   uintptr @8
// flags   uint32 @16  (pad4 @20)
// hinstance uintptr @24
// hwnd    uintptr @32
// ----------------------------------------------------------------------------

type _vkWin32SurfaceCreateInfo struct {
	sType    uint32
	_        [4]byte
	pNext    uintptr
	flags    uint32
	_        [4]byte
	hinstance uintptr
	hwnd     uintptr
}

const _vkSTypeWin32Surface = uint32(1000009000)

// CreateWindowSurface creates a VkSurfaceKHR for this window.
//
// instance must be a VkInstance (cast to unsafe.Pointer).
// allocator may be nil.
// Returns the VkSurfaceKHR as an unsafe.Pointer.
func (w *Window) CreateWindowSurface(instance, allocator unsafe.Pointer) (unsafe.Pointer, error) {
	if err := loadVulkan(); err != nil {
		return nil, fmt.Errorf("Vulkan not available: %w", err)
	}
	fnName, _ := syscall.BytePtrFromString("vkCreateWin32SurfaceKHR")
	addr, _, _ := procVkGetInstProcAddr.Call(
		uintptr(instance),
		uintptr(unsafe.Pointer(fnName)),
	)
	if addr == 0 {
		return nil, errors.New("vkGetInstanceProcAddr: vkCreateWin32SurfaceKHR not found")
	}
	hinstance, err := getModuleHandleW(nil)
	if err != nil {
		hinstance = 0 // non-fatal — some drivers accept NULL
	}
	info := _vkWin32SurfaceCreateInfo{
		sType:     _vkSTypeWin32Surface,
		hinstance: hinstance,
		hwnd:      w.handle,
	}
	var surface uintptr
	r, _, _ := syscall.SyscallN(addr,
		uintptr(instance),
		uintptr(unsafe.Pointer(&info)),
		uintptr(allocator),
		uintptr(unsafe.Pointer(&surface)),
	)
	if r != 0 {
		return nil, fmt.Errorf("vkCreateWin32SurfaceKHR: error %d", r)
	}
	return nativePtrFromUintptr(surface), nil
}
