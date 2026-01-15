// aiken2go generates Go code from Aiken's plutus.json (CIP-0057 Plutus Blueprint).
//
// Usage:
//
//	aiken2go plutus.json -o contracts.go
//	aiken2go plutus.json -t plutus-trace.json -o contracts.go -p mypackage
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pgrange/aiken_to_go/pkg/blueprint"
)

func main() {
	var (
		outfile     string
		tracedFile  string
		packageName string
	)

	flag.StringVar(&outfile, "o", "", "Output file path (required)")
	flag.StringVar(&outfile, "outfile", "", "Output file path (required)")
	flag.StringVar(&tracedFile, "t", "", "Traced blueprint file (optional)")
	flag.StringVar(&tracedFile, "traced-blueprint", "", "Traced blueprint file (optional)")
	flag.StringVar(&packageName, "p", "contracts", "Go package name")
	flag.StringVar(&packageName, "package", "contracts", "Go package name")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <plutus.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Go code from Aiken's CIP-0057 Plutus Blueprint.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s plutus.json -o contracts.go\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s plutus.json -t plutus-trace.json -o contracts.go -p mypackage\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: plutus.json file is required")
		flag.Usage()
		os.Exit(1)
	}

	if outfile == "" {
		fmt.Fprintln(os.Stderr, "Error: output file (-o) is required")
		flag.Usage()
		os.Exit(1)
	}

	infile := flag.Arg(0)

	// Load blueprints
	bp, traceBp, err := blueprint.LoadBlueprintWithTrace(infile, tracedFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading blueprint: %v\n", err)
		os.Exit(1)
	}

	// Generate code
	gen := blueprint.NewGenerator(bp, traceBp, blueprint.GeneratorOptions{
		PackageName: packageName,
		WithTrace:   tracedFile != "",
	})

	code, err := gen.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating code: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if err := os.WriteFile(outfile, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s from %s\n", outfile, infile)
}
