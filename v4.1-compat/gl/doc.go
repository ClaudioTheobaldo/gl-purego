// Package gl provides CGO-less OpenGL 4.1 compatibility-profile bindings for Go.
//
// This package is a drop-in replacement for github.com/go-gl/gl/v4.1-compatibility/gl.
// It uses github.com/ebitengine/purego for dynamic symbol loading instead
// of CGO, which means:
//
//   - No C compiler required at build time
//   - Cross-compilation works out of the box (CGO_ENABLED=0)
//   - Truly static binaries on Linux
//
// The compatibility profile includes all deprecated fixed-function pipeline
// functions (glBegin/glEnd, glMatrixMode, glColor*, glNormal*, display lists,
// etc.) that were removed from the core profile at GL 3.2.  Use this package
// when maintaining legacy GL 1.x/2.x code on modern hardware.  For new code
// prefer the core-profile package at github.com/ClaudioTheobaldo/gl-purego/v4.1/gl.
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
// # Code generation
//
// The bulk of this package (constants, function variables, and wrappers) is
// generated from the official Khronos OpenGL XML registry using the generator
// in cmd/glgen. Do not edit package.go or init.go by hand; run go generate
// instead.
//
//go:generate go run ../../cmd/glgen/ -ver 4.1 -out . -ext -compat
package gl
