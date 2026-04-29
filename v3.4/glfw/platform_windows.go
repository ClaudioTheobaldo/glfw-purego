//go:build windows

package glfw

// GetPlatform returns the platform being used by GLFW.
// On Windows this always returns PlatformWin32.
func GetPlatform() Platform { return PlatformWin32 }

// PlatformSupported reports whether the given platform is supported on this OS.
// On Windows only PlatformWin32 and AnyPlatform are supported.
func PlatformSupported(p Platform) bool {
	return p == PlatformWin32 || p == AnyPlatform
}
