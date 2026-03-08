package combiner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ashep/gcoco/packer"
)

// Write writes one output G-code file per batch to outDir.
func Write(batches []packer.Batch, safeZ float64, outDir string) error {
	for i, batch := range batches {
		path := filepath.Join(outDir, fmt.Sprintf("combined_%d.gcode", i+1))
		if err := writeBatch(batch, safeZ, path, i+1, len(batches)); err != nil {
			return fmt.Errorf("write batch %d: %w", i+1, err)
		}
	}
	return nil
}

func writeBatch(batch packer.Batch, safeZ float64, path string, batchNum, total int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	fmt.Fprintf(w, "; gcoco combined file %d of %d\n", batchNum, total)
	fmt.Fprintf(w, "; generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(w, "G21")
	fmt.Fprintln(w, "G90")
	fmt.Fprintf(w, "G0 Z%.3f\n", safeZ)

	for j, pp := range batch {
		fmt.Fprintf(w, "\n; === piece: %s at X=%.3f Y=%.3f ===\n",
			filepath.Base(pp.File), pp.Offset.X, pp.Offset.Y)

		// Rapid to the working-area position of the piece's bounding box start
		fmt.Fprintf(w, "G0 X%.3f Y%.3f\n", pp.Offset.X+pp.Box.MinX, pp.Offset.Y+pp.Box.MinY)

		ox := pp.Offset.X
		oy := pp.Offset.Y
		if err := transformAndWrite(w, pp.File, ox, oy); err != nil {
			return fmt.Errorf("transform %s: %w", pp.File, err)
		}

		if j < len(batch)-1 {
			fmt.Fprintf(w, "G0 Z%.3f\n", safeZ)
		}
	}

	fmt.Fprintf(w, "\nG0 Z%.3f\n", safeZ)
	fmt.Fprintln(w, "G0 X0 Y0")
	fmt.Fprintln(w, "M2")

	return w.Flush()
}

// transformAndWrite reads a G-code file, offsets X/Y, and writes to w.
// Strips G21, G90, G91, M2, M30 preamble/end codes.
func transformAndWrite(w *bufio.Writer, file string, ox, oy float64) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	for scanner.Scan() {
		line := scanner.Text()
		out, skip := transformLine(line, ox, oy)
		if skip {
			continue
		}
		fmt.Fprintln(w, out)
	}
	return scanner.Err()
}

// isCoordAxis reports whether b is a G-code coordinate axis letter.
func isCoordAxis(b byte) bool {
	switch b {
	case 'X', 'Y', 'Z', 'F', 'I', 'J', 'K', 'A', 'B', 'C':
		return true
	}
	return false
}

var skipCmds = map[string]bool{
	"G20": true, "G21": true,
	"G90": true, "G91": true,
	"G28": true, // homing — machine coordinates, not WCS; strip from combined output
	"M2": true, "M02": true,
	"M30": true,
}

// transformLine offsets X/Y coordinates in move commands.
// Returns (transformed line, skip).
func transformLine(line string, ox, oy float64) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, false
	}

	// Preserve comments as-is
	if trimmed[0] == ';' || trimmed[0] == '(' {
		return line, false
	}

	// Strip inline comment for processing
	comment := ""
	if i := strings.Index(trimmed, ";"); i >= 0 {
		comment = " " + trimmed[i:]
		trimmed = strings.TrimSpace(trimmed[:i])
	}

	words := strings.Fields(strings.ToUpper(trimmed))
	if len(words) == 0 {
		return line, false
	}

	cmd := words[0]

	// Skip preamble/end codes
	if skipCmds[cmd] {
		return "", true
	}

	// Detect modal move: line starts with a coordinate axis letter (e.g. "X131.9 F200")
	isModal := len(cmd) > 0 && isCoordAxis(cmd[0])

	// Only transform explicit move commands or modal moves
	if !isModal {
		switch cmd {
		case "G0", "G00", "G1", "G01", "G2", "G02", "G3", "G03":
		default:
			return line, false
		}
	}

	// Re-parse and offset X/Y. For explicit moves, emit the command word first then params.
	// For modal moves, all words are coordinate parameters.
	origWords := strings.Fields(trimmed)
	result := make([]string, 0, len(origWords))
	startIdx := 0
	if !isModal {
		result = append(result, cmd) // emit command word as-is
		startIdx = 1
	}
	for _, w := range origWords[startIdx:] {
		if len(w) < 2 {
			result = append(result, w)
			continue
		}
		axis := strings.ToUpper(string(w[0]))
		val, err := strconv.ParseFloat(w[1:], 64)
		if err != nil {
			result = append(result, w)
			continue
		}
		switch axis {
		case "X":
			result = append(result, fmt.Sprintf("X%.3f", val+ox))
		case "Y":
			result = append(result, fmt.Sprintf("Y%.3f", val+oy))
		default:
			result = append(result, fmt.Sprintf("%s%.3f", axis, val))
		}
	}
	return strings.Join(result, " ") + comment, false
}
