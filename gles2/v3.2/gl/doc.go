// Package gl provides CGO-less OpenGL ES 3.2 bindings for Go.
//
// It uses github.com/ebitengine/purego for dynamic symbol loading instead of
// CGO, which means no C compiler is required at build time.
//
// # Usage
//
// Before calling any GL function, a current OpenGL ES context must exist and
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
//go:generate go run ../../../cmd/glgen/ -api gles2 -ver 3.2 -out . -ext
package gl

