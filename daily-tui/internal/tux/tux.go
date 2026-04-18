// Package tux renders a small Tux pixel-art mascot using Unicode half-blocks.
//
// Each terminal cell holds two vertical "pixels" by combining the upper-half
// block (▀) — top half rendered as foreground, bottom half as background — and
// the lower-half block (▄) for cells where only the bottom pixel is colored.
// Fully transparent cells render as a literal space so the terminal background
// shows through cleanly at the silhouette edges.
package tux

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// art is the source pixel grid. Every row must be the same width. Characters
// map to colors via palette; '.' is transparent.
var art = []string{
	"...KKKK...",
	"..KWWWWK..",
	"..KWKKWK..",
	"..KKYYKK..",
	"..KKKKKK..",
	".KKWWWWKK.",
	".KWWWWWWK.",
	".KWWWWWWK.",
	".KWWWWWWK.",
	".KKWWWWKK.",
	"..KK..KK..",
	"..KK..KK..",
	".YYY..YYY.",
	".YYY..YYY.",
}

var palette = map[byte]lipgloss.Color{
	'K': lipgloss.Color("#111111"),
	'W': lipgloss.Color("#f5f5f5"),
	'Y': lipgloss.Color("#f7a53b"),
}

// Render returns the multi-line styled string for the Tux art. Each output row
// corresponds to two rows of the source grid, so the result is len(art)/2 lines
// tall and len(art[0]) columns wide.
func Render() string {
	var out strings.Builder
	for y := 0; y < len(art); y += 2 {
		top := art[y]
		var bot string
		if y+1 < len(art) {
			bot = art[y+1]
		}
		for x := 0; x < len(top); x++ {
			tch := top[x]
			var bch byte = '.'
			if x < len(bot) {
				bch = bot[x]
			}
			topT := tch == '.'
			botT := bch == '.'
			switch {
			case topT && botT:
				out.WriteByte(' ')
			case !topT && botT:
				style := lipgloss.NewStyle().Foreground(palette[tch])
				out.WriteString(style.Render("▀"))
			case topT && !botT:
				style := lipgloss.NewStyle().Foreground(palette[bch])
				out.WriteString(style.Render("▄"))
			default:
				style := lipgloss.NewStyle().
					Foreground(palette[tch]).
					Background(palette[bch])
				out.WriteString(style.Render("▀"))
			}
		}
		if y+2 < len(art) {
			out.WriteByte('\n')
		}
	}
	return out.String()
}
