package gl

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Init loads all OpenGL function symbols using the platform's default
// proc-address resolver (wglGetProcAddress on Windows, dlsym on macOS/Linux).
//
// A current OpenGL context must exist before calling Init.
func Init() error {
	if err := initProcAddr(); err != nil {
		return fmt.Errorf("gl: failed to load OpenGL library: %w", err)
	}
	return InitWithProcAddrFunc(getProcAddress)
}

// InitWithProcAddrFunc loads all OpenGL function symbols using the supplied
// resolver. This is useful when the GL context is managed by a third-party
// windowing library that exposes its own GetProcAddress (e.g. GLFW).
//
//	gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
//	    return glfw.GetCurrentContext().GetProcAddress(name)
//	})
func InitWithProcAddrFunc(getProcAddr func(name string) unsafe.Pointer) error {
	var missing []string

	load := func(fptr any, name string, required bool) {
		addr := getProcAddr(name)
		if addr == nil {
			if required {
				missing = append(missing, name)
			}
			return
		}
		purego.RegisterFunc(fptr, uintptr(addr))
	}

	// --- Generated block: one load() call per GL entry point ---
	// (run `go generate ./v2.1/gl/` to regenerate from the OpenGL XML spec)

	load(&gpActiveTexture, "glActiveTexture", true)
	load(&gpAttachShader, "glAttachShader", true)
	load(&gpBindBuffer, "glBindBuffer", true)
	load(&gpBindFramebuffer, "glBindFramebuffer", true)
	load(&gpBindRenderbuffer, "glBindRenderbuffer", true)
	load(&gpBindTexture, "glBindTexture", true)
	load(&gpBindVertexArray, "glBindVertexArray", false) // optional in 2.1 (ARB)
	load(&gpBlendFunc, "glBlendFunc", true)
	load(&gpBufferData, "glBufferData", true)
	load(&gpBufferSubData, "glBufferSubData", true)
	load(&gpClear, "glClear", true)
	load(&gpClearColor, "glClearColor", true)
	load(&gpCompileShader, "glCompileShader", true)
	load(&gpCreateProgram, "glCreateProgram", true)
	load(&gpCreateShader, "glCreateShader", true)
	load(&gpDeleteBuffers, "glDeleteBuffers", true)
	load(&gpDeleteFramebuffers, "glDeleteFramebuffers", true)
	load(&gpDeleteProgram, "glDeleteProgram", true)
	load(&gpDeleteRenderbuffers, "glDeleteRenderbuffers", true)
	load(&gpDeleteShader, "glDeleteShader", true)
	load(&gpDeleteTextures, "glDeleteTextures", true)
	load(&gpDeleteVertexArrays, "glDeleteVertexArrays", false) // optional in 2.1
	load(&gpDisable, "glDisable", true)
	load(&gpDrawArrays, "glDrawArrays", true)
	load(&gpDrawElements, "glDrawElements", true)
	load(&gpEnable, "glEnable", true)
	load(&gpEnableVertexAttribArray, "glEnableVertexAttribArray", true)
	load(&gpFramebufferRenderbuffer, "glFramebufferRenderbuffer", true)
	load(&gpFramebufferTexture2D, "glFramebufferTexture2D", true)
	load(&gpGenBuffers, "glGenBuffers", true)
	load(&gpGenFramebuffers, "glGenFramebuffers", true)
	load(&gpGenRenderbuffers, "glGenRenderbuffers", true)
	load(&gpGenTextures, "glGenTextures", true)
	load(&gpGenVertexArrays, "glGenVertexArrays", false) // optional in 2.1
	load(&gpGetAttribLocation, "glGetAttribLocation", true)
	load(&gpGetError, "glGetError", true)
	load(&gpGetIntegerv, "glGetIntegerv", true)
	load(&gpGetProgramInfoLog, "glGetProgramInfoLog", true)
	load(&gpGetProgramiv, "glGetProgramiv", true)
	load(&gpGetShaderInfoLog, "glGetShaderInfoLog", true)
	load(&gpGetShaderiv, "glGetShaderiv", true)
	load(&gpGetString, "glGetString", true)
	load(&gpGetUniformLocation, "glGetUniformLocation", true)
	load(&gpLinkProgram, "glLinkProgram", true)
	load(&gpPixelStorei, "glPixelStorei", true)
	load(&gpReadBuffer, "glReadBuffer", true)
	load(&gpReadPixels, "glReadPixels", true)
	load(&gpRenderbufferStorage, "glRenderbufferStorage", true)
	load(&gpScissor, "glScissor", true)
	load(&gpShaderSource, "glShaderSource", true)
	load(&gpTexImage2D, "glTexImage2D", true)
	load(&gpTexParameteri, "glTexParameteri", true)
	load(&gpUniform1f, "glUniform1f", true)
	load(&gpUniform1i, "glUniform1i", true)
	load(&gpUniform2f, "glUniform2f", true)
	load(&gpUniform4f, "glUniform4f", true)
	load(&gpUniformMatrix4fv, "glUniformMatrix4fv", true)
	load(&gpUseProgram, "glUseProgram", true)
	load(&gpVertexAttribPointer, "glVertexAttribPointer", true)
	load(&gpViewport, "glViewport", true)

	// --- End generated block ---

	if len(missing) > 0 {
		return fmt.Errorf("gl: %d required functions not found: %v", len(missing), missing)
	}
	return nil
}

