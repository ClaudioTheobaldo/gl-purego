// Package gl provides CGO-less OpenGL 4.6 core-profile bindings for Go.
//
// This package is a drop-in replacement for github.com/go-gl/gl/v4.6-core/gl.
// It uses github.com/ebitengine/purego for dynamic symbol loading instead
// of CGO, which means:
//
//   - No C compiler required at build time
//   - Cross-compilation works out of the box (CGO_ENABLED=0)
//   - Truly static binaries on Linux
//
// OpenGL 4.6 is the latest desktop OpenGL specification, adding direct state
// access (DSA), SPIR-V shader support, and additional compute/geometry
// features.
//
// # Usage
//
// Before calling any GL function, a current OpenGL context must exist and
// Init (or InitWithProcAddrFunc) must be called:
//
//	if err := gl.Init(); err != nil {
//	    log.Fatal(err)
//	}
//
// The rest of the API is identical to github.com/go-gl/gl/v4.6-core/gl.
//
// # Code generation
//
// The bulk of this package (constants, function variables, and wrappers) is
// generated from the official Khronos OpenGL XML registry using the generator
// in cmd/glgen. Do not edit package.go or init.go by hand; run go generate
// instead.
//
//go:generate go run ../../cmd/glgen/ -ver 4.6 -out .
package gl
