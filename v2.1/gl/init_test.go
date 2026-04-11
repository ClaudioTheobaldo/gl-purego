//go:build windows

package gl_test

import (
	"strings"
	"testing"
	"unsafe"

	gl "github.com/ClaudioTheobaldo/gl-purego/v2.1/gl"
)

// TestInitWithProcAddrFunc_NilResolver verifies that InitWithProcAddrFunc
// returns a non-nil error listing every required GL entry point when the
// resolver always returns nil (no OpenGL context available).
func TestInitWithProcAddrFunc_NilResolver(t *testing.T) {
	err := gl.InitWithProcAddrFunc(func(string) unsafe.Pointer { return nil })
	if err == nil {
		t.Fatal("expected error when all required functions are missing, got nil")
	}

	msg := err.Error()
	if !strings.HasPrefix(msg, "gl: ") {
		t.Errorf("error message format unexpected: %q", msg)
	}
	// The error message must mention how many functions were not found.
	if !strings.Contains(msg, "required functions not found") {
		t.Errorf("error message does not mention missing functions: %q", msg)
	}
	t.Logf("nil-resolver error: %s", msg)
}

// TestInitWithProcAddrFunc_OptionalSkipped verifies that InitWithProcAddrFunc
// succeeds when only the optional (required=false) functions are missing —
// i.e., when all *required* functions are "found" and only the optional ones
// return nil.
//
// We distinguish required from optional by the known optional set:
// VAOs, FBOs, glGenerateMipmap, glGetStringi, glClearDepthf.
func TestInitWithProcAddrFunc_OptionalSkipped(t *testing.T) {
	optional := map[string]bool{
		"glGenVertexArrays":         true,
		"glDeleteVertexArrays":       true,
		"glBindVertexArray":          true,
		"glIsVertexArray":            true,
		"glGenFramebuffers":          true,
		"glDeleteFramebuffers":       true,
		"glBindFramebuffer":          true,
		"glCheckFramebufferStatus":   true,
		"glFramebufferTexture2D":     true,
		"glFramebufferRenderbuffer":  true,
		"glGenRenderbuffers":         true,
		"glDeleteRenderbuffers":      true,
		"glBindRenderbuffer":         true,
		"glRenderbufferStorage":      true,
		"glGenerateMipmap":           true,
		"glGetStringi":               true,
		"glClearDepthf":              true,
	}

	// Non-nil address of a real local variable — we never call the functions,
	// so it doesn't matter that the pointer doesn't point to real GL code.
	var sentinel uint8
	fakeAddr := unsafe.Pointer(&sentinel)

	err := gl.InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		if optional[name] {
			return nil // optional → not found, should not cause error
		}
		return fakeAddr // required → pretend found
	})
	if err != nil {
		t.Fatalf("expected nil error when only optional functions are missing, got: %v", err)
	}
}
