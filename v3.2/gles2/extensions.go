package gles2

import "strings"

// IsExtensionSupported reports whether the named OpenGL extension
// (e.g. "GL_ARB_framebuffer_object") is available in the current context.
// Init (or InitWithProcAddrFunc) must have been called before using this function.
//
// On GL 3.0+ and GLES 3.0+ contexts the function uses glGetStringi to
// enumerate extensions one by one.  On older GL 2.1 contexts it falls back
// to the space-separated string returned by glGetString(GL_EXTENSIONS).
func IsExtensionSupported(name string) bool {
	// GL 3.0+ / GLES 3.0+: enumerate extensions via glGetStringi.
	if gpGetStringi != nil {
		var n int32
		GetIntegerv(NUM_EXTENSIONS, &n)
		for i := int32(0); i < n; i++ {
			if GoStr(GetStringi(EXTENSIONS, uint32(i))) == name {
				return true
			}
		}
		return false
	}
	// GL 2.1 fallback: space-separated extension string.
	for _, e := range strings.Fields(GoStr(GetString(EXTENSIONS))) {
		if e == name {
			return true
		}
	}
	return false
}
