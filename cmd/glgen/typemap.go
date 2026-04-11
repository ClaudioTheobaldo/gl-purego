package main

import "strings"

// goFuncName strips the "gl" prefix: "glActiveTexture" → "ActiveTexture".
func goFuncName(glName string) string {
	if strings.HasPrefix(glName, "gl") {
		return glName[2:]
	}
	return glName
}

// varName returns the private function-pointer variable name for a GL entry
// point: "glActiveTexture" → "gpActiveTexture".
func varName(glName string) string {
	if strings.HasPrefix(glName, "gl") {
		return "gp" + glName[2:]
	}
	return "gp" + strings.ToUpper(glName[:1]) + glName[1:]
}

// constName strips the "GL_" prefix from a GL enum name.
func constName(glEnum string) string {
	if strings.HasPrefix(glEnum, "GL_") {
		return glEnum[3:]
	}
	return glEnum
}

// safeParamName escapes Go keyword and built-in name conflicts.
func safeParamName(name string) string {
	switch name {
	case "type":
		return "xtype"
	case "func":
		return "xfunc"
	case "range":
		return "xrange"
	case "map":
		return "xmap"
	case "string":
		return "xstring"
	case "near":
		return "zNear"
	case "far":
		return "zFar"
	case "len":
		return "xlen"
	case "cap":
		return "xcap"
	case "new":
		return "xnew"
	}
	return name
}

// mapCType converts a bare C type string (after XML-tag stripping) to a Go type.
// Returns ("", false) for void return types.
// The second return value is true when the Go type is "unsafe.Pointer".
func mapCType(c string) (goType string, needsUnsafe bool) {
	c = strings.TrimSpace(c)

	// Plain void return.
	if c == "void" || c == "" {
		return "", false
	}

	// Remove const qualifiers for structural analysis.
	noConst := strings.ReplaceAll(c, "const", "")
	noConst = strings.TrimSpace(noConst)

	// Count pointer stars.
	stars := strings.Count(noConst, "*")

	// Get base type name (no stars, no whitespace).
	base := strings.ReplaceAll(noConst, "*", "")
	base = strings.TrimSpace(base)

	// void* → unsafe.Pointer at any depth.
	if base == "void" || base == "GLvoid" {
		return "unsafe.Pointer", true
	}

	goBase := baseGoType(base)

	// GLboolean* → *uint8 rather than *bool: avoids bool-array pitfalls in
	// the purego ABI and matches how go-gl handles it.
	if base == "GLboolean" && stars > 0 {
		goBase = "uint8"
	}

	switch stars {
	case 0:
		return goBase, false
	case 1:
		return "*" + goBase, false
	default:
		return strings.Repeat("*", stars) + goBase, false
	}
}

// baseGoType maps a bare C GL type name to the corresponding Go type.
func baseGoType(c string) string {
	switch c {
	// ── unsigned 32-bit ──────────────────────────────────────────────────────
	case "GLenum", "GLbitfield", "GLuint", "GLhandleARB", "GLuint64EXT":
		return "uint32"
	// ── unsigned 64-bit ──────────────────────────────────────────────────────
	case "GLuint64":
		return "uint64"
	// ── unsigned 16-bit ──────────────────────────────────────────────────────
	case "GLushort", "GLhalf", "GLhalfARB", "GLhalfNV":
		return "uint16"
	// ── unsigned 8-bit / chars ───────────────────────────────────────────────
	case "GLubyte", "GLchar", "GLcharARB":
		return "uint8"
	// ── signed 32-bit ────────────────────────────────────────────────────────
	case "GLint", "GLsizei", "GLfixed", "GLclampx":
		return "int32"
	// ── signed 64-bit ────────────────────────────────────────────────────────
	case "GLint64", "GLint64EXT":
		return "int64"
	// ── signed 16-bit ────────────────────────────────────────────────────────
	case "GLshort":
		return "int16"
	// ── signed 8-bit ─────────────────────────────────────────────────────────
	case "GLbyte":
		return "int8"
	// ── pointer-sized ────────────────────────────────────────────────────────
	case "GLintptr", "GLsizeiptr", "GLintptrARB", "GLsizeiptrARB":
		return "int"
	// ── float ────────────────────────────────────────────────────────────────
	case "GLfloat", "GLclampf":
		return "float32"
	case "GLdouble", "GLclampd":
		return "float64"
	// ── boolean ──────────────────────────────────────────────────────────────
	case "GLboolean":
		return "bool" // pointer case handled in mapCType before calling here
	// ── opaque sync handle ───────────────────────────────────────────────────
	case "GLsync":
		return "uintptr"
	default:
		// Unknown: fall back to uint32 (works for most unknown enums/handles).
		return "uint32"
	}
}