// function pointer slots — populated by InitWithProcAddrFunc.
// Named with a gp prefix (gl pointer) to match the original go-gl convention.
var (
	gpActiveTexture           func(texture uint32)
	gpAttachShader            func(program, shader uint32)
	gpBindBuffer              func(target, buffer uint32)
	gpBindFramebuffer         func(target, framebuffer uint32)
	gpBindRenderbuffer        func(target, renderbuffer uint32)
	gpBindTexture             func(target, texture uint32)
	gpBindVertexArray         func(array uint32)
	gpBlendFunc               func(sfactor, dfactor uint32)
	gpBufferData              func(target uint32, size int, data unsafe.Pointer, usage uint32)
	gpBufferSubData           func(target uint32, offset, size int, data unsafe.Pointer)
	gpClear                   func(mask uint32)
	gpClearColor              func(red, green, blue, alpha float32)
	gpCompileShader           func(shader uint32)
	gpCreateProgram           func() uint32
	gpCreateShader            func(xtype uint32) uint32
	gpDeleteBuffers           func(n int32, buffers *uint32)
	gpDeleteFramebuffers      func(n int32, framebuffers *uint32)
	gpDeleteProgram           func(program uint32)
	gpDeleteRenderbuffers     func(n int32, renderbuffers *uint32)
	gpDeleteShader            func(shader uint32)
	gpDeleteTextures          func(n int32, textures *uint32)
	gpDeleteVertexArrays      func(n int32, arrays *uint32)
	gpDisable                 func(cap uint32)
	gpDrawArrays              func(mode uint32, first, count int32)
	gpDrawElements            func(mode uint32, count int32, xtype uint32, indices unsafe.Pointer)
	gpEnable                  func(cap uint32)
	gpEnableVertexAttribArray func(index uint32)
	gpFramebufferRenderbuffer func(target, attachment, renderbuffertarget, renderbuffer uint32)
	gpFramebufferTexture2D    func(target, attachment, textarget, texture uint32, level int32)
	gpGenBuffers              func(n int32, buffers *uint32)
	gpGenFramebuffers         func(n int32, ids *uint32)
	gpGenRenderbuffers        func(n int32, renderbuffers *uint32)
	gpGenTextures             func(n int32, textures *uint32)
	gpGenVertexArrays         func(n int32, arrays *uint32)
	gpGetAttribLocation       func(program uint32, name *uint8) int32
	gpGetError                func() uint32
	gpGetIntegerv             func(pname uint32, data *int32)
	gpGetProgramInfoLog       func(program uint32, bufSize int32, length *int32, infoLog *uint8)
	gpGetProgramiv            func(program, pname uint32, params *int32)
	gpGetShaderInfoLog        func(shader uint32, bufSize int32, length *int32, infoLog *uint8)
	gpGetShaderiv             func(shader, pname uint32, params *int32)
	gpGetString               func(name uint32) *uint8
	gpGetUniformLocation      func(program uint32, name *uint8) int32
	gpLinkProgram             func(program uint32)
	gpPixelStorei             func(pname uint32, param int32)
	gpReadBuffer              func(src uint32)
	gpReadPixels              func(x, y, width, height int32, format, xtype uint32, pixels unsafe.Pointer)
	gpRenderbufferStorage     func(target, internalformat uint32, width, height int32)
	gpScissor                 func(x, y, width, height int32)
	gpShaderSource            func(shader uint32, count int32, xstring **uint8, length *int32)
	gpTexImage2D              func(target uint32, level, internalformat, width, height, border int32, format, xtype uint32, pixels unsafe.Pointer)
	gpTexParameteri           func(target, pname uint32, param int32)
	gpUniform1f               func(location int32, v0 float32)
	gpUniform1i               func(location, v0 int32)
	gpUniform2f               func(location int32, v0, v1 float32)
	gpUniform4f               func(location int32, v0, v1, v2, v3 float32)
	gpUniformMatrix4fv        func(location, count int32, transpose bool, value *float32)
	gpUseProgram              func(program uint32)
	gpVertexAttribPointer     func(index uint32, size int32, xtype uint32, normalized bool, stride int32, pointer unsafe.Pointer)
	gpViewport                func(x, y, width, height int32)
)
