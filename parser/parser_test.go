package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ashep/gcoco/parser"
)

func TestParseBBox_SimpleMoves(t *testing.T) {
	gcode := strings.NewReader(`
G21
G90
G0 X10 Y5
G1 X30 Y20 F500
G0 X0 Y0
`)
	box, err := parser.ParseBBox(gcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if box.MinX != 0 || box.MinY != 0 || box.MaxX != 30 || box.MaxY != 20 {
		t.Errorf("got %+v, want {0 0 30 20}", box)
	}
}

func TestParseBBox_RelativeMode(t *testing.T) {
	gcode := strings.NewReader(`
G90
G0 X10 Y10
G91
G1 X5 Y5
G1 X-3 Y2
`)
	box, err := parser.ParseBBox(gcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// positions visited: (10,10), (15,15), (12,17)
	if box.MinX != 10 || box.MinY != 10 || box.MaxX != 15 || box.MaxY != 17 {
		t.Errorf("got %+v, want {10 10 15 17}", box)
	}
}

func TestParseBBox_IgnoresZAndOtherCodes(t *testing.T) {
	gcode := strings.NewReader(`
G90
G0 X5 Y5 Z-1
M3 S1000
G1 X10 Y10 Z-2 F300
M5
M2
`)
	box, err := parser.ParseBBox(gcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if box.MinX != 5 || box.MinY != 5 || box.MaxX != 10 || box.MaxY != 10 {
		t.Errorf("got %+v, want {5 5 10 10}", box)
	}
}

func TestParseBBox_Arc(t *testing.T) {
	// Quarter-circle arc from (10,0) to (0,10), center at (0,0), radius 10
	// G3 = counter-clockwise
	gcode := strings.NewReader(`
G90
G0 X10 Y0
G3 X0 Y10 I-10 J0
`)
	box, err := parser.ParseBBox(gcode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Arc passes through approx (10,0)..(0,10), max Y~10, max X~10
	if box.MaxX < 9.9 || box.MaxY < 9.9 {
		t.Errorf("arc bounding box too small: %+v", box)
	}
	if box.MinX < -0.1 || box.MinY < -0.1 {
		t.Errorf("arc bounding box too large: %+v", box)
	}
}

func TestParse_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	f1 := dir + "/a.gcode"
	os.WriteFile(f1, []byte("G90\nG0 X20 Y15\nG1 X40 Y30\n"), 0644)

	f2 := dir + "/b.gcode"
	os.WriteFile(f2, []byte("G90\nG0 X0 Y0\nG1 X10 Y10\n"), 0644)

	pieces, err := parser.Parse([]string{f1, f2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pieces) != 2 {
		t.Fatalf("expected 2 pieces, got %d", len(pieces))
	}
	if pieces[0].Box.MaxX != 40 || pieces[0].Box.MaxY != 30 {
		t.Errorf("piece 0 box wrong: %+v", pieces[0].Box)
	}
	if pieces[1].Box.MaxX != 10 || pieces[1].Box.MaxY != 10 {
		t.Errorf("piece 1 box wrong: %+v", pieces[1].Box)
	}
}
