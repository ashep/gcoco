package packer_test

import (
	"testing"

	"github.com/ashep/gcoco/packer"
	"github.com/ashep/gcoco/parser"
)

func TestPack_SinglePiece(t *testing.T) {
	pieces := []parser.Piece{
		{File: "a.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 50, MaxY: 30}},
	}
	batches := packer.Pack(pieces, 100, 100, 0)
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0]) != 1 {
		t.Fatalf("expected 1 piece in batch, got %d", len(batches[0]))
	}
	p := batches[0][0]
	if p.Offset.X != 0 || p.Offset.Y != 0 {
		t.Errorf("expected offset (0,0), got (%v,%v)", p.Offset.X, p.Offset.Y)
	}
}

func TestPack_TwoPiecesSideBySide(t *testing.T) {
	pieces := []parser.Piece{
		{File: "a.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 40, MaxY: 30}},
		{File: "b.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 40, MaxY: 30}},
	}
	batches := packer.Pack(pieces, 100, 50, 0)
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0]) != 2 {
		t.Fatalf("expected 2 pieces in batch, got %d", len(batches[0]))
	}
}

func TestPack_Overflow_CreatesBatch(t *testing.T) {
	pieces := []parser.Piece{
		{File: "a.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 60, MaxY: 60}},
		{File: "b.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 60, MaxY: 60}},
	}
	// Only one piece fits per batch
	batches := packer.Pack(pieces, 70, 70, 0)
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}
}

func TestPack_SpacingApplied(t *testing.T) {
	// Two pieces of width 50 each with spacing 5, working area 110 wide
	// With spacing 5: need 50+5+50=105 <= 110, should still fit in one batch
	pieces := []parser.Piece{
		{File: "a.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 50, MaxY: 50}},
		{File: "b.gcode", Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 50, MaxY: 50}},
	}
	batches := packer.Pack(pieces, 110, 60, 5)
	if len(batches) != 1 {
		t.Fatalf("with spacing 5 both pieces should fit in one batch, got %d batches", len(batches))
	}
}
