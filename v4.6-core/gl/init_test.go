//go:build windows

package gl

import (
	"strings"
	"testing"
	"unsafe"

)

func TestInitWithProcAddrFunc_NilResolver(t *testing.T) {
	err := InitWithProcAddrFunc(func(string) unsafe.Pointer { return nil })
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

	err := InitWithProcAddrFunc(func(string) unsafe.Pointer { return fakeAddr })
	if err != nil {
		t.Fatalf("expected nil error when all functions present, got: %v", err)
	}
}
