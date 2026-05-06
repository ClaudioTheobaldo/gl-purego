package gles2

// glow-style *WithOffset overloads.  See v2.1/gl/withoffset.go for the
// rationale; only overloads whose underlying entry point is present in
// this package are included (no VertexAttribLPointer in GLES; no
// DrawElementsBaseVertex in GLES 3.0).

import "unsafe"

func offsetPtr(offset uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&offset))
}

// VertexAttribPointerWithOffset — see v2.1/gl/withoffset.go.
func VertexAttribPointerWithOffset(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset uintptr) {
	gpVertexAttribPointer(index, size, xtype, normalized, stride, offsetPtr(offset))
}

// VertexAttribIPointerWithOffset is the offset-typed overload of
// VertexAttribIPointer (integer attributes, GLES 3.0+).
func VertexAttribIPointerWithOffset(index uint32, size int32, xtype uint32, stride int32, offset uintptr) {
	gpVertexAttribIPointer(index, size, xtype, stride, offsetPtr(offset))
}

// DrawElementsWithOffset — see v2.1/gl/withoffset.go.
func DrawElementsWithOffset(mode uint32, count int32, xtype uint32, indices uintptr) {
	gpDrawElements(mode, count, xtype, offsetPtr(indices))
}

// GetVertexAttribPointerWithOffsetv — see v2.1/gl/withoffset.go.
func GetVertexAttribPointerWithOffsetv(index uint32, pname uint32, offset **uintptr) {
	gpGetVertexAttribPointerv(index, pname, unsafe.Pointer(offset))
}
