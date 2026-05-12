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

// In GL 3.3 core, VAO/FBO/GenerateMipmap/GetStringi are all required by the
// spec — there are no optional extras beyond glClearDepthf (GL 4.1+).
func TestInitWithProcAddrFunc_ClearDepthfOptional(t *testing.T) {
	var sentinel uint8
	fakeAddr := unsafe.Pointer(&sentinel)

	err := InitWithProcAddrFunc(func(name string) unsafe.Pointer {
		if name == "glClearDepthf" {
			return nil // optional in 3.3 — must not cause error
		}
		return fakeAddr
	})
	if err != nil {
		t.Fatalf("glClearDepthf should be optional in GL 3.3, got error: %v", err)
	}
}
