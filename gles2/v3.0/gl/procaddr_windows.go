//go:build windows

package gl

import (
	"syscall"
	"unsafe"
)

// On Windows, OpenGL ES is typically provided by ANGLE (libGLESv2.dll).
// The loader resolves symbols via eglGetProcAddress (preferred, covers extensions)
// with a fallback to GetProcAddress on libGLESv2.dll directly.

var (
	libGLESv2dll       = syscall.NewLazyDLL("libGLESv2.dll")
	libEGLdll          = syscall.NewLazyDLL("libEGL.dll")
	procEGLGetProcAddr = libEGLdll.NewProc("eglGetProcAddress")
)

func getProcAddress(name string) unsafe.Pointer {
	cname, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil
	}
	// eglGetProcAddress covers both core and extension symbols on ANGLE.
	addr, _, _ := procEGLGetProcAddr.Call(uintptr(unsafe.Pointer(cname)))
	if addr == 0 {
		// Fallback: look directly in the DLL export table.
		addr, err = syscall.GetProcAddress(syscall.Handle(libGLESv2dll.Handle()), name)
		if err != nil {
			return nil
		}
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

func initProcAddr() error {
	if err := libEGLdll.Load(); err != nil {
		return err
	}
	return libGLESv2dll.Load()
}
