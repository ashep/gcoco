package combiner_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ashep/gcoco/combiner"
	"github.com/ashep/gcoco/packer"
	"github.com/ashep/gcoco/parser"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := dir + "/" + name
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestWrite_NonZeroMinXY verifies that pieces whose G-code doesn't start at (0,0)
// are correctly placed — the double-MinX-subtraction regression test.
func TestWrite_NonZeroMinXY(t *testing.T) {
	dir := t.TempDir()
	// Piece moves from X=0.5,Y=0.5 to X=10.5,Y=10.5 → bbox MinX=0.5 MinY=0.5
	src := writeTempFile(t, dir, "b.gcode", `G21
G90
G0 X0.5 Y0.5
G1 X10.5 Y10.5 F300
M2
`)
	// Packer places bbox at working-area (0,0): Offset = f.x - MinX = 0 - 0.5 = -0.5
	batch := packer.Batch{
		{
			Piece:  parser.Piece{File: src, Box: parser.BBox{MinX: 0.5, MinY: 0.5, MaxX: 10.5, MaxY: 10.5}},
			Offset: packer.Offset{X: -0.5, Y: -0.5},
		},
	}

	outDir := t.TempDir()
	if err := combiner.Write([]packer.Batch{batch}, 3.0, outDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outDir + "/combined_1.gcode")
	out := string(data)

	// Rapid should go to working-area bounding box corner: Offset + MinX = -0.5 + 0.5 = 0.0
	if !strings.Contains(out, "G0 X0.000 Y0.000") {
		t.Errorf("rapid should go to working-area bbox start (0,0), got:\n%s", out)
	}
	// G0 X0.5 + (-0.5) = 0.0; G1 X10.5 + (-0.5) = 10.0
	if !strings.Contains(out, "G0 X0.000 Y0.000") {
		t.Errorf("G0 X0.5 should transform to X0.000, got:\n%s", out)
	}
	if !strings.Contains(out, "G1 X10.000 Y10.000") {
		t.Errorf("G1 X10.5 should transform to X10.000, got:\n%s", out)
	}
	// No negative coordinates anywhere
	if strings.Contains(out, "X-") || strings.Contains(out, "Y-") {
		t.Errorf("output contains negative coordinates:\n%s", out)
	}
}

func TestWrite_SinglePiece(t *testing.T) {
	dir := t.TempDir()
	src := writeTempFile(t, dir, "a.gcode", `G21
G90
G0 X0 Y0
G1 X10 Y10 F500
M2
`)
	batch := packer.Batch{
		{
			Piece:  parser.Piece{File: src, Box: parser.BBox{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}},
			Offset: packer.Offset{X: 5, Y: 8},
		},
	}

	outDir := t.TempDir()
	if err := combiner.Write([]packer.Batch{batch}, 3.0, outDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outDir + "/combined_1.gcode")
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, "G0 X5.000 Y8.000") {
		t.Errorf("expected transformed origin move, got:\n%s", out)
	}
	if !strings.Contains(out, "G1 X15.000 Y18.000") {
		t.Errorf("expected transformed G1 move (10+5=15, 10+8=18), got:\n%s", out)
	}
	if strings.Count(out, "M2") > 1 {
		t.Errorf("M2 should appear only once at the end, got:\n%s", out)
	}
	if !strings.Contains(out, "G0 Z3.000") {
		t.Errorf("expected safe-Z lift, got:\n%s", out)
	}
}
