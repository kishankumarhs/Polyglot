package main

import (
	"strings"
	"testing"
	"unsafe"
)

func TestCreateRecoversFromPanic(t *testing.T) {
	testPanicCreate = true
	defer func() { testPanicCreate = false }()

	h := logger_create_v1(nil)
	if h != nil {
		t.Fatal("expected nil handle after panic")
	}
	msg := goString(logger_last_error(nil))
	if !strings.Contains(msg, "panic") {
		t.Fatalf("expected panic error, got %q", msg)
	}
	_ = unsafe.Sizeof(h)
}
