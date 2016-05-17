package main

import (
	"testing"
)

func Test_Pager(t *testing.T) {
	pager := NewPager(20, 1001)
	if pager.Page() != 51 {
		t.Error("page sould be 51")
	}
	for i := 0; i < 6; i++ {
		pager.CurrentPage = i
		if pager.Offset() != 20*i {
			t.Error("offset should be %d in page %d", 20*i, i)
		}
		if pager.IsFirstVisible() {
			t.Error("first page should not be visible")
		}
		if !pager.IsEndVisible() {
			t.Error("end page should be visible")
		}
	}
	for i := 6; i < 45; i++ {
		pager.CurrentPage = i
		if pager.Offset() != 20*i {
			t.Error("offset should be %d in page %d", 20*i, i)
		}
		if !pager.IsFirstVisible() {
			t.Error("first page should be visible")
		}
		if !pager.IsEndVisible() {
			t.Error("end page should be visible")
		}
	}
	for i := 46; i < 51; i++ {
		pager.CurrentPage = i
		if pager.Offset() != 20*i {
			t.Error("offset should be %d in page %d", 20*i, i)
		}
		if !pager.IsFirstVisible() {
			t.Error("first page should be visible")
		}
		if pager.IsEndVisible() {
			t.Error("end page should not be visible")
		}
	}
}
