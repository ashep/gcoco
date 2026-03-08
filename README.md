# gcoco — G-code Combiner

Combines and nests multiple G-code files into a single output file, automatically arranging pieces within your
machine's working area to minimise material waste. Useful when your CAM software (Fusion 360, FreeCAD, etc.)
generates one file per part and you want to cut them all in one job.

Designed for Grbl and LinuxCNC machines. Handles standard G-code from common CAM tools.

## How it works

Each input file is assumed to have its own origin at (0, 0, 0). gcoco:

1. Parses each file to compute its XY bounding box
2. Arranges all pieces within the specified working area using bin-packing (largest pieces first, minimising wasted
   space)
3. If pieces don't all fit, splits them across multiple output files
4. Writes combined output files with all coordinates translated to their assigned positions

Between pieces the tool lifts to a safe Z height, rapids to the next piece's position, then continues.

## Installation

```bash
go install github.com/ashep/gcoco@latest
```

Or build from source:

```bash
git clone https://github.com/ashep/gcoco
cd gcoco
go build -o gcoco .
```

## Usage

```
gcoco [flags] file1.gcode file2.gcode ...
```

### Flags

| Flag           | Default  | Description                                  |
|----------------|----------|----------------------------------------------|
| `--width`      | required | Working area width in mm                     |
| `--height`     | required | Working area height in mm                    |
| `--safe-z`     | `5`      | Z height for rapid moves between pieces (mm) |
| `--spacing`    | `1`      | Minimum gap between pieces (mm)              |
| `--output-dir` | `.`      | Directory for output files                   |

### Example

```bash
gcoco --width 400 --height 300 --safe-z 5 --spacing 3 --output-dir ./out *.nc
```

Output:

```
Packing 4 piece(s) into 1 batch(es)...
  Batch 1: 4 piece(s)
    Front.nc  -> X=0.00  Y=0.00
    Back.nc   -> X=0.00  Y=76.80
    Left.nc   -> X=141.00 Y=0.00
    Right.nc  -> X=141.00 Y=76.80
Done. Output written to ./out/
```

Output files are named `combined_1.gcode`, `combined_2.gcode`, etc. — one per batch.

## Output file format

Each combined file:

- Sets absolute mode (`G90`) and mm units (`G21`)
- Lifts to safe Z before the first piece and between pieces
- Returns to origin (`G0 X0 Y0`) and ends with `M2` at the end

The following codes are stripped from input files and handled at the combined file level: `G21`, `G90`, `G91`, `G28`,
`M2`, `M30`.

## Requirements

- Input files must use mm units (`G21`)
- Input files must use absolute coordinates (`G90`) — the tool assumes this by default, matching standard CAM output
- Compatible with Grbl and LinuxCNC flavoured G-code
