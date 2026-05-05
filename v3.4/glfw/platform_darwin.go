//go:build darwin

package glfw

// GetPlatform returns the platform being used by GLFW.
// On macOS this always returns PlatformCocoa.
func GetPlatform() Platform { return PlatformCocoa }

// PlatformSupported reports whether the given platform is supported on this OS.
// On macOS only PlatformCocoa and AnyPlatform are supported.
func PlatformSupported(p Platform) bool {
	return p == PlatformCocoa || p == AnyPlatform
}
