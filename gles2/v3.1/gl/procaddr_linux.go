//go:build linux

package gl

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	libGLESv2         uintptr
	eglGetProcAddress func(name *byte) uintptr
)

func initProcAddr() error {
	var err error
	for _, path := range []string{"libGLESv2.so.2", "libGLESv2.so"} {
		libGLESv2, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	// libEGL provides eglGetProcAddress for extension lookup.
	var libEGL uintptr
	for _, path := range []string{"libEGL.so.1", "libEGL.so"} {
		libEGL, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}
	if err != nil {
		// Non-fatal: fall back to dlsym-only resolution.
		return nil
	}
	purego.RegisterLibFunc(&eglGetProcAddress, libEGL, "eglGetProcAddress")
	return nil
}

func getProcAddress(name string) unsafe.Pointer {
	b := append([]byte(name), 0)
	// eglGetProcAddress works for both core and extension symbols on Mesa/ANGLE.
	if eglGetProcAddress != nil {
		if addr := eglGetProcAddress(&b[0]); addr != 0 {
			return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
		}
	}
	// Fallback: dlsym on libGLESv2 directly.
	addr, err := purego.Dlsym(libGLESv2, name)
	if err != nil || addr == 0 {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}
