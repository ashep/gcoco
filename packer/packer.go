package packer

import (
	"sort"

	"github.com/ashep/gcoco/parser"
)

// Offset is a 2D placement offset within the working area.
type Offset struct{ X, Y float64 }

// PlacedPiece is a piece with its assigned placement offset.
type PlacedPiece struct {
	parser.Piece
	Offset Offset
}

// Batch is a set of pieces that fit in one working area.
type Batch []PlacedPiece

// rect is a free rectangle in the working area.
type rect struct{ x, y, w, h float64 }

// Pack arranges pieces into batches using the Maximal Rectangles algorithm (BSSF).
// spacing is added between pieces.
func Pack(pieces []parser.Piece, width, height, spacing float64) []Batch {
	// Sort by area descending for better packing
	sorted := make([]parser.Piece, len(pieces))
	copy(sorted, pieces)
	sort.Slice(sorted, func(i, j int) bool {
		ai := sorted[i].Box.Width() * sorted[i].Box.Height()
		aj := sorted[j].Box.Width() * sorted[j].Box.Height()
		return ai > aj
	})

	var batches []Batch
	remaining := sorted

	for len(remaining) > 0 {
		batch, leftover := packBatch(remaining, width, height, spacing)
		batches = append(batches, batch)
		remaining = leftover
	}
	return batches
}

// packBatch places as many pieces as possible into one working area using MAXRECTS.
// Returns the placed batch and any pieces that didn't fit.
func packBatch(pieces []parser.Piece, width, height, spacing float64) (Batch, []parser.Piece) {
	free := []rect{{0, 0, width, height}}
	var batch Batch
	var leftover []parser.Piece

	for _, p := range pieces {
		pw := p.Box.Width() + spacing
		ph := p.Box.Height() + spacing

		idx := bestFit(free, pw, ph)
		if idx < 0 {
			leftover = append(leftover, p)
			continue
		}

		f := free[idx]
		ox := f.x - p.Box.MinX
		oy := f.y - p.Box.MinY
		batch = append(batch, PlacedPiece{Piece: p, Offset: Offset{ox, oy}})

		// Placed rectangle (including spacing buffer)
		placed := rect{f.x, f.y, pw, ph}

		// MAXRECTS: split every free rectangle around the placed piece,
		// then remove any rect fully contained within another.
		var next []rect
		for _, fr := range free {
			next = append(next, splitAround(fr, placed)...)
		}
		free = removeContained(next)
	}

	return batch, leftover
}

// splitAround splits fr around the placed rectangle, returning up to 4 maximal rects.
// If placed doesn't intersect fr, fr is returned unchanged.
func splitAround(fr, placed rect) []rect {
	// No intersection
	if placed.x >= fr.x+fr.w || placed.x+placed.w <= fr.x ||
		placed.y >= fr.y+fr.h || placed.y+placed.h <= fr.y {
		return []rect{fr}
	}

	var out []rect

	// Left strip
	if placed.x > fr.x {
		out = append(out, rect{fr.x, fr.y, placed.x - fr.x, fr.h})
	}
	// Right strip
	if placed.x+placed.w < fr.x+fr.w {
		out = append(out, rect{placed.x + placed.w, fr.y, fr.x + fr.w - placed.x - placed.w, fr.h})
	}
	// Bottom strip
	if placed.y > fr.y {
		out = append(out, rect{fr.x, fr.y, fr.w, placed.y - fr.y})
	}
	// Top strip
	if placed.y+placed.h < fr.y+fr.h {
		out = append(out, rect{fr.x, placed.y + placed.h, fr.w, fr.y + fr.h - placed.y - placed.h})
	}

	return out
}

// removeContained removes any rect fully contained within another rect in the slice.
func removeContained(rects []rect) []rect {
	var out []rect
	for i, r := range rects {
		contained := false
		for j, other := range rects {
			if i != j && rectContains(other, r) {
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, r)
		}
	}
	return out
}

// rectContains reports whether outer fully contains inner.
func rectContains(outer, inner rect) bool {
	return outer.x <= inner.x &&
		outer.y <= inner.y &&
		outer.x+outer.w >= inner.x+inner.w &&
		outer.y+outer.h >= inner.y+inner.h
}

// bestFit finds the free rectangle with the best short-side fit for pw×ph.
// Returns -1 if no rectangle fits.
func bestFit(free []rect, pw, ph float64) int {
	best := -1
	bestScore := 1e18
	for i, f := range free {
		if pw <= f.w && ph <= f.h {
			score := min64(f.w-pw, f.h-ph)
			if score < bestScore {
				bestScore = score
				best = i
			}
		}
	}
	return best
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
