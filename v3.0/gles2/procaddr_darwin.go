//go:build darwin

package gles2

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

// macOS does not ship OpenGL ES. This file is a stub that returns an error
// at runtime so that programs fail gracefully rather than at link time.
// Use a simulator or ANGLE port if you need GLES on macOS.

var libGLESv2 uintptr

func initProcAddr() error {
	var err error
	// Try ANGLE if available (e.g. via Homebrew or bundled with the app).
	for _, path := range []string{
		"libGLESv2.dylib",
		"@rpath/libGLESv2.dylib",
	} {
		libGLESv2, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			return nil
		}
	}
	return err
}

func getProcAddress(name string) unsafe.Pointer {
	if libGLESv2 == 0 {
		return nil
	}
	sym, err := purego.Dlsym(libGLESv2, name)
	if err != nil || sym == 0 {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&sym))
}
