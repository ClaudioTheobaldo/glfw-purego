module github.com/ClaudioTheobaldo/glfw-purego/examples/linux_smoke

go 1.25.0

replace github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw => ../../v3.3/glfw

require github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw v0.0.0-00010101000000-000000000000

require (
	github.com/ebitengine/purego v0.8.2 // indirect
	golang.org/x/sys v0.43.0 // indirect
)
