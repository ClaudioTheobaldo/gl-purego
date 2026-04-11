//go:build darwin

package gl

import (
	"github.com/ebitengine/purego"
	"unsafe"
)

var libGL uintptr

func initProcAddr() error {
	var err error
	libGL, err = purego.Dlopen(
		"/System/Library/Frameworks/OpenGL.framework/OpenGL",
		purego.RTLD_LAZY|purego.RTLD_GLOBAL,
	)
	return err
}

// getProcAddress resolves an OpenGL symbol via dlsym(RTLD_DEFAULT).
// On macOS all OpenGL symbols are pre-loaded into the process image when
// the OpenGL framework is linked, so RTLD_DEFAULT resolves everything.
func getProcAddress(name string) unsafe.Pointer {
	sym, err := purego.Dlsym(purego.RTLD_DEFAULT, name)
	if err != nil || sym == 0 {
		// Fallback: look directly in the framework handle.
		sym, err = purego.Dlsym(libGL, name)
		if err != nil {
			return nil
		}
	}
	return unsafe.Pointer(sym)
}
