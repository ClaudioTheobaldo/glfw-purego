//go:build linux

package glfw

// GetPlatform returns the platform being used by GLFW.
// On Linux this always returns PlatformX11.
func GetPlatform() Platform { return PlatformX11 }

// PlatformSupported reports whether the given platform is supported on this OS.
// On Linux only PlatformX11 and AnyPlatform are supported.
func PlatformSupported(p Platform) bool {
	return p == PlatformX11 || p == AnyPlatform
}
