package gl

// glow-style *WithOffset overloads.  See v2.1/gl/withoffset.go for the
// rationale.  Desktop GL 3.3+ has the full set: VertexAttrib{,I,L}Pointer,
// DrawElements{,BaseVertex}, GetVertexAttribPointerv.

import "unsafe"

func offsetPtr(offset uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&offset))
}

// VertexAttribPointerWithOffset — see v2.1/gl/withoffset.go.
func VertexAttribPointerWithOffset(index uint32, size int32, xtype uint32, normalized bool, stride int32, offset uintptr) {
	gpVertexAttribPointer(index, size, xtype, normalized, stride, offsetPtr(offset))
}

// VertexAttribIPointerWithOffset — see v3.0/gles2/withoffset.go.
func VertexAttribIPointerWithOffset(index uint32, size int32, xtype uint32, stride int32, offset uintptr) {
	gpVertexAttribIPointer(index, size, xtype, stride, offsetPtr(offset))
}

// VertexAttribLPointerWithOffset — see v2.1/gl/withoffset.go.
func VertexAttribLPointerWithOffset(index uint32, size int32, xtype uint32, stride int32, offset uintptr) {
	gpVertexAttribLPointer(index, size, xtype, stride, offsetPtr(offset))
}

// DrawElementsWithOffset — see v2.1/gl/withoffset.go.
func DrawElementsWithOffset(mode uint32, count int32, xtype uint32, indices uintptr) {
	gpDrawElements(mode, count, xtype, offsetPtr(indices))
}

// DrawElementsBaseVertexWithOffset — see v2.1/gl/withoffset.go.
func DrawElementsBaseVertexWithOffset(mode uint32, count int32, xtype uint32, indices uintptr, basevertex int32) {
	gpDrawElementsBaseVertex(mode, count, xtype, offsetPtr(indices), basevertex)
}

// GetVertexAttribPointerWithOffsetv — see v2.1/gl/withoffset.go.
func GetVertexAttribPointerWithOffsetv(index uint32, pname uint32, offset **uintptr) {
	gpGetVertexAttribPointerv(index, pname, unsafe.Pointer(offset))
}
