package parser

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// BBox is the 2D bounding box of all XY moves in a G-code file.
type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// Width returns the bounding box width.
func (b BBox) Width() float64 { return b.MaxX - b.MinX }

// Height returns the bounding box height.
func (b BBox) Height() float64 { return b.MaxY - b.MinY }

// ParseBBox reads G-code from r and returns the XY bounding box.
func ParseBBox(r io.Reader) (BBox, error) {
	var (
		x, y     float64
		absolute = true // G90 is default
		first    = true
		box      BBox
	)

	update := func() {
		if first {
			box = BBox{x, y, x, y}
			first = false
			return
		}
		if x < box.MinX {
			box.MinX = x
		}
		if x > box.MaxX {
			box.MaxX = x
		}
		if y < box.MinY {
			box.MinY = y
		}
		if y > box.MaxY {
			box.MaxY = y
		}
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == ';' || line[0] == '(' {
			continue
		}
		// Strip inline comments
		if i := strings.Index(line, ";"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if i := strings.Index(line, "("); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}

		words := strings.Fields(strings.ToUpper(line))
		if len(words) == 0 {
			continue
		}

		cmd := words[0]

		// Scan all words for mode changes (handles multi-code lines like "G28 G91 Z0")
		for _, w := range words {
			switch w {
			case "G90":
				absolute = true
			case "G91":
				absolute = false
			}
		}

		switch cmd {
		case "G90", "G91":
			// already handled above
		case "G0", "G00", "G1", "G01":
			nx, ny := x, y
			hasXY := false
			for _, w := range words[1:] {
				if len(w) < 2 {
					continue
				}
				val, err := strconv.ParseFloat(w[1:], 64)
				if err != nil {
					continue
				}
				switch w[0] {
				case 'X':
					if absolute {
						nx = val
					} else {
						nx = x + val
					}
					hasXY = true
				case 'Y':
					if absolute {
						ny = val
					} else {
						ny = y + val
					}
					hasXY = true
				}
			}
			x, y = nx, ny
			if hasXY {
				update()
			}
		default:
			// Modal move: line starts with a coordinate axis (e.g. "X131.9 F200")
			if isCoordAxis(cmd[0]) {
				nx, ny := x, y
				hasXY := false
				for _, w := range words {
					if len(w) < 2 {
						continue
					}
					val, err := strconv.ParseFloat(w[1:], 64)
					if err != nil {
						continue
					}
					switch w[0] {
					case 'X':
						if absolute {
							nx = val
						} else {
							nx = x + val
						}
						hasXY = true
					case 'Y':
						if absolute {
							ny = val
						} else {
							ny = y + val
						}
						hasXY = true
					}
				}
				x, y = nx, ny
				if hasXY {
					update()
				}
			}
		case "G2", "G02", "G3", "G03":
			// Note: modal arc continuations are not handled (rare in practice)
			clockwise := cmd == "G2" || cmd == "G02"
			nx, ny := x, y
			var ix, iy float64
			for _, w := range words[1:] {
				if len(w) < 2 {
					continue
				}
				val, err := strconv.ParseFloat(w[1:], 64)
				if err != nil {
					continue
				}
				switch w[0] {
				case 'X':
					if absolute {
						nx = val
					} else {
						nx = x + val
					}
				case 'Y':
					if absolute {
						ny = val
					} else {
						ny = y + val
					}
				case 'I':
					ix = val
				case 'J':
					iy = val
				}
			}
			// Sample 36 points along the arc to approximate bounding box
			cx, cy := x+ix, y+iy
			r := math.Hypot(x-cx, y-cy)
			startAngle := math.Atan2(y-cy, x-cx)
			endAngle := math.Atan2(ny-cy, nx-cx)
			sweep := endAngle - startAngle
			if clockwise {
				if sweep > 0 {
					sweep -= 2 * math.Pi
				}
			} else {
				if sweep < 0 {
					sweep += 2 * math.Pi
				}
			}
			const steps = 36
			for i := 0; i <= steps; i++ {
				angle := startAngle + sweep*float64(i)/steps
				px := cx + r*math.Cos(angle)
				py := cy + r*math.Sin(angle)
				ox, oy := x, y
				x, y = px, py
				update()
				x, y = ox, oy
			}
			x, y = nx, ny
			update()
		}
	}

	if err := scanner.Err(); err != nil {
		return BBox{}, fmt.Errorf("scan: %w", err)
	}
	if first {
		return BBox{}, fmt.Errorf("no XY moves found")
	}

	return box, nil
}

// isCoordAxis reports whether b is a G-code coordinate axis letter (X, Y, Z, F, I, J, etc.)
// Used to detect modal moves like "X131.9 F200" that have no explicit G command.
func isCoordAxis(b byte) bool {
	switch b {
	case 'X', 'Y', 'Z', 'F', 'I', 'J', 'K', 'A', 'B', 'C':
		return true
	}
	return false
}

// Piece represents a G-code file with its computed bounding box.
type Piece struct {
	File string
	Box  BBox
}

// Parse parses multiple G-code files and returns their bounding boxes.
func Parse(files []string) ([]Piece, error) {
	pieces := make([]Piece, 0, len(files))
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f, err)
		}
		box, err := ParseBBox(fh)
		fh.Close()
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		pieces = append(pieces, Piece{File: f, Box: box})
	}
	return pieces, nil
}
