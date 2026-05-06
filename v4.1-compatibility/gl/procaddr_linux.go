//go:build linux && !gles

package gl

import (
	"github.com/ebitengine/purego"
	"unsafe"
)

var (
	libGL              uintptr
	glXGetProcAddrARB  func(name *byte) uintptr
)

func initProcAddr() error {
	var err error
	// Try versioned name first, then unversioned.
	for _, path := range []string{"libGL.so.1", "libGL.so"} {
		libGL, err = purego.Dlopen(path, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	purego.RegisterLibFunc(&glXGetProcAddrARB, libGL, "glXGetProcAddressARB")
	return nil
}

// getProcAddress resolves an OpenGL symbol via glXGetProcAddressARB.
// This works for both core functions and extensions on Linux/GLX.
func getProcAddress(name string) unsafe.Pointer {
	b := append([]byte(name), 0)
	addr := glXGetProcAddrARB(&b[0])
	if addr == 0 {
		return nil
	}
	return unsafe.Pointer(addr)
}
