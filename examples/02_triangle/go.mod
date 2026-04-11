module github.com/ClaudioTheobaldo/gl-purego/examples/02_triangle

go 1.25.0

replace (
	github.com/ClaudioTheobaldo/gl-purego => ../..
	github.com/ClaudioTheobaldo/glfw-purego => ../../../glfw-purego
	github.com/ClaudioTheobaldo/gl-purego/examples/glutil => ../glutil
)

require (
	github.com/ClaudioTheobaldo/gl-purego v0.0.0-00010101000000-000000000000
	github.com/ClaudioTheobaldo/glfw-purego v0.0.0-00010101000000-000000000000
	github.com/ClaudioTheobaldo/gl-purego/examples/glutil v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/purego v0.8.2 // indirect
	golang.org/x/sys v0.43.0 // indirect
)
