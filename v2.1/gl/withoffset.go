package gl

// glow-style *WithOffset overloads.
//
// Upstream go-gl/gl ships these as "glow overloads" for any GL entry point
// whose final pointer argument is interpreted as a byte offset into the
// currently-bound buffer (VBO/IBO) rather than a real client-memory pointer.
// Passing a real `unsafe.Pointer` cast from a `uintptr` would trip Go's
// `go vet` "possible misuse of unsafe.Pointer" check; the *WithOffset
// variant takes `uintptr` and uses the indirection trick below to convert.
//
// Each helper is a thin wrapper over the existing `gp*` function pointer
// loaded by init().  Only entry points actually present in this package
// have a matching overload — see the per-package coverage matrix in the
// project README.

import "unsafe"

// offsetPtr converts a buffer offset (uintptr) to unsafe.Pointer without
// tripping go vet.  The result is intentionally not a Go-managed address;
// it's a byte offset into the currently-bound buffer object that the GL
// driver dereferences relative to the VBO/IBO base.
func offsetPtr(offset uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&offset))
}

// VertexAttribPointerWithOffset is the offset-typed overload of
// VertexAttribPointer.  The `offset` argument is a byte offset into the
// buffer currently bound to GL_ARRAY_BUFFER.
func VertexAttribPointerWithOffset(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset uintptr) {
	gpVertexAttribPointer(index, size, xtype, normalized, stride, offsetPtr(offset))
}

// VertexAttribLPointerWithOffset is the offset-typed overload of
// VertexAttribLPointer.  Available since GL 4.1; this package's generator
// includes it via the ARB_vertex_attrib_64bit extension header.
func VertexAttribLPointerWithOffset(index uint32, size int32, xtype uint32, stride int32, offset uintptr) {
	gpVertexAttribLPointer(index, size, xtype, stride, offsetPtr(offset))
}

// DrawElementsWithOffset is the offset-typed overload of DrawElements.
// `indices` is a byte offset into the buffer currently bound to
// GL_ELEMENT_ARRAY_BUFFER.
func DrawElementsWithOffset(mode uint32, count int32, xtype uint32, indices uintptr) {
	gpDrawElements(mode, count, xtype, offsetPtr(indices))
}

// DrawElementsBaseVertexWithOffset is the offset-typed overload of
// DrawElementsBaseVertex.
func DrawElementsBaseVertexWithOffset(mode uint32, count int32, xtype uint32, indices uintptr, basevertex int32) {
	gpDrawElementsBaseVertex(mode, count, xtype, offsetPtr(indices), basevertex)
}

// GetVertexAttribPointerWithOffsetv is the offset-typed overload of
// GetVertexAttribPointerv.  `offset` is a pointer to a uintptr that
// receives the buffer-relative offset of the queried attribute.
func GetVertexAttribPointerWithOffsetv(index uint32, pname uint32, offset **uintptr) {
	gpGetVertexAttribPointerv(index, pname, unsafe.Pointer(offset))
}
