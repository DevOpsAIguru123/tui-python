package tux

import (
	"strings"
	"testing"
)

// TestArtIsRectangular guards against accidentally pasting a row with the
// wrong width — the half-block renderer assumes uniform row length.
func TestArtIsRectangular(t *testing.T) {
	if len(art) == 0 {
		t.Fatal("art is empty")
	}
	w := len(art[0])
	for i, row := range art {
		if len(row) != w {
			t.Errorf("row %d has width %d, want %d", i, len(row), w)
		}
	}
}

// TestRenderHasExpectedRowCount verifies the half-block packing: every two
// source rows collapse to one terminal row.
func TestRenderHasExpectedRowCount(t *testing.T) {
	got := strings.Count(Render(), "\n") + 1
	want := (len(art) + 1) / 2
	if got != want {
		t.Errorf("Render produced %d rows, want %d", got, want)
	}
}

// TestRenderContainsHalfBlocks ensures we actually emit styled half-blocks
// rather than degrading to plain spaces (which would mean the palette lookup
// silently failed).
func TestRenderContainsHalfBlocks(t *testing.T) {
	out := Render()
	if !strings.ContainsRune(out, '▀') {
		t.Error("Render output missing upper half-block")
	}
}
