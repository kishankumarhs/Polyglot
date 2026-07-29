package main

import (
	"testing"
	"unsafe"
)

// Handles must stay unique while fewer than maxRetiredHandles have been closed,
// so a stale handle from a closed logger cannot alias a live one.
func TestHandlePoolKeepsRetiredAddressesUnique(t *testing.T) {
	var p handlePool
	seen := map[unsafe.Pointer]bool{}

	for i := 0; i < maxRetiredHandles; i++ {
		h := p.acquire()
		if h == nil {
			t.Fatal("acquire returned nil")
		}
		if seen[h] {
			t.Fatalf("handle address reused after %d retirements", i)
		}
		seen[h] = true
		p.release(h)
	}
}

// Past the retirement ceiling addresses are recycled, so a create/close loop
// does not grow memory without bound.
func TestHandlePoolRecyclesPastCeiling(t *testing.T) {
	var p handlePool
	allocated := map[unsafe.Pointer]bool{}

	for i := 0; i < maxRetiredHandles*3; i++ {
		h := p.acquire()
		if h == nil {
			t.Fatal("acquire returned nil")
		}
		allocated[h] = true
		p.release(h)
	}

	if len(allocated) > maxRetiredHandles+1 {
		t.Fatalf("allocated %d distinct handles, expected at most %d", len(allocated), maxRetiredHandles+1)
	}
	if got := len(p.retired) - p.head; got > 2*maxRetiredHandles {
		t.Fatalf("retired queue grew to %d entries", got)
	}
}
