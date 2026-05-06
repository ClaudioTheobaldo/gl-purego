//go:build windows

package gl_test

import (
	"strings"
	"testing"
	"unsafe"

	gl "github.com/ClaudioTheobaldo/gl-purego/v4.6-core/gl"
)

func TestInitWithProcAddrFunc_NilResolver(t *testing.T) {
	err := gl.InitWithProcAddrFunc(func(string) unsafe.Pointer { return nil })
	if err == nil {
		t.Fatal("expected error when all required functions are missing, got nil")
	}
	if !strings.Contains(err.Error(), "required functions not found") {
		t.Errorf("unexpected error format: %v", err)
	}
	t.Logf("nil-resolver error: %s", err)
}

func TestInitWithProcAddrFunc_AllPresent(t *testing.T) {
	var sentinel uint8
	fakeAddr := unsafe.Pointer(&sentinel)

	err := gl.InitWithProcAddrFunc(func(string) unsafe.Pointer { return fakeAddr })
	if err != nil {
		t.Fatalf("expected nil error when all functions present, got: %v", err)
	}
}
