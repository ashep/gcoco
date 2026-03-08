package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ashep/gcoco/combiner"
	"github.com/ashep/gcoco/packer"
	"github.com/ashep/gcoco/parser"
)

func main() {
	var (
		width   = flag.Float64("width", 0, "Working area width in mm (required)")
		height  = flag.Float64("height", 0, "Working area height in mm (required)")
		safeZ   = flag.Float64("safe-z", 5.0, "Safe Z height for rapid moves between pieces")
		spacing = flag.Float64("spacing", 1.0, "Minimum spacing between pieces in mm")
		outDir  = flag.String("output-dir", ".", "Directory for output files")
	)
	flag.Parse()

	if *width <= 0 || *height <= 0 {
		fmt.Fprintln(os.Stderr, "error: --width and --height are required and must be > 0")
		flag.Usage()
		os.Exit(1)
	}

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one G-code file is required")
		flag.Usage()
		os.Exit(1)
	}

	pieces, err := parser.Parse(files)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	batches := packer.Pack(pieces, *width, *height, *spacing)

	fmt.Printf("Packing %d piece(s) into %d batch(es)...\n", len(pieces), len(batches))
	for i, batch := range batches {
		fmt.Printf("  Batch %d: %d piece(s)\n", i+1, len(batch))
		for _, pp := range batch {
			fmt.Printf("    %s -> X=%.2f Y=%.2f\n", pp.File, pp.Offset.X, pp.Offset.Y)
		}
	}

	if err := combiner.Write(batches, *safeZ, *outDir); err != nil {
		log.Fatalf("write error: %v", err)
	}

	fmt.Printf("Done. Output written to %s/\n", *outDir)
}
