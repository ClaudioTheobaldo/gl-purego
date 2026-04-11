//go:build windows

package gl

import (
	"syscall"
	"unsafe"
)

var (
	opengl32dll       = syscall.NewLazyDLL("opengl32.dll")
	procWglGetProcAddr = opengl32dll.NewProc("wglGetProcAddress")
)

// getProcAddress resolves an OpenGL symbol by name.
//
// Strategy:
//  1. wglGetProcAddress — covers ARB/EXT extensions and core functions
//     promoted to extensions (everything above GL 1.1).
//  2. GetProcAddress on opengl32.dll — covers the base GL 1.1 entry points
//     that wglGetProcAddress refuses to return.
func getProcAddress(name string) unsafe.Pointer {
	cname, err := syscall.BytePtrFromString(name)
	if err != nil {
		return nil
	}
	addr, _, _ := procWglGetProcAddr.Call(uintptr(unsafe.Pointer(cname)))
	if addr == 0 {
		// Fallback to the DLL's own export table (GL 1.1 base functions).
		addr, err = syscall.GetProcAddress(syscall.Handle(opengl32dll.Handle()), name)
		if err != nil {
			return nil
		}
	}
	return unsafe.Pointer(addr)
}

func initProcAddr() error {
	return opengl32dll.Load()
}
