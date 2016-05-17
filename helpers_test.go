package main

import (
	"testing"
)

func Test_ConcurrentSet(t *testing.T) {
	s := NewConcurrentSet()
	if s.Empty() != true || s.Size() != 0 {
		t.Error("set should be empty after initialized")
	}
	s.Put("Key")
	if s.Empty() != false || s.Size() != 1 {
		t.Error("set should not be empty after put")
	}
	s.Put("Key")
	if s.Size() != 1 {
		t.Error("set should handler duplicated items")
	}
	s.Delete("Key")
	if s.Size() != 0 {
		t.Error("set should be empty after delete")
	}
	s.Delete("Key")
}
