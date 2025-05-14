package main

import (
	"flag"
	"fmt"
	"os"

	"limon/parser"
)

func main() {
	// Parse command-line flags similar to the original lemon tool
	var (
		baseFlagPtr = flag.Bool("b", false, "Show only the basis for each parser state")
		noCompressFlagPtr = flag.Bool("c", false, "Do not compress action tables")
		outputDirPtr = flag.String("d", "", "Output directory")
		showHelpPtr = flag.Bool("?", false, "Show help")
		showVersionPtr = flag.Bool("x", false, "Show version")
	)

	flag.Parse()

	if *showHelpPtr {
		fmt.Println("Lemon (Go) LALR(1) Parser Generator")
		flag.PrintDefaults()
		return
	}

	if *showVersionPtr {
		fmt.Println("Lemon (Go) Parser Generator Version 0.1")
		return
	}

	// Check if a grammar file was specified
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Error: No grammar file specified")
		fmt.Println("Usage: limon [options] grammar-file")
		os.Exit(1)
	}

	grammarFile := args[0]
	
	// Create a new parser and process the grammar file
	p := parser.New()
	p.Basisflag = *baseFlagPtr
	p.NoResort = *noCompressFlagPtr
	p.Outdir = *outputDirPtr
	err := p.GenerateParser(grammarFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}