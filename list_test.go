package main

import (
	"testing"

	"github.com/rivo/tview"
)

func TestNormalizeTabClick(t *testing.T) {
	if got := normalizeTabClick(tview.MouseLeftDoubleClick); got != tview.MouseLeftClick {
		t.Fatalf("double click was not normalized: got %v", got)
	}
	if got := normalizeTabClick(tview.MouseLeftClick); got != tview.MouseLeftClick {
		t.Fatalf("single click changed: got %v", got)
	}
}
