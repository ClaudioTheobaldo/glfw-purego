//go:build darwin

// vulkan_darwin.go — Vulkan surface creation via MoltenVK + VK_EXT_metal_surface.
//
// Uses the CAMetalLayer attached to the window's content view (set up in
// CreateWindow) as the native metal surface.  The Vulkan loader / MoltenVK
// library is loaded lazily on first use.

package glfw

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// ── MoltenVK / Vulkan loader ──────────────────────────────────────────────────

// Candidate library paths, tried in order.
var mvkPaths = []string{
	"/opt/homebrew/lib/libMoltenVK.dylib",    // Apple Silicon Homebrew
	"/usr/local/lib/libMoltenVK.dylib",       // Intel Homebrew / manual install
	"libMoltenVK.dylib",                      // bundled alongside the binary
	"libvulkan.dylib",                        // Vulkan SDK loader (preferred)
	"libvulkan.1.dylib",                      // versioned Vulkan SDK loader
}

var (
	vulkanOnce             sync.Once
	vulkanLib              uintptr
	vulkanErr              error
	vkGetInstanceProcAddr  func(instance uintptr, pName string) uintptr
)

func loadVulkan() error {
	vulkanOnce.Do(func() {
		for _, path := range mvkPaths {
			h, err := purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_LOCAL)
			if err == nil {
				vulkanLib = h
				break
			}
		}
		if vulkanLib == 0 {
			vulkanErr = errors.New("glfw: Vulkan/MoltenVK loader not found")
			return
		}
		purego.RegisterLibFunc(&vkGetInstanceProcAddr, vulkanLib, "vkGetInstanceProcAddr")
	})
	return vulkanErr
}

// ── Public Vulkan API ─────────────────────────────────────────────────────────

// VulkanSupported returns true if a Vulkan/MoltenVK loader is available.
func VulkanSupported() bool { return loadVulkan() == nil }

// GetRequiredInstanceExtensions returns the Vulkan instance extensions required
// to create a Metal surface on macOS.
func GetRequiredInstanceExtensions() []string {
	if !VulkanSupported() {
		return nil
	}
	return []string{"VK_KHR_surface", "VK_EXT_metal_surface"}
}

// ── VkMetalSurfaceCreateInfoEXT ────────────────────────────────────────────────
//
// Layout (64-bit):
//   offset  0: sType  uint32   (VK_STRUCTURE_TYPE_METAL_SURFACE_CREATE_INFO_EXT = 1000217000)
//   offset  4: _pad   [4]byte
//   offset  8: pNext  uintptr  (NULL)
//   offset 16: flags  uint32   (0)
//   offset 20: _pad   [4]byte
//   offset 24: pLayer uintptr  (CAMetalLayer*)
//
// Total: 32 bytes.

type _vkMetalSurfaceCreateInfo struct {
	sType  uint32
	_      [4]byte
	pNext  uintptr
	flags  uint32
	_      [4]byte
	pLayer uintptr
}

const _vkSTypeMetalSurface = uint32(1000217000)

// CreateWindowSurface creates a VkSurfaceKHR for this window using
// VK_EXT_metal_surface and the CAMetalLayer attached during CreateWindow.
//
// instance must be a VkInstance cast to unsafe.Pointer.
// allocator may be nil.
// Returns the VkSurfaceKHR as an unsafe.Pointer, or an error.
func (w *Window) CreateWindowSurface(instance, allocator unsafe.Pointer) (unsafe.Pointer, error) {
	if err := loadVulkan(); err != nil {
		return nil, fmt.Errorf("glfw: Vulkan not available: %w", err)
	}
	if w.metalLayer == 0 {
		return nil, errors.New("glfw: window has no CAMetalLayer — Vulkan surface cannot be created")
	}

	// Resolve vkCreateMetalSurfaceEXT through the instance dispatch table.
	addr := vkGetInstanceProcAddr(uintptr(instance), "vkCreateMetalSurfaceEXT")
	if addr == 0 {
		return nil, errors.New("glfw: vkGetInstanceProcAddr: vkCreateMetalSurfaceEXT not found")
	}

	info := _vkMetalSurfaceCreateInfo{
		sType:  _vkSTypeMetalSurface,
		pLayer: w.metalLayer,
	}
	var surface uintptr
	r, _, _ := purego.SyscallN(addr,
		uintptr(instance),
		uintptr(unsafe.Pointer(&info)),
		uintptr(allocator),
		uintptr(unsafe.Pointer(&surface)),
	)
	if r != 0 {
		return nil, fmt.Errorf("glfw: vkCreateMetalSurfaceEXT: VkResult %d", r)
	}
	return nativePtrFromUintptr(surface), nil
}
