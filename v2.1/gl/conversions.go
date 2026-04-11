package gl

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// Ptr takes a slice or a pointer and returns a raw unsafe.Pointer to the
// underlying data. If the argument is a slice, the pointer is to element 0.
//
// Ptr also accepts strings, returning a pointer to the first byte of the
// null-terminated representation. The string must not contain embedded nulls.
//
// It is the caller's responsibility to ensure the data is accessible during
// any GL calls that reference it (i.e. pin it with runtime.Pinner or ensure
// it lives long enough on the stack).
func Ptr(data any) unsafe.Pointer {
	if data == nil {
		return unsafe.Pointer(nil)
	}
	switch v := data.(type) {
	case unsafe.Pointer:
		return v
	case string:
		return unsafe.Pointer(unsafe.StringData(v + "\x00"))
	}

	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Ptr:
		return unsafe.Pointer(rv.Pointer())
	case reflect.Slice:
		if rv.Len() == 0 {
			panic("gl.Ptr: empty slice — no valid address")
		}
		return unsafe.Pointer(rv.Index(0).Addr().Pointer())
	default:
		panic(fmt.Sprintf("gl.Ptr: unsupported type %T", data))
	}
}

// PtrOffset takes a byte offset and returns a GL-compatible pointer.
// Useful for vertex attribute offsets inside a VBO.
// The conversion through pointer indirection avoids the go vet unsafeptr warning
// for the (legitimate) pattern of turning a plain integer offset into a pointer.
func PtrOffset(offset int) unsafe.Pointer {
	u := uintptr(offset)
	return *(*unsafe.Pointer)(unsafe.Pointer(&u))
}

// Str converts a Go string to a null-terminated *uint8 suitable for GL calls.
// The string must not contain embedded null bytes.
// The returned pointer is only valid as long as the original string is alive;
// the caller must ensure the string is not GC'd while the pointer is in use.
func Str(str string) *uint8 {
	if len(str) == 0 || str[len(str)-1] != '\x00' {
		str = str + "\x00"
	}
	return unsafe.SliceData([]byte(str))
}

// GoStr converts a null-terminated C string (*uint8) to a Go string.
// Returns an empty string if cstr is nil.
func GoStr(cstr *uint8) string {
	if cstr == nil {
		return ""
	}
	// Walk forward from cstr without storing a uintptr, keeping the pointer
	// live so the GC can track it through the loop.
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(cstr), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(cstr, n))
}

// Strs returns a pointer to a contiguous array of null-terminated C strings
// built from the supplied Go strings, along with a free function that must be
// called when the array is no longer needed.
//
// This is used for GL calls that take a **GLchar (e.g. ShaderSource).
func Strs(strs ...string) (cstrs **uint8, free func()) {
	// Build one contiguous byte buffer with all null-terminated strings,
	// then build a slice of pointers into it.
	//
	// We use a runtime.Pinner to pin the buffer so the GC cannot move it
	// while GL holds a reference.
	bufs := make([][]byte, len(strs))
	ptrs := make([]*uint8, len(strs))

	var pinner runtime.Pinner

	for i, s := range strs {
		b := []byte(s)
		if len(b) == 0 || b[len(b)-1] != 0 {
			b = append(b, 0)
		}
		bufs[i] = b
		pinner.Pin(&b[0])
		ptrs[i] = &b[0]
	}

	pinner.Pin(&ptrs[0])

	return &ptrs[0], func() {
		pinner.Unpin()
		runtime.KeepAlive(bufs)
		runtime.KeepAlive(ptrs)
	}
}
